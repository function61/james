package main

import (
	"fmt"
	"github.com/apcera/termtables"
	"github.com/spf13/cobra"
	"time"
)

func alertsList(jamesfile *Jamesfile) error {
	var alerts GetAlertsResponse
	if err := httpGetJson(jamesfile.AlertManagerEndpoint+"/alerts", &alerts); err != nil {
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
			alert.Details)
	}

	fmt.Println(tbl.Render())

	return nil
}

func alertsAck(key string, jamesfile *Jamesfile) error {
	ackRequest := &AcknowledgeAlertRequest{
		Key: key,
	}

	if err := httpPostJson(jamesfile.AlertManagerEndpoint+"/alerts/acknowledge", ackRequest); err != nil {
		return err
	}

	return nil
}

func alertEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "List firing alerts",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			reactToError(alertsList(jamesfile))
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "ack [key]",
		Short: "Acknowledge an alert",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			reactToError(alertsAck(args[0], jamesfile))
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
