package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"

	"identity/internal/identity/dto"
)

func TestSeedStartupDoesNotLogDSN(t *testing.T) {
	if os.Getenv("IDENTITY_SEED_LOG_TEST") == "1" {
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSeedStartupDoesNotLogDSN$")
	cmd.Dir = t.TempDir() // Never load the developer's .env.
	cmd.Env = []string{
		"IDENTITY_SEED_LOG_TEST=1",
		"DATABASE_URL=postgres://synthetic-user:synthetic-secret@synthetic-host/identity?port=invalid",
		"OWNER_EMAIL=synthetic-owner@example.invalid", "OWNER_FULL_NAME=synthetic-private-owner", "OWNER_PASSWORD=synthetic-private-password",
	}
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("startup with invalid DSN must fail")
	}
	for _, value := range []string{"synthetic-user", "synthetic-secret", "synthetic-host", "synthetic-owner", "synthetic-private"} {
		if bytes.Contains(output, []byte(value)) {
			t.Fatal("startup logs contain sensitive config")
		}
	}
	if !bytes.Contains(output, []byte(`"reason":"seed_failed"`)) {
		t.Fatal("expected safe startup failure reason")
	}
}

func TestSeedUsesRegisterValidation(t *testing.T) {
	t.Chdir(t.TempDir()) // Never load the developer's .env or connect to a real database.
	t.Setenv("DATABASE_URL", "postgres://synthetic-user:synthetic-secret@synthetic-host/identity?port=invalid")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tt := range []struct {
		name, fullName, email, password, invalidField string
		tooLongPassword                               bool
	}{
		{name: "valid ASCII", fullName: "Synthetic Owner", email: " OWNER@EXAMPLE.INVALID ", password: "12345678"},
		{name: "valid Unicode", fullName: strings.Repeat("é", 100), email: "owner@example.invalid", password: "éééééééé"},
		{name: "72 byte password", fullName: "Owner", email: "owner@example.invalid", password: strings.Repeat("é", 36)},
		{name: "invalid email", fullName: "Owner", email: "invalid-email", password: "12345678", invalidField: "Email"},
		{name: "short Unicode password", fullName: "Owner", email: "owner@example.invalid", password: "éééé", invalidField: "Password"},
		{name: "long name", fullName: strings.Repeat("é", 101), email: "owner@example.invalid", password: "12345678", invalidField: "FullName"},
		{name: "password exceeds bytes", fullName: "Owner", email: "owner@example.invalid", password: strings.Repeat("é", 37), tooLongPassword: true},
		{name: "password spaces preserved", fullName: "Owner", email: "owner@example.invalid", password: " abcdef "},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OWNER_FULL_NAME", tt.fullName)
			t.Setenv("OWNER_EMAIL", tt.email)
			t.Setenv("OWNER_PASSWORD", tt.password)
			err := run(logger)
			if err == nil {
				t.Fatal("invalid DSN must prevent database connection")
			}
			if tt.invalidField != "" {
				var fieldErrors validator.ValidationErrors
				if !errors.As(err, &fieldErrors) || fieldErrors[0].Field() != tt.invalidField {
					t.Fatal("expected the same field validation as register")
				}
			} else if tt.tooLongPassword {
				if !errors.Is(err, dto.ErrPasswordTooLong) {
					t.Fatal("expected register's password byte limit")
				}
			} else if !strings.HasPrefix(err.Error(), "connect database:") {
				t.Fatal("valid input must pass validation and reach DSN parsing")
			}
		})
	}
	t.Setenv("OWNER_EMAIL", "owner@example.invalid")
	t.Setenv("OWNER_PASSWORD", "12345678")
	t.Setenv("OWNER_FULL_NAME", " ")
	if err := run(logger); err == nil || err.Error() != "OWNER_FULL_NAME is required" {
		t.Fatal("seed must still require an OWNER name")
	}
}
