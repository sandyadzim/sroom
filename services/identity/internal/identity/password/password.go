package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
    memory      uint32 = 64 * 1024
    iterations  uint32 = 3
    parallelism uint8  = 2
    saltLength         = 16
    keyLength  uint32  = 32
)

func Hash(plainPassword string) (string, error) {
    salt := make([]byte, saltLength)

    if _, err := rand.Read(salt); err != nil {
        return "", fmt.Errorf("generate salt: %w", err)
    }

    hash := argon2.IDKey(
        []byte(plainPassword),
        salt,
        iterations,
        memory,
        parallelism,
        keyLength,
    )

    encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
    encodedHash := base64.RawStdEncoding.EncodeToString(hash)

    encodedPassword := fmt.Sprintf(
        "$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
        memory,
        iterations,
        parallelism,
        encodedSalt,
        encodedHash,
    )

    return encodedPassword, nil
}


func Verify(plainPassword, encodedPassword string) (bool, error) {
    parts := strings.Split(encodedPassword, "$")

    if len(parts) != 6 ||
        parts[1] != "argon2id" ||
        parts[2] != "v=19" {
        return false, errors.New("invalid password hash format")
    }

    var memory uint32
    var iterations uint32
    var parallelism uint8

    _, err := fmt.Sscanf(
        parts[3],
        "m=%d,t=%d,p=%d",
        &memory,
        &iterations,
        &parallelism,
    )
    if err != nil {
        return false, fmt.Errorf("parse argon2 parameters: %w", err)
    }

    salt, err := base64.RawStdEncoding.DecodeString(parts[4])
    if err != nil {
        return false, fmt.Errorf("decode salt: %w", err)
    }

    expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
    if err != nil {
        return false, fmt.Errorf("decode hash: %w", err)
    }

    actualHash := argon2.IDKey(
        []byte(plainPassword),
        salt,
        iterations,
        memory,
        parallelism,
        uint32(len(expectedHash)),
    )

    return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1, nil
}
