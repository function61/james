package main

import (
	"errors"
	"fmt"
	"github.com/function61/james/pkg/jamestypes"
	"github.com/function61/james/pkg/portainerclient"
	"github.com/function61/james/pkg/servicespec"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
)

func stackDeploy(path string, execute bool) error {
	jctx, err := readJamesfile()
	if err != nil {
		return err
	}

	updated, err := servicespec.SpecToComposeByPath(path)
	if err != nil {
		return err
	}

	portainer, err := makePortainerClient(*jctx, false)
	if err != nil {
		return err
	}

	// "prod5:stacks/hellohttp.hcl"
	jamesRef := jctx.ClusterID + ":" + path

	stacks, err := portainer.ListStacks()
	if err != nil {
		return err
	}

	stack, err := findPortainerStackByRef(jamesRef, stacks)
	if err != nil {
		return err
	}

	stackId := fmt.Sprintf("%d", stack.Id)

	if !execute {
		previous, err := portainer.StackFile(stackId)
		if err != nil {
			return err
		}

		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(previous, updated, false)

		fmt.Println(dmp.DiffPrettyText(diffs))

		return nil
	}

	return portainer.UpdateStack(stackId, jamesRef, updated)
}

func stackDeployEntry() *cobra.Command {
	execute := false

	cmd := &cobra.Command{
		Use:   "deploy <path to .hcl>",
		Short: "Deploys a stack",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			reactToError(stackDeploy(args[0], execute))
		},
	}

	cmd.Flags().BoolVarP(&execute, "execute", "x", execute, "Instead of only diffing, execute the deploy")

	return cmd
}

func stackEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Stack related commands",
	}

	cmd.AddCommand(stackDeployEntry())

	return cmd
}

func makePortainerClient(jctx jamestypes.JamesfileCtx, missingTokOk bool) (*portainerclient.Client, error) {
	if jctx.File.PortainerBaseUrl == "" {
		return nil, errors.New("PortainerBaseUrl not defined")
	}

	tok := ""
	if jctx.File.Credentials.PortainerTok != nil {
		tok = string(*jctx.File.Credentials.PortainerTok)
	} else {
		if !missingTokOk {
			return nil, errors.New("missing PortainerTok")
		}
	}

	return portainerclient.New(jctx.File.PortainerBaseUrl, tok), nil
}

func findPortainerStackByRef(ref string, stacks []portainerclient.Stack) (*portainerclient.Stack, error) {
	for _, stack := range stacks {
		for _, envPair := range stack.Env {
			if envPair.Name == "JAMES_REF" && envPair.Value == ref {
				return &stack, nil
			}
		}
	}

	return nil, fmt.Errorf("stack by JAMES_REF=%s not found", ref)
}
