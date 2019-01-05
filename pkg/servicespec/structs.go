package servicespec

type SpecFile struct {
	Stack          string        `json:"stack"`
	Services       []ServiceSpec `json:"service"`
	GlobalServices []ServiceSpec `json:"global_service"`
}

type ServiceSpec struct {
	Name                  string             `json:"name"`
	Image                 string             `json:"image"`
	Replicas              *uint64            `json:"replicas"`
	HowToUpdate           string             `json:"how_to_update"`
	Version               string             `json:"version"`
	ENVs                  []string           `json:"envs"`
	Command               []string           `json:"command"`
	PlacementNodeHostname string             `json:"placement_node_hostname"`
	IngressPublic         string             `json:"ingress_public"`
	IngressAdmin          string             `json:"ingress_admin"`
	RamMb                 *uint64            `json:"ram_mb"`
	IngressPriority       *uint64            `json:"ingress_priority"`
	PidHost               bool               `json:"pid_host"`
	TcpPorts              []Port             `json:"tcp_port"`
	UdpPorts              []Port             `json:"udp_port"`
	PersistentVolumes     []PersistentVolume `json:"persistentvolume"`
	BindMounts            []BindMount        `json:"bindmount"`
}

type Port struct {
	Public    uint32 `json:"public"`
	Container uint32 `json:"container"`
}

type PersistentVolume struct {
	Name   string `json:"name"`
	Target string `json:"target"`
}

type BindMount struct {
	Host      string `json:"host"`
	Container string `json:"container"`
	ReadOnly  bool   `json:"readonly"`
}

type Defaults struct {
	DockerNetworkName string
}
