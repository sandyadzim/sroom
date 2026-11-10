package dto

import "testing"

func TestValidateRegister(t *testing.T) {
	tests := []struct {
		name    string
		request RegisterRequest
		wantErr bool
	}{
		{
			name: "valid registration",
			request: RegisterRequest{
				FullName: "Budi Santoso",
				Email:    "budi@example.com",
				Password: "Rahasia123!",
			},
			wantErr: false,
		},
		{
			name: "full name is optional",
			request: RegisterRequest{
				Email:    "budi@example.com",
				Password: "Rahasia123!",
			},
			wantErr: false,
		},
		{
			name: "email is required",
			request: RegisterRequest{
				Password: "Rahasia123!",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			request: RegisterRequest{
				Email:    "bukan-email",
				Password: "Rahasia123!",
			},
			wantErr: true,
		},
		{
			name: "password is required",
			request: RegisterRequest{
				Email: "budi@example.com",
			},
			wantErr: true,
		},
		{
			name: "password too short",
			request: RegisterRequest{
				Email:    "budi@example.com",
				Password: "1234567",
			},
			wantErr: true,
		},
		{
			name: "password exceeds 72 bytes",
			request: RegisterRequest{
				Email: "budi@example.com",
				Password: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegister(tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf(
					"ValidateRegister() error = %v, wantErr = %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}
