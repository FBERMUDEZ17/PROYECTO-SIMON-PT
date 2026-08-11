package httpapi

// TAREA (frontend Next.js): JWT en cookie httpOnly — el backend Go la emite
// y valida directamente (decisión tomada junto al usuario), en vez de que
// el frontend la maneje (imposible: httpOnly es invisible para JS).

import (
	"net/http"
	"os"
	"time"

	"proyecto_simon_pt/internal/auth"
)

// AuthCookieName es el nombre de la cookie httpOnly que transporta el JWT.
const AuthCookieName = "auth_token"

// cookieSecure controla el flag Secure de la cookie. En dev (localhost sin
// TLS) debe ir en false; en producción (HTTPS) debe ir en true. Se lee de
// COOKIE_SECURE=true al arrancar el proceso.
func cookieSecure() bool {
	return os.Getenv("COOKIE_SECURE") == "true"
}

// setAuthCookie emite el JWT como cookie httpOnly. SameSite=Lax es
// suficiente para que el navegador la adjunte en requests same-site
// (incluye distintos puertos de "localhost" en desarrollo, y subdominios
// del mismo sitio en producción); para frontends en un dominio registrable
// distinto del backend se necesitaría SameSite=None + Secure=true.
func setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   cookieSecure(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(auth.TokenTTL.Seconds()),
	})
}

// clearAuthCookie invalida la cookie de sesión (logout).
func clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   cookieSecure(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

// tokenFromRequest busca el JWT primero en el header Authorization: Bearer
// (clientes API/tests) y si no está, en la cookie httpOnly (navegador).
func tokenFromRequest(r *http.Request) string {
	if header := r.Header.Get("Authorization"); header != "" {
		const prefix = "Bearer "
		if len(header) > len(prefix) && header[:len(prefix)] == prefix {
			return header[len(prefix):]
		}
		return ""
	}
	if cookie, err := r.Cookie(AuthCookieName); err == nil {
		return cookie.Value
	}
	return ""
}
