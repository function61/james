package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

func iacEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "iac",
		Short: "Enters infrastructureascode container",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			cwd, err := os.Getwd()
			reactToError(err)

			fmt.Printf("Entering infrastructure-as-code container. Press ctrl+c to exit\n")

			runIac := exec.Command(
				"docker",
				"run",
				"--rm",
				"-it",
				// for Packer
				"-e", "DIGITALOCEAN_API_TOKEN="+jamesfile.DigitalOceanApiToken,
				// for Terraform (yes, different key for same thing)
				"-e", "DIGITALOCEAN_TOKEN="+jamesfile.DigitalOceanApiToken,
				"-e", "CLOUDFLARE_EMAIL="+jamesfile.CloudflareEmail,
				"-e", "CLOUDFLARE_TOKEN="+jamesfile.CloudflareToken,
				"-v", cwd+"/secrets.env:/work/secrets.env",
				"-v", cwd+"/nodes.tf:/work/nodes.tf",
				"-v", cwd+"/services.tf:/work/services.tf",
				"-v", cwd+"/terraform.tfstate:/work/terraform.tfstate",
				"-v", cwd+"/terraform.tfstate.backup:/work/terraform.tfstate.backup",
				jamesfile.InfrastructureAsCodeImageVersion)

			runIac.Stdin = os.Stdin
			runIac.Stdout = os.Stdout
			runIac.Stderr = os.Stderr

			reactToError(runIac.Run())
		},
	}
}
