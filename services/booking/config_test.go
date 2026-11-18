package main

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Chdir(t.TempDir())
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
