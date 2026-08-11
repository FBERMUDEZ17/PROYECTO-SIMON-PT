package httpapi

import (
	"errors"
	"net/mail"
	"strings"

	"proyecto_simon_pt/internal/auth"
)

// TAREA (Prompt 4): DTOs y validación de las rutas HTTP de autenticación
// (email inválido, password muy corto, campos faltantes -> 400).

// MinPasswordLength es la longitud mínima aceptada para un password nuevo.
const MinPasswordLength = 8

var (
	ErrMissingFields    = errors.New("email, password y name son requeridos")
	ErrInvalidEmail     = errors.New("email inválido")
	ErrPasswordTooShort = errors.New("el password debe tener al menos 8 caracteres")
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type authResponse struct {
	User  userResponse `json:"user"`
	Token string       `json:"token"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func toUserResponse(u auth.User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, CreatedAt: u.CreatedAt}
}

// validateEmail rechaza cadenas vacías, sin "@", o con formato tipo
// "Nombre <email>" que net/mail acepta pero que no queremos como email crudo.
func validateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != strings.TrimSpace(email) {
		return ErrInvalidEmail
	}
	return nil
}

func validateRegisterRequest(req registerRequest) error {
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return ErrMissingFields
	}
	if err := validateEmail(req.Email); err != nil {
		return err
	}
	if len(req.Password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}

func validateLoginRequest(req loginRequest) error {
	if req.Email == "" || req.Password == "" {
		return ErrMissingFields
	}
	if err := validateEmail(req.Email); err != nil {
		return err
	}
	return nil
}
