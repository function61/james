package main

import (
	"bytes"
	"fmt"
	"github.com/function61/james/pkg/jamestypes"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"log"
	"net"
	"os"
)

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

func sshAuths() []ssh.AuthMethod {
	auths := []ssh.AuthMethod{}
	if sshAgentSock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		// FIXME: sshAgentSock does not get Close()d?
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(sshAgentSock).Signers))
	}

	return auths
}

func sshClientConfig(username string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            username,
		Auth:            sshAuths(),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func doSsh(servname string, jamesfile *jamestypes.JamesfileCtx) error {
	node, err := findNodeByHostname(jamesfile, servname)
	if err != nil {
		return err
	}

	sshClient, err := ssh.Dial("tcp", sshDefaultPort(node.Addr), sshClientConfig(node.Username))
	reactToError(err)
	defer sshClient.Close()

	sshSession, err := sshClient.NewSession()
	reactToError(err)
	defer sshSession.Close()

	fmt.Printf("Connected to %s\n-----\n", servname)

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Request pseudo terminal
	if err := sshSession.RequestPty("xterm", 40, 80, modes); err != nil {
		log.Fatal("request for pseudo terminal failed: ", err)
	}

	sshSession.Stdin = os.Stdin
	sshSession.Stdout = os.Stdout
	sshSession.Stderr = os.Stderr

	if err := sshSession.Shell(); err != nil {
		log.Fatalf("failed to start shell: %s", err.Error())
	}

	if err := sshSession.Wait(); err != nil {
		log.Fatalf("sshSession Wait(): %s", err.Error())
	}

	return nil
}

func sshEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "ssh [server]",
		Short: "Connects to a server",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			reactToError(doSsh(args[0], jamesfile))
		},
	}
}

func sshDefaultPort(addr string) string {
	return addr + ":22"
}
