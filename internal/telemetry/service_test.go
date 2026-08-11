// TAREA: unit tests de "sensor_processing" (orquestación de ingesta) —
// creación de vehículo, alerta por combustible bajo, deduplicación de
// alertas, nivel estable sin alertas.
package telemetry_test

import (
	"errors"
	"testing"
	"time"

	"proyecto_simon_pt/internal/alerts"
	"proyecto_simon_pt/internal/db"
	"proyecto_simon_pt/internal/sensors"
	"proyecto_simon_pt/internal/telemetry"
	"proyecto_simon_pt/internal/vehicles"
)

func newTestService(t *testing.T) (*telemetry.Service, *alerts.Store) {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("abriendo db de test: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	vehicleStore := vehicles.NewStore(database)
	readingStore := sensors.NewStore(database)
	alertStore := alerts.NewStore(database)

	return telemetry.NewService(vehicleStore, readingStore, alertStore), alertStore
}

// newTestServiceWithVehicles es como newTestService pero también expone el
// vehicles.Store (para verificar el dueño asignado por CreateVehicle).
func newTestServiceWithVehicles(t *testing.T) (*telemetry.Service, *vehicles.Store) {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("abriendo db de test: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	vehicleStore := vehicles.NewStore(database)
	readingStore := sensors.NewStore(database)
	alertStore := alerts.NewStore(database)

	return telemetry.NewService(vehicleStore, readingStore, alertStore), vehicleStore
}

func TestIngest_ValidReading_PersistsAndCreatesVehicle(t *testing.T) {
	svc, _ := newTestService(t)

	input := sensors.ReadingInput{
		VehicleID: "DEV-1234-XC54", Lat: 1, Lon: 1, FuelLevel: 80, TemperatureC: 25,
	}
	result, err := svc.Ingest(input, time.Now(), 1, false)
	if err != nil {
		t.Fatalf("Ingest error: %v", err)
	}
	if result.Reading.ID == 0 {
		t.Errorf("se esperaba un ID de lectura asignado")
	}
	if result.Reading.VehicleID != "DEV-1234-XC54" {
		t.Errorf("VehicleID = %q", result.Reading.VehicleID)
	}
}

func TestIngest_InvalidReading_ReturnsValidationError(t *testing.T) {
	svc, _ := newTestService(t)

	input := sensors.ReadingInput{VehicleID: "", Lat: 1, Lon: 1, FuelLevel: 80, TemperatureC: 25}
	_, err := svc.Ingest(input, time.Now(), 1, false)
	if err != sensors.ErrMissingVehicleID {
		t.Fatalf("got %v, want ErrMissingVehicleID", err)
	}
}

func TestIngest_FirstReadingAlone_NoAlertYet(t *testing.T) {
	svc, alertStore := newTestService(t)

	input := sensors.ReadingInput{VehicleID: "DEV-1", Lat: 1, Lon: 1, FuelLevel: 2, TemperatureC: 25}
	result, err := svc.Ingest(input, time.Now(), 1, false)
	if err != nil {
		t.Fatalf("Ingest error: %v", err)
	}
	// Con una sola lectura no hay suficientes datos para predecir autonomía.
	if result.Prediction.Ok {
		t.Errorf("se esperaba Prediction.Ok=false con una sola lectura")
	}
	if len(result.Alerts) != 0 {
		t.Errorf("no debería crearse alerta sin predicción válida")
	}

	stored, err := alertStore.ListByVehicle("DEV-1", 10)
	if err != nil {
		t.Fatalf("ListByVehicle: %v", err)
	}
	if len(stored) != 0 {
		t.Errorf("no se esperaban alertas persistidas, got %d", len(stored))
	}
}

func TestIngest_DecreasingFuelBelowThreshold_CreatesAlert(t *testing.T) {
	svc, alertStore := newTestService(t)
	base := time.Now()

	// Primera lectura: 5% de combustible.
	_, err := svc.Ingest(sensors.ReadingInput{
		VehicleID: "DEV-1", Lat: 1, Lon: 1, FuelLevel: 5, TemperatureC: 25,
	}, base, 1, false)
	if err != nil {
		t.Fatalf("Ingest #1: %v", err)
	}

	// Segunda lectura 30 min después: 2% -> tasa de 6%/h -> autonomía 0.33h (<1h).
	result, err := svc.Ingest(sensors.ReadingInput{
		VehicleID: "DEV-1", Lat: 1, Lon: 1, FuelLevel: 2, TemperatureC: 25,
	}, base.Add(30*time.Minute), 1, false)
	if err != nil {
		t.Fatalf("Ingest #2: %v", err)
	}

	if !result.Prediction.LowFuelAlert() {
		t.Fatalf("se esperaba LowFuelAlert()=true, got %+v", result.Prediction)
	}
	if len(result.Alerts) != 1 {
		t.Fatalf("se esperaba que se creara una alerta, got %d", len(result.Alerts))
	}
	if result.Alerts[0].Type != alerts.TypeLowFuel {
		t.Errorf("Alert.Type = %q, want %q", result.Alerts[0].Type, alerts.TypeLowFuel)
	}

	stored, err := alertStore.ListByVehicle("DEV-1", 10)
	if err != nil {
		t.Fatalf("ListByVehicle: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("len(stored alerts) = %d, want 1", len(stored))
	}
}

func TestIngest_RepeatedLowFuel_DoesNotDuplicateAlert(t *testing.T) {
	svc, alertStore := newTestService(t)
	base := time.Now()

	readings := []struct {
		fuel   float64
		offset time.Duration
	}{
		{5, 0},
		{2, 30 * time.Minute},  // dispara la primera alerta
		{1, 45 * time.Minute},  // sigue bajo, pero dentro de la ventana de dedupe
	}

	var lastResult telemetry.IngestResult
	for _, r := range readings {
		res, err := svc.Ingest(sensors.ReadingInput{
			VehicleID: "DEV-1", Lat: 1, Lon: 1, FuelLevel: r.fuel, TemperatureC: 25,
		}, base.Add(r.offset), 1, false)
		if err != nil {
			t.Fatalf("Ingest: %v", err)
		}
		lastResult = res
	}

	if len(lastResult.Alerts) != 0 {
		t.Errorf("la tercera lectura no debería generar una alerta nueva (dedupe), got %+v", lastResult.Alerts)
	}

	stored, err := alertStore.ListByVehicle("DEV-1", 10)
	if err != nil {
		t.Fatalf("ListByVehicle: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("len(stored alerts) = %d, want 1 (deduplicadas)", len(stored))
	}
}

func TestIngest_NewVehicle_OwnedByFirstSubmitter(t *testing.T) {
	svc, _ := newTestService(t)

	input := sensors.ReadingInput{VehicleID: "DEV-OWN-1", Lat: 1, Lon: 1, FuelLevel: 80, TemperatureC: 25}
	if _, err := svc.Ingest(input, time.Now(), 42, false); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
}

func TestIngest_DifferentUser_Forbidden(t *testing.T) {
	svc, _ := newTestService(t)
	input := sensors.ReadingInput{VehicleID: "DEV-OWN-2", Lat: 1, Lon: 1, FuelLevel: 80, TemperatureC: 25}

	// Usuario 1 reclama el vehículo con la primera lectura.
	if _, err := svc.Ingest(input, time.Now(), 1, false); err != nil {
		t.Fatalf("Ingest #1: %v", err)
	}

	// Usuario 2 (no-admin) intenta mandar datos del mismo vehículo: rechazado.
	_, err := svc.Ingest(input, time.Now(), 2, false)
	if !errors.Is(err, vehicles.ErrForbidden) {
		t.Fatalf("got %v, want vehicles.ErrForbidden", err)
	}
}

func TestIngest_AdminBypassesOwnership(t *testing.T) {
	svc, _ := newTestService(t)
	input := sensors.ReadingInput{VehicleID: "DEV-OWN-3", Lat: 1, Lon: 1, FuelLevel: 80, TemperatureC: 25}

	if _, err := svc.Ingest(input, time.Now(), 1, false); err != nil {
		t.Fatalf("Ingest #1: %v", err)
	}

	// Un admin puede mandar datos de un vehículo que no es suyo.
	if _, err := svc.Ingest(input, time.Now(), 99, true); err != nil {
		t.Fatalf("Ingest como admin: %v", err)
	}
}

func TestCreateVehicle_AssignsChosenOwner(t *testing.T) {
	svc, vehicleStore := newTestServiceWithVehicles(t)

	input := sensors.ReadingInput{VehicleID: "DEV-NEW-1", Lat: 1, Lon: 1, FuelLevel: 80, TemperatureC: 25, SpeedKmh: 20}
	result, err := svc.CreateVehicle(input, time.Now(), 7)
	if err != nil {
		t.Fatalf("CreateVehicle: %v", err)
	}
	if result.Reading.VehicleID != "DEV-NEW-1" {
		t.Errorf("VehicleID = %q", result.Reading.VehicleID)
	}

	v, err := vehicleStore.GetByID("DEV-NEW-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !v.IsOwnedBy(7) {
		t.Errorf("el vehículo debería quedar a nombre del owner_user_id elegido (7), got owner=%+v", v.OwnerUserID)
	}
}

func TestCreateVehicle_AlreadyExists_Fails(t *testing.T) {
	svc, _ := newTestServiceWithVehicles(t)

	input := sensors.ReadingInput{VehicleID: "DEV-NEW-2", Lat: 1, Lon: 1, FuelLevel: 80, TemperatureC: 25}
	if _, err := svc.CreateVehicle(input, time.Now(), 7); err != nil {
		t.Fatalf("CreateVehicle #1: %v", err)
	}

	// Segundo intento con el mismo id (incluso a otro dueño): rechazado, no
	// reasigna ni agrega datos por accidente.
	_, err := svc.CreateVehicle(input, time.Now(), 8)
	if !errors.Is(err, telemetry.ErrVehicleAlreadyExists) {
		t.Fatalf("got %v, want telemetry.ErrVehicleAlreadyExists", err)
	}
}

func TestCreateVehicle_InvalidReading_ReturnsValidationError(t *testing.T) {
	svc, _ := newTestServiceWithVehicles(t)

	input := sensors.ReadingInput{VehicleID: "DEV-NEW-3", Lat: 999, Lon: 1, FuelLevel: 80, TemperatureC: 25}
	_, err := svc.CreateVehicle(input, time.Now(), 7)
	if !errors.Is(err, sensors.ErrInvalidLatitude) {
		t.Fatalf("got %v, want sensors.ErrInvalidLatitude", err)
	}
}

func TestIngest_StableFuel_NoAlert(t *testing.T) {
	svc, alertStore := newTestService(t)
	base := time.Now()

	for i, offset := range []time.Duration{0, time.Hour, 2 * time.Hour} {
		_, err := svc.Ingest(sensors.ReadingInput{
			VehicleID: "DEV-1", Lat: 1, Lon: 1, FuelLevel: 50, TemperatureC: 25,
		}, base.Add(offset), 1, false)
		if err != nil {
			t.Fatalf("Ingest #%d: %v", i, err)
		}
	}

	stored, err := alertStore.ListByVehicle("DEV-1", 10)
	if err != nil {
		t.Fatalf("ListByVehicle: %v", err)
	}
	if len(stored) != 0 {
		t.Errorf("nivel estable no debería generar alertas, got %d", len(stored))
	}
}

func TestIngest_SpeedAboveThreshold_CreatesAlert(t *testing.T) {
	svc, alertStore := newTestService(t)

	result, err := svc.Ingest(sensors.ReadingInput{
		VehicleID: "DEV-1", Lat: 1, Lon: 1, FuelLevel: 80, TemperatureC: 25, SpeedKmh: 95,
	}, time.Now(), 1, false)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	if len(result.Alerts) != 1 {
		t.Fatalf("se esperaba una alerta, got %d", len(result.Alerts))
	}
	if result.Alerts[0].Type != alerts.TypeHighSpeed {
		t.Errorf("Alert.Type = %q, want %q", result.Alerts[0].Type, alerts.TypeHighSpeed)
	}

	stored, err := alertStore.ListByVehicle("DEV-1", 10)
	if err != nil {
		t.Fatalf("ListByVehicle: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("len(stored alerts) = %d, want 1", len(stored))
	}
}

func TestIngest_SpeedAtOrBelowThreshold_NoAlert(t *testing.T) {
	svc, alertStore := newTestService(t)

	result, err := svc.Ingest(sensors.ReadingInput{
		VehicleID: "DEV-1", Lat: 1, Lon: 1, FuelLevel: 80, TemperatureC: 25, SpeedKmh: 80,
	}, time.Now(), 1, false)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(result.Alerts) != 0 {
		t.Errorf("80 km/h no debería disparar la alerta (umbral es 'superior a 80'), got %+v", result.Alerts)
	}

	stored, err := alertStore.ListByVehicle("DEV-1", 10)
	if err != nil {
		t.Fatalf("ListByVehicle: %v", err)
	}
	if len(stored) != 0 {
		t.Errorf("no se esperaban alertas persistidas, got %d", len(stored))
	}
}

func TestIngest_RepeatedHighSpeed_DoesNotDuplicateAlert(t *testing.T) {
	svc, alertStore := newTestService(t)
	base := time.Now()

	for i, offset := range []time.Duration{0, 5 * time.Minute, 10 * time.Minute} {
		_, err := svc.Ingest(sensors.ReadingInput{
			VehicleID: "DEV-1", Lat: 1, Lon: 1, FuelLevel: 80, TemperatureC: 25, SpeedKmh: 100,
		}, base.Add(offset), 1, false)
		if err != nil {
			t.Fatalf("Ingest #%d: %v", i, err)
		}
	}

	stored, err := alertStore.ListByVehicle("DEV-1", 10)
	if err != nil {
		t.Fatalf("ListByVehicle: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("len(stored alerts) = %d, want 1 (deduplicadas)", len(stored))
	}
}

func TestIngest_LowFuelAndHighSpeed_BothAlertsFireTogether(t *testing.T) {
	svc, alertStore := newTestService(t)
	base := time.Now()

	// Primera lectura: establece un nivel de combustible alto.
	_, err := svc.Ingest(sensors.ReadingInput{
		VehicleID: "DEV-1", Lat: 1, Lon: 1, FuelLevel: 5, TemperatureC: 25, SpeedKmh: 10,
	}, base, 1, false)
	if err != nil {
		t.Fatalf("Ingest #1: %v", err)
	}

	// Segunda lectura: combustible cae a niveles críticos (autonomía <1h) Y
	// va a más de 80 km/h en la misma lectura -> deben dispararse ambas.
	result, err := svc.Ingest(sensors.ReadingInput{
		VehicleID: "DEV-1", Lat: 1, Lon: 1, FuelLevel: 2, TemperatureC: 25, SpeedKmh: 100,
	}, base.Add(30*time.Minute), 1, false)
	if err != nil {
		t.Fatalf("Ingest #2: %v", err)
	}

	if len(result.Alerts) != 2 {
		t.Fatalf("se esperaban 2 alertas simultáneas, got %d: %+v", len(result.Alerts), result.Alerts)
	}

	stored, err := alertStore.ListByVehicle("DEV-1", 10)
	if err != nil {
		t.Fatalf("ListByVehicle: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("len(stored alerts) = %d, want 2", len(stored))
	}
}
