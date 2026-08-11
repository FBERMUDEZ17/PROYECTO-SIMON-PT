package httpapi

import (
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"proyecto_simon_pt/internal/alerts"
	"proyecto_simon_pt/internal/auth"
	"proyecto_simon_pt/internal/devices"
	"proyecto_simon_pt/internal/push"
	"proyecto_simon_pt/internal/sensors"
	"proyecto_simon_pt/internal/telemetry"
	"proyecto_simon_pt/internal/vehicles"
	"proyecto_simon_pt/internal/ws"
)

// Deps agrupa todas las dependencias necesarias para armar el router HTTP.
type Deps struct {
	Auth      *auth.Service
	Telemetry *telemetry.Service
	Vehicles  *vehicles.Store
	Readings  *sensors.Store
	Alerts    *alerts.Store
	Devices   *devices.Store
	Hub       *ws.Hub
}

// allowedOrigins lee FRONTEND_ORIGIN (coma-separado) para CORS. Con
// credentials:true, el navegador exige un origen explícito — no se puede
// usar "*". Default: el origen típico de `next dev`.
// TAREA (frontend Next.js): CORS con credenciales, necesario para que el
// navegador envíe/reciba la cookie httpOnly entre localhost:3000 y :8080.
func allowedOrigins() []string {
	raw := os.Getenv("FRONTEND_ORIGIN")
	if raw == "" {
		return []string{"http://localhost:3000"}
	}
	origins := strings.Split(raw, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}
	return origins
}

// NewRouter arma el router chi con todas las rutas de la API:
//   - públicas:   POST /auth/register, POST /auth/login, POST /auth/logout
//   - protegidas: GET /auth/me, POST /sensors/data, GET /vehicles/{id},
//                 POST /devices/register (registro de push tokens mobile)
//   - solo-admin: GET /admin/users, POST /admin/vehicles
//   - /ws:        autenticación propia (header, query token, o cookie)
func NewRouter(deps Deps) http.Handler {
	authHandler := NewHandler(deps.Auth)

	// notifier queda nil si no se pasó un Devices store (ej. wiring viejo
	// / algunos tests): broadcastIngestResult lo trata como "push
	// deshabilitado" en ese caso, sin romper nada.
	var notifier *pushNotifier
	if deps.Devices != nil {
		notifier = newPushNotifier(deps.Vehicles, deps.Devices, push.NewClient())
	}

	sensorsHandler := NewSensorsHandler(deps.Telemetry, deps.Hub, notifier)
	vehiclesHandler := NewVehiclesHandler(deps.Vehicles, deps.Readings, deps.Alerts)
	adminHandler := NewAdminHandler(deps.Auth, deps.Telemetry, deps.Hub, notifier)
	devicesHandler := NewDevicesHandler(deps.Devices)

	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true, // imprescindible para que el navegador envíe/reciba la cookie httpOnly
		MaxAge:           300,
	}))

	// TAREA (Prompt 4): rutas públicas de autenticación.
	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)
	r.Post("/auth/logout", authHandler.Logout) // TAREA (frontend): logout limpia la cookie.
	r.Post("/auth/forgot-password", authHandler.ForgotPassword) // TAREA: restaurador de contraseña.
	r.Post("/auth/reset-password", authHandler.ResetPassword)

	// TAREA (sensores/IoT): WebSocket para actualizaciones en tiempo real.
	// /ws gestiona su propia autenticación (header, ?token=, o cookie),
	// por eso no va detrás de RequireAuth.
	r.Get("/ws", ws.NewHandler(deps.Hub, deps.Auth).ServeHTTP)

	// TAREA (Prompt 6): endpoints protegidos por el middleware de auth
	// (Prompt 5). Los de sensores/vehículos son de la tarea de telemetría IoT.
	r.Group(func(protected chi.Router) {
		protected.Use(RequireAuth(deps.Auth))
		protected.Get("/auth/me", authHandler.Me) // TAREA (frontend): rol del usuario logueado.
		protected.Post("/sensors/data", sensorsHandler.IngestData)
		protected.Get("/vehicles", vehiclesHandler.List)
		protected.Get("/vehicles/{id}", vehiclesHandler.GetByID)
		// TAREA (mobile): registro de push tokens del cliente mobile.
		protected.Post("/devices/register", devicesHandler.Register)
		// TAREA: solo-admin — crear vehículo asignado a un propietario
		// elegido (el handler valida el rol; RequireAuth ya exige login).
		protected.Get("/admin/users", adminHandler.ListUsers)
		protected.Post("/admin/vehicles", adminHandler.CreateVehicle)
	})

	return r
}
