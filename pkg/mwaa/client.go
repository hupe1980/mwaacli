package mwaa

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsmwaa "github.com/aws/aws-sdk-go-v2/service/mwaa"
	"github.com/aws/aws-sdk-go-v2/service/mwaa/document"
	"github.com/aws/aws-sdk-go-v2/service/mwaa/types"
	"github.com/hupe1980/mwaacli/pkg/config"
)

// Client provides methods to interact with AWS MWAA (Managed Workflows for Apache Airflow).
type Client struct {
	client *awsmwaa.Client
}

// NewClient initializes a new MWAA client with the provided configuration.
func NewClient(cfg *config.Config) (*Client, error) {
	return &Client{
		client: awsmwaa.NewFromConfig(cfg.AWSConfig),
	}, nil
}

// CreateCliToken generates a CLI token for the specified MWAA environment.
func (c *Client) CreateCliToken(ctx context.Context, environmentName string) (*awsmwaa.CreateCliTokenOutput, error) {
	input := &awsmwaa.CreateCliTokenInput{
		Name: aws.String(environmentName),
	}

	return c.client.CreateCliToken(ctx, input)
}

// CreateWebLoginToken generates a web login token for the specified MWAA environment.
func (c *Client) CreateWebLoginToken(ctx context.Context, environmentName string) (*awsmwaa.CreateWebLoginTokenOutput, error) {
	input := &awsmwaa.CreateWebLoginTokenInput{
		Name: aws.String(environmentName),
	}

	return c.client.CreateWebLoginToken(ctx, input)
}

// InvokeRestAPI sends a REST API request to the MWAA environment with the specified method and payload.
func (c *Client) InvokeRestAPI(method types.RestApiMethod, name, path string, payload any) (*awsmwaa.InvokeRestApiOutput, error) {
	input := &awsmwaa.InvokeRestApiInput{
		Method: method,
		Name:   aws.String(name),
		Path:   aws.String(path),
		Body:   document.NewLazyDocument(payload),
	}

	return c.client.InvokeRestApi(context.TODO(), input)
}

// InvokeCliCommand executes a CLI command on the specified MWAA environment.
// It creates a CLI token, prepares the request, and sends it to the MWAA web server.
func (c *Client) InvokeCliCommand(ctx context.Context, mwaaEnvName, command string) (int, string, string, error) {
	// Generate CLI token
	cliTokenOutput, err := c.CreateCliToken(ctx, mwaaEnvName)
	if err != nil {
		return 0, "", "", err
	}

	// Construct request details
	mwaaAuthToken := "Bearer " + *cliTokenOutput.CliToken
	mwaaWebserverHostname := fmt.Sprintf("https://%s/aws_mwaa/cli", *cliTokenOutput.WebServerHostname)

	// Create HTTP request
	req, err := http.NewRequest("POST", mwaaWebserverHostname, strings.NewReader(command))
	if err != nil {
		return 0, "", "", err
	}

	req.Header.Set("Authorization", mwaaAuthToken)
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", "", err
	}
	defer resp.Body.Close()

	// Decode response body
	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, "", "", err
	}

	mwaaStdErrMessage, err := base64.StdEncoding.DecodeString(response["stderr"])
	if err != nil {
		return 0, "", "", err
	}

	mwaaStdOutMessage, err := base64.StdEncoding.DecodeString(response["stdout"])
	if err != nil {
		return 0, "", "", err
	}

	return resp.StatusCode, string(mwaaStdErrMessage), string(mwaaStdOutMessage), nil
}

// ListEnvironments retrieves a list of all MWAA environments in the AWS account.
func (c *Client) ListEnvironments(ctx context.Context) ([]string, error) {
	input := &awsmwaa.ListEnvironmentsInput{}

	output, err := c.client.ListEnvironments(ctx, input)
	if err != nil {
		return nil, err
	}

	return output.Environments, nil
}

// GetEnvironment fetches details for a specific MWAA environment.
func (c *Client) GetEnvironment(ctx context.Context, environmentName string) (*types.Environment, error) {
	input := &awsmwaa.GetEnvironmentInput{
		Name: aws.String(environmentName),
	}

	output, err := c.client.GetEnvironment(ctx, input)
	if err != nil {
		return nil, err
	}

	return output.Environment, nil
}

// DeleteEnvironment removes an MWAA environment by its name.
func (c *Client) DeleteEnvironment(ctx context.Context, environmentName string) error {
	input := &awsmwaa.DeleteEnvironmentInput{
		Name: aws.String(environmentName),
	}

	_, err := c.client.DeleteEnvironment(ctx, input)

	return err
}
