// Package db maneja la conexión e inicialización del schema de SQLite.
//
// TAREA: tabla "users" en la inicialización de SQLite (id, email único,
// password_hash, name, created_at + role agregado luego para el
// enmascarado por rol). Las tablas vehicles/sensor_readings/alerts son de
// la tarea de telemetría IoT (sensores/predicción de combustible/alertas).
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const schema = `
-- TAREA (Prompt 2): tabla users.
CREATE TABLE IF NOT EXISTS users (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	email         TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	name          TEXT NOT NULL,
	role          TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
	created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

-- TAREA (sensores/IoT): catálogo de vehículos, lecturas de sensores y alertas.
-- owner_user_id: dueño del vehículo (quien mandó su primera lectura de
-- sensor); NULL = sin reclamar. Un usuario no-admin solo ve/opera sobre
-- los vehículos que le pertenecen; admin ve todos.
CREATE TABLE IF NOT EXISTS vehicles (
	id            TEXT PRIMARY KEY, -- device id, ej. "DEV-1234-XC54"
	name          TEXT NOT NULL,
	owner_user_id INTEGER REFERENCES users(id),
	created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_vehicles_owner ON vehicles (owner_user_id);

-- fuel_level en porcentaje (0-100) del tanque. speed_kmh: velocidad
-- instantánea reportada por el sensor GPS (0 = detenido); alimenta el
-- histórico de velocidad y la alerta predictiva de exceso de velocidad.
CREATE TABLE IF NOT EXISTS sensor_readings (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	vehicle_id    TEXT NOT NULL REFERENCES vehicles(id),
	lat           REAL NOT NULL,
	lon           REAL NOT NULL,
	fuel_level    REAL NOT NULL,
	temperature_c REAL NOT NULL,
	speed_kmh     REAL NOT NULL DEFAULT 0,
	recorded_at   TEXT NOT NULL,
	created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_sensor_readings_vehicle_recorded
	ON sensor_readings (vehicle_id, recorded_at);

-- TAREA: restaurador de contraseña — tokens de un solo uso con expiración.
CREATE TABLE IF NOT EXISTS password_resets (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id    INTEGER NOT NULL REFERENCES users(id),
	token      TEXT NOT NULL UNIQUE,
	expires_at TEXT NOT NULL,
	used_at    TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_password_resets_token ON password_resets (token);

CREATE TABLE IF NOT EXISTS alerts (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	vehicle_id TEXT NOT NULL REFERENCES vehicles(id),
	type       TEXT NOT NULL,
	message    TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_alerts_vehicle_created
	ON alerts (vehicle_id, created_at);

-- TAREA (mobile): push tokens (Expo push token, envuelve FCM/APNs)
-- registrados por los clientes mobile vía POST /devices/register, para
-- poder avisarles por push cuando se crea una alerta de uno de sus
-- vehículos (ver internal/devices, internal/push).
CREATE TABLE IF NOT EXISTS device_tokens (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id    INTEGER NOT NULL REFERENCES users(id),
	push_token TEXT NOT NULL UNIQUE,
	platform   TEXT NOT NULL CHECK (platform IN ('ios', 'android')),
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_device_tokens_user ON device_tokens (user_id);
`

// Open abre (o crea) la base de datos en dsn e inicializa el schema.
// dsn puede ser una ruta de archivo o ":memory:" para tests.
func Open(dsn string) (*sql.DB, error) {
	if dsn != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
			return nil, fmt.Errorf("creando directorio de datos: %w", err)
		}
	}

	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("abriendo sqlite: %w", err)
	}

	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("conectando a sqlite: %w", err)
	}

	if _, err := database.Exec(schema); err != nil {
		return nil, fmt.Errorf("inicializando schema: %w", err)
	}

	if err := migrate(database); err != nil {
		return nil, fmt.Errorf("migrando schema: %w", err)
	}

	return database, nil
}

// migrate aplica cambios de schema sobre bases de datos creadas antes de que
// existieran (no hay herramienta de migraciones en el proyecto). Tolera que
// la columna ya exista (bases de datos nuevas la crean directamente en el
// CREATE TABLE de arriba).
func migrate(database *sql.DB) error {
	_, err := database.Exec(`ALTER TABLE vehicles ADD COLUMN owner_user_id INTEGER REFERENCES users(id)`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}
	_, err = database.Exec(`ALTER TABLE sensor_readings ADD COLUMN speed_kmh REAL NOT NULL DEFAULT 0`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}
	return nil
}
