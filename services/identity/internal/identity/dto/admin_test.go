package dto

import (
	"strings"
	"testing"
)

func TestUpdateAdminValidation(t *testing.T) {
	empty, badEmail, validEmail := "", "invalid", "admin@example.invalid"
	short, unicodeShort, tooLong := "1234567", "éééé", strings.Repeat("é", 37)
	active, badStatus := "active", "disabled"
	for _, tt := range []struct {
		name  string
		req   UpdateAdminRequest
		valid bool
	}{
		{"empty patch", UpdateAdminRequest{}, false},
		{"empty name allowed", UpdateAdminRequest{FullName: &empty}, true},
		{"empty email", UpdateAdminRequest{Email: &empty}, false},
		{"bad email", UpdateAdminRequest{Email: &badEmail}, false},
		{"valid email", UpdateAdminRequest{Email: &validEmail}, true},
		{"empty password", UpdateAdminRequest{Password: &empty}, false},
		{"short password", UpdateAdminRequest{Password: &short}, false},
		{"short Unicode", UpdateAdminRequest{Password: &unicodeShort}, false},
		{"byte limit", UpdateAdminRequest{Password: &tooLong}, false},
		{"empty status", UpdateAdminRequest{Status: &empty}, false},
		{"bad status", UpdateAdminRequest{Status: &badStatus}, false},
		{"valid status", UpdateAdminRequest{Status: &active}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateUpdateAdmin(tt.req); (err == nil) != tt.valid {
				t.Fatalf("validation error=%v, valid=%v", err, tt.valid)
			}
		})
	}
}
