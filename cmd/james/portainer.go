package main

import (
	"fmt"
	"github.com/function61/james/pkg/jamestypes"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

func portainerDetails(jamesfile *jamestypes.JamesfileCtx) {
	fmt.Printf(
		"Portainer connection details:\n"+
			"                  Name: %s\n"+
			"          Endpoint URL: dockersockproxy.%s.%s:4431\n"+
			"                   TLS: Yes\n"+
			"              TLS mode: TLS with server and client verification\n"+
			"    TLS CA certificate: Download from https://fn61.net/ca.crt\n"+
			"       TLS certificate: client-bundle.crt\n"+
			"               TLS key: client-bundle.crt\n",
		jamesfile.Cluster.ID,
		jamesfile.Cluster.ID,
		jamesfile.File.Domain)
}

func portainerLaunch() error {
	startPortainer := exec.Command(
		"docker",
		"run",
		"-d",
		"--name", "portainer",
		"-p", "9000:9000",
		"-v", "portainer_data:/data",
		"--restart", "always",
		"portainer/portainer")

	startPortainer.Stdout = os.Stdout
	startPortainer.Stderr = os.Stderr

	if err := startPortainer.Run(); err != nil {
		return err
	}

	fmt.Println("Portainer should now be usable at http://localhost:9000/")

	return nil
}

func portainerEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portainer",
		Short: "Commands related to Portainer",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "launch",
		Short: "Deploys Portainer on localhost",
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
