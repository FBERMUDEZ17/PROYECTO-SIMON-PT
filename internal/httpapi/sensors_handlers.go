package httpapi

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"proyecto_simon_pt/internal/alerts"
	"proyecto_simon_pt/internal/devices"
	"proyecto_simon_pt/internal/push"
	"proyecto_simon_pt/internal/sensors"
	"proyecto_simon_pt/internal/telemetry"
	"proyecto_simon_pt/internal/vehicles"
	"proyecto_simon_pt/internal/ws"
)

// SensorsHandler expone la ingesta de datos de sensores.
type SensorsHandler struct {
	telemetry *telemetry.Service
	hub       *ws.Hub
	notifier  *pushNotifier
}

func NewSensorsHandler(t *telemetry.Service, hub *ws.Hub, notifier *pushNotifier) *SensorsHandler {
	return &SensorsHandler{telemetry: t, hub: hub, notifier: notifier}
}

// pushNotifier agrupa lo necesario para avisar por push al dueño de un
// vehículo cuando se le crea una alerta nueva (ver broadcastIngestResult,
// más abajo). Puede pasarse nil (ej. en tests que no necesitan probar
// push): en ese caso broadcastIngestResult simplemente no manda nada.
//
// TAREA (mobile): "notificaciones push para alertas", lado servidor —
// junto con internal/devices (persistencia de tokens) e internal/push
// (cliente de la Expo Push API), es lo que le da contenido real al
// endpoint POST /devices/register que registra el cliente mobile.
type pushNotifier struct {
	vehicles *vehicles.Store
	devices  *devices.Store
	client   *push.Client
}

func newPushNotifier(v *vehicles.Store, d *devices.Store, c *push.Client) *pushNotifier {
	return &pushNotifier{vehicles: v, devices: d, client: c}
}

type sensorDataRequest struct {
	VehicleID    string  `json:"vehicle_id"`
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
	FuelLevel    float64 `json:"fuel_level"`
	TemperatureC float64 `json:"temperature_c"`
	SpeedKmh     float64 `json:"speed_kmh"`
	// RecordedAt es opcional (RFC3339); si se omite, se usa la hora del servidor.
	RecordedAt *string `json:"recorded_at,omitempty"`
}

type sensorDataResponse struct {
	VehicleID      string   `json:"vehicle_id"`
	RecordedAt     string   `json:"recorded_at"`
	AutonomyHours  *float64 `json:"autonomy_hours,omitempty"`
	AlertTriggered bool     `json:"alert_triggered"`
}

func isSensorValidationError(err error) bool {
	return errors.Is(err, sensors.ErrMissingVehicleID) ||
		errors.Is(err, sensors.ErrInvalidVehicleID) ||
		errors.Is(err, sensors.ErrInvalidLatitude) ||
		errors.Is(err, sensors.ErrInvalidLongitude) ||
		errors.Is(err, sensors.ErrInvalidFuelLevel) ||
		errors.Is(err, sensors.ErrInvalidTemp) ||
		errors.Is(err, sensors.ErrInvalidSpeed)
}

// TAREA: "API REST: POST /sensors/data" — valida, persiste, calcula
// predicción de combustible (vía telemetry.Service) y difunde el evento
// por WebSocket (h.broadcast, más abajo).
//
// IngestData maneja POST /sensors/data.
func (h *SensorsHandler) IngestData(w http.ResponseWriter, r *http.Request) {
	var req sensorDataRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo de la petición inválido: "+err.Error())
		return
	}

	recordedAt := time.Now().UTC()
	if req.RecordedAt != nil && *req.RecordedAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.RecordedAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "recorded_at debe tener formato RFC3339")
			return
		}
		recordedAt = parsed
	}

	input := sensors.ReadingInput{
		VehicleID:    req.VehicleID,
		Lat:          req.Lat,
		Lon:          req.Lon,
		FuelLevel:    req.FuelLevel,
		TemperatureC: req.TemperatureC,
		SpeedKmh:     req.SpeedKmh,
	}

	// TAREA: ownership — el usuario autenticado que manda la lectura es
	// quien queda como dueño si el vehículo es nuevo.
	claims, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "no autenticado")
		return
	}
	userID, err := strconv.ParseInt(claims.Sub, 10, 64)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token con sub inválido")
		return
	}

	result, err := h.telemetry.Ingest(input, recordedAt, userID, claims.IsAdmin())
	if err != nil {
		if isSensorValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, vehicles.ErrForbidden) {
			writeError(w, http.StatusForbidden, "este vehículo pertenece a otro usuario")
			return
		}
		writeError(w, http.StatusInternalServerError, "error interno procesando la lectura")
		return
	}

	broadcastIngestResult(h.hub, result, h.notifier)

	resp := sensorDataResponse{
		VehicleID:      result.Reading.VehicleID,
		RecordedAt:     result.Reading.RecordedAt.Format(time.RFC3339),
		AlertTriggered: len(result.Alerts) > 0,
	}
	if autonomy, ok := finiteAutonomy(result.Prediction); ok {
		resp.AutonomyHours = &autonomy
	}

	writeJSON(w, http.StatusCreated, resp)
}

// finiteAutonomy devuelve la autonomía estimada solo si es un número finito
// (una tasa de consumo estable/positiva produce +Inf, que no es serializable
// de forma útil en JSON).
func finiteAutonomy(p sensors.PredictionResult) (float64, bool) {
	if !p.Ok || math.IsInf(p.AutonomyHours, 1) {
		return 0, false
	}
	return p.AutonomyHours, true
}

// broadcastIngestResult difunde por WebSocket el resultado de procesar una
// lectura (evento "sensor_reading", y "alert" por cada alerta creada), y
// además dispara un push (best-effort, no bloqueante) al dueño del
// vehículo por cada alerta nueva, si notifier no es nil. Compartido por
// POST /sensors/data y POST /admin/vehicles (creación de vehículo con su
// primera lectura) — misma forma de evento/push en ambos casos.
func broadcastIngestResult(hub *ws.Hub, result telemetry.IngestResult, notifier *pushNotifier) {
	var autonomyPtr *float64
	if autonomy, ok := finiteAutonomy(result.Prediction); ok {
		autonomyPtr = &autonomy
	}

	hub.Broadcast(ws.Event{
		Type:      "sensor_reading",
		VehicleID: result.Reading.VehicleID,
		Payload: ws.SensorReadingPayload{
			Lat:           result.Reading.Lat,
			Lon:           result.Reading.Lon,
			FuelLevel:     result.Reading.FuelLevel,
			TemperatureC:  result.Reading.TemperatureC,
			SpeedKmh:      result.Reading.SpeedKmh,
			RecordedAt:    result.Reading.RecordedAt.Format(time.RFC3339),
			AutonomyHours: autonomyPtr,
		},
	})

	for _, a := range result.Alerts {
		hub.Broadcast(ws.Event{
			Type:      "alert",
			VehicleID: a.VehicleID,
			Payload: ws.AlertPayload{
				AlertType: a.Type,
				Message:   a.Message,
			},
		})
		notifyOwnerByPush(notifier, a)
	}
}

// notifyOwnerByPush busca los push tokens del dueño del vehículo de la
// alerta y les manda la notificación vía la Expo Push API, en su propia
// goroutine para no retrasar la respuesta de POST /sensors/data. No hace
// nada si notifier es nil (push deshabilitado, p.ej. en tests) o si el
// vehículo no tiene dueño asignado todavía.
func notifyOwnerByPush(notifier *pushNotifier, a alerts.Alert) {
	if notifier == nil {
		return
	}

	go func() {
		vehicle, err := notifier.vehicles.GetByID(a.VehicleID)
		if err != nil || !vehicle.OwnerUserID.Valid {
			return
		}

		tokens, err := notifier.devices.TokensForUser(vehicle.OwnerUserID.Int64)
		if err != nil || len(tokens) == 0 {
			return
		}

		notifier.client.Send(tokens, "Alerta: "+vehicle.Name, a.Message, map[string]string{
			"vehicleId": a.VehicleID,
			"alertType": a.Type,
		})
	}()
}
