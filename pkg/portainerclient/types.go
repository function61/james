package portainerclient

type EnvPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Stack struct {
	ID         int `json:"Id"`
	EndpointID int
	Name       string
	Env        []EnvPair
}

type Endpoint struct {
	ID        int `json:"Id"`
	Name      string
	Status    int
	Snapshots []struct {
		DockerVersion         string
		RunningContainerCount int
		ServiceCount          int
		StackCount            int
		TotalCPU              int
		TotalMemory           int64
	}
}
