package httpapi

import (
	"context"
	"net/http"

	"proyecto_simon_pt/internal/auth"
)

type contextKey string

const userContextKey contextKey = "authUser"

// TAREA (Prompt 5): middleware de autenticación — lee el header
// Authorization (Bearer), verifica el JWT y adjunta el usuario a la
// request; 401 si no hay token o es inválido. TAREA (frontend): además
// acepta el JWT vía cookie httpOnly como fallback (ver tokenFromRequest
// en cookie.go).
//
// RequireAuth exige un JWT válido, tomado del header "Authorization: Bearer
// <token>" (clientes API/tests) o, si no está presente, de la cookie
// httpOnly auth_token (navegador). Si falta o es inválido/expirado/
// malformado, responde 401 y corta la cadena. Si es válido, adjunta los
// claims del usuario al contexto de la request y continúa.
func RequireAuth(service *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := tokenFromRequest(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "no autenticado: falta token")
				return
			}

			claims, err := service.ParseToken(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "token inválido: "+err.Error())
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext recupera los claims del usuario autenticado adjuntados
// por RequireAuth. ok es false si la request no pasó por el middleware.
func UserFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(userContextKey).(*auth.Claims)
	return claims, ok
}
