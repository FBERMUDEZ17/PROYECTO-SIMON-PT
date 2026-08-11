// TAREA (mobile): tests de integración de POST /devices/register — auth
// requerida, validación de payload, y persistencia real vía
// internal/devices.Store (chequeada aquí a través de Deps, ya que el
// endpoint no expone un GET de lectura).
package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"proyecto_simon_pt/internal/httpapi"
)

// userIDFromToken decodifica el JWT devuelto por login/register para
// obtener el user id (claims.Sub), igual que hacen los handlers HTTP.
func userIDFromToken(t *testing.T, deps httpapi.Deps, token string) int64 {
	t.Helper()
	claims, err := deps.Auth.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	id, err := strconv.ParseInt(claims.Sub, 10, 64)
	if err != nil {
		t.Fatalf("claims.Sub no es un id válido: %v", err)
	}
	return id
}

func TestIntegration_DevicesRegister_RequiresAuth(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	resp, _ := doAuthedRequest(t, http.MethodPost, server.URL+"/devices/register", "", map[string]any{
		"push_token": "ExponentPushToken[abc123]", "platform": "android",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestIntegration_DevicesRegister_ValidToken_Created(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	token := registerAndLogin(t, server.URL, "mobile-user@example.com")

	resp, body := doAuthedRequest(t, http.MethodPost, server.URL+"/devices/register", token, map[string]any{
		"push_token": "ExponentPushToken[abc123]", "platform": "android",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%v", resp.StatusCode, body)
	}
	if body["status"] != "ok" {
		t.Errorf("status body = %v, want \"ok\"", body["status"])
	}

	// Verifica persistencia real: el token quedó asociado al usuario que
	// se acaba de registrar/loguear.
	userID := userIDFromToken(t, deps, token)
	tokens, err := deps.Devices.TokensForUser(userID)
	if err != nil {
		t.Fatalf("TokensForUser: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "ExponentPushToken[abc123]" {
		t.Errorf("tokens = %v, want [ExponentPushToken[abc123]]", tokens)
	}
}

func TestIntegration_DevicesRegister_MissingPushToken_400(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	token := registerAndLogin(t, server.URL, "no-token@example.com")

	resp, _ := doAuthedRequest(t, http.MethodPost, server.URL+"/devices/register", token, map[string]any{
		"push_token": "", "platform": "android",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestIntegration_DevicesRegister_InvalidPlatform_400(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	token := registerAndLogin(t, server.URL, "bad-platform@example.com")

	resp, _ := doAuthedRequest(t, http.MethodPost, server.URL+"/devices/register", token, map[string]any{
		"push_token": "ExponentPushToken[abc123]", "platform": "windows-phone",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestIntegration_DevicesRegister_ReRegisterSameToken_ReassignsOwner(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	tokenA := registerAndLogin(t, server.URL, "user-a@example.com")
	tokenB := registerAndLogin(t, server.URL, "user-b@example.com")

	sharedPushToken := "ExponentPushToken[shared-device]"

	if resp, body := doAuthedRequest(t, http.MethodPost, server.URL+"/devices/register", tokenA, map[string]any{
		"push_token": sharedPushToken, "platform": "ios",
	}); resp.StatusCode != http.StatusCreated {
		t.Fatalf("registro de A: status = %d, body=%v", resp.StatusCode, body)
	}

	// Mismo dispositivo, otro usuario se loguea después (ej. logout/login):
	// el token pasa a pertenecer a B, no queda duplicado bajo A.
	if resp, body := doAuthedRequest(t, http.MethodPost, server.URL+"/devices/register", tokenB, map[string]any{
		"push_token": sharedPushToken, "platform": "ios",
	}); resp.StatusCode != http.StatusCreated {
		t.Fatalf("registro de B: status = %d, body=%v", resp.StatusCode, body)
	}

	userIDA := userIDFromToken(t, deps, tokenA)
	userIDB := userIDFromToken(t, deps, tokenB)

	tokensA, err := deps.Devices.TokensForUser(userIDA)
	if err != nil {
		t.Fatalf("TokensForUser A: %v", err)
	}
	if len(tokensA) != 0 {
		t.Errorf("A ya no debería tener el token registrado, got %v", tokensA)
	}

	tokensB, err := deps.Devices.TokensForUser(userIDB)
	if err != nil {
		t.Fatalf("TokensForUser B: %v", err)
	}
	if len(tokensB) != 1 || tokensB[0] != sharedPushToken {
		t.Errorf("tokens de B = %v, want [%s]", tokensB, sharedPushToken)
	}
}
