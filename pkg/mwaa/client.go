// Package mwaa provides a client for interacting with AWS Managed Workflows for Apache Airflow (MWAA).
// It includes methods for managing MWAA environments, invoking REST API commands, generating CLI tokens,
// and executing CLI commands on MWAA environments.
package mwaa

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
func NewClient(cfg *config.Config) *Client {
	return &Client{
		client: awsmwaa.NewFromConfig(cfg.AWSConfig),
	}
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
func (c *Client) InvokeRestAPI(ctx context.Context, method types.RestApiMethod, environmentName, path string, queryParams, body any) (*awsmwaa.InvokeRestApiOutput, error) {
	input := &awsmwaa.InvokeRestApiInput{
		Method:          method,
		Name:            aws.String(environmentName),
		Path:            aws.String(path),
		QueryParameters: document.NewLazyDocument(queryParams),
		Body:            document.NewLazyDocument(body),
	}

	output, err := c.client.InvokeRestApi(ctx, input)
	if err != nil {
		return nil, c.handleRestAPIError(err)
	}

	return output, nil
}

// handleRestAPIError processes and formats REST API errors.
func (c *Client) handleRestAPIError(err error) error {
	var (
		clientErr *types.RestApiClientException
		serverErr *types.RestApiServerException
	)

	if errors.As(err, &clientErr) {
		return c.formatRestAPIError(clientErr.RestApiResponse, clientErr.RestApiStatusCode)
	}

	if errors.As(err, &serverErr) {
		return c.formatRestAPIError(serverErr.RestApiResponse, serverErr.RestApiStatusCode)
	}

	return err
}

// formatRestAPIError extracts error details from the response.
func (c *Client) formatRestAPIError(response document.Interface, statusCode *int32) error {
	var errorRsp map[string]any
	if err := response.UnmarshalSmithyDocument(&errorRsp); err == nil {
		return fmt.Errorf("%s: %s (HTTP StatusCode %d)", errorRsp["title"], errorRsp["detail"], aws.ToInt32(statusCode))
	}

	return fmt.Errorf("%s (HTTP StatusCode %d)", response, aws.ToInt32(statusCode))
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
	mwaaAuthToken := "Bearer " + aws.ToString(cliTokenOutput.CliToken)
	mwaaWebserverHostname := fmt.Sprintf("https://%s/aws_mwaa/cli", aws.ToString(cliTokenOutput.WebServerHostname))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mwaaWebserverHostname, strings.NewReader(command))
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

	if resp.StatusCode != http.StatusOK {
		// Print response body if an error occurred
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, "", "", err
		}

		return resp.StatusCode, "", "", fmt.Errorf("%s (HTTP StatusCode %d)", string(body), resp.StatusCode)
	}

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
