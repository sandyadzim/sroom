package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const tokenSize = 32

func Generate() (string, error) {
	randomBytes := make([]byte, tokenSize)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func Hash(plainToken string) string {
	sum := sha256.Sum256([]byte(plainToken))

	return hex.EncodeToString(sum[:])
}
