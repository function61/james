package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/function61/james/pkg/jamestypes"
	"github.com/spf13/cobra"
	"os"
)

const (
	terraformFileName = "nodes/terraform.tfstate"
)

func digitalOceanNodeResolver(resource TerraformResource, sshUsername string) *jamestypes.Node {
	if resource.Type != "digitalocean_droplet" {
		return nil
	}

	return &jamestypes.Node{
		Name:     resource.Primary.Attributes["name"],
		Addr:     resource.Primary.Attributes["ipv4_address"],
		Username: sshUsername,
	}
}

func hetznerNodeResolver(resource TerraformResource, sshUsername string) *jamestypes.Node {
	if resource.Type != "hcloud_server" {
		return nil
	}

	return &jamestypes.Node{
		Name:     resource.Primary.Attributes["name"],
		Addr:     resource.Primary.Attributes["ipv4_address"],
		Username: sshUsername,
	}
}

type nodeDefinitionResolver func(TerraformResource, string) *jamestypes.Node

var nodeDefinitionResolvers = []nodeDefinitionResolver{
	digitalOceanNodeResolver,
	hetznerNodeResolver,
}

func importNodesEntry() *cobra.Command {
	sshUsername := "core"

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import nodes from Terraform",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			exitIfError(err)

			addedNodes := []*jamestypes.Node{}

			terraformFile, err := os.Open(terraformFileName)
			exitIfError(err)
			defer terraformFile.Close()

			tf := TerraformFile{}
			exitIfError(json.NewDecoder(terraformFile).Decode(&tf))

			for _, module := range tf.Modules {
				for _, resource := range module.Resources {
					for _, resolver := range nodeDefinitionResolvers {
						nodeSpec := resolver(resource, sshUsername)
						if nodeSpec != nil {
							existingBox, _ := findNodeByHostname(jamesfile, nodeSpec.Name)
							isNewBox := existingBox == nil

							if isNewBox {
								addedNodes = append(addedNodes, nodeSpec)
							}

							break // no need to try other resolvers
						}
					}
				}
			}

			if len(addedNodes) == 0 {
				exitIfError(errors.New("no new nodes found"))
			}

			jamesfile.Cluster.Nodes = append(jamesfile.Cluster.Nodes, addedNodes...)

			exitIfError(writeJamesfile(&jamesfile.File))

			fmt.Printf("Updated Jamesfile with %d nodes\nRemember to bootstrap added nodes\n", len(addedNodes))
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
