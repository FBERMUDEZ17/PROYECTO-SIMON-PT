package sensors

import (
	"errors"
	"regexp"
)

var (
	ErrMissingVehicleID = errors.New("vehicle_id es requerido")
	ErrInvalidVehicleID = errors.New("vehicle_id tiene un formato inválido")
	ErrInvalidLatitude  = errors.New("lat debe estar entre -90 y 90")
	ErrInvalidLongitude = errors.New("lon debe estar entre -180 y 180")
	ErrInvalidFuelLevel = errors.New("fuel_level debe estar entre 0 y 100")
	ErrInvalidTemp      = errors.New("temperature está fuera de rango físico plausible")
	ErrInvalidSpeed     = errors.New("speed_kmh debe estar entre 0 y 300")
)

// deviceIDPattern acepta ids alfanuméricos con segmentos separados por guion
// (ej. "DEV-1234-XC54"), 3-64 caracteres.
var deviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9]+(-[A-Za-z0-9]+)*$`)

// ReadingInput son los datos crudos recibidos en POST /sensors/data, antes
// de validar.
type ReadingInput struct {
	VehicleID    string
	Lat          float64
	Lon          float64
	FuelLevel    float64
	TemperatureC float64
	SpeedKmh     float64
}

// TAREA: "validación de entrada en todos los endpoints" — reglas de
// negocio para POST /sensors/data (lat/lon/fuel_level/temperatura en
// rango, vehicle_id con formato válido).
//
// Validate aplica las reglas de negocio sobre una lectura entrante.
func (in ReadingInput) Validate() error {
	if in.VehicleID == "" {
		return ErrMissingVehicleID
	}
	if len(in.VehicleID) > 64 || !deviceIDPattern.MatchString(in.VehicleID) {
		return ErrInvalidVehicleID
	}
	if in.Lat < -90 || in.Lat > 90 {
		return ErrInvalidLatitude
	}
	if in.Lon < -180 || in.Lon > 180 {
		return ErrInvalidLongitude
	}
	if in.FuelLevel < 0 || in.FuelLevel > 100 {
		return ErrInvalidFuelLevel
	}
	// Rango físico plausible para un sensor de temperatura de motor/ambiente.
	if in.TemperatureC < -50 || in.TemperatureC > 150 {
		return ErrInvalidTemp
	}
	if in.SpeedKmh < 0 || in.SpeedKmh > 300 {
		return ErrInvalidSpeed
	}
	return nil
}
