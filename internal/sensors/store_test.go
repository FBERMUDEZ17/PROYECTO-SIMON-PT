// TAREA: tests de persistencia de lecturas (sensor_processing) — orden e
// historial acotado por límite.
package sensors_test

import (
	"testing"
	"time"

	"proyecto_simon_pt/internal/db"
	"proyecto_simon_pt/internal/sensors"
	"proyecto_simon_pt/internal/vehicles"
)

func TestStore_InsertAndRecent_OrderedAscending(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("abriendo db de test: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	vehicleStore := vehicles.NewStore(database)
	if err := vehicleStore.EnsureExists("DEV-1234-XC54", "DEV-1234-XC54", 1); err != nil {
		t.Fatalf("EnsureExists: %v", err)
	}

	store := sensors.NewStore(database)
	base := time.Now().Truncate(time.Second)

	for i, fuel := range []float64{80, 70, 60} {
		r := sensors.Reading{
			VehicleID:    "DEV-1234-XC54",
			Lat:          1, Lon: 1,
			FuelLevel:    fuel,
			TemperatureC: 20,
			RecordedAt:   base.Add(time.Duration(i) * time.Hour),
		}
		if _, err := store.Insert(r); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	recent, err := store.Recent("DEV-1234-XC54", 10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(recent) != 3 {
		t.Fatalf("len(recent) = %d, want 3", len(recent))
	}
	// Orden ascendente por tiempo.
	if recent[0].FuelLevel != 80 || recent[1].FuelLevel != 70 || recent[2].FuelLevel != 60 {
		t.Errorf("orden inesperado: %+v", recent)
	}

	latest, err := store.Latest("DEV-1234-XC54")
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if latest.FuelLevel != 60 {
		t.Errorf("Latest.FuelLevel = %v, want 60", latest.FuelLevel)
	}
}

func TestStore_Recent_LimitsToLastN(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("abriendo db de test: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	vehicleStore := vehicles.NewStore(database)
	if err := vehicleStore.EnsureExists("DEV-1", "DEV-1", 1); err != nil {
		t.Fatalf("EnsureExists: %v", err)
	}

	store := sensors.NewStore(database)
	base := time.Now().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		r := sensors.Reading{
			VehicleID: "DEV-1", Lat: 1, Lon: 1,
			FuelLevel: float64(100 - i*10), TemperatureC: 20,
			RecordedAt: base.Add(time.Duration(i) * time.Hour),
		}
		if _, err := store.Insert(r); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	recent, err := store.Recent("DEV-1", 2)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("len(recent) = %d, want 2", len(recent))
	}
	// Deben ser las 2 más recientes (fuel 70@t3h, 60@t4h), en orden ascendente.
	if recent[0].FuelLevel != 70 || recent[1].FuelLevel != 60 {
		t.Errorf("lecturas inesperadas: %+v", recent)
	}
}
