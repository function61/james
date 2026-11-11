package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/function61/gokit/osutil"
	"github.com/function61/james/pkg/jamestypes"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

func runSSHBash(addr string, username string, bashScript string, stdout io.Writer) error {
	sshClient, err := dialSSH(addr, username)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	sshSession, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("create SSH session: %w", err)
	}
	defer sshSession.Close()

	sshSession.Stdin = bytes.NewBufferString(bashScript)
	sshSession.Stdout = stdout
	sshSession.Stderr = os.Stderr

	// without --login with some boxes the environment (e.g. PATH) is not set up properly.
	// alternative would probably be to hardcode full paths to binary-to-invoke
	return sshSession.Run("/bin/bash --login")
}

func dialSSH(addr string, username string) (*ssh.Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("find home directory for SSH known hosts: %w", err)
	}
	callback, err := knownhosts.New(filepath.Join(home, ".ssh", "known_hosts"))
	if err != nil {
		return nil, fmt.Errorf("load SSH known hosts: %w", err)
	}

	config := &ssh.ClientConfig{User: username, HostKeyCallback: callback}
	// SSH_AUTH_SOCK is the conventional, user-controlled local SSH agent socket.
	// #nosec G704 -- only a Unix socket is dialed, not a network address.
	if sshAgentSock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		defer sshAgentSock.Close()
		config.Auth = append(config.Auth, ssh.PublicKeysCallback(agent.NewClient(sshAgentSock).Signers))
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("connect SSH to %s: %w", addr, err)
	}
	return client, nil
}

func doSSH(servname string, jamesfile *jamestypes.JamesfileCtx) error {
	node, err := findNodeByHostname(jamesfile, servname)
	if err != nil {
		return err
	}

	sshClient, err := dialSSH(sshDefaultPort(node.Addr), node.Username)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	sshSession, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("create SSH session: %w", err)
	}
	defer sshSession.Close()

	fmt.Printf("Connected to %s\n-----\n", servname)

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Request pseudo terminal
	if err := sshSession.RequestPty("xterm", 40, 80, modes); err != nil {
		return fmt.Errorf("request for pseudo terminal failed: %w", err)
	}

	sshSession.Stdin = os.Stdin
	sshSession.Stdout = os.Stdout
	sshSession.Stderr = os.Stderr

	if err := sshSession.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	if err := sshSession.Wait(); err != nil {
		return fmt.Errorf("SSH session wait: %w", err)
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
			osutil.ExitIfError(err)

			osutil.ExitIfError(doSSH(args[0], jamesfile))
		},
	}
}

func sshDefaultPort(addr string) string {
	return addr + ":22"
}
