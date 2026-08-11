// Package ws implementa el hub de WebSockets para difundir actualizaciones
// de telemetría en tiempo real a clientes autenticados.
//
// TAREA: "WebSockets para actualizaciones en tiempo real" (Gorilla
// WebSocket). renderForRole() también cumple el enmascarado de device ids
// para clientes no-admin conectados al socket.
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"proyecto_simon_pt/internal/auth"
	"proyecto_simon_pt/internal/vehicles"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// La API es sin cookies (auth por Bearer token/JWT), así que no hay
	// riesgo de CSRF vía origen cruzado en el handshake.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Event es un evento de telemetría en tiempo real emitido a los clientes.
type Event struct {
	Type      string `json:"type"` // "sensor_reading" | "alert"
	VehicleID string `json:"vehicle_id"`
	Payload   any    `json:"payload"`
}

// SensorReadingPayload es el payload de un evento "sensor_reading".
type SensorReadingPayload struct {
	Lat           float64  `json:"lat"`
	Lon           float64  `json:"lon"`
	FuelLevel     float64  `json:"fuel_level"`
	TemperatureC  float64  `json:"temperature_c"`
	SpeedKmh      float64  `json:"speed_kmh"`
	RecordedAt    string   `json:"recorded_at"`
	AutonomyHours *float64 `json:"autonomy_hours,omitempty"`
}

// AlertPayload es el payload de un evento "alert".
type AlertPayload struct {
	AlertType string `json:"alert_type"`
	Message   string `json:"message"`
}

// client es una conexión WebSocket registrada, junto con el rol del usuario
// autenticado (determina si se enmascara el device id en los eventos).
type client struct {
	conn *websocket.Conn
	send chan []byte
	role string
}

// Hub mantiene el conjunto de clientes conectados y reenvía eventos,
// enmascarando el device id para clientes no-admin.
type Hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}

	register   chan *client
	unregister chan *client
	broadcast  chan Event
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*client]struct{}),
		register:   make(chan *client),
		unregister: make(chan *client),
		broadcast:  make(chan Event, 256),
	}
}

// Run procesa el loop del hub. Debe lanzarse en su propia goroutine antes
// de aceptar conexiones.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			h.deliver(event)
		}
	}
}

// TAREA: punto de entrada que usan los handlers HTTP (sensors_handlers.go)
// para difundir cada lectura/alerta nueva a todos los clientes conectados.
//
// Broadcast encola un evento para enviarlo a todos los clientes conectados.
// No bloquea: si el buffer está lleno, el evento se descarta y se loggea
// (preferible a bloquear la ingesta de sensores por un lector lento).
func (h *Hub) Broadcast(event Event) {
	select {
	case h.broadcast <- event:
	default:
		log.Printf("ws: buffer de broadcast lleno, evento descartado (type=%s vehicle=%s)", event.Type, event.VehicleID)
	}
}

// ClientCount devuelve el número de clientes conectados (uso en tests/salud).
func (h *Hub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *Hub) deliver(event Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for c := range h.clients {
		rendered := renderForRole(event, c.role)
		data, err := json.Marshal(rendered)
		if err != nil {
			log.Printf("ws: error serializando evento: %v", err)
			continue
		}
		select {
		case c.send <- data:
		default:
			// Cliente lento/atascado: lo desconectamos en vez de bloquear
			// la entrega al resto de clientes.
			delete(h.clients, c)
			close(c.send)
		}
	}
}

// renderForRole enmascara el vehicle_id para roles distintos de admin.
func renderForRole(event Event, role string) Event {
	if role == auth.RoleAdmin {
		return event
	}
	masked := event
	masked.VehicleID = vehicles.MaskDeviceID(event.VehicleID)
	return masked
}
