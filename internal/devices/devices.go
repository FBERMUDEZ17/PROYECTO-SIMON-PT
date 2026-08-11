// Package devices persiste los push tokens (Expo push tokens — envuelven
// el token nativo FCM/APNs, ver internal/push) registrados por los
// clientes mobile, para poder notificarlos cuando se crea una alerta de
// uno de sus vehículos.
//
// TAREA (mobile): respaldo de "POST /devices/register".
package devices

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

// Register guarda (o reasigna, si el token ya estaba registrado por otra
// sesión) un push token para userID. push_token es único: si el mismo
// dispositivo se loguea con otro usuario, el token pasa a pertenecer al
// nuevo dueño en vez de quedar duplicado apuntando a ambos.
func (s *Store) Register(userID int64, pushToken, platform string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO device_tokens (user_id, push_token, platform, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(push_token) DO UPDATE SET
		   user_id = excluded.user_id,
		   platform = excluded.platform,
		   updated_at = excluded.updated_at`,
		userID, pushToken, platform, now, now,
	)
	return err
}

// TokensForUser devuelve los push tokens registrados de userID (puede
// haber más de uno: varios dispositivos con la misma cuenta).
func (s *Store) TokensForUser(userID int64) ([]string, error) {
	rows, err := s.db.Query(`SELECT push_token FROM device_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}
