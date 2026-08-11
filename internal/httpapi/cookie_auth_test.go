// TAREA (frontend Next.js): tests de la cookie httpOnly — login la setea,
// autentica endpoints protegidos sin header Authorization, logout la limpia.
package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"proyecto_simon_pt/internal/httpapi"
)

// newCookieClient devuelve un *http.Client con cookie jar, para simular un
// navegador que retiene automáticamente la cookie httpOnly entre requests.
func newCookieClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("creando cookie jar: %v", err)
	}
	return &http.Client{Jar: jar}
}

func TestIntegration_Login_SetsHttpOnlyCookie(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	client := newCookieClient(t)

	registerBody, _ := json.Marshal(map[string]string{
		"email": "cookie-user@example.com", "password": "correct-pw", "name": "Cookie User",
	})
	resp, err := client.Post(server.URL+"/auth/register", "application/json", bytes.NewReader(registerBody))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	resp.Body.Close()

	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == httpapi.AuthCookieName {
			found = true
			if !c.HttpOnly {
				t.Errorf("la cookie %s debería ser HttpOnly", httpapi.AuthCookieName)
			}
		}
	}
	if !found {
		t.Fatalf("no se encontró la cookie %s en la respuesta de /auth/register", httpapi.AuthCookieName)
	}
}

func TestIntegration_CookieAuth_AllowsProtectedRoute_WithoutAuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	client := newCookieClient(t)

	registerBody, _ := json.Marshal(map[string]string{
		"email": "cookie-only@example.com", "password": "correct-pw", "name": "Cookie Only",
	})
	resp, err := client.Post(server.URL+"/auth/register", "application/json", bytes.NewReader(registerBody))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	resp.Body.Close()

	// El client.Jar adjunta la cookie automáticamente; no seteamos ningún
	// header Authorization a propósito, para probar el fallback a cookie.
	meResp, err := client.Get(server.URL + "/auth/me")
	if err != nil {
		t.Fatalf("GET /auth/me: %v", err)
	}
	defer meResp.Body.Close()

	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", meResp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(meResp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["email"] != "cookie-only@example.com" {
		t.Errorf("email = %v", body["email"])
	}
	if body["role"] != "user" {
		t.Errorf("role = %v, want user", body["role"])
	}
}

func TestIntegration_Me_WithoutAuth_Returns401(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL + "/auth/me")
	if err != nil {
		t.Fatalf("GET /auth/me: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestIntegration_Logout_ClearsCookie(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	client := newCookieClient(t)

	registerBody, _ := json.Marshal(map[string]string{
		"email": "logout-user@example.com", "password": "correct-pw", "name": "Logout User",
	})
	resp, err := client.Post(server.URL+"/auth/register", "application/json", bytes.NewReader(registerBody))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	resp.Body.Close()

	logoutResp, err := client.Post(server.URL+"/auth/logout", "application/json", nil)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	logoutResp.Body.Close()
	if logoutResp.StatusCode != http.StatusNoContent {
		t.Fatalf("status logout = %d, want 204", logoutResp.StatusCode)
	}

	// Tras el logout, la cookie ya no debería dar acceso a /auth/me.
	meResp, err := client.Get(server.URL + "/auth/me")
	if err != nil {
		t.Fatalf("GET /auth/me: %v", err)
	}
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status tras logout = %d, want 401", meResp.StatusCode)
	}
}
