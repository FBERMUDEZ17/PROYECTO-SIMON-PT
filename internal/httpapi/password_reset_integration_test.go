// TAREA: tests de integración del restaurador de contraseña —
// forgot-password no revela si el email existe, reset-password exige un
// token válido, y el password nuevo funciona para loguear.
package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"proyecto_simon_pt/internal/httpapi"
)

func TestIntegration_ForgotPassword_UnknownEmail_StillReturns200(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	resp, _ := postJSON(t, server.URL+"/auth/forgot-password", map[string]string{
		"email": "no-existe@example.com",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (no debe revelar si el email existe)", resp.StatusCode)
	}
}

func TestIntegration_ResetPassword_FullFlow(t *testing.T) {
	// Este test necesita generar el token directamente vía el Service (no
	// hay email real): simula lo que el usuario obtendría del log del
	// servidor tras POST /auth/forgot-password.
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	registerAndLogin(t, server.URL, "reset-me@example.com")

	token, found, err := deps.Auth.RequestPasswordReset("reset-me@example.com")
	if err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}
	if !found {
		t.Fatalf("se esperaba found=true")
	}

	resp, body := postJSON(t, server.URL+"/auth/reset-password", map[string]string{
		"token": token, "new_password": "new-correct-pw",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%v", resp.StatusCode, body)
	}

	// El password viejo ya no funciona.
	oldLoginResp, _ := postJSON(t, server.URL+"/auth/login", map[string]string{
		"email": "reset-me@example.com", "password": "correct-pw",
	})
	if oldLoginResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login con password viejo: status = %d, want 401", oldLoginResp.StatusCode)
	}

	// El password nuevo sí funciona.
	newLoginResp, _ := postJSON(t, server.URL+"/auth/login", map[string]string{
		"email": "reset-me@example.com", "password": "new-correct-pw",
	})
	if newLoginResp.StatusCode != http.StatusOK {
		t.Fatalf("login con password nuevo: status = %d, want 200", newLoginResp.StatusCode)
	}
}

func TestIntegration_ResetPassword_TokenReuse_Fails(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	registerAndLogin(t, server.URL, "reuse-token@example.com")
	token, _, err := deps.Auth.RequestPasswordReset("reuse-token@example.com")
	if err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}

	postJSON(t, server.URL+"/auth/reset-password", map[string]string{
		"token": token, "new_password": "first-new-pw",
	})

	// Reusar el mismo token una segunda vez debe fallar.
	resp, _ := postJSON(t, server.URL+"/auth/reset-password", map[string]string{
		"token": token, "new_password": "second-new-pw",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (token ya usado)", resp.StatusCode)
	}
}

func TestIntegration_ResetPassword_InvalidToken_Returns400(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	resp, _ := postJSON(t, server.URL+"/auth/reset-password", map[string]string{
		"token": "no-existe-este-token", "new_password": "whatever-pw",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestIntegration_ResetPassword_PasswordTooShort_Returns400(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	registerAndLogin(t, server.URL, "short-pw@example.com")
	token, _, _ := deps.Auth.RequestPasswordReset("short-pw@example.com")

	resp, _ := postJSON(t, server.URL+"/auth/reset-password", map[string]string{
		"token": token, "new_password": "short",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
