// TAREA: tests del middleware de auth (Prompt 5) — token válido pasa, sin
// token 401, token expirado/malformado 401.
package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"proyecto_simon_pt/internal/auth"
	"proyecto_simon_pt/internal/db"
	"proyecto_simon_pt/internal/httpapi"
)

var testJWTSecret = []byte("middleware-test-secret")

// newProtectedTestServer levanta un servidor HTTP real con una ruta protegida
// por RequireAuth. El handler protegido, si se alcanza, expone en la
// respuesta los claims del usuario que el middleware adjuntó al contexto.
func newProtectedTestServer(t *testing.T) (*httptest.Server, *auth.Service) {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("abriendo db de test: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	service := auth.NewService(database, testJWTSecret)

	protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := httpapi.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "no user in context", http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-User-Email", claims.Email)
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(httpapi.RequireAuth(service)(protected))
	t.Cleanup(server.Close)

	return server, service
}

func doRequestWithAuth(t *testing.T, url, authHeader string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("creando request: %v", err)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("petición HTTP falló: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })

	return resp
}

func TestRequireAuth_ValidToken_Passes(t *testing.T) {
	server, service := newProtectedTestServer(t)

	_, token, err := service.Register("user@example.com", "correct-pw", "User")
	if err != nil {
		t.Fatalf("registro falló: %v", err)
	}

	resp := doRequestWithAuth(t, server.URL, "Bearer "+token)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("X-User-Email"); got != "user@example.com" {
		t.Errorf("info de usuario no adjuntada correctamente: X-User-Email = %q", got)
	}
}

func TestRequireAuth_NoToken_Returns401(t *testing.T) {
	server, _ := newProtectedTestServer(t)

	resp := doRequestWithAuth(t, server.URL, "")

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestRequireAuth_InvalidTokens_Return401(t *testing.T) {
	server, _ := newProtectedTestServer(t)

	expiredToken, err := auth.GenerateToken(
		testJWTSecret, "1", "user@example.com", "User", "user",
		time.Now().Add(-25*time.Hour), // emitido hace 25h, TTL 24h -> ya expiró
	)
	if err != nil {
		t.Fatalf("generando token expirado: %v", err)
	}

	cases := []struct {
		name       string
		authHeader string
	}{
		{"token expirado", "Bearer " + expiredToken},
		{"token malformado", "Bearer no-es-un-jwt-valido"},
		{"token con firma alterada", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIn0.firma-invalida"},
		{"header sin esquema Bearer", "Token abcdef"},
		{"header Bearer sin token", "Bearer "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doRequestWithAuth(t, server.URL, tc.authHeader)
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", resp.StatusCode)
			}
		})
	}
}
