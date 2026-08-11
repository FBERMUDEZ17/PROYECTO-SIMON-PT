package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"proyecto_simon_pt/internal/auth"
	"proyecto_simon_pt/internal/sensors"
	"proyecto_simon_pt/internal/telemetry"
	"proyecto_simon_pt/internal/ws"
)

// AdminHandler expone acciones administrativas: listar usuarios (para
// elegir el propietario) y crear un vehículo nuevo asignado a un usuario
// puntual. Ambas rutas son solo-admin.
//
// TAREA: "botón admin para agregar un vehículo nuevo, asignado a un
// propietario" — no agrega tablas ni columnas nuevas, solo reutiliza
// users/vehicles/sensor_readings/alerts ya existentes.
type AdminHandler struct {
	auth      *auth.Service
	telemetry *telemetry.Service
	hub       *ws.Hub
	notifier  *pushNotifier
}

func NewAdminHandler(a *auth.Service, t *telemetry.Service, hub *ws.Hub, notifier *pushNotifier) *AdminHandler {
	return &AdminHandler{auth: a, telemetry: t, hub: hub, notifier: notifier}
}

type userSummaryResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// ListUsers maneja GET /admin/users: usuarios registrados, para poblar el
// selector de "propietario" al crear un vehículo. Solo-admin.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	claims, ok := UserFromContext(r.Context())
	if !ok || !claims.IsAdmin() {
		writeError(w, http.StatusForbidden, "requiere rol admin")
		return
	}

	users, err := h.auth.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error interno consultando usuarios")
		return
	}

	resp := make([]userSummaryResponse, len(users))
	for i, u := range users {
		resp[i] = userSummaryResponse{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role}
	}
	writeJSON(w, http.StatusOK, resp)
}

type createVehicleRequest struct {
	VehicleID    string  `json:"vehicle_id"`
	OwnerUserID  int64   `json:"owner_user_id"`
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
	FuelLevel    float64 `json:"fuel_level"`
	TemperatureC float64 `json:"temperature_c"`
	SpeedKmh     float64 `json:"speed_kmh"`
}

type createVehicleResponse struct {
	VehicleID   string `json:"vehicle_id"`
	OwnerUserID int64  `json:"owner_user_id"`
	RecordedAt  string `json:"recorded_at"`
}

// CreateVehicle maneja POST /admin/vehicles: crea un vehículo nuevo (con
// su primera lectura, mismas características que POST /sensors/data:
// lat/lon/fuel_level/temperature_c/speed_kmh) asignado al propietario que
// el admin elija. Solo-admin.
func (h *AdminHandler) CreateVehicle(w http.ResponseWriter, r *http.Request) {
	claims, ok := UserFromContext(r.Context())
	if !ok || !claims.IsAdmin() {
		writeError(w, http.StatusForbidden, "requiere rol admin")
		return
	}

	var req createVehicleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo de la petición inválido: "+err.Error())
		return
	}

	if req.OwnerUserID == 0 {
		writeError(w, http.StatusBadRequest, "owner_user_id es requerido")
		return
	}
	if _, err := h.auth.GetUserByID(req.OwnerUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "el propietario elegido no existe")
			return
		}
		writeError(w, http.StatusInternalServerError, "error interno consultando el propietario")
		return
	}

	input := sensors.ReadingInput{
		VehicleID:    req.VehicleID,
		Lat:          req.Lat,
		Lon:          req.Lon,
		FuelLevel:    req.FuelLevel,
		TemperatureC: req.TemperatureC,
		SpeedKmh:     req.SpeedKmh,
	}

	result, err := h.telemetry.CreateVehicle(input, time.Now().UTC(), req.OwnerUserID)
	if err != nil {
		if isSensorValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, telemetry.ErrVehicleAlreadyExists) {
			writeError(w, http.StatusConflict, "ya existe un vehículo con ese id")
			return
		}
		writeError(w, http.StatusInternalServerError, "error interno creando el vehículo")
		return
	}

	broadcastIngestResult(h.hub, result, h.notifier)

	writeJSON(w, http.StatusCreated, createVehicleResponse{
		VehicleID:   result.Reading.VehicleID,
		OwnerUserID: req.OwnerUserID,
		RecordedAt:  result.Reading.RecordedAt.Format(time.RFC3339),
	})
}
