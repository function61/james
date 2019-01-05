package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"log"
	"net"
	"os"
)

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

func doSsh(servname string, jamesfile *JamesfileCtx) error {
	box, errFindBox := jamesfile.findBoxByName(servname)
	if errFindBox != nil {
		return errFindBox
	}

	sshClient, err := ssh.Dial("tcp", sshDefaultPort(box.Addr), sshClientConfig(box.Username))
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
