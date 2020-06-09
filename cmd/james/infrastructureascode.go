package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/function61/james/pkg/jamestypes"
	"github.com/spf13/cobra"
)

func dnsEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns",
		Short: "DNS tools",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "console",
		Short: "Enters DNS console",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			iacCommon("dns")
		},
	})

	return cmd
}

func iacEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "iac [namespace]",
		Short: "Custom infrastructure-as-code container",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			iacCommon(args[0])
		},
	}
}

func iacCommon(namespace string) {
	jamesfile, err := readJamesfile()
	exitIfError(err)

	cwd, err := os.Getwd()
	exitIfError(err)

	pathToNamespaceFile := func(filename string) string {
		return cwd + "/" + namespace + "/" + filename
	}

	exitIfError(touch(pathToNamespaceFile("terraform.tfstate")))
	exitIfError(touch(pathToNamespaceFile("terraform.tfstate.backup")))

	dockerArgs := []string{
		"docker",
		"run",
		"--rm",
		"-it",
		"-v", pathToNamespaceFile("state/") + ":/work/state/", // state directory
		"-v", pathToNamespaceFile("terraform.tfstate") + ":/work/terraform.tfstate",
		"-v", pathToNamespaceFile("terraform.tfstate.backup") + ":/work/terraform.tfstate.backup",
	}

	// expose all API credentials needed by Terraform/Packer
	for key, value := range credentialsToTerraformAndPackerEnvs(jamesfile.File.Credentials) {
		dockerArgs = append(dockerArgs, "-e", key+"="+value)
	}

	tfFilesFromNamespace, err := filepath.Glob(namespace + "/*.tf")
	exitIfError(err)

	for _, tfFile := range tfFilesFromNamespace {
		// remove the path to the dir and keep only filename
		filename := filepath.Base(tfFile)

		mapping := pathToNamespaceFile(filename) + ":/work/" + filename

		dockerArgs = append(dockerArgs, "-v", mapping)
	}

	dockerArgs = append(dockerArgs, jamesfile.File.InfrastructureAsCodeImageVersion)

	fmt.Printf("Entering infrastructure-as-code container. Press ctrl+c to exit\n")

	runIac := exec.Command(dockerArgs[0], dockerArgs[1:]...)
	runIac.Stdin = os.Stdin
	runIac.Stdout = os.Stdout
	runIac.Stderr = os.Stderr

	exitIfError(runIac.Run())
}

// creates an empty file if it does not exist
func touch(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			emptyFile, err := os.Create(path)
			if err != nil {
				return err
			}

			return emptyFile.Close()
		}

		return err // an actual error
	}

	return nil
}

func credentialsToTerraformAndPackerEnvs(creds jamestypes.Credentials) map[string]string {
	envs := map[string]string{}

	if creds.DigitalOcean != nil {
		// 1st is for Packer
		// 2nd for Terraform (yes, different key for same thing)
		envs["DIGITALOCEAN_API_TOKEN"] = string(*creds.DigitalOcean)
		envs["DIGITALOCEAN_TOKEN"] = string(*creds.DigitalOcean)
	}

	if creds.Cloudflare != nil {
		envs["CLOUDFLARE_EMAIL"] = creds.Cloudflare.Username
		envs["CLOUDFLARE_TOKEN"] = creds.Cloudflare.Password
	}

	if creds.AWS != nil {
		envs["AWS_ACCESS_KEY_ID"] = creds.AWS.Username
		envs["AWS_SECRET_ACCESS_KEY"] = creds.AWS.Password
	}

	if creds.Hetzner != nil {
		envs["HCLOUD_TOKEN"] = string(*creds.Hetzner)
	}

	return envs
}
