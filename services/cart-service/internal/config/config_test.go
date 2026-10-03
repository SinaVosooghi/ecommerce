package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validSecret = "0123456789abcdef0123456789abcdef"

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", validSecret)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
	assert.True(t, cfg.AuthEnabled)
	assert.Equal(t, []string{"*"}, cfg.CORSAllowedOrigins, "dev allows any origin by default")
}

func TestLoad_MalformedValuesAreErrors(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", validSecret)
	t.Setenv("APP_PORT", "abc")
	t.Setenv("IDEMPOTENCY_ENABLED", "maybe")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_PORT")
	assert.Contains(t, err.Error(), "IDEMPOTENCY_ENABLED")
}

func TestLoad_SecurityRules(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name:    "auth enabled requires a secret",
			env:     map[string]string{"JWT_SECRET_KEY": ""},
			wantErr: "JWT_SECRET_KEY",
		},
		{
			name:    "short secret",
			env:     map[string]string{"JWT_SECRET_KEY": "too-short"},
			wantErr: "JWT_SECRET_KEY",
		},
		{
			name:    "auth can be disabled in dev",
			env:     map[string]string{"AUTH_ENABLED": "false"},
			wantErr: "",
		},
		{
			name:    "auth cannot be disabled outside dev",
			env:     map[string]string{"ENV_NAME": "prod", "AUTH_ENABLED": "false", "JWT_SECRET_KEY": validSecret},
			wantErr: "AUTH_ENABLED",
		},
		{
			name:    "wildcard CORS rejected outside dev",
			env:     map[string]string{"ENV_NAME": "prod", "JWT_SECRET_KEY": validSecret, "CORS_ALLOWED_ORIGINS": "*"},
			wantErr: "CORS_ALLOWED_ORIGINS",
		},
		{
			name:    "prod defaults to no CORS",
			env:     map[string]string{"ENV_NAME": "prod", "JWT_SECRET_KEY": validSecret},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			_, err := Load()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLoad_StringSliceTrimsEntries(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", validSecret)
	t.Setenv("ENV_NAME", "prod")
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://a.example , ,https://b.example")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, []string{"https://a.example", "https://b.example"}, cfg.CORSAllowedOrigins)
}
