package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/function61/gokit/jsonfile"
	"os"
	"path/filepath"
)

const jamesfileFilename = "../jamesfile.json"

type BoxDefinition struct {
	Name     string `json:"Name"`
	Addr     string `json:"Addr"`
	Username string `json:"Username"`
}

type Jamesfile struct {
	Domain                           string                    `json:"domain"`
	Clusters                         map[string]*ClusterConfig `json:"clusters"`
	AlertManagerEndpoint             string                    `json:"AlertManagerEndpoint"`
	InfrastructureAsCodeImageVersion string                    `json:"InfrastructureAsCodeImageVersion"`
	DockerSockProxyServerCertKey     string                    `json:"DockerSockProxyServerCertKey"`
	DockerSockProxyVersion           string                    `json:"DockerSockProxyVersion"`
	CanaryEndpoint                   string                    `json:"canary_endpoint"`
	Credentials                      Credentials               `json:"credentials"`
}

type ClusterConfig struct {
	ID                   string          `json:"id"`
	SwarmManagerName     string          `json:"swarm_manager_name"`
	SwarmJoinTokenWorker string          `json:"swarm_jointoken_worker"`
	Nodes                []BoxDefinition `json:"nodes"`
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
}

type JamesfileCtx struct {
	File      Jamesfile
	ClusterID string
	Cluster   *ClusterConfig
}

func readJamesfile() (*JamesfileCtx, error) {
	jf := Jamesfile{}
	if err := jsonfile.Read(jamesfileFilename, &jf, true); err != nil {
		return nil, err
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	clusterId := filepath.Base(wd)
	if _, exists := jf.Clusters[clusterId]; !exists {
		return nil, fmt.Errorf("unknown cluster: %s", clusterId)
	}

	return &JamesfileCtx{
		File:      jf,
		ClusterID: clusterId,
		Cluster:   jf.Clusters[clusterId],
	}, nil
}

func writeJamesfile(jamesfile *Jamesfile) error {
	file, err := os.Create(jamesfileFilename)
	if err != nil {
		return err
	}
	defer file.Close()

	jsonEncoder := json.NewEncoder(file)
	jsonEncoder.SetIndent("", "\t")
	if err := jsonEncoder.Encode(jamesfile); err != nil {
		return err
	}

	return nil
}

func (j *JamesfileCtx) findBoxByName(name string) (*BoxDefinition, error) {
	for _, node := range j.File.Clusters[j.ClusterID].Nodes {
		if node.Name == name {
			return &node, nil
		}
	}

	return nil, errors.New("Node not found: " + name)
}

func reactToError(err error) {
	if err != nil {
		panic(err)
	}
}
