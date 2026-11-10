package main

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func TestStartupLoggingDoesNotExposeConfig(t *testing.T) {
	if os.Getenv("IDENTITY_API_LOG_TEST") == "1" {
		main()
		return
	}
	for _, port := range []string{"8081", "synthetic-private-invalid-port"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestStartupLoggingDoesNotExposeConfig$")
		cmd.Dir = t.TempDir() // Isolate .env loading; DSN parsing fails before networking.
		cmd.Env = []string{
			"IDENTITY_API_LOG_TEST=1", "APP_PORT=" + port,
			"DATABASE_URL=postgres://synthetic-user:synthetic-secret@synthetic-host/identity?port=invalid",
		}
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("startup with invalid config must fail")
		}
		for _, value := range []string{"synthetic-user", "synthetic-secret", "synthetic-host", "synthetic-private"} {
			if bytes.Contains(output, []byte(value)) {
				t.Fatal("startup logs contain sensitive config")
			}
		}
		if !bytes.Contains(output, []byte(`"operation":`)) || !bytes.Contains(output, []byte(`"reason":`)) {
			t.Fatal("startup must retain a safe operation and reason")
		}
	}
}
