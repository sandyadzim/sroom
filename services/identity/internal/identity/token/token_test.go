package token

import (
	"encoding/base64"
	"testing"
)

func TestGenerate(t *testing.T) {
	generatedToken, err := Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if generatedToken == "" {
		t.Fatal("generated token must not be empty")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(generatedToken)
	if err != nil {
		t.Fatalf("generated token is not valid base64url: %v", err)
	}

	if len(decoded) != tokenSize {
		t.Fatalf(
			"expected %d bytes, got %d",
			tokenSize,
			len(decoded),
		)
	}
}

func TestGenerateUniqueTokens(t *testing.T) {
	first, err := Generate()
	if err != nil {
		t.Fatalf("first Generate() failed: %v", err)
	}

	second, err := Generate()
	if err != nil {
		t.Fatalf("second Generate() failed: %v", err)
	}

	if first == second {
		t.Fatal("generated tokens must be different")
	}
}

func TestHash(t *testing.T) {
	plainToken := "example-token"

	firstHash := Hash(plainToken)
	secondHash := Hash(plainToken)

	if firstHash != secondHash {
		t.Fatal("hashing the same token must produce the same hash")
	}

	if len(firstHash) != 64 {
		t.Fatalf("expected hash length 64, got %d", len(firstHash))
	}

	if firstHash == plainToken {
		t.Fatal("hash must not equal the plain token")
	}

	if Hash("different-token") == firstHash {
		t.Fatal("different tokens must produce different hashes")
	}
}
