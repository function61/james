package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

func portainerDetails(jamesfile *Jamesfile) {
	fmt.Printf(
		"Portainer connection details:\n"+
			"                  Name: %s\n"+
			"          Endpoint URL: portainer.%s.%s:4431\n"+
			"                   TLS: Yes\n"+
			"              TLS mode: TLS with server and client verification\n"+
			"    TLS CA certificate: Download from https://function61.com/ca-certificate.crt\n"+
			"       TLS certificate: client-bundle.crt\n"+
			"               TLS key: client-bundle.crt\n",
		jamesfile.ClusterID,
		jamesfile.ClusterID,
		jamesfile.Domain)
}

func portainerLaunch() error {
	startPortainer := exec.Command(
		"docker",
		"run",
		"-d",
		"--name", "portainer",
		"-p", "9000:9000",
		"-v", "portainer_data:/data",
		"portainer/portainer")

	startPortainer.Stdout = os.Stdout
	startPortainer.Stderr = os.Stderr

	return startPortainer.Run()
}

func portainerEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portainer",
		Short: "Commands related to Portainer",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "launch",
		Short: "Launches Portainer on localhost",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			reactToError(portainerLaunch())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "details",
		Short: "Docker Swarm connection details for Portainer",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			portainerDetails(jamesfile)
		},
	})

	return cmd
}
