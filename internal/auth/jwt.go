// Package auth implementa registro/login de usuarios y un JWT manual
// (HS256) construido únicamente con la librería estándar de Go —
// sin dependencias externas de validación/firma.
//
// TAREA: "arquitecto backend Go" — JWT manual sin librerías externas de
// validación (jwt.go completo), consumido por el servicio de auth
// (service.go, register/login) y por el middleware HTTP (httpapi.RequireAuth).
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// TokenTTL es la vigencia del JWT emitido por register/login.
const TokenTTL = 24 * time.Hour

var (
	ErrMalformedToken      = errors.New("token malformado")
	ErrUnsupportedAlgorithm = errors.New("algoritmo de token no soportado")
	ErrInvalidSignature    = errors.New("firma de token inválida")
	ErrTokenExpired        = errors.New("token expirado")
)

// jwtHeader es fijo: siempre HS256/JWT. Al no aceptar el "alg" del token
// entrante como variable de confianza evitamos ataques de confusión de
// algoritmo (p.ej. un token con "alg":"none").
const jwtHeaderJSON = `{"alg":"HS256","typ":"JWT"}`

// Claims son los datos que viajan firmados dentro del JWT.
type Claims struct {
	Sub   string `json:"sub"`   // id del usuario
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"` // "user" o "admin" — TAREA (sensores/IoT): habilita enmascarar device ids solo para no-admin.
	Iat   int64  `json:"iat"`  // issued at (unix seconds)
	Exp   int64  `json:"exp"`  // expiration (unix seconds)
}

// IsAdmin es un helper de conveniencia para checks de autorización.
// TAREA (sensores/IoT): usado por los handlers de /vehicles y por el hub
// de WebSockets para decidir si enmascarar el device id en la respuesta.
func (c Claims) IsAdmin() bool {
	return c.Role == "admin"
}

func b64Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func b64Decode(s string) ([]byte, error) {
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrMalformedToken
	}
	return data, nil
}

func sign(signingInput string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	return mac.Sum(nil)
}

// GenerateToken construye y firma manualmente un JWT HS256:
// base64url(header) + "." + base64url(payload) + "." + base64url(hmac).
// TAREA: emisión del JWT (register/login lo llaman con TTL de 24h).
func GenerateToken(secret []byte, sub, email, name, role string, now time.Time) (string, error) {
	payload := Claims{
		Sub:   sub,
		Email: email,
		Name:  name,
		Role:  role,
		Iat:   now.Unix(),
		Exp:   now.Add(TokenTTL).Unix(),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	headerB64 := b64Encode([]byte(jwtHeaderJSON))
	payloadB64 := b64Encode(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	signature := sign(signingInput, secret)
	sigB64 := b64Encode(signature)

	return signingInput + "." + sigB64, nil
}

// VerifyToken valida firma, algoritmo y expiración, y devuelve los claims.
// TAREA: verificación manual (sin librería) — es lo que usa el middleware
// de autenticación (httpapi.RequireAuth) y el handler de WebSocket.
func VerifyToken(secret []byte, token string, now time.Time) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrMalformedToken
	}
	headerB64, payloadB64, sigB64 := parts[0], parts[1], parts[2]

	headerBytes, err := b64Decode(headerB64)
	if err != nil {
		return nil, err
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrMalformedToken
	}
	if header.Alg != "HS256" {
		return nil, ErrUnsupportedAlgorithm
	}

	expectedSig := sign(headerB64+"."+payloadB64, secret)
	actualSig, err := b64Decode(sigB64)
	if err != nil {
		return nil, err
	}
	// hmac.Equal compara en tiempo constante para evitar timing attacks.
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, ErrInvalidSignature
	}

	payloadBytes, err := b64Decode(payloadB64)
	if err != nil {
		return nil, err
	}
	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, ErrMalformedToken
	}

	if now.Unix() > claims.Exp {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}
