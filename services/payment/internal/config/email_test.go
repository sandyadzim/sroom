package config

import "testing"

func TestEmailConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DATABASE_URL", "postgres://localhost/synthetic")
	t.Setenv("RABBITMQ_URL", "")
	t.Setenv("APP_PORT", "")
	for _, test := range []struct {
		name, smtp, from, web string
		valid                 bool
	}{
		{"disabled defaults", "", "", "", true}, {"local Mailpit", "127.0.0.1:1025", "", "", true},
		{"custom", "mailpit.internal:1025", "SROOM <sender@example.test>", "https://web.example.test/", true},
		{"bad SMTP", "smtp://mailpit:1025", "", "", false}, {"bad port", "mailpit:0", "", "", false},
		{"header injection", "", "sender@example.test\r\nBcc: other@example.test", "", false},
		{"URL credentials", "", "", "https://user:secret@web.example.test", false}, {"URL path", "", "", "http://localhost:5173/path", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SMTP_ADDR", test.smtp)
			t.Setenv("MAIL_FROM", test.from)
			t.Setenv("WEB_BASE_URL", test.web)
			cfg, err := Load()
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v error=%v", test.valid, err)
			}
			if !test.valid {
				return
			}
			if cfg.SMTPAddress != test.smtp {
				t.Fatal("SMTP config changed")
			}
			if test.web == "" && cfg.WebBaseURL != "http://localhost:5173" {
				t.Fatal("missing frontend default")
			}
			if test.from == "" && cfg.MailFrom != "SROOM <noreply@sroom.test>" {
				t.Fatal("missing sender default")
			}
		})
	}
}
