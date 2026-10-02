// Package config provides configuration loading and validation for the cart service.
package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
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
	CircuitBreakerEnabled          bool
	CircuitBreakerFailureThreshold int           `validate:"min=1,max=100"`
	CircuitBreakerSuccessThreshold int           `validate:"min=1,max=100"`
	CircuitBreakerTimeout          time.Duration `validate:"min=1s,max=5m"`

	// Retry Configuration
	RetryMaxAttempts  int           `validate:"min=1,max=10"`
	RetryInitialDelay time.Duration `validate:"min=10ms,max=10s"`
	RetryMaxDelay     time.Duration `validate:"min=100ms,max=1m"`

	// Timeouts
	DynamoDBReadTimeout  time.Duration `validate:"min=50ms,max=30s"`
	DynamoDBWriteTimeout time.Duration `validate:"min=50ms,max=30s"`

	// EventBridge Configuration
	EventBridgeEnabled bool
	EventBridgeBusName string
	EventBridgeSource  string

	// Feature Flags
	FeatureFlagsEnabled bool

	// Metrics (CloudWatch Embedded Metric Format on stdout)
	MetricsEnabled bool

	// Secrets Manager
	SecretsManagerEnabled bool
	JWTSecretKey          string // Injected by ECS from Secrets Manager

	// Authentication. May only be disabled in dev.
	AuthEnabled bool

	// CORS
	CORSAllowedOrigins []string
	CORSAllowedMethods []string
	CORSAllowedHeaders []string

	// JWT Configuration
	JWTIssuer   string
	JWTAudience string
}

// minJWTSecretBytes is the minimum HS256 key length (RFC 7518 §3.2).
const minJWTSecretBytes = 32

// Load loads configuration from .env file (if present) and environment variables, then validates it.
// Environment variables take precedence over .env file values. Malformed values are reported as
// errors rather than silently replaced with defaults.
func Load() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	env := &envReader{}
	environment := env.String("ENV_NAME", "dev")

	// Allow any origin by default only in dev; other environments must opt in explicitly.
	var defaultOrigins []string
	if environment == "dev" {
		defaultOrigins = []string{"*"}
	}

	cfg := &Config{
		// Server defaults
		Port:        env.Int("APP_PORT", 8080),
		Environment: environment,
		ServiceName: env.String("SERVICE_NAME", "cart-service"),

		// Logging defaults
		LogLevel: env.String("LOG_LEVEL", "info"),

		// AWS defaults
		AWSRegion:   env.String("AWS_REGION", "us-east-1"),
		XRayEnabled: env.Bool("AWS_XRAY_ENABLED", false),

		// DynamoDB defaults
		DynamoDBTable:    env.String("DYNAMODB_TABLE", "cart-service-carts"),
		DynamoDBEndpoint: env.String("DYNAMODB_ENDPOINT", ""),

		// Redis defaults
		RedisURL:     env.String("REDIS_URL", ""),
		RedisEnabled: env.Bool("REDIS_ENABLED", false),

		// Rate limiting defaults
		RateLimitRPS:   env.Int("RATE_LIMIT_RPS", 100),
		RateLimitBurst: env.Int("RATE_LIMIT_BURST", 200),

		// Request limits defaults
		MaxRequestSize: env.Int64("MAX_REQUEST_SIZE", 1048576), // 1MB

		// Idempotency defaults
		IdempotencyEnabled: env.Bool("IDEMPOTENCY_ENABLED", true),
		IdempotencyTTL:     env.Duration("IDEMPOTENCY_TTL", 24*time.Hour),

		// Circuit breaker defaults
		CircuitBreakerEnabled:          env.Bool("CIRCUIT_BREAKER_ENABLED", true),
		CircuitBreakerFailureThreshold: env.Int("CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5),
		CircuitBreakerSuccessThreshold: env.Int("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 3),
		CircuitBreakerTimeout:          env.Duration("CIRCUIT_BREAKER_TIMEOUT", 30*time.Second),

		// Retry defaults
		RetryMaxAttempts:  env.Int("RETRY_MAX_ATTEMPTS", 3),
		RetryInitialDelay: env.Duration("RETRY_INITIAL_DELAY", 100*time.Millisecond),
		RetryMaxDelay:     env.Duration("RETRY_MAX_DELAY", 5*time.Second),

		// Timeout defaults
		DynamoDBReadTimeout:  env.Duration("DYNAMODB_READ_TIMEOUT", 500*time.Millisecond),
		DynamoDBWriteTimeout: env.Duration("DYNAMODB_WRITE_TIMEOUT", 1*time.Second),

		// EventBridge defaults
		EventBridgeEnabled: env.Bool("EVENTBRIDGE_ENABLED", true),
		EventBridgeBusName: env.String("EVENTBRIDGE_BUS_NAME", "default"),
		EventBridgeSource:  env.String("EVENTBRIDGE_SOURCE", "cart-service"),

		// Feature flags defaults
		FeatureFlagsEnabled: env.Bool("FEATURE_FLAGS_ENABLED", false),

		// Metrics defaults
		MetricsEnabled: env.Bool("METRICS_ENABLED", false),

		// Secrets Manager defaults
		SecretsManagerEnabled: env.Bool("SECRETS_MANAGER_ENABLED", false),
		JWTSecretKey:          env.String("JWT_SECRET_KEY", ""),

		// Auth defaults
		AuthEnabled: env.Bool("AUTH_ENABLED", true),

		// CORS defaults
		CORSAllowedOrigins: env.StringSlice("CORS_ALLOWED_ORIGINS", defaultOrigins),
		CORSAllowedMethods: env.StringSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
		CORSAllowedHeaders: env.StringSlice("CORS_ALLOWED_HEADERS", []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "Idempotency-Key"}),

		// JWT defaults
		JWTIssuer:   env.String("JWT_ISSUER", ""),
		JWTAudience: env.String("JWT_AUDIENCE", ""),
	}

	if err := env.Err(); err != nil {
		return nil, fmt.Errorf("invalid environment: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks field constraints and cross-field security rules.
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	var errs []error
	if !c.AuthEnabled && !c.IsDevelopment() {
		errs = append(errs, errors.New("AUTH_ENABLED=false is only allowed when ENV_NAME=dev"))
	}
	if c.AuthEnabled && len(c.JWTSecretKey) < minJWTSecretBytes {
		errs = append(errs, fmt.Errorf("JWT_SECRET_KEY must be at least %d bytes when auth is enabled", minJWTSecretBytes))
	}
	if !c.IsDevelopment() && slices.Contains(c.CORSAllowedOrigins, "*") {
		errs = append(errs, errors.New("CORS_ALLOWED_ORIGINS must list explicit origins outside dev"))
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	return nil
}

// IsDevelopment returns true if running in development environment.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "dev"
}

// IsProduction returns true if running in production environment.
func (c *Config) IsProduction() bool {
	return c.Environment == "prod"
}

// envReader reads typed environment variables and collects parse errors.
type envReader struct {
	errs []error
}

// Err returns all parse errors encountered so far.
func (e *envReader) Err() error {
	return errors.Join(e.errs...)
}

func (e *envReader) fail(key, value string, err error) {
	e.errs = append(e.errs, fmt.Errorf("%s=%q: %w", key, value, err))
}

func (e *envReader) String(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (e *envReader) Int(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		e.fail(key, value, err)
		return defaultValue
	}
	return intValue
}

func (e *envReader) Int64(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		e.fail(key, value, err)
		return defaultValue
	}
	return intValue
}

func (e *envReader) Bool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		e.fail(key, value, err)
		return defaultValue
	}
	return boolValue
}

func (e *envReader) Duration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		e.fail(key, value, err)
		return defaultValue
	}
	return duration
}

// StringSlice parses a comma-separated list, trimming spaces and dropping empty entries.
func (e *envReader) StringSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var out []string
	for part := range strings.SplitSeq(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
