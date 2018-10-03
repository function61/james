package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func httpPostJson(url string, body interface{}) error {
	bodyAsJson := &bytes.Buffer{}
	if err := json.NewEncoder(bodyAsJson).Encode(body); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bodyAsJson)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, errTransport := client.Do(req)
	if errTransport != nil {
		return errTransport
	}

	return statusOk(response)
}

func httpGetJson(url string, to interface{}) error {
	httpClient := http.Client{}
	response, errTransport := httpClient.Get(url)
	if errTransport != nil {
		return errTransport
	}

	if err := statusOk(response); err != nil {
		return err
	}

	if err := json.NewDecoder(response.Body).Decode(to); err != nil {
		return err
	}

	return nil
}

func statusOk(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("Response not 2xx: got %d", response.StatusCode)
	}

	return nil
}
