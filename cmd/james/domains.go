package main

import (
	"errors"
	"fmt"

	"github.com/function61/james/pkg/domainwhois/domainwhoiswhoisxmlapi"
	"github.com/scylladb/termtables"
	"github.com/spf13/cobra"
)

func domainsEntry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Domain related commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "List tracked domains",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			exitIfError(listDomains())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "add [domain]",
		Short: "Start tracking whois data of domain",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			exitIfError(addDomain(args[0]))
		},
	})

	return cmd
}

func listDomains() error {
	jamesfile, err := readJamesfile()
	if err != nil {
		return err
	}

	tbl := termtables.CreateTable()
	tbl.AddHeaders("Domain", "Registrant", "Registrar", "Expires")

	for _, domain := range jamesfile.File.Domains {
		tbl.AddRow(
			domain.Domain,
			domain.RegistrantName,
			domain.Registrar,
			domain.Expires.Format("2006-01-02"))
	}

	fmt.Println(tbl.Render())

	return nil
}

func addDomain(name string) error {
	jamesfile, err := readJamesfile()
	exitIfError(err)

	if jamesfile.File.Credentials.WhoisXmlApi == nil {
		return errors.New("credentials not set")
	}
	svc := domainwhoiswhoisxmlapi.New(string(*jamesfile.File.Credentials.WhoisXmlApi))

	whoisData, err := svc.Whois(name)
	if err != nil {
		return err
	}

	jamesfile.File.Domains = append(jamesfile.File.Domains, *whoisData)

	return writeJamesfile(&jamesfile.File)
}
