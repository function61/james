package servicespec

import (
	"errors"
	"fmt"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/function61/gokit/envvar"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/james/pkg/hcltojson"
	"github.com/go-yaml/yaml"
	"io"
	"os"
	"regexp"
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

func convertOneService(service ServiceSpec, isGlobal bool, compose *composetypes.Config) error {
	envs, err := convertEnvs(service)
	if err != nil {
		return err
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

	if service.IngressPublic != "" && service.IngressAdmin != "" {
		return fmt.Errorf("both IngressPublic and IngressAdmin cannot be defined")
	}

	labels := composetypes.Labels{}
	if service.IngressPublic != "" {
		port, rule, err := parseIngressConfig(service.IngressPublic)
		if err != nil {
			return err
		}

		labels["traefik.enable"] = "true"
		labels["traefik.frontend.rule"] = rule
		labels["traefik.port"] = port
		labels["traefik.frontend.entryPoints"] = "public_http,public_https"

		if service.IngressPriority != nil {
			labels["traefik.frontend.priority"] = fmt.Sprintf("%d", *service.IngressPriority)
		}
	}

	if service.IngressAdmin != "" {
		port, rule, err := parseIngressConfig(service.IngressAdmin)
		if err != nil {
			return err
		}

		labels["traefik.enable"] = "true"
		labels["traefik.frontend.rule"] = rule
		labels["traefik.port"] = port
		labels["traefik.frontend.entryPoints"] = "backoffice"
	}

	if service.RamMb == nil {
		return errors.New("RAM limit is required")
	}

	ramBytes := composetypes.UnitBytes(*service.RamMb) * 1024 * 1024

	composeService := composetypes.ServiceConfig{
		Name:        service.Name,
		Image:       service.Image + ":" + service.Version,
		Command:     service.Command,
		Environment: *envs,
		Volumes:     volumes,
		Ports:       convertPorts(service),
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
	}

	if isGlobal && service.Replicas != nil {
		return errors.New("global services cannot have 'replicas' defined")
	}

	composeService.Deploy.Replicas = service.Replicas

	if service.PidHost {
		composeService.Pid = "host"
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

	// TODO: test fails with this
	if len(service.PersistentVolumes) > 0 && service.HowToUpdate != "stop-old-first" {
		return errors.New("expecting stop-old-first when have PersistentVolumes")
	}

	compose.Services = append(compose.Services, composeService)

	return nil
}

func specToComposeConfig(spec *SpecFile, defaults Defaults) (*composetypes.Config, error) {
	compose := &composetypes.Config{
		Version: "3.5",
		Volumes: map[string]composetypes.VolumeConfig{},
		Networks: map[string]composetypes.NetworkConfig{
			"default": composetypes.NetworkConfig{
				External: composetypes.External{
					Name: defaults.DockerNetworkName,
				},
			},
		},
	}

	for _, service := range spec.Services {
		if err := convertOneService(service, false, compose); err != nil {
			return nil, err
		}
	}

	for _, service := range spec.GlobalServices {
		if err := convertOneService(service, true, compose); err != nil {
			return nil, err
		}
	}

	return compose, nil
}

func parseSpecFile(content io.Reader) (*SpecFile, error) {
	specFileAsJson, err := hcltojson.Convert(content)
	if err != nil {
		return nil, err
	}

	spec := &SpecFile{}
	if err := jsonfile.Unmarshal(specFileAsJson, spec, true); err != nil {
		return nil, err
	}

	return spec, nil
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

func convertEnvs(service ServiceSpec) (*composetypes.MappingWithEquals, error) {
	envs := composetypes.MappingWithEquals{}
	for _, envSerialized := range service.ENVs {
		key, value := envvar.Parse(envSerialized)
		if key == "" {
			return nil, fmt.Errorf("Invalid format for ENV variable: %s", envSerialized)
		}

		envs[key] = &value
	}

	return &envs, nil
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

var parseIngressConfigRe = regexp.MustCompile("^([0-9]+)/(.+)")

func parseIngressConfig(serialized string) (string, string, error) {
	matches := parseIngressConfigRe.FindStringSubmatch(serialized)
	if matches == nil {
		return "", "", errors.New("incorrect format for ingress config")
	}

	return matches[1], matches[2], nil
}
