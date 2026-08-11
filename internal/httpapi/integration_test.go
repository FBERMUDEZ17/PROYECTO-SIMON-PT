// TAREA: tests de integración HTTP reales (Prompt 4) — POST /auth/register
// y /auth/login vía httptest.NewServer + peticiones HTTP de verdad.
package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"proyecto_simon_pt/internal/httpapi"
)

// newTestServer levanta un servidor HTTP real (loopback) respaldado por una
// base de datos SQLite en memoria, aislada por test.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	router := httpapi.NewRouter(newTestDeps(t))

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server
}

// postJSON hace una petición HTTP real (TCP sobre loopback) al servidor de test.
func postJSON(t *testing.T, url string, payload any) (*http.Response, map[string]any) {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("petición HTTP falló: %v", err)
	}
	defer resp.Body.Close()

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decodificando respuesta JSON: %v", err)
	}

	return resp, decoded
}

func TestIntegration_Register_Success(t *testing.T) {
	server := newTestServer(t)

	resp, body := postJSON(t, server.URL+"/auth/register", map[string]string{
		"email":    "Alice@Example.com",
		"password": "s3cret-pw",
		"name":     "Alice",
	})

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%v", resp.StatusCode, body)
	}
	user, ok := body["user"].(map[string]any)
	if !ok {
		t.Fatalf("respuesta sin campo 'user': %v", body)
	}
	if user["email"] != "alice@example.com" {
		t.Errorf("email no normalizado en la respuesta: %v", user["email"])
	}
	if body["token"] == "" || body["token"] == nil {
		t.Errorf("se esperaba un token no vacío")
	}
}

func TestIntegration_Register_DuplicateEmail(t *testing.T) {
	server := newTestServer(t)
	payload := map[string]string{"email": "bob@example.com", "password": "password1", "name": "Bob"}

	first, _ := postJSON(t, server.URL+"/auth/register", payload)
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("primer registro: status = %d, want 201", first.StatusCode)
	}

	second, body := postJSON(t, server.URL+"/auth/register", payload)
	if second.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%v", second.StatusCode, body)
	}
}

func TestIntegration_Register_InvalidEmail(t *testing.T) {
	server := newTestServer(t)

	resp, body := postJSON(t, server.URL+"/auth/register", map[string]string{
		"email":    "no-es-un-email",
		"password": "password1",
		"name":     "X",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%v", resp.StatusCode, body)
	}
}

func TestIntegration_Register_PasswordTooShort(t *testing.T) {
	server := newTestServer(t)

	resp, body := postJSON(t, server.URL+"/auth/register", map[string]string{
		"email":    "short@example.com",
		"password": "123",
		"name":     "X",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%v", resp.StatusCode, body)
	}
}

func TestIntegration_Register_MissingFields(t *testing.T) {
	server := newTestServer(t)

	resp, body := postJSON(t, server.URL+"/auth/register", map[string]string{
		"email": "missing@example.com",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%v", resp.StatusCode, body)
	}
}

func TestIntegration_Login_Success(t *testing.T) {
	server := newTestServer(t)
	postJSON(t, server.URL+"/auth/register", map[string]string{
		"email": "carol@example.com", "password": "correct-pw", "name": "Carol",
	})

	resp, body := postJSON(t, server.URL+"/auth/login", map[string]string{
		"email": "carol@example.com", "password": "correct-pw",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%v", resp.StatusCode, body)
	}
	if body["token"] == "" || body["token"] == nil {
		t.Errorf("se esperaba un token no vacío")
	}
}

func TestIntegration_Login_WrongPassword(t *testing.T) {
	server := newTestServer(t)
	postJSON(t, server.URL+"/auth/register", map[string]string{
		"email": "dave@example.com", "password": "correct-pw", "name": "Dave",
	})

	resp, body := postJSON(t, server.URL+"/auth/login", map[string]string{
		"email": "dave@example.com", "password": "wrong-pw",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%v", resp.StatusCode, body)
	}
}

func TestIntegration_Login_NonexistentEmail(t *testing.T) {
	server := newTestServer(t)

	resp, body := postJSON(t, server.URL+"/auth/login", map[string]string{
		"email": "nadie@example.com", "password": "cualquiera",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%v", resp.StatusCode, body)
	}
}

func TestIntegration_Login_EmailUppercase(t *testing.T) {
	server := newTestServer(t)
	postJSON(t, server.URL+"/auth/register", map[string]string{
		"email": "erin@example.com", "password": "correct-pw", "name": "Erin",
	})

	resp, body := postJSON(t, server.URL+"/auth/login", map[string]string{
		"email": "ERIN@EXAMPLE.COM", "password": "correct-pw",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%v", resp.StatusCode, body)
	}
}

func TestIntegration_Login_InvalidEmailFormat(t *testing.T) {
	server := newTestServer(t)

	resp, body := postJSON(t, server.URL+"/auth/login", map[string]string{
		"email": "not-an-email", "password": "whatever",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%v", resp.StatusCode, body)
	}
}
