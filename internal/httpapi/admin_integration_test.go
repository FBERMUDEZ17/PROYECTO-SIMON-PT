// TAREA: tests de integración de las rutas solo-admin (GET /admin/users,
// POST /admin/vehicles) — auth requerida, rechazo para no-admin, creación
// asignada al propietario elegido, y rechazo de ids duplicados.
package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"proyecto_simon_pt/internal/auth"
	"proyecto_simon_pt/internal/httpapi"
)

// promoteToAdmin promueve el email a admin y devuelve un token fresco (el
// rol se codifica en el JWT, así que hay que volver a loguear).
func promoteToAdmin(t *testing.T, deps httpapi.Deps, serverURL, email, password string) string {
	t.Helper()
	if err := deps.Auth.SetRole(email, auth.RoleAdmin); err != nil {
		t.Fatalf("SetRole: %v", err)
	}
	_, body := postJSON(t, serverURL+"/auth/login", map[string]string{"email": email, "password": password})
	token, _ := body["token"].(string)
	if token == "" {
		t.Fatalf("no se obtuvo token tras promover a admin: %v", body)
	}
	return token
}

func TestIntegration_AdminListUsers_RequiresAuth(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	resp, _ := doAuthedRequest(t, http.MethodGet, server.URL+"/admin/users", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestIntegration_AdminListUsers_ForbiddenForNonAdmin(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	token := registerAndLogin(t, server.URL, "not-admin@example.com")

	resp, _ := doAuthedRequest(t, http.MethodGet, server.URL+"/admin/users", token, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestIntegration_AdminListUsers_ReturnsRegisteredUsers(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	registerAndLogin(t, server.URL, "owner-a@example.com")
	registerAndLogin(t, server.URL, "owner-b@example.com")
	adminToken := promoteToAdmin(t, deps, server.URL, "owner-a@example.com", "correct-pw")

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /admin/users: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var users []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
}

func TestIntegration_AdminCreateVehicle_RequiresAuth(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	resp, _ := doAuthedRequest(t, http.MethodPost, server.URL+"/admin/vehicles", "", map[string]any{
		"vehicle_id": "DEV-ADMIN-1", "owner_user_id": 1, "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestIntegration_AdminCreateVehicle_ForbiddenForNonAdmin(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	token := registerAndLogin(t, server.URL, "regular@example.com")

	resp, _ := doAuthedRequest(t, http.MethodPost, server.URL+"/admin/vehicles", token, map[string]any{
		"vehicle_id": "DEV-ADMIN-2", "owner_user_id": 1, "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestIntegration_AdminCreateVehicle_AssignsChosenOwner(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	// El propietario elegido es un usuario distinto del admin que crea el
	// vehículo — ese es justamente el caso de uso del selector de dueño.
	_, ownerBody := postJSON(t, server.URL+"/auth/register", map[string]string{
		"email": "fleet-owner@example.com", "password": "correct-pw", "name": "Fleet Owner",
	})
	ownerUser, _ := ownerBody["user"].(map[string]any)
	ownerID := ownerUser["id"].(float64)

	registerAndLogin(t, server.URL, "the-admin@example.com")
	adminToken := promoteToAdmin(t, deps, server.URL, "the-admin@example.com", "correct-pw")

	resp, body := doAuthedRequest(t, http.MethodPost, server.URL+"/admin/vehicles", adminToken, map[string]any{
		"vehicle_id": "DEV-ADMIN-3", "owner_user_id": ownerID, "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25, "speed_kmh": 10,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%v", resp.StatusCode, body)
	}
	if body["vehicle_id"] != "DEV-ADMIN-3" {
		t.Errorf("vehicle_id = %v", body["vehicle_id"])
	}

	// El vehículo debe aparecer en el listado del propietario elegido...
	ownerToken := registerAndLoginExisting(t, server.URL, "fleet-owner@example.com", "correct-pw")
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/vehicles", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	ownerResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /vehicles (owner): %v", err)
	}
	defer ownerResp.Body.Close()
	var ownerVehicles []map[string]any
	if err := json.NewDecoder(ownerResp.Body).Decode(&ownerVehicles); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(ownerVehicles) != 1 {
		t.Fatalf("len(ownerVehicles) = %d, want 1 (el vehículo creado por el admin)", len(ownerVehicles))
	}
}

func TestIntegration_AdminCreateVehicle_DuplicateID_Returns409(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	registerAndLogin(t, server.URL, "dup-owner@example.com")
	adminToken := promoteToAdmin(t, deps, server.URL, "dup-owner@example.com", "correct-pw")

	body := map[string]any{
		"vehicle_id": "DEV-ADMIN-DUP", "owner_user_id": 1, "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	}
	resp1, b1 := doAuthedRequest(t, http.MethodPost, server.URL+"/admin/vehicles", adminToken, body)
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("primera creación: status = %d, want 201; body=%v", resp1.StatusCode, b1)
	}

	resp2, b2 := doAuthedRequest(t, http.MethodPost, server.URL+"/admin/vehicles", adminToken, body)
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("segunda creación (mismo id): status = %d, want 409; body=%v", resp2.StatusCode, b2)
	}
}

func TestIntegration_AdminCreateVehicle_UnknownOwner_Returns400(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	registerAndLogin(t, server.URL, "unknown-owner-admin@example.com")
	adminToken := promoteToAdmin(t, deps, server.URL, "unknown-owner-admin@example.com", "correct-pw")

	resp, body := doAuthedRequest(t, http.MethodPost, server.URL+"/admin/vehicles", adminToken, map[string]any{
		"vehicle_id": "DEV-ADMIN-4", "owner_user_id": 999999, "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%v", resp.StatusCode, body)
	}
}

func TestIntegration_AdminCreateVehicle_InvalidReading_Returns400(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	registerAndLogin(t, server.URL, "invalid-reading-admin@example.com")
	adminToken := promoteToAdmin(t, deps, server.URL, "invalid-reading-admin@example.com", "correct-pw")

	resp, body := doAuthedRequest(t, http.MethodPost, server.URL+"/admin/vehicles", adminToken, map[string]any{
		"vehicle_id": "DEV-ADMIN-5", "owner_user_id": 1, "lat": 999, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%v", resp.StatusCode, body)
	}
}

// registerAndLoginExisting loguea un usuario ya registrado (a diferencia de
// registerAndLogin, que registra uno nuevo).
func registerAndLoginExisting(t *testing.T, serverURL, email, password string) string {
	t.Helper()
	_, body := postJSON(t, serverURL+"/auth/login", map[string]string{"email": email, "password": password})
	token, _ := body["token"].(string)
	if token == "" {
		t.Fatalf("no se obtuvo token al loguear: %v", body)
	}
	return token
}
