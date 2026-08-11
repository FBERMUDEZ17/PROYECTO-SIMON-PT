package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"proyecto_simon_pt/internal/alerts"
	"proyecto_simon_pt/internal/sensors"
	"proyecto_simon_pt/internal/vehicles"
)

// TAREA: "API REST: GET /vehicles/{id}" (+ GET /vehicles, agregado luego
// para que el frontend pueda listar/pintar el mapa) — enmascara device
// ids para no-admin y expone la predicción de autonomía por vehículo.

// VehiclesHandler expone la consulta de vehículos.
type VehiclesHandler struct {
	vehicles *vehicles.Store
	readings *sensors.Store
	alerts   *alerts.Store
}

func NewVehiclesHandler(v *vehicles.Store, r *sensors.Store, a *alerts.Store) *VehiclesHandler {
	return &VehiclesHandler{vehicles: v, readings: r, alerts: a}
}

// historyWindow: cuántas lecturas recientes se devuelven en GET
// /vehicles/{id} para alimentar gráficos históricos en el frontend.
const historyWindow = 50

type sensorReadingResponse struct {
	Lat           float64  `json:"lat"`
	Lon           float64  `json:"lon"`
	FuelLevel     float64  `json:"fuel_level"`
	TemperatureC  float64  `json:"temperature_c"`
	SpeedKmh      float64  `json:"speed_kmh"`
	RecordedAt    string   `json:"recorded_at"`
	AutonomyHours *float64 `json:"autonomy_hours,omitempty"`
}

type alertResponse struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type vehicleResponse struct {
	ID             string                  `json:"id"`
	Name           string                  `json:"name"`
	LatestReading  *sensorReadingResponse  `json:"latest_reading,omitempty"`
	RecentReadings []sensorReadingResponse `json:"recent_readings,omitempty"`
	RecentAlerts   []alertResponse         `json:"recent_alerts"`
}

// currentUser extrae (userID, isAdmin) del contexto de la request
// (adjuntado por RequireAuth). Todas las rutas de este handler están
// protegidas, así que ok debería ser siempre true; se maneja igual por
// robustez.
func currentUser(r *http.Request) (userID int64, isAdmin bool, err error) {
	claims, ok := UserFromContext(r.Context())
	if !ok {
		return 0, false, errors.New("no autenticado")
	}
	userID, err = strconv.ParseInt(claims.Sub, 10, 64)
	if err != nil {
		return 0, false, errors.New("token con sub inválido")
	}
	return userID, claims.IsAdmin(), nil
}

// List maneja GET /vehicles: devuelve los vehículos del usuario (todos si
// es admin) con su historial reciente (para gráficos) y predicción de
// autonomía. Enmascara el device id para no-admin.
//
// TAREA: ownership — "si un usuario posee más de un vehículo, que
// seleccione a cuál monitorear": este listado ya viene filtrado a solo
// los vehículos del usuario autenticado (salvo admin, que ve todos), así
// que la selección en el frontend siempre es entre vehículos propios.
//
// Nota: incluye historial también acá (no solo en GetByID) a propósito:
// para un usuario no-admin, el id que ve en este listado ya viene
// enmascarado ("DEV-****-XC54"), así que no existe forma de volver a
// pedir GET /vehicles/{id} con ese id enmascarado (el backend no lo
// reconoce, 404). El frontend depende de este único listado como fuente
// completa de datos; no hace un segundo fetch por id.
func (h *VehiclesHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, isAdmin, err := currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	all, err := h.vehicles.List(userID, isAdmin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error interno consultando vehículos")
		return
	}

	resp := make([]vehicleResponse, 0, len(all))
	for _, v := range all {
		vr, err := h.buildVehicleResponse(v, isAdmin, historyWindow, true)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "error interno consultando lecturas")
			return
		}
		resp = append(resp, vr)
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetByID maneja GET /vehicles/{id}. Requiere autenticación (RequireAuth);
// para usuarios no-admin, el device id se enmascara en la respuesta
// (ej. "DEV-1234-XC54" -> "DEV-****-XC54"). Incluye recent_readings
// (historial) para alimentar gráficos en el frontend.
//
// TAREA: ownership — si el vehículo no es del usuario (y no es admin),
// responde 404 en vez de 403: no confirma la existencia de vehículos
// ajenos (mismo criterio que Login: no revelar información de otros).
func (h *VehiclesHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	userID, isAdmin, err := currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	v, err := h.vehicles.GetByID(id)
	if err != nil {
		if errors.Is(err, vehicles.ErrNotFound) {
			writeError(w, http.StatusNotFound, "vehículo no encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "error interno consultando el vehículo")
		return
	}

	if !isAdmin && !v.IsOwnedBy(userID) {
		writeError(w, http.StatusNotFound, "vehículo no encontrado")
		return
	}

	resp, err := h.buildVehicleResponse(v, isAdmin, historyWindow, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error interno consultando lecturas")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// buildVehicleResponse arma la respuesta de un vehículo (con id/nombre
// enmascarados si corresponde, la última lectura + predicción de
// autonomía, y opcionalmente el historial de lecturas), usando hasta
// `readingWindow` lecturas recientes para la regresión de combustible.
//
// Incluye siempre las últimas 10 alertas (List() las necesita para el
// panel de alertas predictivas del admin, no solo GetByID).
func (h *VehiclesHandler) buildVehicleResponse(v vehicles.Vehicle, isAdmin bool, readingWindow int, includeHistory bool) (vehicleResponse, error) {
	resp := vehicleResponse{ID: v.ID, Name: v.Name, RecentAlerts: []alertResponse{}}

	recentAlerts, err := h.alerts.ListByVehicle(v.ID, 10)
	if err != nil {
		return vehicleResponse{}, err
	}
	for _, a := range recentAlerts {
		resp.RecentAlerts = append(resp.RecentAlerts, alertResponse{Type: a.Type, Message: a.Message, CreatedAt: a.CreatedAt})
	}
	if !isAdmin {
		resp.ID = vehicles.MaskDeviceID(v.ID)
		resp.Name = vehicles.MaskDeviceID(v.Name)
	}

	recent, err := h.readings.Recent(v.ID, readingWindow)
	if err != nil {
		return vehicleResponse{}, err
	}
	if len(recent) == 0 {
		return resp, nil
	}

	points := make([]sensors.FuelPoint, len(recent))
	for i, reading := range recent {
		points[i] = sensors.FuelPoint{RecordedAt: reading.RecordedAt, FuelLevel: reading.FuelLevel}
	}
	prediction := sensors.PredictAutonomy(points)

	toResponse := func(reading sensors.Reading, withAutonomy bool) sensorReadingResponse {
		sr := sensorReadingResponse{
			Lat:          reading.Lat,
			Lon:          reading.Lon,
			FuelLevel:    reading.FuelLevel,
			TemperatureC: reading.TemperatureC,
			SpeedKmh:     reading.SpeedKmh,
			RecordedAt:   reading.RecordedAt.Format(time.RFC3339),
		}
		if withAutonomy {
			if autonomy, ok := finiteAutonomy(prediction); ok {
				sr.AutonomyHours = &autonomy
			}
		}
		return sr
	}

	latest := toResponse(recent[len(recent)-1], true)
	resp.LatestReading = &latest

	if includeHistory {
		resp.RecentReadings = make([]sensorReadingResponse, len(recent))
		for i, reading := range recent {
			resp.RecentReadings[i] = toResponse(reading, false)
		}
	}

	return resp, nil
}
