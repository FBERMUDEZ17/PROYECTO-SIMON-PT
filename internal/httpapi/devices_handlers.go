package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"proyecto_simon_pt/internal/devices"
)

// DevicesHandler expone el registro de push tokens del cliente mobile.
//
// TAREA (mobile): "POST /devices/register" — persiste el Expo push token
// del dispositivo autenticado (ver mobile/src/notifications/push.ts), para
// poder avisarle por push cuando se cree una alerta de uno de sus
// vehículos (ver broadcastIngestResult en sensors_handlers.go, que
// dispara internal/push.Client.Send con los tokens de devices.Store).
type DevicesHandler struct {
	devices *devices.Store
}

func NewDevicesHandler(d *devices.Store) *DevicesHandler {
	return &DevicesHandler{devices: d}
}

type registerDeviceRequest struct {
	PushToken string `json:"push_token"`
	Platform  string `json:"platform"`
}

// Register maneja POST /devices/register (protegida por RequireAuth).
// El token se guarda asociado al usuario autenticado, no a uno elegido
// por el cliente, para que nadie pueda registrar un token a nombre de
// otro usuario.
func (h *DevicesHandler) Register(w http.ResponseWriter, r *http.Request) {
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

	var req registerDeviceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo de la petición inválido: "+err.Error())
		return
	}

	req.PushToken = strings.TrimSpace(req.PushToken)
	if req.PushToken == "" {
		writeError(w, http.StatusBadRequest, "push_token es requerido")
		return
	}
	if req.Platform != "ios" && req.Platform != "android" {
		writeError(w, http.StatusBadRequest, "platform debe ser 'ios' o 'android'")
		return
	}

	if err := h.devices.Register(userID, req.PushToken, req.Platform); err != nil {
		writeError(w, http.StatusInternalServerError, "error interno registrando el dispositivo")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}
