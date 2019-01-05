package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

const (
	terraformFileName = "nodes/terraform.tfstate"
)

func digitalOceanBoxDefinitionResolver(resource TerraformResource) *BoxDefinition {
	if resource.Type != "digitalocean_droplet" {
		return nil
	}

	return &BoxDefinition{
		Name:     resource.Primary.Attributes["name"],
		Addr:     resource.Primary.Attributes["ipv4_address"],
		Username: "core", // FIXME: assumption about underlying image
	}
}

func hetznerBoxDefinitionResolver(resource TerraformResource) *BoxDefinition {
	if resource.Type != "hcloud_server" {
		return nil
	}

	return &BoxDefinition{
		Name:     resource.Primary.Attributes["name"],
		Addr:     resource.Primary.Attributes["ipv4_address"],
		Username: "root", // FIXME: assumption about underlying image
	}
}

type boxDefinitionResolver func(TerraformResource) *BoxDefinition

var boxDefinitionResolvers = []boxDefinitionResolver{
	digitalOceanBoxDefinitionResolver,
	hetznerBoxDefinitionResolver,
}

func importBoxesEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Import boxes from Terraform",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			newBoxesFound := []BoxDefinition{}

			terraformFile, err := os.Open(terraformFileName)
			reactToError(err)
			defer terraformFile.Close()

			tf := TerraformFile{}
			reactToError(json.NewDecoder(terraformFile).Decode(&tf))

			for _, module := range tf.Modules {
				for _, resource := range module.Resources {
					for _, resolver := range boxDefinitionResolvers {
						boxDefinition := resolver(resource)
						if boxDefinition != nil {
							existingBox, _ := jamesfile.findBoxByName(boxDefinition.Name)
							isNewBox := existingBox == nil

							if isNewBox {
								newBoxesFound = append(newBoxesFound, *boxDefinition)
							}

							break // no need to try other resolvers
						}
					}
				}
			}

			if len(newBoxesFound) == 0 {
				reactToError(errors.New("no new boxes found"))
			}

			jamesfile.Cluster.Nodes = append(jamesfile.Cluster.Nodes, newBoxesFound...)

			reactToError(writeJamesfile(&jamesfile.File))

			fmt.Printf("Updated Jamesfile with %d boxes\nRemember to bootstrap added boxes\n", len(newBoxesFound))
		},
	}
}

// TODO: piggyback Terraform data structures

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
