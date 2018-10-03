package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

const (
	terraformFileName = "terraform.tfstate"
)

func importBoxesEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Import boxes from Terraform",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			if len(jamesfile.Boxes) != 0 {
				reactToError(errors.New("jamesfile.Boxes not empty"))
			}

			foundBoxes := []BoxDefinition{}

			terraformFile, err := os.Open(terraformFileName)
			reactToError(err)
			defer terraformFile.Close()

			tf := TerraformFile{}
			reactToError(json.NewDecoder(terraformFile).Decode(&tf))

			for _, module := range tf.Modules {
				for _, resource := range module.Resources {
					if resource.Type != "digitalocean_droplet" {
						continue
					}

					foundBoxes = append(foundBoxes, BoxDefinition{
						Name:     resource.Primary.Attributes["name"],
						Addr:     resource.Primary.Attributes["ipv4_address"],
						Username: "core", // FIXME: assumption about underlying image
					})
				}
			}

			if len(foundBoxes) == 0 {
				reactToError(errors.New("no boxes found"))
			}

			jamesfile.Boxes = append(jamesfile.Boxes, foundBoxes...)

			reactToError(writeJamesfile(jamesfile))

			fmt.Printf("Wrote jamesfile with %d found boxes\n", len(foundBoxes))
		},
	}
}

// TODO: import these from Terraform

type TerraformResourcePrimary struct {
	Id         string            `json:"id"`
	Attributes map[string]string `json:"attributes"`
}

type TerraformResource struct {
	Type    string                   `json:"type"`
	Primary TerraformResourcePrimary `json:"primary"`
}

type TerraformModule struct {
	Resources map[string]TerraformResource `json:"resources"`
}

type TerraformFile struct {
	Modules []TerraformModule `json:"modules"`
}
