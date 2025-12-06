package secrets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// AWSSecretsManagerConfig holds configuration for AWS Secrets Manager.
type AWSSecretsManagerConfig struct {
	Region   string
	Endpoint string // Optional, for local testing
}

// AWSSecretsManager implements Manager using AWS Secrets Manager.
type AWSSecretsManager struct {
	client *secretsmanager.Client
}

// NewAWSSecretsManager creates a new AWS Secrets Manager client.
func NewAWSSecretsManager(ctx context.Context, cfg AWSSecretsManagerConfig) (*AWSSecretsManager, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var client *secretsmanager.Client
	if cfg.Endpoint != "" {
		client = secretsmanager.NewFromConfig(awsCfg, func(o *secretsmanager.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	} else {
		client = secretsmanager.NewFromConfig(awsCfg)
	}

	return &AWSSecretsManager{client: client}, nil
}

// GetSecret retrieves a secret from AWS Secrets Manager.
func (m *AWSSecretsManager) GetSecret(ctx context.Context, key string) (string, error) {
	result, err := m.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", key, err)
	}

	if result.SecretString != nil {
		return *result.SecretString, nil
	}

	return "", fmt.Errorf("secret %s has no string value", key)
}

// GetSecretJSON retrieves a secret and unmarshals it as JSON.
func (m *AWSSecretsManager) GetSecretJSON(ctx context.Context, key string, target interface{}) error {
	value, err := m.GetSecret(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(value), target)
}
