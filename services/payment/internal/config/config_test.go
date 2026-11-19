package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("RABBITMQ_URL", "")
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
				t.Fatalf("unexpected port/database configuration: %v", err)
			}
		})
	}
}

func TestDotEnvPreservesEnvironment(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("RABBITMQ_URL", "")
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
		t.Fatalf("unexpected dotenv configuration: %v", err)
	}
}

func TestInvalidDotEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("RABBITMQ_URL", "")
	if err := os.WriteFile(".env", []byte("APP_PORT=\"unterminated"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil || err.Error() != "could not load .env" {
		t.Fatalf("invalid .env must fail safely, got %v", err)
	}
}

func TestIntegrationConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("RABBITMQ_URL", "")
	t.Setenv("DATABASE_URL", "postgres://localhost/payment-test")
	t.Setenv("APP_PORT", "")
	t.Setenv("IDENTITY_BASE_URL", "")
	t.Setenv("BOOKING_BASE_URL", "")
	t.Setenv("PAYMENT_BOOKING_TOKEN", "")
	t.Setenv("MIDTRANS_SERVER_KEY", "")
	cfg, err := Load()
	if err != nil || cfg.IdentityBaseURL != "http://localhost:8081" || cfg.BookingBaseURL != "http://localhost:8083" || cfg.BookingToken != "" || cfg.MidtransServerKey != "" {
		t.Fatal("invalid integration defaults")
	}
	t.Setenv("PAYMENT_BOOKING_TOKEN", " test-only-pair-token ")
	t.Setenv("MIDTRANS_SERVER_KEY", " test-only-key ")
	cfg, err = Load()
	if err != nil || cfg.BookingToken != "test-only-pair-token" || cfg.MidtransServerKey != "test-only-key" {
		t.Fatal("integration config was not loaded correctly")
	}
}

func TestRabbitMQConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DATABASE_URL", "postgres://localhost/payment-test")
	t.Setenv("APP_PORT", "")
	for _, value := range []string{"", "amqp://demo:synthetic-password@localhost:5672/sroom", "amqps://localhost/sroom", "https://localhost/sroom", "amqp:///sroom", "amqp://demo:synthetic-password@localhost:bad/sroom"} {
		t.Setenv("RABBITMQ_URL", value)
		cfg, err := Load()
		invalid := value == "https://localhost/sroom" || value == "amqp:///sroom" || value == "amqp://demo:synthetic-password@localhost:bad/sroom"
		if invalid {
			if err == nil || strings.Contains(err.Error(), "synthetic-password") {
				t.Fatal("invalid broker URL not safely rejected")
			}
		} else if err != nil || cfg.RabbitMQURL != value {
			t.Fatal("valid broker config rejected")
		}
	}
}
