package godfather

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parameters (adjust based on your security requirements)
const (
	argonTime = 1         // Number of iterations
	memory    = 64 * 1024 // Memory usage in KiB (64MB)
	threads   = 4         // Number of threads
	keyLen    = 32        // Length of the generated key
)

// ----------------------------------------------------------------
// Generate a hashed version of the password using Argon2
// ----------------------------------------------------------------
func GenerateHash(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Generate hash
	hash := argon2.IDKey([]byte(password), salt, argonTime, memory, threads, keyLen)

	// Encode hash and salt for storage
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: algorithm$time$memory$threads$salt$hash
	return fmt.Sprintf("argon2id$%d$%d$%d$%s$%s",
		argonTime, memory, threads, b64Salt, b64Hash), nil
}

// ----------------------------------------------------------------
// Verify a password against a stored hash
// ----------------------------------------------------------------
func VerifyPassword(password, storedHash string) (bool, error) {
	parts := strings.Split(storedHash, "$")
	if len(parts) != 6 || parts[0] != "argon2id" {
		return false, errors.New("invalid hash format")
	}

	// Parse parameters
	var time, memory uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[1], "%d", &time); err != nil {
		return false, errors.New("invalid time parameter")
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &memory); err != nil {
		return false, errors.New("invalid memory parameter")
	}
	if _, err := fmt.Sscanf(parts[3], "%d", &threads); err != nil {
		return false, errors.New("invalid threads parameter")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	// Regenerate hash with stored parameters
	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return b64Hash == parts[5], nil
}
