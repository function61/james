package domainwhoiswhoisxmlapi

import (
	"context"
	"fmt"
	"time"

	"github.com/function61/gokit/ezhttp"
	"github.com/function61/james/pkg/domainwhois"
)

// jsonwhois.com didn't seem to be able to parse .fi ccTLD dates
// whoisxmlapi.com worked better
func New(apiKey string) domainwhois.Service {
	return &WhoisXmlApi{apiKey}
}

type WhoisXmlApi struct {
	apiKey string
}

func (w *WhoisXmlApi) Whois(domain string) (*domainwhois.Data, error) {
	// responses routinely last > 10 s
	ctx, cancel := context.WithTimeout(context.TODO(), 20*time.Second)
	defer cancel()

	endpoint := fmt.Sprintf(
		"https://www.whoisxmlapi.com/whoisserver/WhoisService?apiKey=%s&domainName=%s&outputFormat=JSON",
		w.apiKey,
		domain)

	result := &WhoisXmlApiData{}
	_, err := ezhttp.Get(
		ctx,
		endpoint,
		ezhttp.RespondsJson(result, true))
	if err != nil {
		return nil, err
	}

	normalized := normalizeWhoisXmlApi(*result)

	return &normalized, nil
}

type whoisXmlApiStupidDate time.Time

func (w *whoisXmlApiStupidDate) UnmarshalJSON(input []byte) error {
	t, err := time.Parse(`"2006-01-02 15:04:05 UTC"`, string(input))
	if err != nil {
		return err
	}
	*w = whoisXmlApiStupidDate(t)
	return nil
}

type WhoisXmlApiData struct {
	Record struct {
		Domain       string `json:"domainName"`
		Registrar    string `json:"registrarName"`
		RegistryData struct {
			Registrant struct {
				Name    string `json:"name"`
				RawText string `json:"rawText"`
			} `json:"registrant"`
			Created whoisXmlApiStupidDate `json:"createdDateNormalized"`
			Expires whoisXmlApiStupidDate `json:"expiresDateNormalized"`
		} `json:"registryData"`
	} `json:"WhoisRecord"`
}

func normalizeWhoisXmlApi(w WhoisXmlApiData) domainwhois.Data {
	return domainwhois.Data{
		Domain:            w.Record.Domain,
		Registrar:         w.Record.Registrar,
		RegistrantName:    w.Record.RegistryData.Registrant.Name,
		RegistrantDetails: w.Record.RegistryData.Registrant.RawText,
		Created:           time.Time(w.Record.RegistryData.Created),
		Expires:           time.Time(w.Record.RegistryData.Expires),
	}
}
