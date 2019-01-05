package main

import (
	"fmt"
	"github.com/function61/james/pkg/servicespec"
	"github.com/spf13/cobra"
)

func specToComposeEntry() *cobra.Command {
	return &cobra.Command{
		Use:   "spec-to-compose <path>",
		Short: "Spec to JSON",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			yamlContent, err := servicespec.SpecToComposeByPath(args[0])
			reactToError(err)

			fmt.Printf("%s\n", yamlContent)
		},
	}
}