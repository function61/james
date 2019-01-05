package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/apcera/termtables"
	"github.com/function61/gokit/cryptorandombytes"
	"github.com/function61/gokit/ezhttp"
	"github.com/spf13/cobra"
)

var (
	errUnableToFindMonitor = errors.New("unable to find monitor by id")
	errCanaryNotConfigured = errors.New("canary not configured")
)

func monitorsList(jamesfile Jamesfile) error {
	config, err := getConfig(jamesfile)
	if err != nil {
		return err
	}

	if len(config.Monitors) == 0 {
		return nil
	}

	tbl := termtables.CreateTable()
	tbl.AddHeaders("ID", "Enabled", "URL", "Find string")

	for _, monitor := range config.Monitors {
		tbl.AddRow(
			monitor.Id,
			boolToCheckmarkString(monitor.Enabled),
			monitor.Url,
			monitor.Find)
	}

	fmt.Println(tbl.Render())

	return nil
}

func monitorsCreate(url string, findString string, jamesfile Jamesfile) error {
	config, err := getConfig(jamesfile)
	if err != nil {
		return err
	}

	monitor := Monitor{
		Id:      cryptorandombytes.Hex(3),
		Enabled: true,
		Url:     url,
		Find:    findString,
	}

	config.Monitors = append(config.Monitors, monitor)

	return setConfig(config, jamesfile)
}

func monitorsDelete(id string, jamesfile Jamesfile) error {
	config, err := getConfig(jamesfile)
	if err != nil {
		return err
	}

	monitorsWithOneDeleted, err := deleteMonitor(id, config.Monitors)
	if err != nil {
		return err
	}

	config.Monitors = monitorsWithOneDeleted

	return setConfig(config, jamesfile)
}

func monitorsEnableOrDisable(id string, enable bool, jamesfile Jamesfile) error {
	config, err := getConfig(jamesfile)
	if err != nil {
		return err
	}

	monitorsUpdated, err := enableOrDisableMonitor(id, enable, config.Monitors)
	if err != nil {
		return err
	}

	config.Monitors = monitorsUpdated

	return setConfig(config, jamesfile)
}

func monitorsEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitors",
		Short: "AlertManager-canary related commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "Lists monitors",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			reactToError(monitorsList(jamesfile.File))
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create [url] [findString]",
		Short: "Create a new monitor",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			reactToError(monitorsCreate(args[0], args[1], jamesfile.File))
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "rm [id]",
		Short: "Removes a monitor",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			reactToError(monitorsDelete(args[0], jamesfile.File))
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "enable [id]",
		Short: "Enables a monitor",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			reactToError(monitorsEnableOrDisable(args[0], true, jamesfile.File))
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "disable [id]",
		Short: "Disables a monitor",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jamesfile, err := readJamesfile()
			reactToError(err)

			reactToError(monitorsEnableOrDisable(args[0], false, jamesfile.File))
		},
	})

	return cmd
}

func boolToCheckmarkString(input bool) string {
	if input {
		return "✓"
	}
	return "✗"
}

func enableOrDisableMonitor(id string, enable bool, monitors []Monitor) ([]Monitor, error) {
	newMonitors := []Monitor{}

	found := false
	for _, monitor := range monitors {
		if monitor.Id == id {
			monitor.Enabled = enable
			found = true
		}

		newMonitors = append(newMonitors, monitor)
	}

	if !found {
		return nil, errUnableToFindMonitor
	}

	return newMonitors, nil
}

func deleteMonitor(id string, monitors []Monitor) ([]Monitor, error) {
	for idx, monitor := range monitors {
		if monitor.Id == id {
			return append(monitors[:idx], monitors[idx+1:]...), nil
		}
	}

	return nil, errUnableToFindMonitor
}

func configEndpointFor(jamesfile Jamesfile) (string, error) {
	if jamesfile.CanaryEndpoint == "" {
		return "", errCanaryNotConfigured
	}

	return jamesfile.CanaryEndpoint + "/config", nil
}

type Config struct {
	SnsTopicIngest string    `json:"sns_topic_ingest"`
	Monitors       []Monitor `json:"monitors"`
}

type Monitor struct {
	Id      string `json:"id"`
	Enabled bool   `json:"enabled"`
	Url     string `json:"url"`
	Find    string `json:"find"`
}

func getConfig(jamesfile Jamesfile) (*Config, error) {
	configEndpoint, err := configEndpointFor(jamesfile)
	if err != nil {
		return nil, err
	}

	config := &Config{}

	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	// can't allow unknown fields in JSON because since we're doing mutations, we could lose data
	if _, err := ezhttp.Get(ctx, configEndpoint, ezhttp.RespondsJson(config, false)); err != nil {
		return nil, err
	}

	return config, nil
}

func setConfig(config *Config, jamesfile Jamesfile) error {
	configEndpoint, err := configEndpointFor(jamesfile)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.TODO(), ezhttp.DefaultTimeout10s)
	defer cancel()

	_, err = ezhttp.Put(ctx, configEndpoint, ezhttp.SendJson(config))

	return err
}
