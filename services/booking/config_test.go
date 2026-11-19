package main

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("RABBITMQ_URL", "")
	const testDatabaseURL = "postgres://localhost/rent-car-booking"

	for _, test := range []struct {
		name        string
		port        string
		databaseURL string
		wantPort    int
		wantError   string
	}{
		{name: "default port", databaseURL: testDatabaseURL, wantPort: 8083},
		{name: "explicit port", port: "8183", databaseURL: testDatabaseURL, wantPort: 8183},
		{name: "invalid port", port: "abc", databaseURL: testDatabaseURL, wantError: "APP_PORT must be between 1 and 65535"},
		{name: "port out of range", port: "65536", databaseURL: testDatabaseURL, wantError: "APP_PORT must be between 1 and 65535"},
		{name: "missing database URL", wantError: "DATABASE_URL is required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("APP_PORT", test.port)
			t.Setenv("DATABASE_URL", test.databaseURL)
			cfg, err := loadConfig()
			if test.wantError != "" {
				if err == nil || err.Error() != test.wantError {
					t.Fatalf("error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil || cfg.appPort != test.wantPort || cfg.databaseURL != test.databaseURL {
				t.Fatalf("loadConfig() = %+v, %v", cfg, err)
			}
		})
	}
}

func TestLoadDotEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("RABBITMQ_URL", "")
	t.Setenv("APP_PORT", "")
	if err := os.Unsetenv("APP_PORT"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "postgres://localhost/from-environment")
	if err := os.WriteFile(".env", []byte("APP_PORT=8183\nDATABASE_URL=postgres://localhost/from-file\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.appPort != 8183 || cfg.databaseURL != "postgres://localhost/from-environment" {
		t.Fatalf("loadConfig() = %+v", cfg)
	}
}

func TestDemoPaymentsConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("RABBITMQ_URL", "")
	t.Setenv("DATABASE_URL", "postgres://localhost/booking-test")
	t.Setenv("APP_PORT", "8083")
	for _, value := range []string{"", "false", "true", "TRUE", "invalid"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("BOOKING_DEMO_PAYMENTS", value)
			cfg, err := loadConfig()
			if err != nil || cfg.demoPayments != (value == "true") {
				t.Fatalf("demo payments must be enabled only by true (%v)", err)
			}
		})
	}
}

func TestPaymentConsumerConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DATABASE_URL", "postgres://localhost/booking-test")
	t.Setenv("APP_PORT", "")
	t.Setenv("PAYMENT_BASE_URL", "")
	t.Setenv("PAYMENT_BOOKING_TOKEN", "")
	t.Setenv("BOOKING_DEMO_PAYMENTS", "false")
	t.Setenv("RABBITMQ_URL", "")
	cfg, err := loadConfig()
	if err != nil || cfg.paymentURL != "http://localhost:8084" || cfg.rabbitURL != "" {
		t.Fatal("invalid consumer defaults")
	}
	t.Setenv("RABBITMQ_URL", "amqp://local:synthetic-password@localhost:5672/sroom")
	if _, err := loadConfig(); err == nil {
		t.Fatal("consumer accepted missing pair token")
	}
	t.Setenv("PAYMENT_BOOKING_TOKEN", "synthetic-pair")
	t.Setenv("BOOKING_DEMO_PAYMENTS", "true")
	if _, err := loadConfig(); err == nil {
		t.Fatal("consumer accepted unsafe demo expiry mode")
	}
	t.Setenv("BOOKING_DEMO_PAYMENTS", "false")
	if _, err := loadConfig(); err != nil {
		t.Fatal(err)
	}
}
