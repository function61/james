package portainerclient

import (
	"context"
	"fmt"
	"github.com/function61/gokit/ezhttp"
	"io/ioutil"
)

type Client struct {
	baseUrl     string
	bearerToken string
}

func New(baseUrl string, bearerToken string) *Client {
	return &Client{
		baseUrl:     baseUrl,
		bearerToken: bearerToken,
	}
}

func (p *Client) Auth(username, password string) (string, error) {
	type request struct {
		Username string
		Password string
	}
	type response struct {
		Jwt string `json:"jwt"`
	}

	res := response{}
	if _, err := ezhttp.Post(
		context.TODO(),
		p.baseUrl+"/api/auth",
		ezhttp.SendJson(&request{Username: username, Password: password}),
		ezhttp.RespondsJson(&res, true)); err != nil {
		return "", err
	}

	return res.Jwt, nil
}

func (p *Client) ListStacks() ([]Stack, error) {
	stacks := []Stack{}
	if _, err := ezhttp.Get(
		context.TODO(),
		p.baseUrl+"/api/stacks",
		ezhttp.AuthBearer(p.bearerToken),
		ezhttp.RespondsJson(&stacks, true)); err != nil {
		return nil, err
	}

	return stacks, nil
}

func (p *Client) StackFile(stackId string) (string, error) {
	type response struct {
		StackFileContent string
	}

	res := response{}
	if _, err := ezhttp.Get(
		context.TODO(),
		fmt.Sprintf("%s/api/stacks/%s/file", p.baseUrl, stackId),
		ezhttp.AuthBearer(p.bearerToken),
		ezhttp.RespondsJson(&res, true)); err != nil {
		return "", err
	}

	return res.StackFileContent, nil
}

func (p *Client) UpdateStack(stackId string, jamesRef string, stackFile string) error {
	req := struct {
		StackFileContent string
		Env              []EnvPair
		Prune            bool
	}{
		StackFileContent: stackFile,
		Env: []EnvPair{
			{
				Name:  "JAMES_REF",
				Value: jamesRef,
			},
		},
		Prune: true,
	}

	if res, err := ezhttp.Put(
		context.TODO(),
		fmt.Sprintf("%s/api/stacks/%s?endpointId=7", p.baseUrl, stackId),
		ezhttp.AuthBearer(p.bearerToken),
		ezhttp.SendJson(&req)); err != nil {
		resp, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("%v: %s", err, resp)
	}

	return nil
}
