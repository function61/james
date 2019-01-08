package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/james/pkg/jamestypes"
	"os"
	"path/filepath"
)

const jamesfileFilename = "../jamesfile.json"

func readJamesfile() (*jamestypes.JamesfileCtx, error) {
	jf := jamestypes.Jamesfile{}
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

	return &jamestypes.JamesfileCtx{
		File:      jf,
		ClusterID: clusterId,
		Cluster:   jf.Clusters[clusterId],
	}, nil
}

func writeJamesfile(jamesfile *jamestypes.Jamesfile) error {
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

func findNodeByHostname(j *jamestypes.JamesfileCtx, name string) (*jamestypes.Node, error) {
	for _, node := range j.File.Clusters[j.ClusterID].Nodes {
		if node.Name == name {
			return node, nil
		}
	}

	return nil, errors.New("Node not found: " + name)
}

func reactToError(err error) {
	if err != nil {
		panic(err)
	}
}
