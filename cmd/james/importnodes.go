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

func digitalOceanNodeResolver(resource TerraformResource, sshUsername string) *Node {
	if resource.Type != "digitalocean_droplet" {
		return nil
	}

	return &Node{
		Name:     resource.Primary.Attributes["name"],
		Addr:     resource.Primary.Attributes["ipv4_address"],
		Username: sshUsername,
	}
}

func hetznerNodeResolver(resource TerraformResource, sshUsername string) *Node {
	if resource.Type != "hcloud_server" {
		return nil
	}

	return &Node{
		Name:     resource.Primary.Attributes["name"],
		Addr:     resource.Primary.Attributes["ipv4_address"],
		Username: sshUsername,
	}
}

type nodeDefinitionResolver func(TerraformResource, string) *Node

var nodeDefinitionResolvers = []nodeDefinitionResolver{
	digitalOceanNodeResolver,
	hetznerNodeResolver,
}

func importNodesEntry() *cobra.Command {
	sshUsername := "core"

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import boxes from Terraform",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			newBoxesFound := []*Node{}

			terraformFile, err := os.Open(terraformFileName)
			reactToError(err)
			defer terraformFile.Close()

			tf := TerraformFile{}
			reactToError(json.NewDecoder(terraformFile).Decode(&tf))

			for _, module := range tf.Modules {
				for _, resource := range module.Resources {
					for _, resolver := range nodeDefinitionResolvers {
						boxDefinition := resolver(resource, sshUsername)
						if boxDefinition != nil {
							existingBox, _ := jamesfile.findNodeByHostname(boxDefinition.Name)
							isNewBox := existingBox == nil

							if isNewBox {
								newBoxesFound = append(newBoxesFound, boxDefinition)
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

	cmd.Flags().StringVarP(&sshUsername, "ssh-username", "", sshUsername, "Username to use when logging in with SSH")

	return cmd
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
