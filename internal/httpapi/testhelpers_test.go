// Helper compartido por los tests de integración de httpapi (auth,
// sensores/IoT, cookie httpOnly): arma un Deps completo con SQLite en memoria.
package httpapi_test

import (
	"testing"

	"proyecto_simon_pt/internal/alerts"
	"proyecto_simon_pt/internal/auth"
	"proyecto_simon_pt/internal/db"
	"proyecto_simon_pt/internal/devices"
	"proyecto_simon_pt/internal/httpapi"
	"proyecto_simon_pt/internal/sensors"
	"proyecto_simon_pt/internal/telemetry"
	"proyecto_simon_pt/internal/vehicles"
	"proyecto_simon_pt/internal/ws"
)

// newTestDeps arma un httpapi.Deps completo respaldado por SQLite en
// memoria, aislado por test.
func newTestDeps(t *testing.T) httpapi.Deps {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("abriendo db de test: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	authService := auth.NewService(database, []byte("test-secret"))
	vehicleStore := vehicles.NewStore(database)
	readingStore := sensors.NewStore(database)
	alertStore := alerts.NewStore(database)
	deviceStore := devices.NewStore(database)
	telemetryService := telemetry.NewService(vehicleStore, readingStore, alertStore)

	hub := ws.NewHub()
	go hub.Run()

	return httpapi.Deps{
		Auth:      authService,
		Telemetry: telemetryService,
		Vehicles:  vehicleStore,
		Readings:  readingStore,
		Alerts:    alertStore,
		Devices:   deviceStore,
		Hub:       hub,
	}
}
