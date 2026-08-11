// Package telemetry orquesta el procesamiento de una lectura de sensor
// entrante: valida, persiste, calcula la predicción de autonomía de
// combustible y genera alertas cuando corresponde (combustible bajo,
// exceso de velocidad).
//
// TAREA: "sensor_processing" — capa de orquestación que junta validación
// (sensors.Validate), predicción (sensors.PredictAutonomy) y alertas
// (alerts.Store), llamada desde POST /sensors/data.
package telemetry

import (
	"errors"
	"fmt"
	"time"

	"proyecto_simon_pt/internal/alerts"
	"proyecto_simon_pt/internal/sensors"
	"proyecto_simon_pt/internal/vehicles"
)

const (
	// predictionWindow: cuántas lecturas recientes se usan para la regresión.
	predictionWindow = 10
	// alertDedupeWindow: no repetir la misma alerta para el mismo vehículo
	// dentro de esta ventana.
	alertDedupeWindow = 15 * time.Minute
	// HighSpeedThresholdKmh: por encima de esto se dispara la alerta
	// predictiva de exceso de velocidad (solo visible para admin, ver
	// AlertsPanel en el frontend — igual que la de combustible bajo).
	HighSpeedThresholdKmh = 80.0
)

// ErrVehicleAlreadyExists: se intentó crear (no reportar telemetría de) un
// vehículo cuyo id ya está en uso. Ver CreateVehicle.
var ErrVehicleAlreadyExists = errors.New("ya existe un vehículo con ese id")

// IngestResult es lo que resulta de procesar una lectura entrante.
type IngestResult struct {
	Reading    sensors.Reading
	Prediction sensors.PredictionResult
	Alerts     []alerts.Alert // vacío si no se creó ninguna alerta nueva
}

type Service struct {
	vehicles *vehicles.Store
	readings *sensors.Store
	alerts   *alerts.Store
	now      func() time.Time // inyectable para tests
}

func NewService(v *vehicles.Store, r *sensors.Store, a *alerts.Store) *Service {
	return &Service{vehicles: v, readings: r, alerts: a, now: time.Now}
}

// Ingest valida la lectura, la persiste, recalcula la predicción de
// autonomía de combustible del vehículo, y crea alertas si corresponde
// (combustible bajo y/o exceso de velocidad; ambas deduplicadas dentro de
// alertDedupeWindow — pueden dispararse las dos en la misma lectura).
//
// TAREA: ownership — submittedByUserID es el usuario autenticado que
// manda la lectura (POST /sensors/data ya requiere login). Si el vehículo
// es nuevo, queda a su nombre; si ya existe y pertenece a otro usuario (y
// quien reporta no es admin), se rechaza con vehicles.ErrForbidden.
func (s *Service) Ingest(input sensors.ReadingInput, recordedAt time.Time, submittedByUserID int64, isAdmin bool) (IngestResult, error) {
	if err := input.Validate(); err != nil { // TAREA: validación de entrada.
		return IngestResult{}, err
	}

	existing, err := s.vehicles.GetByID(input.VehicleID)
	switch {
	case errors.Is(err, vehicles.ErrNotFound):
		// Vehículo nuevo: queda a nombre de quien manda la primera lectura.
		if err := s.vehicles.EnsureExists(input.VehicleID, input.VehicleID, submittedByUserID); err != nil {
			return IngestResult{}, fmt.Errorf("asegurando vehículo: %w", err)
		}
	case err != nil:
		return IngestResult{}, fmt.Errorf("consultando vehículo: %w", err)
	default:
		if !isAdmin && !existing.IsOwnedBy(submittedByUserID) {
			return IngestResult{}, vehicles.ErrForbidden
		}
	}

	return s.persistAndEvaluate(input, recordedAt)
}

// CreateVehicle crea un vehículo NUEVO asignado explícitamente a
// ownerUserID (elegido por un admin), con su primera lectura de sensor —
// a diferencia de Ingest, ownerUserID no es el usuario autenticado sino el
// propietario que el admin eligió. Falla con ErrVehicleAlreadyExists si el
// id ya está en uso (acción explícita de "crear", no debe reasignar ni
// agregar datos a un vehículo existente por accidente).
//
// TAREA: "botón para crear un vehículo nuevo" (solo admin) — reutiliza
// vehicles.Store/sensors.Store/alerts.Store existentes, sin cambios de
// schema; comparte la misma validación y evaluación de alertas que Ingest
// (vía persistAndEvaluate).
func (s *Service) CreateVehicle(input sensors.ReadingInput, recordedAt time.Time, ownerUserID int64) (IngestResult, error) {
	if err := input.Validate(); err != nil {
		return IngestResult{}, err
	}

	if _, err := s.vehicles.GetByID(input.VehicleID); err == nil {
		return IngestResult{}, ErrVehicleAlreadyExists
	} else if !errors.Is(err, vehicles.ErrNotFound) {
		return IngestResult{}, fmt.Errorf("consultando vehículo: %w", err)
	}

	if err := s.vehicles.EnsureExists(input.VehicleID, input.VehicleID, ownerUserID); err != nil {
		return IngestResult{}, fmt.Errorf("creando vehículo: %w", err)
	}

	return s.persistAndEvaluate(input, recordedAt)
}

// persistAndEvaluate guarda la lectura, recalcula la predicción de
// autonomía y crea las alertas que correspondan (combustible bajo y/o
// exceso de velocidad). Común a Ingest y CreateVehicle: la única
// diferencia entre ambos es cómo se decide el dueño del vehículo.
func (s *Service) persistAndEvaluate(input sensors.ReadingInput, recordedAt time.Time) (IngestResult, error) {
	// TAREA: persistencia de la lectura cruda (GPS, combustible, temperatura, velocidad).
	reading := sensors.Reading{
		VehicleID:    input.VehicleID,
		Lat:          input.Lat,
		Lon:          input.Lon,
		FuelLevel:    input.FuelLevel,
		TemperatureC: input.TemperatureC,
		SpeedKmh:     input.SpeedKmh,
		RecordedAt:   recordedAt,
	}
	id, err := s.readings.Insert(reading)
	if err != nil {
		return IngestResult{}, fmt.Errorf("guardando lectura: %w", err)
	}
	reading.ID = id

	recent, err := s.readings.Recent(input.VehicleID, predictionWindow)
	if err != nil {
		return IngestResult{}, fmt.Errorf("leyendo histórico: %w", err)
	}

	// TAREA: cálculo predictivo de combustible sobre el historial reciente.
	points := make([]sensors.FuelPoint, len(recent))
	for i, r := range recent {
		points[i] = sensors.FuelPoint{RecordedAt: r.RecordedAt, FuelLevel: r.FuelLevel}
	}
	prediction := sensors.PredictAutonomy(points)

	result := IngestResult{Reading: reading, Prediction: prediction}

	if prediction.LowFuelAlert() { // TAREA: alerta si autonomía < 1h.
		createdAlert, err := s.maybeCreateLowFuelAlert(input.VehicleID, prediction)
		if err != nil {
			return IngestResult{}, fmt.Errorf("creando alerta: %w", err)
		}
		if createdAlert != nil {
			result.Alerts = append(result.Alerts, *createdAlert)
		}
	}

	if reading.SpeedKmh > HighSpeedThresholdKmh { // TAREA: alerta predictiva de exceso de velocidad.
		createdAlert, err := s.maybeCreateHighSpeedAlert(input.VehicleID, reading.SpeedKmh)
		if err != nil {
			return IngestResult{}, fmt.Errorf("creando alerta: %w", err)
		}
		if createdAlert != nil {
			result.Alerts = append(result.Alerts, *createdAlert)
		}
	}

	return result, nil
}

func (s *Service) maybeCreateLowFuelAlert(vehicleID string, prediction sensors.PredictionResult) (*alerts.Alert, error) {
	since := s.now().Add(-alertDedupeWindow)
	count, err := s.alerts.RecentCount(vehicleID, alerts.TypeLowFuel, since)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		// Ya se avisó recientemente para este vehículo: evita spam de alertas.
		return nil, nil
	}

	message := fmt.Sprintf("Autonomía estimada: %.1f minutos", prediction.AutonomyHours*60)
	created, err := s.alerts.Create(vehicleID, alerts.TypeLowFuel, message)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// maybeCreateHighSpeedAlert crea una alerta de exceso de velocidad
// (deduplicada dentro de alertDedupeWindow, igual que la de combustible
// bajo) cuando la lectura supera HighSpeedThresholdKmh.
func (s *Service) maybeCreateHighSpeedAlert(vehicleID string, speedKmh float64) (*alerts.Alert, error) {
	since := s.now().Add(-alertDedupeWindow)
	count, err := s.alerts.RecentCount(vehicleID, alerts.TypeHighSpeed, since)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, nil
	}

	message := fmt.Sprintf("Velocidad detectada: %.0f km/h (límite %.0f km/h)", speedKmh, HighSpeedThresholdKmh)
	created, err := s.alerts.Create(vehicleID, alerts.TypeHighSpeed, message)
	if err != nil {
		return nil, err
	}
	return &created, nil
}
