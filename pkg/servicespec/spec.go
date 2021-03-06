package servicespec

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

func convertOneService(service ServiceSpec, isGlobal bool, compose *composetypes.Config, defaults Defaults) error {
	// most of the "not empty" checks carried out by HCL layer

	envs, err := convertEnvs(service)
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

		envs["BACKUP_COMMAND"] = &backup.Command // deprecated, remove later
	}

	volumes := []composetypes.ServiceVolumeConfig{}
	for _, bindMount := range service.BindMounts {
		volumes = append(volumes, composetypes.ServiceVolumeConfig{
			Type:     "bind",
			Source:   bindMount.Host,
			Target:   bindMount.Container,
			ReadOnly: bindMount.ReadOnly,
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

	composeService := composetypes.ServiceConfig{
		Name:        service.Name,
		Image:       service.Image + ":" + service.Version,
		Command:     service.Command,
		Environment: envs,
		Volumes:     volumes,
		Devices:     service.Devices,
		CapAdd:      service.Caps,
		Privileged:  service.Privileged,
		User:        service.User,
		Ports:       convertPorts(service),
		Labels:      labels, // TODO: duplicated here. needed for Edgerouter when in host networking mode
		Deploy: composetypes.DeployConfig{
			Mode:         deployMode,
			Labels:       labels,
			Placement:    composetypes.Placement{},
			UpdateConfig: updateConfig,
			Resources: composetypes.Resources{
				Limits: &composetypes.Resource{
					MemoryBytes: ramBytes,
				},
			},
		},
		Networks: map[string]*composetypes.ServiceNetworkConfig{},
	}

	if isGlobal && service.Replicas != nil {
		return errors.New("global services cannot have 'replicas' defined")
	}

	composeService.Deploy.Replicas = service.Replicas

	if service.PidHost {
		composeService.Pid = "host"
	}

	if service.NetHost {
		createNetworkConfigIfNotExists(compose, "host", composetypes.NetworkConfig{
			External: composetypes.External{
				Name: "host",
			},
		})

		composeService.Networks["host"] = nil
	} else {
		createNetworkConfigIfNotExists(compose, "default", composetypes.NetworkConfig{
			External: composetypes.External{
				Name: defaults.DockerNetworkName,
			},
		})

		// not required, but better to be explicit
		composeService.Networks["default"] = nil
	}

	if service.PlacementNodeHostname != "" {
		composeService.Deploy.Placement.Constraints = []string{
			"node.hostname == " + service.PlacementNodeHostname,
		}
	} else {
		// prevent human errors
		if len(service.PersistentVolumes) > 0 {
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

func specToComposeConfig(spec *SpecFile, defaults Defaults) (*composetypes.Config, error) {
	compose := &composetypes.Config{
		Version:  "3.5",
		Volumes:  map[string]composetypes.VolumeConfig{},
		Networks: map[string]composetypes.NetworkConfig{},
	}

	for _, service := range spec.Services {
		if err := convertOneService(service, false, compose, defaults); err != nil {
			return nil, err
		}
	}

	for _, service := range spec.GlobalServices {
		if err := convertOneService(service, true, compose, defaults); err != nil {
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

	return specToCompose(specFile)
}

func specToCompose(content io.Reader) (string, error) {
	defaults := Defaults{
		DockerNetworkName: "fn61",
	}

	spec, err := parseSpecFile(content)
	if err != nil {
		return "", err
	}

	composeConfig, err := specToComposeConfig(spec, defaults)
	if err != nil {
		return "", err
	}

	yamlBytes, err := yaml.Marshal(&composeConfig)
	if err != nil {
		return "", err
	}

	return string(yamlBytes), nil
}

func convertEnvs(service ServiceSpec) (composetypes.MappingWithEquals, error) {
	envs := composetypes.MappingWithEquals{}
	for _, env := range service.ENVs {
		env := env // pin
		envs[env.Key] = &env.Value
	}

	return envs, nil
}

func convertPorts(service ServiceSpec) []composetypes.ServicePortConfig {
	composePorts := []composetypes.ServicePortConfig{}

	convertOneType := func(ports []Port, tcpOrUdp string) {
		for _, port := range ports {
			composePorts = append(composePorts, composetypes.ServicePortConfig{
				Mode:      "ingress",
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
