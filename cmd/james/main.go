package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"regexp"
)

var version = "dev"

func runSshBash(addr string, username string, bashScript string, stdout io.Writer) error {
	sshClient, err := ssh.Dial("tcp", addr, sshClientConfig(username))
	reactToError(err)
	defer sshClient.Close()

	sshSession, err := sshClient.NewSession()
	reactToError(err)
	defer sshSession.Close()

	sshSession.Stdin = bytes.NewBufferString(bashScript)
	sshSession.Stdout = stdout
	sshSession.Stderr = os.Stderr

	// without --login with some boxes the environment (e.g. PATH) is not set up properly.
	// alternative would probably be to hardcode full paths to binary-to-invoke
	return sshSession.Run("/bin/bash --login")
}

var swarmTokenParseRegex = regexp.MustCompile("(SWMTKN-1-[^ ]+)")

func bootstrap(box *BoxDefinition, jamesfile *JamesfileCtx) error {
	var managerBox *BoxDefinition

	if jamesfile.Cluster.SwarmManagerName != "" {
		var err error
		managerBox, err = jamesfile.findBoxByName(jamesfile.Cluster.SwarmManagerName)
		if err != nil {
			return err
		}
	}

	bootstrappingManagerBox := managerBox == nil

	script := ""

	if bootstrappingManagerBox {
		log.Printf("Bootstrapping manager %s", box.Addr)

		// detach does not wait for "service to converge" (it'll print loads of data to stdout)
		script = fmt.Sprintf(
			`set -eu

docker swarm init --advertise-addr %s --listen-addr %s

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
			box.Addr,
			box.Addr,
			jamesfile.File.DockerSockProxyServerCertKey,
			jamesfile.File.DockerSockProxyVersion)
	} else {
		log.Printf("Bootstrapping worker %s", box.Addr)

		script = fmt.Sprintf(
			`set -eu

docker swarm join --token %s %s:2377`,
			jamesfile.Cluster.SwarmJoinTokenWorker,
			managerBox.Addr)
	}

	sshStdoutCopy := &bytes.Buffer{}

	stdoutTee := io.MultiWriter(os.Stdout, sshStdoutCopy)

	if err := runSshBash(
		sshDefaultPort(box.Addr),
		box.Username,
		script,
		stdoutTee); err != nil {
		return err
	}

	if bootstrappingManagerBox {
		match := swarmTokenParseRegex.FindStringSubmatch(sshStdoutCopy.String())
		if match == nil {
			return errors.New("unable to find swarm token")
		}

		token := match[1]

		// update manager details to jamesfile
		jamesfile.Cluster.SwarmManagerName = box.Name
		jamesfile.Cluster.SwarmJoinTokenWorker = token

		if err := writeJamesfile(&jamesfile.File); err != nil {
			return err
		}

		portainerDetails(jamesfile)
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
			reactToError(err)

			node, errFindBox := jamesfile.findBoxByName(args[0])
			reactToError(errFindBox)

			reactToError(bootstrap(node, jamesfile))
		},
	}
}

func nodesEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "List nodes",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			for _, node := range jamesfile.Cluster.Nodes {
				fmt.Printf("%s\n", node.Name)
			}
		},
	}

	cmd.AddCommand(importNodesEntry())
	cmd.AddCommand(&cobra.Command{
		Use:   "console",
		Short: "Enters node management console",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			iacCommon("nodes")
		},
	})

	return cmd
}

func main() {
	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   "James is your friendly infrastructure tool",
		Version: version,
	}

	commands := []*cobra.Command{
		bootstrapEntry(),
		alertEntry(),
		monitorsEntry(),
		nodesEntry(),
		sshEntry(),
		portainerEntry(),
		iacEntry(),
		dnsEntry(),
		specToComposeEntry(),
	}

	for _, cmd := range commands {
		app.AddCommand(cmd)
	}

	if err := app.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
