package secrets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// SSMParameterStoreConfig holds configuration for SSM Parameter Store.
type SSMParameterStoreConfig struct {
	Region   string
	Endpoint string // Optional, for local testing
}

// SSMParameterStore implements Manager using AWS SSM Parameter Store.
type SSMParameterStore struct {
	client *ssm.Client
}

// NewSSMParameterStore creates a new SSM Parameter Store client.
func NewSSMParameterStore(ctx context.Context, cfg SSMParameterStoreConfig) (*SSMParameterStore, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var client *ssm.Client
	if cfg.Endpoint != "" {
		client = ssm.NewFromConfig(awsCfg, func(o *ssm.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	} else {
		client = ssm.NewFromConfig(awsCfg)
	}

	return &SSMParameterStore{client: client}, nil
}

// GetSecret retrieves a parameter from SSM Parameter Store.
func (m *SSMParameterStore) GetSecret(ctx context.Context, key string) (string, error) {
	result, err := m.client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(key),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get parameter %s: %w", key, err)
	}

	if result.Parameter != nil && result.Parameter.Value != nil {
		return *result.Parameter.Value, nil
	}

	return "", fmt.Errorf("parameter %s has no value", key)
}

// GetSecretJSON retrieves a parameter and unmarshals it as JSON.
func (m *SSMParameterStore) GetSecretJSON(ctx context.Context, key string, target interface{}) error {
	value, err := m.GetSecret(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(value), target)
}

// GetParameters retrieves multiple parameters by path prefix.
func (m *SSMParameterStore) GetParameters(ctx context.Context, path string) (map[string]string, error) {
	params := make(map[string]string)
	var nextToken *string

	for {
		result, err := m.client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
			Path:           aws.String(path),
			WithDecryption: aws.Bool(true),
			Recursive:      aws.Bool(true),
			NextToken:      nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get parameters by path %s: %w", path, err)
		}

		for _, param := range result.Parameters {
			if param.Name != nil && param.Value != nil {
				params[*param.Name] = *param.Value
			}
		}

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return params, nil
}
