package httpapi

// TAREA: "crear un restaurador de contraseña" — rutas públicas
// POST /auth/forgot-password y POST /auth/reset-password.

import (
	"errors"
	"log"
	"net/http"

	"proyecto_simon_pt/internal/auth"
)

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ForgotPassword maneja POST /auth/forgot-password. Responde 200 con un
// mensaje genérico exista o no el email (no revela qué emails están
// registrados); si existe, el token se loguea en la consola del servidor
// — no hay servicio de email configurado en este proyecto, así que esta
// es la forma de probar el flujo completo en desarrollo. En producción
// esto NUNCA debe ir a la respuesta HTTP ni a logs sin control de acceso:
// iría en el link de un email real.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo de la petición inválido: "+err.Error())
		return
	}
	if err := validateEmail(req.Email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, found, err := h.service.RequestPasswordReset(req.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error interno generando el token de restablecimiento")
		return
	}
	if found {
		log.Printf("[password-reset] token para %s: %s (expira en %s)", req.Email, token, auth.ResetTokenTTL)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "si el email está registrado, se generó un token de restablecimiento",
	})
}

// ResetPassword maneja POST /auth/reset-password: canjea el token por un
// password nuevo.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo de la petición inválido: "+err.Error())
		return
	}
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token es requerido")
		return
	}
	if len(req.NewPassword) < MinPasswordLength {
		writeError(w, http.StatusBadRequest, ErrPasswordTooShort.Error())
		return
	}

	err := h.service.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidResetToken) || errors.Is(err, auth.ErrResetTokenExpired) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "error interno restableciendo el password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password actualizado"})
}
