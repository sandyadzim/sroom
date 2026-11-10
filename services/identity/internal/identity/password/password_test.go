package password

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	plainPassword := "Rahasia123!"

	hashedPassword, err := Hash(plainPassword)
	if err != nil {
		t.Fatalf("Hash() failed: %v", err)
	}

	// Pastikan password tidak disimpan sebagai plaintext.
	if hashedPassword == plainPassword {
		t.Fatal("hash must not equal plaintext")
	}

	// Pastikan format hash menggunakan Argon2id.
	if !strings.HasPrefix(hashedPassword, "$argon2id$") {
		t.Fatal("expected Argon2id hash format")
	}

	// Password benar harus lolos verifikasi.
	valid, err := Verify(plainPassword, hashedPassword)
	if err != nil {
		t.Fatalf("Verify() failed: %v", err)
	}

	if !valid {
		t.Fatal("correct password should be valid")
	}

	// Password salah harus ditolak.
	valid, err = Verify("PasswordSalah!", hashedPassword)
	if err != nil {
		t.Fatalf("Verify() failed: %v", err)
	}

	if valid {
		t.Fatal("incorrect password should be invalid")
	}
}

func TestHashUsesRandomSalt(t *testing.T) {
	password := "Rahasia123!"

	hash1, err := Hash(password)
	if err != nil {
		t.Fatal(err)
	}

	hash2, err := Hash(password)
	if err != nil {
		t.Fatal(err)
	}

	if hash1 == hash2 {
		t.Fatal("same password should produce different hashes")
	}
}

func TestVerifyInvalidHash(t *testing.T) {
	valid, err := Verify("Rahasia123!", "invalid-hash")

	if err == nil {
		t.Fatal("expected error for invalid hash")
	}

	if valid {
		t.Fatal("invalid hash must not be valid")
	}
}
