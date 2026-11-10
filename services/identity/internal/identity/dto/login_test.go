package dto

import (
	"testing"
)

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name    string
		request LoginRequest
		wantErr bool
	}{
		{
			name: "valid login",
			request: LoginRequest{
				Email:    "budi@example.com",
				Password: "Rahasia123!",
			},
			wantErr: false,
		},
		{
			name: "email required",
			request: LoginRequest{
				Email:    "",
				Password: "Rahasia123!",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			request: LoginRequest{
				Email:    "budi-email-salah",
				Password: "Rahasia123!",
			},
			wantErr: true,
		},
		{
			name: "password required",
			request: LoginRequest{
				Email:    "budi@example.com",
				Password: "",
			},
			wantErr: true,
		},
		{
			name: "short password allowed at login",
			request: LoginRequest{
				Email:    "budi@example.com",
				Password: "123",
			},
			wantErr: false,
		},
		{
			name: "email exceeds 255 characters",
			request: LoginRequest{
				Email:    string(make([]byte, 250)) + "@example.com",
				Password: "Rahasia123!",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLogin(tt.request)

			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
