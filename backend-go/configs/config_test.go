package configs

import (
	"strings"
	"testing"
)

const strongSecret = "9f2c7a41d0b84e6f8c3a5b7e1d2f4a6c8e0b1d3f5a7c9e2b"

// setEnv sets a clean, production-valid environment; individual tests override keys.
func setEnv(t *testing.T, overrides map[string]string) {
	t.Helper()
	base := map[string]string{
		"ENVIRONMENT":        "production",
		"JWT_SECRET":         strongSecret,
		"DB_PASSWORD":        "s3cure-db-pass",
		"CORS_ORIGINS":       "https://lab.example.com",
		"GIN_MODE":           "",
		"COOKIE_SECURE":      "",
		"TRUSTED_PROXIES":    "",
		"MAX_SESSIONS_TOTAL": "",
		"DOCKER_HOST":        "",
		"DOCKER_SOCKET":      "",
		"JWT_EXPIRATION":     "",
	}
	for k, v := range overrides {
		base[k] = v
	}
	for k, v := range base {
		t.Setenv(k, v)
	}
}

func TestProductionValidConfig(t *testing.T) {
	setEnv(t, nil)
	cfg := Load()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
	if !cfg.Security.CookieSecure {
		t.Error("COOKIE_SECURE must default to true")
	}
	if cfg.Docker.MaxSessionsTotal != 20 {
		t.Errorf("MaxSessionsTotal = %d, want 20", cfg.Docker.MaxSessionsTotal)
	}
	if len(cfg.Security.TrustedProxies) == 0 {
		t.Error("TrustedProxies default must not be empty")
	}
	if cfg.Docker.Host != "unix:///var/run/docker.sock" {
		t.Errorf("Docker.Host = %q", cfg.Docker.Host)
	}
	if cfg.Server.GinMode != "release" {
		t.Errorf("production GinMode = %q, want release", cfg.Server.GinMode)
	}
}

func TestProductionRejectsInsecureSettings(t *testing.T) {
	cases := map[string]struct {
		env  map[string]string
		want string
	}{
		"missing jwt secret":   {map[string]string{"JWT_SECRET": ""}, "JWT_SECRET"},
		"short jwt secret":     {map[string]string{"JWT_SECRET": "too-short"}, "JWT_SECRET"},
		"placeholder secret":   {map[string]string{"JWT_SECRET": "your-super-secret-jwt-key-change-in-production"}, "placeholder"},
		"changeme in secret":   {map[string]string{"JWT_SECRET": "changeme-changeme-changeme-changeme-1"}, "placeholder"},
		"default db password":  {map[string]string{"DB_PASSWORD": "devopslab_secret"}, "DB_PASSWORD"},
		"postgres db password": {map[string]string{"DB_PASSWORD": "postgres"}, "DB_PASSWORD"},
		"password db password": {map[string]string{"DB_PASSWORD": "password"}, "DB_PASSWORD"},
		"unset db password":    {map[string]string{"DB_PASSWORD": ""}, "DB_PASSWORD"}, // falls back to default
		"wildcard cors":        {map[string]string{"CORS_ORIGINS": "https://lab.example.com,*"}, "CORS_ORIGINS"},
		"bad gin mode":         {map[string]string{"GIN_MODE": "verbose", "ENVIRONMENT": "development"}, "GIN_MODE"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			setEnv(t, tc.env)
			err := Load().Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

func TestDevelopmentIsLenient(t *testing.T) {
	setEnv(t, map[string]string{
		"ENVIRONMENT":  "development",
		"JWT_SECRET":   "",
		"DB_PASSWORD":  "devopslab_secret",
		"CORS_ORIGINS": "http://localhost:3000",
		"GIN_MODE":     "debug",
	})
	cfg := Load()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development config rejected: %v", err)
	}
	if !cfg.JWT.Generated || len(cfg.JWT.Secret) < MinJWTSecretLength {
		t.Fatalf("expected generated ephemeral secret, got generated=%v len=%d", cfg.JWT.Generated, len(cfg.JWT.Secret))
	}
	if cfg.Server.GinMode != "debug" {
		t.Errorf("GinMode = %q", cfg.Server.GinMode)
	}
}

func TestDockerHostAndLists(t *testing.T) {
	setEnv(t, map[string]string{
		"DOCKER_HOST":     "tcp://docker-proxy:2375",
		"TRUSTED_PROXIES": "10.0.0.0/8, 172.16.0.0/12 ,",
		"CORS_ORIGINS":    "https://a.example.com/, https://b.example.com",
		"COOKIE_SECURE":   "false",
		"JWT_EXPIRATION":  "9999h",
	})
	cfg := Load()
	if cfg.Docker.Host != "tcp://docker-proxy:2375" {
		t.Errorf("DOCKER_HOST not preferred: %q", cfg.Docker.Host)
	}
	if got := strings.Join(cfg.Security.TrustedProxies, "|"); got != "10.0.0.0/8|172.16.0.0/12" {
		t.Errorf("TrustedProxies = %q", got)
	}
	if got := strings.Join(cfg.CORS.Origins, "|"); got != "https://a.example.com|https://b.example.com" {
		t.Errorf("Origins = %q", got)
	}
	if cfg.Security.CookieSecure {
		t.Error("COOKIE_SECURE=false ignored")
	}
	if cfg.JWT.Expiration != maxJWTExpiration {
		t.Errorf("JWT expiration not capped: %v", cfg.JWT.Expiration)
	}
}
