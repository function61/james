package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
)

func iacEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "iac [namespace]",
		Short: "Enters infrastructureascode container",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			namespace := args[0]

			jamesfile, err := readJamesfile()
			reactToError(err)

			cwd, err := os.Getwd()
			reactToError(err)

			pathToNamespaceFile := func(filename string) string {
				return cwd + "/" + namespace + "/" + filename
			}

			reactToError(touch(pathToNamespaceFile("terraform.tfstate")))
			reactToError(touch(pathToNamespaceFile("terraform.tfstate.backup")))

			dockerArgs := []string{
				"docker",
				"run",
				"--rm",
				"-it",
				// for Packer
				"-e", "DIGITALOCEAN_API_TOKEN=" + jamesfile.Credentials.DigitalOcean.Password,
				// for Terraform (yes, different key for same thing)
				"-e", "DIGITALOCEAN_TOKEN=" + jamesfile.Credentials.DigitalOcean.Password,
				"-e", "CLOUDFLARE_EMAIL=" + jamesfile.Credentials.Cloudflare.Username,
				"-e", "CLOUDFLARE_TOKEN=" + jamesfile.Credentials.Cloudflare.Password,
				"-e", "AWS_ACCESS_KEY_ID=" + jamesfile.Credentials.AWS.Username,
				"-e", "AWS_SECRET_ACCESS_KEY=" + jamesfile.Credentials.AWS.Password,
				"-v", cwd + "/scripts:/work/scripts",
				"-v", pathToNamespaceFile("terraform.tfstate") + ":/work/terraform.tfstate",
				"-v", pathToNamespaceFile("terraform.tfstate.backup") + ":/work/terraform.tfstate.backup",
			}

			tfFilesFromNamespace, err := filepath.Glob(namespace + "/*.tf")
			reactToError(err)

			if len(tfFilesFromNamespace) == 0 {
				reactToError(errors.New("no *.tf files found"))
			}

			for _, tfFile := range tfFilesFromNamespace {
				// remove the path to the dir and keep only filename
				filename := filepath.Base(tfFile)

				mapping := pathToNamespaceFile(filename) + ":/work/" + filename

				dockerArgs = append(dockerArgs, "-v", mapping)
			}

			dockerArgs = append(dockerArgs, jamesfile.InfrastructureAsCodeImageVersion)

			fmt.Printf("Entering infrastructure-as-code container. Press ctrl+c to exit\n")

			runIac := exec.Command(dockerArgs[0], dockerArgs[1:]...)
			runIac.Stdin = os.Stdin
			runIac.Stdout = os.Stdout
			runIac.Stderr = os.Stderr

			reactToError(runIac.Run())
		},
	}
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
