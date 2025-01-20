package servicespec

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/go-yaml/yaml"
	"github.com/hashicorp/hcl/v2/hclsimple"
)

var one = uint64(1)
var knownUpdateConfigs = map[string]*composetypes.UpdateConfig{
	"parallel-one-at-a-time": {
		Parallelism: &one,
		Order:       "start-first",
	},
	"stop-old-first": {
		Order: "stop-first",
	},
}

func convertOneService(service ServiceSpec, isGlobal bool, compose *composetypes.Config, defaults Defaults, secrets Secrets) error {
	// most of the "not empty" checks carried out by HCL layer

	envs, err := convertEnvs(service, secrets)
	if err != nil {
		return err
	}

	labels := composetypes.Labels{}

	// try to tell the image's logger system to omit timestamps, since Docker adds those anyway
	// https://github.com/function61/gokit/blob/7397b370de1295275a4670bce87cc8f5f64e33fa/logex/helpers.go#L27
	one := "1"
	envs["LOGGER_SUPPRESS_TIMESTAMPS"] = &one

	if backup := service.Backup; backup != nil {
		labels["ubackup.command"] = backup.Command
	}

	volumes := []composetypes.ServiceVolumeConfig{}
	for _, bindMount := range service.BindMounts {
		bindConfig := func() *composetypes.ServiceVolumeBind {
			if bindMount.Propagation != "" {
				return &composetypes.ServiceVolumeBind{
					Propagation: bindMount.Propagation,
				}
			} else {
				return nil
			}
		}()

		volumes = append(volumes, composetypes.ServiceVolumeConfig{
			Type:     "bind",
			Source:   bindMount.Host,
			Target:   bindMount.Container,
			ReadOnly: bindMount.ReadOnly,
			Bind:     bindConfig,
		})
	}
	for _, pv := range service.PersistentVolumes {
		// things like requiring these dummy entries is the reason I wrote this
		// transpiler
		compose.Volumes[pv.Name] = composetypes.VolumeConfig{}

		volumes = append(volumes, composetypes.ServiceVolumeConfig{
			Type:   "volume",
			Source: pv.Name,
			Target: pv.Target,
		})
	}

	deployMode := "" // will default to "replicated"
	if isGlobal {
		deployMode = "global"
	}

	// forcing user explicitly to tell this because incorrect config is dangerous
	updateConfig := knownUpdateConfigs[service.HowToUpdate]
	if updateConfig == nil {
		return fmt.Errorf("unknown HowToUpdate: %s", service.HowToUpdate)
	}

	netMode := service.Net.Mode

	if netMode != "own-bridge-namespace" && netMode != "host" {
		return fmt.Errorf("unsupported network: %s (expecting own-bridge-namespace | host)", netMode)
	}

	ownNamespace := netMode == "own-bridge-namespace"
	netHost := netMode == "host"

	if countTrue(service.IngressPublic != nil, service.IngressBearer != nil, service.IngressSso != nil) > 1 {
		return errors.New("maximum of one ingress per service exceeded")
	}

	if ingress := service.IngressPublic; ingress != nil {
		labels["traefik.frontend.rule"] = ingress.Rule
		if ingress.Port != nil {
			labels["traefik.port"] = strconv.Itoa(*ingress.Port)
		}

		// requires explicit opt-in, so a key missing does not accidentally expose endpoints to public
		labels["edgerouter.auth"] = "public"
	}

	if ingress := service.IngressBearer; ingress != nil {
		labels["traefik.frontend.rule"] = ingress.Rule
		if ingress.Port != nil {
			labels["traefik.port"] = strconv.Itoa(*ingress.Port)
		}

		labels["edgerouter.auth"] = "bearer_token"
		labels["edgerouter.auth_bearer_token"] = ingress.Token
	}

	if ingress := service.IngressSso; ingress != nil {
		labels["traefik.frontend.rule"] = ingress.Rule
		if ingress.Port != nil {
			labels["traefik.port"] = strconv.Itoa(*ingress.Port)
		}

		labels["edgerouter.auth"] = "sso"
		labels["edgerouter.auth_sso.tenant"] = ingress.Tenant
		labels["edgerouter.auth_sso.users"] = strings.Join(ingress.Users, ",")
	}

	ramBytes := composetypes.UnitBytes(service.RamMb) * 1024 * 1024

	securityOpts := []string{}
	for _, item := range service.SecurityOpts {
		securityOpts = append(securityOpts, item.Value)
	}

	// https://docs.docker.com/engine/containers/start-containers-automatically/#use-a-restart-policy
	restartPolicy := func() string {
		if explicitPolicy := service.RestartPolicy; explicitPolicy != "" {
			return explicitPolicy
		} else { // no explicit defined => default to on-failure
			return "on-failure"
		}
	}()

	composeService := composetypes.ServiceConfig{
		Name: service.Name,
		Image: func() string {
			if strings.HasPrefix(service.Version, "sha256:") {
				return service.Image + "@" + service.Version // redis@sha256:...
			} else {
				return service.Image + ":" + service.Version // redis:latest
			}
		}(),
		Command:     service.Command,
		Environment: envs,
		Volumes:     volumes,
		Devices:     service.Devices,
		CapAdd:      service.Caps,
		Privileged:  service.Privileged,
		User:        service.User,
		Ports:       convertPorts(service, ownNamespace),
		Labels:      labels, // TODO: duplicated here. needed for Edgerouter when in host networking mode
		Restart:     restartPolicy,
		Deploy: composetypes.DeployConfig{
			Mode:         deployMode,
			Labels:       labels,
			Placement:    composetypes.Placement{},
			UpdateConfig: updateConfig,
			/* do not use as this is under deploy config
			RestartPolicy: &composetypes.RestartPolicy{
				Condition: "on-failure",
			},
			*/
			Resources: composetypes.Resources{
				Limits: &composetypes.ResourceLimit{
					MemoryBytes: ramBytes,
				},
			},
		},
		ShmSize: func() string {
			if service.ShmSizeMb > 0 {
				return fmt.Sprintf("%dM", service.ShmSizeMb)
			} else {
				return ""
			}
		}(),
		SecurityOpt: securityOpts,
		Networks:    map[string]*composetypes.ServiceNetworkConfig{},
	}

	if isGlobal && service.Replicas != nil {
		return errors.New("global services cannot have 'replicas' defined")
	}

	composeService.Deploy.Replicas = service.Replicas

	if service.PidHost {
		composeService.Pid = "host"
	}

	if netHost || ownNamespace {
		/*
			createNetworkConfigIfNotExists(compose, "host", composetypes.NetworkConfig{
				External: composetypes.External{
					Name: "host",
				},
			})

			// composeService.Networks["host"] = nil
		*/
		// for network mode host we shouldn't use the "networks:\n  host:" idiom and then define
		// on top level `networks:\n  host:\n    external:\n      name: "host"`
		//
		// because with docker-compose (no with Docker Swarm) this results in error:
		// "Network-scoped alias is supported only for containers in user defined networks"
		//  https://github.com/docker/compose/issues/3012
		//
		// however `network_mode: "host"` doesn't seem to work for Swarm: https://github.com/docker/compose/issues/3012#issuecomment-312414338
		if netHost {
			composeService.NetworkMode = "host"
		} else { // ownNamespace
			ip := net.ParseIP(service.Net.IP).To4()
			if ip == nil {
				return fmt.Errorf("when net.mode == '%s' need IP; empty given", netMode)
			}

			if ip[3] == 1 {
				return fmt.Errorf("IP address cannot be x.y.z.1, is reserved for gateway; got %s", ip.String())
			}

			// "10.61.3.2" => "10.61.3.1/24"
			subnet := net.IPv4(ip[0], ip[1], ip[2], 1).String() + "/24"

			createNetworkConfigIfNotExists(compose, "default", composetypes.NetworkConfig{
				Ipam: composetypes.IPAMConfig{
					Config: []*composetypes.IPAMPool{
						{
							Subnet: subnet,
						},
					},
				},
			})

			composeService.Networks["default"] = &composetypes.ServiceNetworkConfig{
				Ipv4Address: service.Net.IP,
			}
		}
	} else {
		createNetworkConfigIfNotExists(compose, "default", composetypes.NetworkConfig{
			External: composetypes.External{
				Name: defaults.DockerNetworkName,
			},
		})

		// not required, but better to be explicit
		composeService.Networks["default"] = nil
	}

	couldHavePlacementNodeHostname := !ownNamespace

	if service.PlacementNodeHostname != "" {
		if !couldHavePlacementNodeHostname {
			return errors.New("PlacementNodeHostname not supported when using new convention (own-namespace)")
		}

		composeService.Deploy.Placement.Constraints = []string{
			"node.hostname == " + service.PlacementNodeHostname,
		}
	} else {
		// prevent human errors
		if couldHavePlacementNodeHostname && len(service.PersistentVolumes) > 0 {
			return errors.New("persistent volumes defined but no placement hostname defined")
		}
	}

	/*
		there are stateful volumes that can be shared between concurrent containers (file uploads @ erotuomari.com)
		if len(service.PersistentVolumes) > 0 && service.HowToUpdate != "stop-old-first" {
			return errors.New("expecting stop-old-first when have PersistentVolumes")
		}
	*/

	if len(service.PersistentVolumes) > 0 && service.Backup == nil {
		return errors.New(`stateful service - define at least empty backup section if you really don't want backups`)
	}

	compose.Services = append(compose.Services, composeService)

	return nil
}

func specToComposeConfig(spec *SpecFile, defaults Defaults, secrets Secrets) (*composetypes.Config, error) {
	compose := &composetypes.Config{
		Version:  "3.5",
		Volumes:  map[string]composetypes.VolumeConfig{},
		Networks: map[string]composetypes.NetworkConfig{},
	}

	for _, service := range spec.Services {
		if err := convertOneService(service, false, compose, defaults, secrets); err != nil {
			return nil, err
		}
	}

	for _, service := range spec.GlobalServices {
		if err := convertOneService(service, true, compose, defaults, secrets); err != nil {
			return nil, err
		}
	}

	return compose, nil
}

func parseSpecFile(content io.Reader) (*SpecFile, error) {
	buf, err := ioutil.ReadAll(content)
	if err != nil {
		return nil, err
	}

	spec := &SpecFile{}
	return spec, hclsimple.Decode("dummy.hcl", buf, nil, spec)
}

func SpecToComposeByPath(path string) (string, error) {
	specFile, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer specFile.Close()

	return specToCompose(specFile, readSecrets())
}

func specToCompose(content io.Reader, secrets Secrets) (string, error) {
	defaults := Defaults{
		DockerNetworkName: "fn61",
	}

	spec, err := parseSpecFile(content)
	if err != nil {
		return "", err
	}

	composeConfig, err := specToComposeConfig(spec, defaults, secrets)
	if err != nil {
		return "", err
	}

	yamlBytes, err := yaml.Marshal(&composeConfig)
	if err != nil {
		return "", err
	}

	return string(yamlBytes), nil
}

type Secrets map[string]string

func convertEnvs(service ServiceSpec, secrets Secrets) (composetypes.MappingWithEquals, error) {
	envs := composetypes.MappingWithEquals{}
	withErr := func(err error) (composetypes.MappingWithEquals, error) {
		return envs, fmt.Errorf("convertEnvs: %w", err)
	}

	for _, env := range service.ENVs {
		env := env // pin
		value := env.Value

		if secretKey := env.Secret; secretKey != "" {

			const secretPlaceholder = "{secret}"

			if !strings.Contains(value, secretPlaceholder) {
				return withErr(fmt.Errorf("trying to use secret '%s' but placeholder '%s' not found", secretKey, secretPlaceholder))
			}

			secret, found := secrets[secretKey]
			if !found {
				return withErr(fmt.Errorf("secret not found: %s", secretKey))
			}

			value = strings.ReplaceAll(value, secretPlaceholder, secret)
		}

		envs[env.Key] = &value
	}

	return envs, nil
}

func convertPorts(service ServiceSpec, ownNamespace bool) []composetypes.ServicePortConfig {
	composePorts := []composetypes.ServicePortConfig{}

	convertOneType := func(ports []Port, tcpOrUdp string) {
		for _, port := range ports {
			mode := func() string {
				if ownNamespace {
					return ""
				} else {
					return "ingress" // Swarm concept
				}
			}()

			composePorts = append(composePorts, composetypes.ServicePortConfig{
				Mode:      mode,
				Target:    port.Container,
				Published: port.Public,
				Protocol:  tcpOrUdp,
			})
		}
	}

	convertOneType(service.TcpPorts, "tcp")
	convertOneType(service.UdpPorts, "udp")

	return composePorts
}

func createNetworkConfigIfNotExists(compose *composetypes.Config, networkName string, config composetypes.NetworkConfig) {
	if _, alreadyExists := compose.Networks[networkName]; alreadyExists {
		return
	}

	compose.Networks[networkName] = config
}

func countTrue(items ...bool) int {
	sum := 0
	for _, item := range items {
		if item {
			sum++
		}
	}

	return sum
}
