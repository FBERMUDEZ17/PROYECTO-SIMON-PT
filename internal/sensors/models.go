// Package sensors procesa lecturas de sensores de vehículos (GPS,
// combustible, temperatura, velocidad): validación, persistencia y
// predicción de autonomía de combustible.
//
// TAREA: "arquitecto backend Go" — ingesta de datos de sensores +
// cálculo predictivo de combustible (alerta si autonomía < 1h) + alerta
// predictiva de exceso de velocidad (> 80 km/h).
package sensors

import "time"

// Reading es una lectura cruda de sensor asociada a un vehículo.
type Reading struct {
	ID           int64
	VehicleID    string
	Lat          float64
	Lon          float64
	FuelLevel    float64 // porcentaje del tanque, 0-100
	TemperatureC float64
	SpeedKmh     float64 // velocidad instantánea reportada por el sensor GPS
	RecordedAt   time.Time
}
