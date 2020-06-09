package domainwhoiswhoisxmlapi

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/function61/gokit/assert"
)

func TestParseAndNormalize(t *testing.T) {
	raw := &WhoisXmlApiData{}
	assert.Assert(t, json.Unmarshal([]byte(exampleResponse), raw) == nil)

	data := normalizeWhoisXmlApi(*raw)

	assert.EqualString(t, data.Domain, "example.com")
	assert.EqualString(t, data.Registrar, "Gandi SAS")
	assert.EqualString(t, data.RegistrantName, "function61.com")
	assert.Assert(t, strings.HasPrefix(data.RegistrantDetails, "function61.com\nregister number: 1234567-2\n"))
	assert.EqualString(t, data.Created.Format(time.RFC3339), "2006-09-03T00:00:00Z")
	assert.EqualString(t, data.Expires.Format(time.RFC3339), "2021-09-03T00:00:00Z")
}

const exampleResponse = `{
   "WhoisRecord": {
      "domainName": "example.com",
      "parseCode": 8,
      "audit": {
         "createdDate": "2019-01-08 12:41:48.552 UTC",
         "updatedDate": "2019-01-08 12:41:48.552 UTC"
      },
      "registrarName": "Gandi SAS",
      "registrarIANAID": "81",
      "registryData": {
         "createdDate": "3.9.2006 00:00:00",
         "updatedDate": "8.9.2017",
         "expiresDate": "3.9.2021 11:01:00",
         "registrant": {
            "name": "function61.com",
            "street1": "Nice address 1",
            "street2": "33100",
            "city": "com",
            "state": "register",
            "postalCode": "Tampere",
            "country": "FINLAND",
            "countryCode": "FI",
            "telephone": "0000000000",
            "rawText": "function61.com\nregister number: 1234567-2\naddress: Nice address 1\naddress: 33100\naddress: Tampere\ncountry: Finland\nphone: 0000000000\nregistrant email:\nRegistrar",
            "unparsable": "function61: 1234567-2\naddress: Nice address 1\naddress: 33100\naddress: Tampere\ncountry: Finland\nphone: 0000000000\nregistrant email:\nRegistrar"
         },
         "domainName": "example.com",
         "nameServers": {
            "rawText": "gina.ns.cloudflare.com\nben.ns.cloudflare.com\n",
            "hostNames": [
               "gina.ns.cloudflare.com",
               "ben.ns.cloudflare.com"
            ],
            "ips": []
         },
         "status": "Registered",
         "rawText": "domain: example.com\nstatus: Registered\ncreated: 3.9.2006 00:00:00\nexpires: 3.9.2021 11:01:00\navailable: 3.10.2021 11:01:00\nmodified: 8.9.2017\nRegistryLock: no\n\nNameservers\n\nnserver: gina.ns.cloudflare.com \nnserver: ben.ns.cloudflare.com \ndnssec: unsigned delegation\n\nRegistrant:\n\nfunction61.com\nregister number: 1234567-2\naddress: Nice address 1\naddress: 33100\naddress: Tampere\ncountry: Finland\nphone: 0000000000\nregistrant email: \n\nRegistrar\n\nregistrar: Gandi SAS\nwww: www.gandi.net\n\n>>> Last update of WHOIS database: 8.1.2019 14:30:18 (EET) <<<\n\n\nCopyright (c) Finnish Communications Regulatory Authority",
         "parseCode": 1275,
         "header": "",
         "strippedText": "domain: example.com\nstatus: Registered\ncreated: 3.9.2006 00:00:00\nexpires: 3.9.2021 11:01:00\navailable: 3.10.2021 11:01:00\nmodified: 8.9.2017\nRegistryLock: no\nNameservers\nnserver: gina.ns.cloudflare.com\nnserver: ben.ns.cloudflare.com\ndnssec: unsigned delegation\nRegistrant:\nfunction61.com\nregister number: 1234567-2\naddress: Nice address 1\naddress: 33100\naddress: Tampere\ncountry: Finland\nphone: 0000000000\nregistrant email:\nRegistrar\nregistrar: Gandi SAS\nwww: www.gandi.net\n>>> Last update of WHOIS database: 8.1.2019 14:30:18 (EET) <<<\nCopyright (c) Finnish Communications Regulatory Authority\n",
         "audit": {
            "createdDate": "2019-01-08 12:41:48.552 UTC",
            "updatedDate": "2019-01-08 12:41:48.552 UTC"
         },
         "registrarName": "Gandi SAS",
         "registrarIANAID": "81",
         "createdDateNormalized": "2006-09-03 00:00:00 UTC",
         "updatedDateNormalized": "2017-09-08 00:00:00 UTC",
         "expiresDateNormalized": "2021-09-03 00:00:00 UTC",
         "whoisServer": "whois.ficora.fi"
      },
      "domainNameExt": ".fi",
      "estimatedDomainAge": 4510
   }
}`
