package godfather

import (
	"strings"
	"testing"
)

// ----------------------------------------------------------------
func TestGenerateHashAndVerifyPassword_Success(t *testing.T) {
	password := "mySecretPassword123!"

	hash, err := GenerateHash(password)
	if err != nil {
		t.Fatalf("GenerateHash failed: %v", err)
	}
	if !strings.HasPrefix(hash, "argon2id$") {
		t.Errorf("Hash does not start with expected prefix: %s", hash)
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !ok {
		t.Error("VerifyPassword returned false for correct password")
	}
}

// ----------------------------------------------------------------
func TestVerifyPassword_WrongPassword(t *testing.T) {
	password := "correctPassword"
	wrongPassword := "wrongPassword"

	hash, err := GenerateHash(password)
	if err != nil {
		t.Fatalf("GenerateHash failed: %v", err)
	}

	ok, err := VerifyPassword(wrongPassword, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if ok {
		t.Error("VerifyPassword returned true for incorrect password")
	}
}

// ----------------------------------------------------------------
func TestVerifyPassword_InvalidHashFormat(t *testing.T) {
	invalidHash := "notargon2id$1$2$3$4$5"
	ok, err := VerifyPassword("password", invalidHash)
	if err == nil {
		t.Error("Expected error for invalid hash format")
	}
	if ok {
		t.Error("VerifyPassword returned true for invalid hash format")
	}
}

func TestVerifyPassword_InvalidBase64Salt(t *testing.T) {
	// Valid format but invalid base64 salt
	invalidSaltHash := "argon2id$1$65536$4$invalidbase64$hash"
	ok, err := VerifyPassword("password", invalidSaltHash)
	if err == nil {
		t.Error("Expected error for invalid base64 salt")
	}
	if ok {
		t.Error("VerifyPassword returned true for invalid base64 salt")
	}
}

func TestGenerateHash_DifferentSalts(t *testing.T) {
	password := "samePassword"
	hash1, err := GenerateHash(password)
	if err != nil {
		t.Fatalf("GenerateHash failed: %v", err)
	}
	hash2, err := GenerateHash(password)
	if err != nil {
		t.Fatalf("GenerateHash failed: %v", err)
	}
	if hash1 == hash2 {
		t.Error("Hashes should be different for same password due to random salt")
	}
}
