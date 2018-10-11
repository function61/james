package main

import (
	"encoding/json"
	"errors"
	"os"
)

const jamesfileFilename = "jamesfile.json"

type BoxDefinition struct {
	Name     string `json:"Name"`
	Addr     string `json:"Addr"`
	Username string `json:"Username"`
}

type Jamesfile struct {
	ClusterID                        string          `json:"ClusterID"`
	Domain                           string          `json:"Domain"`
	SwarmManagerName                 string          `json:"SwarmManagerName"`
	SwarmJoinTokenWorker             string          `json:"SwarmJoinTokenWorker"`
	AlertManagerEndpoint             string          `json:"AlertManagerEndpoint"`
	InfrastructureAsCodeImageVersion string          `json:"InfrastructureAsCodeImageVersion"`
	DockerSockProxyServerCertKey     string          `json:"DockerSockProxyServerCertKey"`
	DockerSockProxyVersion           string          `json:"DockerSockProxyVersion"`
	DigitalOceanApiToken             string          `json:"DigitalOceanApiToken"`
	CloudflareEmail                  string          `json:"CloudflareEmail"`
	CloudflareToken                  string          `json:"CloudflareToken"`
	Boxes                            []BoxDefinition `json:"Boxes"`
}

func readJamesfile() (*Jamesfile, error) {
	file, err := os.Open(jamesfileFilename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	jsonDecoder := json.NewDecoder(file)
	jsonDecoder.DisallowUnknownFields()

	jf := &Jamesfile{}
	if err := jsonDecoder.Decode(jf); err != nil {
		return nil, err
	}

	return jf, nil
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

func (j *Jamesfile) findBoxByName(name string) (*BoxDefinition, error) {
	for _, box := range j.Boxes {
		if box.Name == name {
			return &box, nil
		}
	}

	return nil, errors.New("box not found: " + name)
}

func reactToError(err error) {
	if err != nil {
		panic(err)
	}
}
