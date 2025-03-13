package secretsbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/service/mwaa/types"
	"github.com/hupe1980/mwaacli/pkg/config"
)

const (
	SecretsManagerBackend               = "airflow.providers.amazon.aws.secrets.secrets_manager.SecretsManagerBackend"
	SystemsManagerParameterStoreBackend = "airflow.providers.amazon.aws.secrets.systems_manager.SystemsManagerParameterStoreBackend"
)

// Kwargs defines the structure for secrets backend configuration.
type Kwargs struct {
	ConnectionsPrefix        string `json:"connections_prefix"`
	ConnectionsLookupPattern string `json:"connections_lookup_pattern"`
	VariablesPrefix          string `json:"variables_prefix"`
	VariablesLookupPattern   string `json:"variables_lookup_pattern"`
}

// SecretsBackend defines the interface for managing secrets.
type SecretsBackend interface {
	ListSecrets(ctx context.Context, prefix string) ([]string, error)
	GetSecretValue(ctx context.Context, secretID string) (string, error)
	UpdateSecretValue(ctx context.Context, secretID, secretValue string) error
}

// Client manages the interaction with the secrets backend.
type Client struct {
	secretsBackend SecretsBackend
	kwargs         *Kwargs
}

// NewClient initializes a new secrets backend client.
func NewClient(cfg *config.Config, environment *types.Environment) (*Client, error) {
	var kwargs Kwargs
	if err := json.Unmarshal([]byte(environment.AirflowConfigurationOptions["secrets.backend_kwargs"]), &kwargs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secrets backend kwargs: %w", err)
	}

	var secretsBackend SecretsBackend

	switch environment.AirflowConfigurationOptions["secrets.backend"] {
	case SecretsManagerBackend:
		client, err := NewSecretsManagerClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create Secrets Manager Client: %w", err)
		}

		secretsBackend = client
	default:
		return nil, fmt.Errorf("unsupported secrets backend: %s", environment.AirflowConfigurationOptions["secrets.backend"])
	}

	return &Client{
		secretsBackend: secretsBackend,
		kwargs:         &kwargs,
	}, nil
}

// ListConnections retrieves a list of connection secrets.
func (c *Client) ListConnections(ctx context.Context) ([]string, error) {
	prefix := c.kwargs.ConnectionsPrefix
	pattern := c.kwargs.ConnectionsLookupPattern

	secrets, err := c.secretsBackend.ListSecrets(ctx, prefix)
	if err != nil {
		return nil, err
	}

	if pattern == "" {
		return secrets, nil
	}

	var matchedSecrets []string

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	for _, secret := range secrets {
		if re.MatchString(secret) {
			matchedSecrets = append(matchedSecrets, secret)
		}
	}

	return matchedSecrets, nil
}

// ListVariables retrieves a list of variable secrets.
func (c *Client) ListVariables(ctx context.Context) ([]string, error) {
	prefix := c.kwargs.VariablesPrefix
	pattern := c.kwargs.VariablesLookupPattern

	secrets, err := c.secretsBackend.ListSecrets(ctx, prefix)
	if err != nil {
		return nil, err
	}

	if pattern == "" {
		return secrets, nil
	}

	var matchedSecrets []string

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	for _, secret := range secrets {
		if re.MatchString(secret) {
			matchedSecrets = append(matchedSecrets, secret)
		}
	}

	return matchedSecrets, nil
}

// GetConnection retrieves a specific connection secret.
func (c *Client) GetConnection(ctx context.Context, connectionID string) (string, error) {
	secretID := fmt.Sprintf("%s/%s", c.kwargs.ConnectionsPrefix, connectionID)
	return c.secretsBackend.GetSecretValue(ctx, secretID)
}

// GetVariable retrieves a specific variable secret.
func (c *Client) GetVariable(ctx context.Context, variableID string) (string, error) {
	secretID := fmt.Sprintf("%s/%s", c.kwargs.VariablesPrefix, variableID)
	return c.secretsBackend.GetSecretValue(ctx, secretID)
}
