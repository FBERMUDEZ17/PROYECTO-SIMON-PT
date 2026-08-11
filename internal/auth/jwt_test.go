// TAREA: tests del JWT manual (Prompt 1) — round-trip, firma alterada,
// secreto incorrecto, expiración, algoritmo no soportado, malformado.
package auth

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateAndVerifyToken_RoundTrip(t *testing.T) {
	secret := []byte("s3cr3t")
	now := time.Now()

	token, err := GenerateToken(secret, "42", "user@example.com", "User", "user", now)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}

	claims, err := VerifyToken(secret, token, now)
	if err != nil {
		t.Fatalf("VerifyToken error: %v", err)
	}
	if claims.Sub != "42" || claims.Email != "user@example.com" {
		t.Errorf("claims inesperados: %+v", claims)
	}
}

func TestVerifyToken_RejectsTamperedSignature(t *testing.T) {
	secret := []byte("s3cr3t")
	now := time.Now()

	token, _ := GenerateToken(secret, "1", "a@b.com", "A", "user", now)
	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + parts[1] + ".AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

	if _, err := VerifyToken(secret, tampered, now); err != ErrInvalidSignature {
		t.Fatalf("esperaba ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyToken_RejectsWrongSecret(t *testing.T) {
	now := time.Now()
	token, _ := GenerateToken([]byte("secret-a"), "1", "a@b.com", "A", "user", now)

	if _, err := VerifyToken([]byte("secret-b"), token, now); err != ErrInvalidSignature {
		t.Fatalf("esperaba ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyToken_RejectsExpiredToken(t *testing.T) {
	secret := []byte("s3cr3t")
	issuedAt := time.Now().Add(-25 * time.Hour) // emitido hace 25h -> ya expiró (TTL 24h)

	token, _ := GenerateToken(secret, "1", "a@b.com", "A", "user", issuedAt)

	if _, err := VerifyToken(secret, token, time.Now()); err != ErrTokenExpired {
		t.Fatalf("esperaba ErrTokenExpired, got %v", err)
	}
}

func TestVerifyToken_RejectsUnsupportedAlgorithm(t *testing.T) {
	secret := []byte("s3cr3t")
	now := time.Now()

	token, _ := GenerateToken(secret, "1", "a@b.com", "A", "user", now)
	parts := strings.Split(token, ".")

	// Header con alg "none" (base64url de {"alg":"none","typ":"JWT"})
	noneHeader := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0"
	forged := noneHeader + "." + parts[1] + "." + parts[2]

	if _, err := VerifyToken(secret, forged, now); err != ErrUnsupportedAlgorithm {
		t.Fatalf("esperaba ErrUnsupportedAlgorithm, got %v", err)
	}
}

func TestVerifyToken_RejectsMalformedToken(t *testing.T) {
	if _, err := VerifyToken([]byte("s"), "no-es-un-jwt", time.Now()); err != ErrMalformedToken {
		t.Fatalf("esperaba ErrMalformedToken, got %v", err)
	}
}
