package domainwhois

import (
	"time"
)

type Data struct {
	Domain            string    `json:"domain"`
	Registrar         string    `json:"registrar"`
	RegistrantName    string    `json:"registrant_name"`
	RegistrantDetails string    `json:"registrant_details"`
	Created           time.Time `json:"created"`
	Expires           time.Time `json:"expires"`
}

type Service interface {
	Whois(domain string) (*Data, error)
}
