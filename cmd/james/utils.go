package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/function61/gokit/jsonfile"
	"github.com/function61/james/pkg/jamestypes"
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
	return jsonfile.Write(jamesfileFilename, jamesfile)
}

func findNodeByHostname(j *jamestypes.JamesfileCtx, name string) (*jamestypes.Node, error) {
	for _, node := range j.File.Clusters[j.ClusterID].Nodes {
		if node.Name == name {
			return node, nil
		}
	}

	return nil, errors.New("Node not found: " + name)
}

func exitIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
