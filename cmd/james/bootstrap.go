package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/function61/james/pkg/jamestypes"
	"github.com/function61/james/pkg/shellmultipart"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"regexp"
)

var swarmTokenParseRegex = regexp.MustCompile("(SWMTKN-1-[^ ]+)")

func bootstrap(node *jamestypes.Node, jamesfile *jamestypes.JamesfileCtx) error {
	var managerNode *jamestypes.Node

	if jamesfile.Cluster.SwarmManagerName != "" {
		var err error
		managerNode, err = findNodeByHostname(jamesfile, jamesfile.Cluster.SwarmManagerName)
		if err != nil {
			return err
		}
	}

	commands := shellmultipart.New()
	commands.AddPart("set -eu")
	commands.AddPart("sudo hostnamectl set-hostname " + node.Name)

	parseDetectionResults := attachDetectors(commands)

	var swarmInitCmd *shellmultipart.Part

	if managerNode == nil {
		log.Printf("Bootstrapping manager %s", node.Addr)

		// detach does not wait for "service to converge" (it'll print loads of data to stdout)
		script := fmt.Sprintf(
			`docker swarm init --advertise-addr %s --listen-addr %s

# for some reason we've to opt-in for encryption..
docker network create --driver overlay --opt encrypted --attachable fn61

SERVERCERT_KEY="%s"
DOCKERSOCKPROXY_VERSION="%s"

docker service create \
	--detach \
	--name dockersockproxy \
	--constraint node.role==manager \
	--publish 4431:4431 \
	--env "SERVERCERT_KEY=$SERVERCERT_KEY" \
	--mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
	--network fn61 \
	"fn61/dockersockproxy:$DOCKERSOCKPROXY_VERSION"

`,
			node.Addr,
			node.Addr,
			jamesfile.File.DockerSockProxyServerCertKey,
			jamesfile.File.DockerSockProxyVersion)

		swarmInitCmd = commands.AddPart(script)
	} else {
		log.Printf("Bootstrapping worker %s", node.Addr)

		script := fmt.Sprintf(
			`docker swarm join --token %s %s:2377`,
			jamesfile.Cluster.SwarmJoinTokenWorker,
			managerNode.Addr)

		commands.AddPart(script)
	}

	sshStdoutCopy := &bytes.Buffer{}

	stdoutTee := io.MultiWriter(os.Stdout, sshStdoutCopy)

	if err := runSshBash(
		sshDefaultPort(node.Addr),
		node.Username,
		commands.GetMultipartShellScript(),
		stdoutTee); err != nil {
		return err
	}

	if err := commands.ParseShellOutput(sshStdoutCopy); err != nil {
		return fmt.Errorf("ParseShellOutput: %v", err)
	}

	if swarmInitCmd != nil {
		match := swarmTokenParseRegex.FindStringSubmatch(swarmInitCmd.Output())
		if match == nil {
			return errors.New("unable to find swarm token")
		}

		token := match[1]

		// update manager details to jamesfile
		jamesfile.Cluster.SwarmManagerName = node.Name
		jamesfile.Cluster.SwarmJoinTokenWorker = token

		portainerDetails(jamesfile)
	}

	nodeSpecs, err := parseDetectionResults()
	if err != nil {
		return err
	}

	node.Specs = nodeSpecs

	if err := writeJamesfile(&jamesfile.File); err != nil {
		return err
	}

	return nil
}

func bootstrapEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap [hostname]",
		Short: "Bootstraps a node",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			exitIfError(err)

			node, err := findNodeByHostname(jamesfile, args[0])
			exitIfError(err)

			exitIfError(bootstrap(node, jamesfile))
		},
	}
}
