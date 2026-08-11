package ws

// TAREA: upgrade HTTP -> WebSocket con autenticación (sin esto, cualquiera
// podría conectarse y recibir telemetría de todos los vehículos).
// TAREA (frontend): agrega el fallback a cookie httpOnly (bearerToken).

import (
	"net/http"
	"strings"

	"proyecto_simon_pt/internal/auth"
)

// AuthCookieName es el nombre de la cookie httpOnly que transporta el JWT
// (debe coincidir con httpapi.AuthCookieName). Se duplica aquí en vez de
// importarlo para no crear una dependencia ws -> httpapi.
const AuthCookieName = "auth_token"

// NewHandler crea el http.HandlerFunc que hace el upgrade a WebSocket,
// autenticando antes del handshake vía, en orden: header
// "Authorization: Bearer <token>" (clientes API), query param ?token=
// (fallback explícito), o la cookie httpOnly auth_token (navegador: el
// handshake WS es una request HTTP normal, así que el navegador adjunta la
// cookie automáticamente sin que el JS del cliente tenga que leerla).
func NewHandler(hub *Hub, service *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			http.Error(w, "falta token de autenticación", http.StatusUnauthorized)
			return
		}

		claims, err := service.ParseToken(token)
		if err != nil {
			http.Error(w, "token inválido: "+err.Error(), http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return // el upgrader ya escribió la respuesta de error
		}

		c := &client{conn: conn, send: make(chan []byte, 32), role: claims.Role}
		hub.register <- c

		go c.writePump()
		go c.readPump(hub)
	}
}

func bearerToken(r *http.Request) string {
	if header := r.Header.Get("Authorization"); header != "" {
		if strings.HasPrefix(header, "Bearer ") {
			return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		}
		return ""
	}
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	if cookie, err := r.Cookie(AuthCookieName); err == nil {
		return cookie.Value
	}
	return ""
}
