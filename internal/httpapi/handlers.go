package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"proyecto_simon_pt/internal/auth"
)

const maxRequestBody = 1 << 20 // 1 MiB

// Handler agrupa las rutas HTTP de autenticación.
type Handler struct {
	service *auth.Service
}

func NewHandler(service *auth.Service) *Handler {
	return &Handler{service: service}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// TAREA (Prompt 4): rutas HTTP de autenticación — devuelven usuario+token,
// con códigos de error apropiados (400 validación, 409 duplicado, 401
// credenciales). TAREA (frontend): además setean la cookie httpOnly.

// Register maneja POST /auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo de la petición inválido: "+err.Error())
		return
	}

	if err := validateRegisterRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, token, err := h.service.Register(req.Email, req.Password, req.Name)
	if err != nil {
		if errors.Is(err, auth.ErrEmailAlreadyExists) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "error interno registrando el usuario")
		return
	}

	setAuthCookie(w, token) // TAREA (frontend): cookie httpOnly, además del token en el body.
	writeJSON(w, http.StatusCreated, authResponse{User: toUserResponse(user), Token: token})
}

// Login maneja POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo de la petición inválido: "+err.Error())
		return
	}

	if err := validateLoginRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, token, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "error interno autenticando el usuario")
		return
	}

	setAuthCookie(w, token) // TAREA (frontend): cookie httpOnly, además del token en el body.
	writeJSON(w, http.StatusOK, authResponse{User: toUserResponse(user), Token: token})
}

// TAREA (frontend Next.js): Logout y Me son endpoints nuevos que no pedía
// el backend original — los necesita el frontend porque el JWT vive en
// una cookie httpOnly (invisible para JS del navegador).

// Logout maneja POST /auth/logout: invalida la cookie de sesión. No
// requiere estar autenticado (es idempotente: si no había sesión, no pasa
// nada).
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	clearAuthCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// Me maneja GET /auth/me (protegido por RequireAuth). Devuelve la info del
// usuario autenticado a partir de los claims del JWT, sin ida a la base de
// datos. El frontend lo usa para saber el rol (admin/user) ya que un JWT
// httpOnly no se puede leer/decodificar desde JavaScript del navegador.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no autenticado")
		return
	}

	id, err := strconv.ParseInt(claims.Sub, 10, 64)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token con sub inválido")
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID:    id,
		Email: claims.Email,
		Name:  claims.Name,
		Role:  claims.Role,
	})
}
