package main

import (
	"context"
	"fmt"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/james/pkg/jamestypes"
	"github.com/scylladb/termtables"
	"github.com/spf13/cobra"
	"strings"
	"time"
)

func alertsList(jamesfile jamestypes.Jamesfile) error {
	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	alerts := GetAlertsResponse{}
	if _, err := ezhttp.Get(
		ctx,
		jamesfile.AlertManagerEndpoint+"/alerts",
		ezhttp.RespondsJson(&alerts, true),
	); err != nil {
		return err
	}

	if len(alerts) == 0 {
		return nil
	}

	tbl := termtables.CreateTable()
	tbl.AddHeaders("Key", "Time", "Subject", "Details")

	for _, alert := range alerts {
		tbl.AddRow(
			alert.Key,
			alert.Timestamp.Format(time.RFC3339),
			alert.Subject,
			truncate(onelinerize(alert.Details), 96))
	}

	fmt.Println(tbl.Render())

	return nil
}

func alertsAck(key string, jamesfile jamestypes.Jamesfile) error {
	ackRequest := &AcknowledgeAlertRequest{
		Key: key,
	}

	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()
	_, err := ezhttp.Post(
		ctx,
		jamesfile.AlertManagerEndpoint+"/alerts/acknowledge",
		ezhttp.SendJson(ackRequest))

	return err
}

func alertEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "List firing alerts",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			exitIfError(err)

			exitIfError(alertsList(jamesfile.File))
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "ack [key]",
		Short: "Acknowledge an alert",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			exitIfError(err)

			exitIfError(alertsAck(args[0], jamesfile.File))
		},
	})

	return cmd
}

type AlertItem struct {
	Key       string    `json:"alert_key"`
	Subject   string    `json:"subject"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

type GetAlertsResponse []AlertItem

type AcknowledgeAlertRequest struct {
	Key string `json:"alert_key"`
}

func truncate(input string, maxLen int) string {
	if len(input) > maxLen {
		return input[0:maxLen-2] + ".."
	}

	return input
}

func onelinerize(input string) string {
	return strings.Replace(input, "\n", "\\n", -1)
}
