// Package dynamodb provides a DynamoDB implementation of the cart repository.
package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// ClientConfig holds configuration for the DynamoDB client.
type ClientConfig struct {
	Region    string
	Endpoint  string // Optional, for local development
	TableName string
}

// Client wraps the DynamoDB client with configuration.
type Client struct {
	db        *dynamodb.Client
	tableName string
}

// NewClient creates a new DynamoDB client.
func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create DynamoDB client with optional endpoint override
	var dbClient *dynamodb.Client
	if cfg.Endpoint != "" {
		dbClient = dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	} else {
		dbClient = dynamodb.NewFromConfig(awsCfg)
	}

	return &Client{
		db:        dbClient,
		tableName: cfg.TableName,
	}, nil
}

// DB returns the underlying DynamoDB client.
func (c *Client) DB() *dynamodb.Client {
	return c.db
}

// TableName returns the configured table name.
func (c *Client) TableName() string {
	return c.tableName
}

// HealthCheck verifies connectivity to DynamoDB.
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.db.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(c.tableName),
	})
	if err != nil {
		return fmt.Errorf("DynamoDB health check failed: %w", err)
	}
	return nil
}
