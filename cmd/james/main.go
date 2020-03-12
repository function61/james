package main

import (
	"fmt"
	"github.com/function61/gokit/dynversion"
	"github.com/function61/james/pkg/jamestypes"
	"github.com/scylladb/termtables"
	"github.com/spf13/cobra"
	"os"
)

func nodesEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "List nodes",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			exitIfError(err)

			tbl := termtables.CreateTable()
			tbl.AddHeaders("Node", "RAM (GB)", "Disk (GB)", "OS", "Docker", "Kernel")

			for _, node := range jamesfile.Cluster.Nodes {
				specs := node.Specs

				if specs == nil {
					specs = &jamestypes.NodeSpecs{} // dummy
				}

				tbl.AddRow(
					node.Name,
					fmt.Sprintf("%.1f", specs.RamGb),
					fmt.Sprintf("%.1f", specs.DiskGb),
					specs.OsRelease,
					specs.DockerVersion,
					specs.KernelVersion)
			}

			fmt.Println(tbl.Render())
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

func vmImagesEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "vm-images",
		Short: "Enters VM image management console",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			iacCommon("vm-images")
		},
	}
}

func main() {
	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   "James is your friendly infrastructure tool",
		Version: dynversion.Version,
	}

	// TODO: certificate authority
	commands := []*cobra.Command{
		alertEntry(),
		monitorsEntry(),
		nodesEntry(),
		sshEntry(),
		vmImagesEntry(),
		portainerEntry(),
		iacEntry(),
		dnsEntry(),
		specToComposeEntry(),
		domainsEntry(),
		stackEntry(),
	}

	for _, cmd := range commands {
		app.AddCommand(cmd)
	}

	if err := app.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
