package portainerclient

import (
	"context"
	"errors"
	"fmt"
	"github.com/function61/gokit/ezhttp"
	"io/ioutil"
)

type Client struct {
	baseUrl     string
	bearerToken string
	endpointId  string
}

func New(baseUrl string, bearerToken string, endpointId string) (*Client, error) {
	if endpointId == "" {
		return nil, errors.New("empty endpointId")
	}

	return &Client{
		baseUrl:     baseUrl,
		bearerToken: bearerToken,
		endpointId:  endpointId,
	}, nil
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
		ezhttp.RespondsJson(&res, true),
	); err != nil {
		return "", fmt.Errorf("Auth: %w", err)
	}

	return res.Jwt, nil
}

type DockerInfoResponse struct {
	Swarm struct {
		Cluster struct {
			ID string
		}
	}
}

func (p *Client) DockerInfo(ctx context.Context) (*DockerInfoResponse, error) {
	res := &DockerInfoResponse{}

	if _, err := ezhttp.Get(
		ctx,
		p.baseUrl+"/api/endpoints/"+p.endpointId+"/docker/info",
		ezhttp.AuthBearer(p.bearerToken),
		ezhttp.RespondsJson(res, true),
	); err != nil {
		return nil, fmt.Errorf("DockerInfo: %w", err)
	}

	return res, nil
}

func (p *Client) ListStacks(ctx context.Context) ([]Stack, error) {
	stacks := []Stack{}
	if _, err := ezhttp.Get(
		ctx,
		p.baseUrl+"/api/stacks",
		ezhttp.AuthBearer(p.bearerToken),
		ezhttp.RespondsJson(&stacks, true),
	); err != nil {
		return nil, fmt.Errorf("ListStacks: %w", err)
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
		ezhttp.RespondsJson(&res, true),
	); err != nil {
		return "", fmt.Errorf("StackFile: %s: %w", stackId, err)
	}

	return res.StackFileContent, nil
}

func (p *Client) CreateStack(ctx context.Context, name string, jamesRef string, stackFile string) error {
	// we need to provide stupid details that Portainer itself would be able to resolve
	dockerInfo, err := p.DockerInfo(ctx)
	if err != nil {
		return err
	}

	req := struct {
		Name             string
		SwarmID          string
		StackFileContent string
		Env              []EnvPair
	}{
		Name:             name,
		SwarmID:          dockerInfo.Swarm.Cluster.ID,
		StackFileContent: stackFile,
		Env: []EnvPair{
			{
				Name:  "JAMES_REF",
				Value: jamesRef,
			},
		},
	}

	if res, err := ezhttp.Post(
		ctx,
		fmt.Sprintf("%s/api/stacks?endpointId=%s&type=1&method=string", p.baseUrl, p.endpointId),
		ezhttp.AuthBearer(p.bearerToken),
		ezhttp.SendJson(&req),
	); err != nil {
		resp, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("%v: %s", err, resp)
	}

	return nil
}

func (p *Client) UpdateStack(ctx context.Context, stackId string, jamesRef string, stackFile string) error {
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
		ctx,
		fmt.Sprintf("%s/api/stacks/%s?endpointId=%s", p.baseUrl, stackId, p.endpointId),
		ezhttp.AuthBearer(p.bearerToken),
		ezhttp.SendJson(&req),
	); err != nil {
		resp, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("%v: %s", err, resp)
	}

	return nil
}

func (p *Client) DeleteStack(ctx context.Context, stackId int) error {
	if res, err := ezhttp.Del(
		ctx,
		fmt.Sprintf("%s/api/stacks/%d", p.baseUrl, stackId),
		ezhttp.AuthBearer(p.bearerToken),
	); err != nil {
		resp, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("%v: %s", err, resp)
	}

	return nil
}
