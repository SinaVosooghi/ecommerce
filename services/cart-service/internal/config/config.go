// Package config provides configuration loading and validation for the cart service.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config holds all configuration values for the cart service.
type Config struct {
	// Server configuration
	Port        int    `validate:"required,min=1024,max=65535"`
	Environment string `validate:"required,oneof=dev staging prod"`
	ServiceName string `validate:"required"`

	// Logging
	LogLevel string `validate:"required,oneof=debug info warn error"`

	// AWS Configuration
	AWSRegion   string `validate:"required"`
	XRayEnabled bool

	// DynamoDB Configuration
	DynamoDBTable    string `validate:"required"`
	DynamoDBEndpoint string // Optional, for local development

	// Redis Configuration (for idempotency)
	RedisURL     string
	RedisEnabled bool

	// Rate Limiting
	RateLimitRPS   int `validate:"min=1,max=10000"`
	RateLimitBurst int `validate:"min=1,max=10000"`

	// Request Limits
	MaxRequestSize int64 `validate:"min=1024,max=10485760"`

	// Idempotency
	IdempotencyEnabled bool
	IdempotencyTTL     time.Duration `validate:"min=1m,max=168h"`

	// Circuit Breaker
	CircuitBreakerEnabled         bool
	CircuitBreakerFailureThreshold int `validate:"min=1,max=100"`
	CircuitBreakerSuccessThreshold int `validate:"min=1,max=100"`
	CircuitBreakerTimeout         time.Duration `validate:"min=1s,max=5m"`

	// Retry Configuration
	RetryMaxAttempts int           `validate:"min=1,max=10"`
	RetryInitialDelay time.Duration `validate:"min=10ms,max=10s"`
	RetryMaxDelay    time.Duration `validate:"min=100ms,max=1m"`

	// Timeouts
	DynamoDBReadTimeout  time.Duration `validate:"min=50ms,max=30s"`
	DynamoDBWriteTimeout time.Duration `validate:"min=50ms,max=30s"`

	// EventBridge Configuration
	EventBridgeEnabled  bool
	EventBridgeBusName  string
	EventBridgeSource   string

	// Feature Flags
	FeatureFlagsEnabled bool

	// Secrets Manager
	SecretsManagerEnabled bool
	JWTSecretKey         string // Can be loaded from Secrets Manager

	// CORS
	CORSAllowedOrigins []string
	CORSAllowedMethods []string
	CORSAllowedHeaders []string

	// JWT Configuration
	JWTIssuer   string
	JWTAudience string
}

// Load loads configuration from environment variables and validates it.
func Load() (*Config, error) {
	cfg := &Config{
		// Server defaults
		Port:        getEnvInt("APP_PORT", 8080),
		Environment: getEnvString("ENV_NAME", "dev"),
		ServiceName: getEnvString("SERVICE_NAME", "cart-service"),

		// Logging defaults
		LogLevel: getEnvString("LOG_LEVEL", "info"),

		// AWS defaults
		AWSRegion:   getEnvString("AWS_REGION", "us-east-1"),
		XRayEnabled: getEnvBool("AWS_XRAY_ENABLED", false),

		// DynamoDB defaults
		DynamoDBTable:    getEnvString("DYNAMODB_TABLE", "cart-service-carts"),
		DynamoDBEndpoint: getEnvString("DYNAMODB_ENDPOINT", ""),

		// Redis defaults
		RedisURL:     getEnvString("REDIS_URL", ""),
		RedisEnabled: getEnvBool("REDIS_ENABLED", false),

		// Rate limiting defaults
		RateLimitRPS:   getEnvInt("RATE_LIMIT_RPS", 100),
		RateLimitBurst: getEnvInt("RATE_LIMIT_BURST", 200),

		// Request limits defaults
		MaxRequestSize: getEnvInt64("MAX_REQUEST_SIZE", 1048576), // 1MB

		// Idempotency defaults
		IdempotencyEnabled: getEnvBool("IDEMPOTENCY_ENABLED", true),
		IdempotencyTTL:     getEnvDuration("IDEMPOTENCY_TTL", 24*time.Hour),

		// Circuit breaker defaults
		CircuitBreakerEnabled:         getEnvBool("CIRCUIT_BREAKER_ENABLED", true),
		CircuitBreakerFailureThreshold: getEnvInt("CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5),
		CircuitBreakerSuccessThreshold: getEnvInt("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 3),
		CircuitBreakerTimeout:         getEnvDuration("CIRCUIT_BREAKER_TIMEOUT", 30*time.Second),

		// Retry defaults
		RetryMaxAttempts:  getEnvInt("RETRY_MAX_ATTEMPTS", 3),
		RetryInitialDelay: getEnvDuration("RETRY_INITIAL_DELAY", 100*time.Millisecond),
		RetryMaxDelay:     getEnvDuration("RETRY_MAX_DELAY", 5*time.Second),

		// Timeout defaults
		DynamoDBReadTimeout:  getEnvDuration("DYNAMODB_READ_TIMEOUT", 500*time.Millisecond),
		DynamoDBWriteTimeout: getEnvDuration("DYNAMODB_WRITE_TIMEOUT", 1*time.Second),

		// EventBridge defaults
		EventBridgeEnabled: getEnvBool("EVENTBRIDGE_ENABLED", true),
		EventBridgeBusName: getEnvString("EVENTBRIDGE_BUS_NAME", "default"),
		EventBridgeSource:  getEnvString("EVENTBRIDGE_SOURCE", "cart-service"),

		// Feature flags defaults
		FeatureFlagsEnabled: getEnvBool("FEATURE_FLAGS_ENABLED", false),

		// Secrets Manager defaults
		SecretsManagerEnabled: getEnvBool("SECRETS_MANAGER_ENABLED", false),
		JWTSecretKey:         getEnvString("JWT_SECRET_KEY", ""),

		// CORS defaults
		CORSAllowedOrigins: getEnvStringSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
		CORSAllowedMethods: getEnvStringSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
		CORSAllowedHeaders: getEnvStringSlice("CORS_ALLOWED_HEADERS", []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID", "Idempotency-Key"}),

		// JWT defaults
		JWTIssuer:   getEnvString("JWT_ISSUER", ""),
		JWTAudience: getEnvString("JWT_AUDIENCE", ""),
	}

	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// IsDevelopment returns true if running in development environment.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "dev"
}

// IsProduction returns true if running in production environment.
func (c *Config) IsProduction() bool {
	return c.Environment == "prod"
}

// Helper functions for environment variable parsing

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
