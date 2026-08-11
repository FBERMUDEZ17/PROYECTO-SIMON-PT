// TAREA (sensores/IoT): tests de WebSocket real — dial real, auth por
// token, y recepción de un evento de broadcast enmascarado.
package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"proyecto_simon_pt/internal/httpapi"
)

func TestIntegration_WebSocket_BroadcastsSensorReading(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	token := registerAndLogin(t, server.URL, "ws-user@example.com")

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=" + token
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial ws: %v (status=%v)", err, respStatus(resp))
	}
	defer conn.Close()

	// Da tiempo a que el hub registre la conexión antes de emitir el evento.
	time.Sleep(50 * time.Millisecond)

	doAuthedRequest(t, http.MethodPost, server.URL+"/sensors/data", token, map[string]any{
		"vehicle_id": "DEV-1234-XC54", "lat": 1, "lon": 1, "fuel_level": 80, "temperature_c": 25,
	})

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("no se recibió el broadcast por websocket: %v", err)
	}

	var event map[string]any
	if err := json.Unmarshal(message, &event); err != nil {
		t.Fatalf("evento no es JSON válido: %v", err)
	}
	if event["type"] != "sensor_reading" {
		t.Errorf("type = %v, want sensor_reading", event["type"])
	}
	// El usuario no es admin: el vehicle_id debe llegar enmascarado.
	if event["vehicle_id"] != "DEV-****-XC54" {
		t.Errorf("vehicle_id = %v, want DEV-****-XC54 (enmascarado)", event["vehicle_id"])
	}
}

func TestIntegration_WebSocket_RequiresToken(t *testing.T) {
	deps := newTestDeps(t)
	server := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatalf("se esperaba que el dial fallara sin token")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %v, want 401", respStatus(resp))
	}
}

func respStatus(resp *http.Response) any {
	if resp == nil {
		return nil
	}
	return resp.StatusCode
}
