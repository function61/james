package servicespec

type SpecFile struct {
	// Stack          string        `json:"stack" hcl:"stack"`
	Services       []ServiceSpec `json:"service" hcl:"service,block"`
	GlobalServices []ServiceSpec `json:"global_service" hcl:"global_service,block"`
}

type ServiceSpec struct {
	Name        string  `json:"name" hcl:"name,label"`
	Image       string  `json:"image" hcl:"image"`
	Replicas    *uint64 `json:"replicas" hcl:"replicas"`
	HowToUpdate string  `json:"how_to_update" hcl:"how_to_update"`
	Version     string  `json:"version" hcl:"version"`
	ENVs        []struct {
		Key   string `json:"key" hcl:"key,label"`
		Value string `json:"value" hcl:"value"`
	} `json:"env" hcl:"env,block"`
	Command               []string `json:"command" hcl:"command,optional"`
	Privileged            bool     `json:"privileged" hcl:"privileged,optional"`
	Devices               []string `json:"devices" hcl:"devices,optional"`
	User                  string   `json:"user" hcl:"user,optional"`
	Caps                  []string `json:"caps" hcl:"caps,optional"`
	PlacementNodeHostname string   `json:"placement_node_hostname" hcl:"placement_node_hostname,optional"`
	IngressPublic         *struct {
		SharedIngressSettings `hcl:",remain"`
	} `json:"ingress_public" hcl:"ingress_public,block"`
	IngressBearer *struct {
		SharedIngressSettings `hcl:",remain"`
		Token                 string `json:"token" hcl:"token"`
	} `json:"ingress_bearer" hcl:"ingress_bearer,block"`
	IngressSso *struct {
		SharedIngressSettings `hcl:",remain"`
		Users                 []string `json:"users" hcl:"users"`
		Tenant                string   `json:"tenant" hcl:"tenant"`
	} `json:"ingress_sso" hcl:"ingress_sso,block"`
	Backup *struct {
		Command string `json:"command" hcl:"command"`
		// TODO: file extension
	} `json:"backup" hcl:"backup,block"`
	RamMb             uint64             `json:"ram_mb" hcl:"ram_mb"`
	PidHost           bool               `json:"pid_host" hcl:"pid_host,optional"`
	NetHost           bool               `json:"net_host" hcl:"net_host,optional"`
	TcpPorts          []Port             `json:"tcp_port" hcl:"tcp_port,block"`
	UdpPorts          []Port             `json:"udp_port" hcl:"udp_port,block"`
	PersistentVolumes []PersistentVolume `json:"persistentvolume" hcl:"persistentvolume,block"`
	BindMounts        []BindMount        `json:"bindmount" hcl:"bindmount,block"`
}

// common to all ingresses (public/password/SSO)
type SharedIngressSettings struct {
	Rule string `json:"rule" hcl:"rule"`
	Port *int   `json:"port" hcl:"port"`
}

type Port struct {
	Public    uint32 `json:"public" hcl:"public"`
	Container uint32 `json:"container" hcl:"container"`
}

type PersistentVolume struct {
	Name   string `json:"name" hcl:"name"`
	Target string `json:"target" hcl:"target"`
}

type BindMount struct {
	Host      string `json:"host" hcl:"host"`
	Container string `json:"container" hcl:"container"`
	ReadOnly  bool   `json:"readonly" hcl:"readonly"`
}

type Defaults struct {
	DockerNetworkName string
}
