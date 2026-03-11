package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("testpassword")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword() returned empty hash")
	}
	if hash == "testpassword" {
		t.Fatal("HashPassword() returned plaintext password")
	}
}

func TestCheckPassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if !CheckPassword("mypassword", hash) {
		t.Error("CheckPassword() should return true for correct password")
	}

	if CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword() should return false for wrong password")
	}
}

func TestGenerateJWT(t *testing.T) {
	secret := "test-secret"
	token, err := GenerateJWT(secret, "user-123", true)
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateJWT() returned empty token")
	}
}

func TestValidateJWT(t *testing.T) {
	secret := "test-secret"
	token, err := GenerateJWT(secret, "user-456", false)
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}

	claims, err := ValidateJWT(secret, token)
	if err != nil {
		t.Fatalf("ValidateJWT() error = %v", err)
	}

	if claims.UserID != "user-456" {
		t.Errorf("claims.UserID = %q, want %q", claims.UserID, "user-456")
	}
	if claims.IsAdmin {
		t.Error("claims.IsAdmin should be false")
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	token, _ := GenerateJWT("secret-1", "user-789", false)

	_, err := ValidateJWT("secret-2", token)
	if err == nil {
		t.Error("ValidateJWT() should fail with wrong secret")
	}
}

func TestValidateJWT_InvalidToken(t *testing.T) {
	_, err := ValidateJWT("secret", "not-a-real-token")
	if err == nil {
		t.Error("ValidateJWT() should fail with invalid token")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	key1, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}
	if len(key1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("GenerateAPIKey() length = %d, want 64", len(key1))
	}

	key2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}
	if key1 == key2 {
		t.Error("GenerateAPIKey() should produce unique keys")
	}
}
