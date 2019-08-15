package jamestypes

import (
	"github.com/function61/james/pkg/domainwhois"
)

type Node struct {
	Name     string     `json:"Name"`
	Addr     string     `json:"Addr"`
	Username string     `json:"Username"`
	Specs    *NodeSpecs `json:"specs"` // fetched on bootstrap
}

type NodeSpecs struct {
	KernelVersion string  `json:"kernel_version"`
	OsRelease     string  `json:"os_release"`
	DockerVersion string  `json:"docker_version"`
	RamGb         float64 `json:"ram_gb"`
	DiskGb        float64 `json:"disk_gb"`
}

type Jamesfile struct {
	Domain                           string                    `json:"domain"`
	PortainerBaseUrl                 string                    `json:"portainer_baseurl"`
	Clusters                         map[string]*ClusterConfig `json:"clusters"`
	AlertManagerEndpoint             string                    `json:"AlertManagerEndpoint"`
	InfrastructureAsCodeImageVersion string                    `json:"InfrastructureAsCodeImageVersion"`
	DockerSockProxyServerCertKey     string                    `json:"DockerSockProxyServerCertKey"`
	DockerSockProxyVersion           string                    `json:"DockerSockProxyVersion"`
	CanaryEndpoint                   string                    `json:"canary_endpoint"`
	Domains                          []domainwhois.Data        `json:"domains"`
	Credentials                      Credentials               `json:"credentials"`
}

type ClusterConfig struct {
	ID                   string  `json:"id"`
	SwarmManagerName     string  `json:"swarm_manager_name"`
	SwarmJoinTokenWorker string  `json:"swarm_jointoken_worker"`
	PortainerEndpointId  string  `json:"portainer_endpoint_id"`
	Nodes                []*Node `json:"nodes"`
}

type UsernamePasswordCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type BareTokenCredential string

type Credentials struct {
	AWS          *UsernamePasswordCredentials `json:"aws"`
	Cloudflare   *UsernamePasswordCredentials `json:"cloudflare"`
	DigitalOcean *BareTokenCredential         `json:"digitalocean"`
	Hetzner      *BareTokenCredential         `json:"hetzner"`
	WhoisXmlApi  *BareTokenCredential         `json:"whoisxmlapi"`
	Portainer    *UsernamePasswordCredentials `json:"portainer"`
	PortainerTok *BareTokenCredential         `json:"portainer_shortlived_bearertoken"`
}

type JamesfileCtx struct {
	File      Jamesfile
	ClusterID string
	Cluster   *ClusterConfig
}
