// Package logging provides structured JSON logging for the cart service.
package logging

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger with additional functionality.
type Logger struct {
	zl zerolog.Logger
}

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// Context keys for trace propagation
	traceIDKey     contextKey = "trace_id"
	requestIDKey   contextKey = "request_id"
	userIDKey      contextKey = "user_id"
	correlationKey contextKey = "correlation_id"
)

// Config holds logger configuration.
type Config struct {
	Level       string
	ServiceName string
	Environment string
	Output      io.Writer
}

// New creates a new Logger instance.
func New(cfg Config) *Logger {
	// Set output
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	// Parse log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DurationFieldUnit = time.Millisecond

	// Create base logger with service context
	zl := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Str("service_name", cfg.ServiceName).
		Str("environment", cfg.Environment).
		Logger()

	return &Logger{zl: zl}
}

// WithContext returns a new logger with context values.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	zl := l.zl.With().Logger()

	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		zl = zl.With().Str("trace_id", traceID).Logger()
	}

	if requestID, ok := ctx.Value(requestIDKey).(string); ok && requestID != "" {
		zl = zl.With().Str("request_id", requestID).Logger()
	}

	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		zl = zl.With().Str("user_id", userID).Logger()
	}

	if correlationID, ok := ctx.Value(correlationKey).(string); ok && correlationID != "" {
		zl = zl.With().Str("correlation_id", correlationID).Logger()
	}

	return &Logger{zl: zl}
}

// With returns a new logger with additional fields.
func (l *Logger) With() zerolog.Context {
	return l.zl.With()
}

// WithFields returns a new logger with the given fields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.zl.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{zl: ctx.Logger()}
}

// WithField returns a new logger with a single field.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{zl: l.zl.With().Interface(key, value).Logger()}
}

// WithError returns a new logger with the error field.
func (l *Logger) WithError(err error) *Logger {
	return &Logger{zl: l.zl.With().Err(err).Logger()}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string) {
	l.zl.Debug().Msg(msg)
}

// Debugf logs a formatted debug message.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.zl.Debug().Msgf(format, args...)
}

// Info logs an info message.
func (l *Logger) Info(msg string) {
	l.zl.Info().Msg(msg)
}

// Infof logs a formatted info message.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.zl.Info().Msgf(format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string) {
	l.zl.Warn().Msg(msg)
}

// Warnf logs a formatted warning message.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.zl.Warn().Msgf(format, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string) {
	l.zl.Error().Msg(msg)
}

// Errorf logs a formatted error message.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.zl.Error().Msgf(format, args...)
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(msg string) {
	l.zl.Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal message and exits.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.zl.Fatal().Msgf(format, args...)
}

// Panic logs a panic message and panics.
func (l *Logger) Panic(msg string) {
	l.zl.Panic().Msg(msg)
}

// LogRequest logs an HTTP request with standard fields.
func (l *Logger) LogRequest(ctx context.Context, method, path string, statusCode int, latency time.Duration, clientIP string) {
	l.WithContext(ctx).zl.Info().
		Str("method", method).
		Str("path", path).
		Int("status_code", statusCode).
		Dur("latency_ms", latency).
		Str("client_ip", clientIP).
		Msg("request completed")
}

// Context helper functions

// ContextWithTraceID returns a new context with the trace ID.
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// ContextWithRequestID returns a new context with the request ID.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// ContextWithUserID returns a new context with the user ID.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// ContextWithCorrelationID returns a new context with the correlation ID.
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationKey, correlationID)
}

// TraceIDFromContext extracts the trace ID from context.
func TraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// RequestIDFromContext extracts the request ID from context.
func RequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// UserIDFromContext extracts the user ID from context.
func UserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}
