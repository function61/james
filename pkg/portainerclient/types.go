package portainerclient

type EnvPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Stack struct {
	Id         int
	EndpointID int
	Env        []EnvPair
}
