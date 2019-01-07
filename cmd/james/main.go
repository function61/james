package main

import (
	"fmt"
	"github.com/apcera/termtables"
	"github.com/spf13/cobra"
	"os"
)

var version = "dev"

func nodesEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "List nodes",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			tbl := termtables.CreateTable()
			tbl.AddHeaders("Node", "RAM (GB)", "Disk (GB)", "OS", "Docker")

			for _, node := range jamesfile.Cluster.Nodes {
				specs := node.Specs

				if specs == nil {
					specs = &NodeSpecs{} // dummy
				}

				tbl.AddRow(
					node.Name,
					fmt.Sprintf("%.1f", specs.RamGb),
					fmt.Sprintf("%.1f", specs.DiskGb),
					specs.OsRelease,
					specs.DockerVersion)
			}

			fmt.Println(tbl.Render())

			for _, node := range jamesfile.Cluster.Nodes {
				fmt.Printf("%s\n", node.Name)
			}
		},
	}

	cmd.AddCommand(importNodesEntry())
	cmd.AddCommand(bootstrapEntry())
	cmd.AddCommand(&cobra.Command{
		Use:   "console",
		Short: "Enters node management console",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			iacCommon("nodes")
		},
	})

	return cmd
}

func main() {
	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   "James is your friendly infrastructure tool",
		Version: version,
	}

	commands := []*cobra.Command{
		alertEntry(),
		monitorsEntry(),
		nodesEntry(),
		sshEntry(),
		portainerEntry(),
		iacEntry(),
		dnsEntry(),
		specToComposeEntry(),
	}

	for _, cmd := range commands {
		app.AddCommand(cmd)
	}

	if err := app.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
