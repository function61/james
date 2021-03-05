package portainerclient

type EnvPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Stack struct {
	Id         int
	EndpointID int
	Name       string
	Env        []EnvPair
}

type Endpoint struct {
	Id        int
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
