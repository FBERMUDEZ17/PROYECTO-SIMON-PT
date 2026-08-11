// TAREA (sensores/IoT): tests de integración de POST /sensors/data y
// GET /vehicles(/{id}) — auth requerida, validación, y enmascarado de
// device ids admin vs no-admin verificado end-to-end.
package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"proyecto_simon_pt/internal/auth"
	"proyecto_simon_pt/internal/httpapi"
)

// registerAndLogin crea un usuario (rol "user" por defecto) y devuelve su
// token JWT, usando una petición HTTP real.
func registerAndLogin(t *testing.T, serverURL, email string) string {
	t.Helper()
	_, body := postJSON(t, serverURL+"/auth/register", map[string]string{
		"email": email, "password": "correct-pw", "name": "Test User",
	})
	token, _ := body["token"].(string)
	if token == "" {
		t.Fatalf("no se obtuvo token al registrar: %v", body)
	}
	return token
}

func doAuthedRequest(t *testing.T, method, url, token string, payload any) (*http.Response, map[string]any) {
	t.Helper()

	var reqBody *bytes.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("creando request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("petición HTTP falló: %v", err)
	}
	defer resp.Body.Close()

	var decoded map[string]any
	if resp.ContentLength != 0 {
		_ = json.NewDecoder(resp.Body).Decode(&decoded)
	}
	return resp, decoded
}

func TestIntegration_SensorsData_RequiresAuth(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	resp, _ := doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", "", map[string]any{
		"vehicle_id": "DEV-1", "lat": 1, "lon": 1, "fuel_level": 50, "temperature_c": 20,
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestIntegration_SensorsData_ValidReading_Created(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	token := registerAndLogin(t, server.URL, "device-owner@example.com")

	resp, body := doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", token, map[string]any{
		"vehicle_id": "DEV-1234-XC54", "lat": 10.5, "lon": -20.1, "fuel_level": 80, "temperature_c": 30,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%v", resp.StatusCode, body)
	}
	if body["vehicle_id"] != "DEV-1234-XC54" {
		t.Errorf("vehicle_id = %v", body["vehicle_id"])
	}
}

func TestIntegration_SensorsData_InvalidPayload_400(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	token := registerAndLogin(t, server.URL, "bad-data@example.com")

	resp, body := doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", token, map[string]any{
		"vehicle_id": "DEV-1", "lat": 999, "lon": 1, "fuel_level": 50, "temperature_c": 20,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%v", resp.StatusCode, body)
	}
}

func TestIntegration_VehiclesList_RequiresAuth(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	resp, _ := doAuthedRequest(t, http.MethodGet, server.URL+"/vehicles", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestIntegration_VehiclesList_ReturnsMaskedForNonAdmin(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	token := registerAndLogin(t, server.URL, "list-user@example.com")

	doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", token, map[string]any{
		"vehicle_id": "DEV-1234-XC54", "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})
	doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", token, map[string]any{
		"vehicle_id": "DEV-9999-ZZ01", "lat": 2, "lon": 2, "fuel_level": 60, "temperature_c": 22,
	})

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/vehicles", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /vehicles: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var list []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
	for _, v := range list {
		id, _ := v["id"].(string)
		if id == "DEV-1234-XC54" || id == "DEV-9999-ZZ01" {
			t.Errorf("id no enmascarado para no-admin: %v", id)
		}
	}
}

func TestIntegration_VehiclesList_OnlyShowsOwnVehicles(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	aliceToken := registerAndLogin(t, server.URL, "alice-owner@example.com")
	bobToken := registerAndLogin(t, server.URL, "bob-owner@example.com")

	doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", aliceToken, map[string]any{
		"vehicle_id": "DEV-ALICE-1", "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})
	doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", bobToken, map[string]any{
		"vehicle_id": "DEV-BOB-1", "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})

	// doAuthedRequest decodifica en map[string]any; para una respuesta que
	// es un array usamos una request manual para decodificar como []any.
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/vehicles", nil)
	req.Header.Set("Authorization", "Bearer "+aliceToken)
	rawResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /vehicles: %v", err)
	}
	defer rawResp.Body.Close()
	var aliceVehicles []map[string]any
	if err := json.NewDecoder(rawResp.Body).Decode(&aliceVehicles); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(aliceVehicles) != 1 {
		t.Fatalf("len(aliceVehicles) = %d, want 1 (solo su propio vehículo)", len(aliceVehicles))
	}
}

func TestIntegration_SensorsData_DifferentUser_Forbidden(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	ownerToken := registerAndLogin(t, server.URL, "vehicle-owner@example.com")
	otherToken := registerAndLogin(t, server.URL, "other-user@example.com")

	doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", ownerToken, map[string]any{
		"vehicle_id": "DEV-SHARED-1", "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})

	resp, body := doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", otherToken, map[string]any{
		"vehicle_id": "DEV-SHARED-1", "lat": 2, "lon": 2, "fuel_level": 50, "temperature_c": 20,
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%v", resp.StatusCode, body)
	}
}

func TestIntegration_VehicleByID_OtherUsersVehicle_ReturnsNotFound(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	ownerToken := registerAndLogin(t, server.URL, "vehicle-owner-2@example.com")
	otherToken := registerAndLogin(t, server.URL, "other-user-2@example.com")

	doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", ownerToken, map[string]any{
		"vehicle_id": "DEV-SHARED-2", "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})

	resp, _ := doAuthedRequest(t, http.MethodGet, server.URL+"/vehicles/DEV-SHARED-2", otherToken, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (no revela vehículos ajenos)", resp.StatusCode)
	}
}

func TestIntegration_VehicleByID_RequiresAuth(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	resp, _ := doAuthedRequest(t, http.MethodGet, server.URL+"/vehicles/DEV-1", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestIntegration_VehicleByID_NotFound(t *testing.T) {
	server := httptest.NewServer(httpapi.NewRouter(newTestDeps(t)))
	t.Cleanup(server.Close)

	token := registerAndLogin(t, server.URL, "nf@example.com")

	resp, _ := doAuthedRequest(t, http.MethodGet, server.URL+"/vehicles/DEV-NOPE", token, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestIntegration_VehicleByID_MasksDeviceID_ForNonAdmin(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	token := registerAndLogin(t, server.URL, "regular-user@example.com")

	// Crea el vehículo enviando una lectura de sensor.
	doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", token, map[string]any{
		"vehicle_id": "DEV-1234-XC54", "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})

	resp, body := doAuthedRequest(t, http.MethodGet, server.URL+"/vehicles/DEV-1234-XC54", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%v", resp.StatusCode, body)
	}
	if body["id"] != "DEV-****-XC54" {
		t.Errorf("id = %v, want id enmascarado DEV-****-XC54", body["id"])
	}
}

func TestIntegration_VehicleByID_ShowsFullDeviceID_ForAdmin(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	registerAndLogin(t, server.URL, "admin-user@example.com")
	if err := deps.Auth.SetRole("admin-user@example.com", auth.RoleAdmin); err != nil {
		t.Fatalf("SetRole: %v", err)
	}
	// El rol se codifica en el JWT emitido en el login, así que hay que
	// volver a autenticar después de promover al usuario a admin.
	_, loginBody := postJSON(t, server.URL+"/auth/login", map[string]string{
		"email": "admin-user@example.com", "password": "correct-pw",
	})
	adminToken, _ := loginBody["token"].(string)

	doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", adminToken, map[string]any{
		"vehicle_id": "DEV-1234-XC54", "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})

	resp, body := doAuthedRequest(t, http.MethodGet, server.URL+"/vehicles/DEV-1234-XC54", adminToken, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%v", resp.StatusCode, body)
	}
	if body["id"] != "DEV-1234-XC54" {
		t.Errorf("id = %v, want el device id completo para admin", body["id"])
	}
}
