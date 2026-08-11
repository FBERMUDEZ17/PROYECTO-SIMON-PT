// Package alerts persiste alertas generadas por el procesamiento de
// telemetría (ej. combustible bajo, exceso de velocidad).
//
// TAREA: soporte de persistencia para "sistema de alertas predictivas" —
// RecentCount implementa la deduplicación (no repetir la misma alerta de
// un vehículo dentro de una ventana de tiempo).
package alerts

import (
	"database/sql"
	"time"
)

const (
	TypeLowFuel   = "low_fuel"
	TypeHighSpeed = "high_speed"
)

type Alert struct {
	ID        int64
	VehicleID string
	Type      string
	Message   string
	CreatedAt string
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(vehicleID, alertType, message string) (Alert, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO alerts (vehicle_id, type, message, created_at) VALUES (?, ?, ?, ?)`,
		vehicleID, alertType, message, now,
	)
	if err != nil {
		return Alert{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Alert{}, err
	}
	return Alert{ID: id, VehicleID: vehicleID, Type: alertType, Message: message, CreatedAt: now}, nil
}

// RecentCount cuenta cuántas alertas del mismo tipo se crearon para un
// vehículo desde `since`. Se usa para deduplicar alertas repetidas.
func (s *Store) RecentCount(vehicleID, alertType string, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM alerts WHERE vehicle_id = ? AND type = ? AND created_at >= ?`,
		vehicleID, alertType, since.UTC().Format(time.RFC3339),
	).Scan(&count)
	return count, err
}

// ListByVehicle devuelve hasta `limit` alertas más recientes de un vehículo.
func (s *Store) ListByVehicle(vehicleID string, limit int) ([]Alert, error) {
	rows, err := s.db.Query(
		`SELECT id, vehicle_id, type, message, created_at
		 FROM alerts WHERE vehicle_id = ? ORDER BY created_at DESC, id DESC LIMIT ?`,
		vehicleID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.VehicleID, &a.Type, &a.Message, &a.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
