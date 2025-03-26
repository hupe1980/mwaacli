package secretsbackend

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/hupe1980/mwaacli/pkg/config"
)

// SecretsManagerClient is a wrapper around AWS Secrets Manager client.
type SecretsManagerClient struct {
	client *secretsmanager.Client
}

// NewSecretsManagerClient initializes a new SecretsManagerClient.
func NewSecretsManagerClient(cfg *config.Config) (*SecretsManagerClient, error) {
	return &SecretsManagerClient{
		client: secretsmanager.NewFromConfig(cfg.AWSConfig),
	}, nil
}

// ListSecrets retrieves a list of secret IDs that match the given prefix.
func (s *SecretsManagerClient) ListSecrets(ctx context.Context, prefix string) ([]string, error) {
	var secretIDs []string

	paginator := secretsmanager.NewListSecretsPaginator(s.client, &secretsmanager.ListSecretsInput{
		Filters: []types.Filter{
			{
				Key:    types.FilterNameStringTypeName,
				Values: []string{prefix},
			},
		},
	})

	for paginator.HasMorePages() {
		result, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, secret := range result.SecretList {
			if secret.Name != nil {
				secretIDs = append(secretIDs, *secret.Name)
			}
		}
	}

	return secretIDs, nil
}

// GetSecretValue retrieves the value of a given secret ID.
func (s *SecretsManagerClient) GetSecretValue(ctx context.Context, secretID string) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}

	result, err := s.client.GetSecretValue(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret value: %w", err)
	}

	if result.SecretString == nil {
		return "", fmt.Errorf("secret value is nil")
	}

	return aws.ToString(result.SecretString), nil
}

// UpdateSecretValue updates the value of a given secret ID.
func (s *SecretsManagerClient) UpdateSecretValue(ctx context.Context, secretID, secretValue string) error {
	input := &secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(secretID),
		SecretString: aws.String(secretValue),
	}

	_, err := s.client.UpdateSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update secret value: %w", err)
	}

	return nil
}
