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
	reactToError(sshSession.Run("/bin/bash"))

	return nil
}

var swarmTokenParseRegex = regexp.MustCompile("(SWMTKN-1-[^ ]+)")

func bootstrap(box *BoxDefinition, jamesfile *Jamesfile) error {
	managerBox, err := jamesfile.findBoxByName(jamesfile.SwarmManagerName)
	if err != nil {
		return err
	}

	bootstrappingManagerBox := managerBox == nil

	script := ""

	if bootstrappingManagerBox {
		log.Printf("Bootstrapping manager %s", box.Addr)

		script = fmt.Sprintf(
			`set -eu

docker swarm init --advertise-addr %s --listen-addr %s

docker network create --driver overlay --attachable fn61

SERVERCERT_KEY="%s"
DOCKERSOCKPROXY_VERSION="%s"

docker service create \
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
			jamesfile.DockerSockProxyServerCertKey,
			jamesfile.DockerSockProxyVersion)
	} else {
		log.Printf("Bootstrapping worker %s", box.Addr)

		script = fmt.Sprintf(
			`set -eu

docker swarm join --token %s %s:2377`,
			jamesfile.SwarmJoinTokenWorker,
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
		jamesfile.SwarmManagerName = box.Name
		jamesfile.SwarmJoinTokenWorker = token

		if err := writeJamesfile(jamesfile); err != nil {
			return err
		}

		portainerDetails(jamesfile)
	}

	return nil
}

func bootstrapEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap [server]",
		Short: "Bootstraps a server",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			box, errFindBox := jamesfile.findBoxByName(args[0])
			reactToError(errFindBox)

			reactToError(bootstrap(box, jamesfile))
		},
	}
}

func boxesEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boxes",
		Short: "List boxes",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			for _, box := range jamesfile.Boxes {
				fmt.Printf("%s\n", box.Name)
			}
		},
	}

	cmd.AddCommand(importBoxesEntry())
	cmd.AddCommand(&cobra.Command{
		Use:   "console",
		Short: "Enters box management console",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			iacCommon("boxes")
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

	app.AddCommand(bootstrapEntry())
	app.AddCommand(alertEntry())
	app.AddCommand(boxesEntry())
	app.AddCommand(sshEntry())
	app.AddCommand(portainerEntry())
	app.AddCommand(iacEntry())
	app.AddCommand(dnsEntry())

	if err := app.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
