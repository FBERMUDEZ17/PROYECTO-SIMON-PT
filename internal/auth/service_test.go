// TAREA: tests del servicio de auth (Prompt 3) — registro exitoso/email
// duplicado, login exitoso/password incorrecto/email inexistente/email en
// mayúsculas (los 6 casos pedidos explícitamente).
package auth

import (
	"database/sql"
	"errors"
	"testing"

	"proyecto_simon_pt/internal/db"
)

func newTestService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("abriendo db de test: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	return NewService(database, []byte("test-secret")), database
}

func TestRegister_Success(t *testing.T) {
	svc, _ := newTestService(t)

	user, token, err := svc.Register("Alice@Example.com", "s3cret-pw", "Alice")
	if err != nil {
		t.Fatalf("Register devolvió error inesperado: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email no normalizado: got %q", user.Email)
	}
	if user.ID == 0 {
		t.Errorf("se esperaba un ID asignado")
	}
	if token == "" {
		t.Errorf("se esperaba un JWT no vacío")
	}

	claims, err := VerifyToken([]byte("test-secret"), token, svc.now())
	if err != nil {
		t.Fatalf("el token emitido no verifica: %v", err)
	}
	if claims.Email != "alice@example.com" {
		t.Errorf("claims.Email = %q, want alice@example.com", claims.Email)
	}
	if claims.Exp-claims.Iat != int64(TokenTTL.Seconds()) {
		t.Errorf("el token no expira en 24h: iat=%d exp=%d", claims.Iat, claims.Exp)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, _ := newTestService(t)

	if _, _, err := svc.Register("bob@example.com", "password1", "Bob"); err != nil {
		t.Fatalf("primer registro falló: %v", err)
	}

	_, _, err := svc.Register("BOB@example.com", "otraPassword", "Bob2")
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("esperaba ErrEmailAlreadyExists, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	svc, _ := newTestService(t)
	if _, _, err := svc.Register("carol@example.com", "correct-pw", "Carol"); err != nil {
		t.Fatalf("registro falló: %v", err)
	}

	user, token, err := svc.Login("carol@example.com", "correct-pw")
	if err != nil {
		t.Fatalf("Login devolvió error inesperado: %v", err)
	}
	if user.Email != "carol@example.com" {
		t.Errorf("email inesperado: %q", user.Email)
	}
	if token == "" {
		t.Errorf("se esperaba un JWT no vacío")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _ := newTestService(t)
	if _, _, err := svc.Register("dave@example.com", "correct-pw", "Dave"); err != nil {
		t.Fatalf("registro falló: %v", err)
	}

	_, _, err := svc.Login("dave@example.com", "wrong-pw")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("esperaba ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_NonexistentEmail(t *testing.T) {
	svc, _ := newTestService(t)

	_, _, err := svc.Login("nadie@example.com", "cualquiera")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("esperaba ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_EmailUppercase(t *testing.T) {
	svc, _ := newTestService(t)
	if _, _, err := svc.Register("erin@example.com", "correct-pw", "Erin"); err != nil {
		t.Fatalf("registro falló: %v", err)
	}

	user, token, err := svc.Login("ERIN@EXAMPLE.COM", "correct-pw")
	if err != nil {
		t.Fatalf("Login con email en mayúsculas debería funcionar, got: %v", err)
	}
	if user.Email != "erin@example.com" {
		t.Errorf("email no normalizado en login: %q", user.Email)
	}
	if token == "" {
		t.Errorf("se esperaba un JWT no vacío")
	}
}

func TestListUsers_ReturnsAllRegisteredUsers(t *testing.T) {
	svc, _ := newTestService(t)

	if _, _, err := svc.Register("bob@example.com", "correct-pw", "Bob"); err != nil {
		t.Fatalf("registro falló: %v", err)
	}
	if _, _, err := svc.Register("alice@example.com", "correct-pw", "Alice"); err != nil {
		t.Fatalf("registro falló: %v", err)
	}

	users, err := svc.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
	// Ordenado por email.
	if users[0].Email != "alice@example.com" || users[1].Email != "bob@example.com" {
		t.Errorf("orden inesperado: %+v", users)
	}
}

func TestGetUserByID_ExistingAndMissing(t *testing.T) {
	svc, _ := newTestService(t)

	created, _, err := svc.Register("owner@example.com", "correct-pw", "Owner")
	if err != nil {
		t.Fatalf("registro falló: %v", err)
	}

	found, err := svc.GetUserByID(created.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if found.Email != "owner@example.com" {
		t.Errorf("email = %q, want owner@example.com", found.Email)
	}

	if _, err := svc.GetUserByID(999999); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("got %v, want sql.ErrNoRows", err)
	}
}
