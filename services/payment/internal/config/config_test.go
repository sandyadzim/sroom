package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, test := range []struct {
		name        string
		port        string
		databaseURL string
		wantPort    int
		wantError   string
	}{
		{name: "default port", databaseURL: "postgres://localhost/payment-test", wantPort: 8084},
		{name: "explicit port", port: "8184", databaseURL: "postgres://localhost/payment-test", wantPort: 8184},
		{name: "invalid port", port: "abc", databaseURL: "postgres://localhost/payment-test", wantError: "APP_PORT must be between 1 and 65535"},
		{name: "zero port", port: "0", databaseURL: "postgres://localhost/payment-test", wantError: "APP_PORT must be between 1 and 65535"},
		{name: "negative port", port: "-1", databaseURL: "postgres://localhost/payment-test", wantError: "APP_PORT must be between 1 and 65535"},
		{name: "port out of range", port: "65536", databaseURL: "postgres://localhost/payment-test", wantError: "APP_PORT must be between 1 and 65535"},
		{name: "missing database URL", wantError: "DATABASE_URL is required"},
		{name: "blank database URL", databaseURL: "   ", wantError: "DATABASE_URL is required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("APP_PORT", test.port)
			t.Setenv("DATABASE_URL", test.databaseURL)
			cfg, err := Load()
			if test.wantError != "" {
				if err == nil || err.Error() != test.wantError {
					t.Fatalf("error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil || cfg.AppPort != test.wantPort || cfg.DatabaseURL != test.databaseURL {
				t.Fatalf("Load() = %+v, %v", cfg, err)
			}
		})
	}
}

func TestDotEnvPreservesEnvironment(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("APP_PORT", "")
	if err := os.Unsetenv("APP_PORT"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "postgres://localhost/from-environment")
	if err := os.WriteFile(".env", []byte("APP_PORT=8184\nDATABASE_URL=postgres://localhost/from-file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || cfg.AppPort != 8184 || cfg.DatabaseURL != "postgres://localhost/from-environment" {
		t.Fatalf("Load() = %+v, %v", cfg, err)
	}
}

func TestInvalidDotEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".env", []byte("APP_PORT=\"unterminated"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil || err.Error() != "could not load .env" {
		t.Fatalf("invalid .env must fail safely, got %v", err)
	}
}
