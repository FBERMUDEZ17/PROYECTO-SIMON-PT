package sensors

// TAREA: "persistencia en base de datos (SQLite)" — CRUD de lecturas de
// sensores; Recent() alimenta tanto la predicción de combustible como el
// historial que consume el frontend para sus gráficos.

import (
	"database/sql"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Insert persiste una lectura. recordedAt se guarda en RFC3339 UTC.
func (s *Store) Insert(r Reading) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO sensor_readings (vehicle_id, lat, lon, fuel_level, temperature_c, speed_kmh, recorded_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.VehicleID, r.Lat, r.Lon, r.FuelLevel, r.TemperatureC, r.SpeedKmh, r.RecordedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Recent devuelve hasta `limit` lecturas más recientes de un vehículo,
// ordenadas de más antigua a más nueva (orden natural para regresión).
func (s *Store) Recent(vehicleID string, limit int) ([]Reading, error) {
	rows, err := s.db.Query(
		`SELECT id, vehicle_id, lat, lon, fuel_level, temperature_c, speed_kmh, recorded_at
		 FROM (
		   SELECT id, vehicle_id, lat, lon, fuel_level, temperature_c, speed_kmh, recorded_at
		   FROM sensor_readings
		   WHERE vehicle_id = ?
		   ORDER BY recorded_at DESC, id DESC
		   LIMIT ?
		 )
		 ORDER BY recorded_at ASC, id ASC`,
		vehicleID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []Reading
	for rows.Next() {
		var (
			r          Reading
			recordedAt string
		)
		if err := rows.Scan(&r.ID, &r.VehicleID, &r.Lat, &r.Lon, &r.FuelLevel, &r.TemperatureC, &r.SpeedKmh, &recordedAt); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339, recordedAt)
		if err != nil {
			return nil, err
		}
		r.RecordedAt = parsed
		readings = append(readings, r)
	}
	return readings, rows.Err()
}

// Latest devuelve la lectura más reciente de un vehículo, o
// (Reading{}, sql.ErrNoRows) si no hay ninguna.
func (s *Store) Latest(vehicleID string) (Reading, error) {
	readings, err := s.Recent(vehicleID, 1)
	if err != nil {
		return Reading{}, err
	}
	if len(readings) == 0 {
		return Reading{}, sql.ErrNoRows
	}
	return readings[0], nil
}
