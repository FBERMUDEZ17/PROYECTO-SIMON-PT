// Package vehicles gestiona el catálogo de vehículos (identificados por su
// device id, ej. "DEV-1234-XC54").
//
// TAREA: catálogo de vehículos para GET /vehicles y GET /vehicles/{id};
// mask.go implementa el enmascarado de device ids para no-admin.
//
// TAREA: ownership por usuario — "si un usuario posee más de un vehículo,
// que pueda elegir cuál monitorear, viendo solo los suyos". El dueño es
// quien mandó la primera lectura de sensor de ese vehicle_id
// (EnsureExists); un usuario no-admin solo ve/opera sus propios vehículos,
// admin ve todos.
package vehicles

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("vehículo no encontrado")
	// ErrForbidden: el vehículo existe pero pertenece a otro usuario.
	ErrForbidden = errors.New("el vehículo pertenece a otro usuario")
)

type Vehicle struct {
	ID          string
	Name        string
	OwnerUserID sql.NullInt64
	CreatedAt   string
}

// IsOwnedBy indica si userID es el dueño de este vehículo.
func (v Vehicle) IsOwnedBy(userID int64) bool {
	return v.OwnerUserID.Valid && v.OwnerUserID.Int64 == userID
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// EnsureExists crea el vehículo si no existe todavía (idempotente),
// asignando ownerUserID como dueño. Si el vehículo ya existe, no cambia de
// dueño (ON CONFLICT DO NOTHING): el primer usuario que reportó datos para
// ese device id sigue siendo el dueño.
func (s *Store) EnsureExists(id, name string, ownerUserID int64) error {
	_, err := s.db.Exec(
		`INSERT INTO vehicles (id, name, owner_user_id, created_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		id, name, ownerUserID, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// List devuelve los vehículos visibles para el usuario: todos si isAdmin,
// o solo los que le pertenecen si no.
func (s *Store) List(userID int64, isAdmin bool) ([]Vehicle, error) {
	query := `SELECT id, name, owner_user_id, created_at FROM vehicles`
	args := []any{}
	if !isAdmin {
		query += ` WHERE owner_user_id = ?`
		args = append(args, userID)
	}
	query += ` ORDER BY id`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Vehicle
	for rows.Next() {
		var v Vehicle
		if err := rows.Scan(&v.ID, &v.Name, &v.OwnerUserID, &v.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

// GetByID devuelve el vehículo o ErrNotFound si no existe. No aplica
// ownership por sí solo — el caller decide (ver ErrForbidden) para poder
// distinguir "no existe" (404) de "no es tuyo" en el handler HTTP.
func (s *Store) GetByID(id string) (Vehicle, error) {
	var v Vehicle
	err := s.db.QueryRow(
		`SELECT id, name, owner_user_id, created_at FROM vehicles WHERE id = ?`, id,
	).Scan(&v.ID, &v.Name, &v.OwnerUserID, &v.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return Vehicle{}, ErrNotFound
	}
	if err != nil {
		return Vehicle{}, err
	}
	return v, nil
}
