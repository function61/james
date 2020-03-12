package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/james/pkg/jamestypes"
	"github.com/function61/james/pkg/portainerclient"
	"github.com/function61/james/pkg/servicespec"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"strconv"
)

func stackDeploy(path string, dryRun bool, stackName string, retriesLeft int) error {
	if retriesLeft <= 0 {
		return errors.New("stackDeploy retries exceeded")
	}

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
		// display pro-tip
		if rse, isResponseStatusError := err.(*ezhttp.ResponseStatusError); isResponseStatusError && rse.StatusCode() == http.StatusUnauthorized {
			// try to renew the token
			if err := portainerRenewAuthToken(); err != nil {
				return err
			}

			// try running the whole func again (we need to reload jamesfile and make new portainer client)
			return stackDeploy(path, dryRun, stackName, retriesLeft-1)
		} else {
			return err
		}
	}

	diffAndAck := func(previous string, needAck bool) error {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(previous, updated, false)

		// or DiffCleanupSemantic?
		diffs = dmp.DiffCleanupMerge(diffs)

		fmt.Println(dmp.DiffPrettyText(diffs))

		if !needAck {
			return nil
		}

		fmt.Printf("deploy y/n: ")

		line, _, err := bufio.NewReader(os.Stdin).ReadLine()
		if err != nil {
			return err
		}

		if string(line) != "y" {
			return fmt.Errorf("ack not 'y'; got %s", line)
		}

		fmt.Println("HOLD ON TO YOUR BUTTS")

		return nil
	}

	stack := findPortainerStackByRef(jamesRef, jctx.Cluster.PortainerEndpointId, stacks)
	if stack == nil { // new stack
		if stackName == "" {
			return errors.New("creation of new stack requires --name CLI arg")
		}

		fmt.Printf("NOTE! stack by JAMES_REF=%s not found - creating new\n", jamesRef)

		if err := diffAndAck("", !dryRun); err != nil {
			return err
		}

		if dryRun {
			return nil
		}

		if err := portainer.CreateStack(context.TODO(), stackName, jamesRef, updated); err != nil {
			return err
		}
	} else { // update existing stack
		stackId := fmt.Sprintf("%d", stack.Id)

		previous, err := portainer.StackFile(stackId)
		if err != nil {
			return err
		}

		if err := diffAndAck(previous, !dryRun); err != nil {
			return err
		}

		if dryRun {
			return nil
		}

		if err := portainer.UpdateStack(context.TODO(), stackId, jamesRef, updated); err != nil {
			return err
		}
	}

	return nil
}

func stackRm(path string) error {
	jctx, err := readJamesfile()
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

	stack := findPortainerStackByRef(jamesRef, jctx.Cluster.PortainerEndpointId, stacks)
	if stack == nil {
		return fmt.Errorf("stack to delete not found: %s", path)
	}

	return portainer.DeleteStack(context.TODO(), stack.Id)
}


func stackDeployEntry() *cobra.Command {
	dry := false
	name := ""

	cmd := &cobra.Command{
		Use:   "deploy <path to .hcl>",
		Short: "Deploys a stack",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			reactToError(stackDeploy(args[0], dry, name, 2))
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", name, "Name of the stack (needed when deploying new stack)")
	cmd.Flags().BoolVarP(&dry, "dry", "d", dry, "Instead of deploying, just make a dry run (do a diff)")

	return cmd
}

func stackRmEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <stackId>",
		Short: "Removes a stack",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			reactToError(stackRm(args[0]))
		},
	}

	return cmd
}

func stackEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Stack related commands",
	}

	cmd.AddCommand(stackDeployEntry())
	cmd.AddCommand(stackRmEntry())

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

	return portainerclient.New(jctx.File.PortainerBaseUrl, tok, jctx.Cluster.PortainerEndpointId)
}

func findPortainerStackByRef(ref string, endpointID string, stacks []portainerclient.Stack) *portainerclient.Stack {
	for _, stack := range stacks {
		if strconv.Itoa(stack.EndpointID) != endpointID {
			continue
		}

		for _, envPair := range stack.Env {
			if envPair.Name == "JAMES_REF" && envPair.Value == ref {
				return &stack
			}
		}
	}

	return nil
}
