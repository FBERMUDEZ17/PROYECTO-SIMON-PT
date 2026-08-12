package main

import (
	"log"
	"net/http"
	"os"

	"proyecto_simon_pt/internal/alerts"
	"proyecto_simon_pt/internal/auth"
	"proyecto_simon_pt/internal/db"
	"proyecto_simon_pt/internal/devices"
	"proyecto_simon_pt/internal/httpapi"
	"proyecto_simon_pt/internal/sensors"
	"proyecto_simon_pt/internal/telemetry"
	"proyecto_simon_pt/internal/vehicles"
	"proyecto_simon_pt/internal/ws"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/app.db"
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET debe estar definido")
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("no se pudo abrir la base de datos: %v", err)
	}
	defer database.Close()

	authService := auth.NewService(database, []byte(secret)) // TAREA (Prompt 3): register/login + JWT.

	// TAREA: cuentas demo cargadas en cada arranque (idempotente) — no hay
	// endpoint HTTP para promoverse a admin (por diseño, ver auth.SetRole),
	// así que sin esto un evaluador nuevo no tendría forma de ver la vista
	// admin sin editar la base de datos a mano. Credenciales documentadas
	// en README.md. Desactivable con SEED_DEMO_USERS=false (p.ej. si se
	// apunta este binario a una base de datos real).
	if os.Getenv("SEED_DEMO_USERS") != "false" {
		seedDemoUsers(authService)
	}

	// TAREA (sensores/IoT): wiring de ingesta de sensores, vehículos y alertas.
	vehicleStore := vehicles.NewStore(database)
	readingStore := sensors.NewStore(database)
	alertStore := alerts.NewStore(database)
	deviceStore := devices.NewStore(database) // TAREA (mobile): push tokens, ver POST /devices/register.
	telemetryService := telemetry.NewService(vehicleStore, readingStore, alertStore)

	// TAREA: WebSockets en tiempo real — el hub corre en su propia goroutine.
	hub := ws.NewHub()
	go hub.Run()

	router := httpapi.NewRouter(httpapi.Deps{
		Auth:      authService,
		Telemetry: telemetryService,
		Vehicles:  vehicleStore,
		Readings:  readingStore,
		Alerts:    alertStore,
		Devices:   deviceStore,
		Hub:       hub,
	})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("escuchando en %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}

// seedDemoUsers crea (si no existen ya) una cuenta admin y una cuenta user
// de demostración, para que evaluar el proyecto no dependa de registrarse
// a mano ni de editar la base de datos para ver la vista admin. Idempotente:
// Register devuelve ErrEmailAlreadyExists en corridas siguientes, y SetRole
// es un simple UPDATE por email, seguro de repetir.
func seedDemoUsers(authService *auth.Service) {
	type demoUser struct {
		email    string
		password string
		name     string
		role     string
	}
	demoUsers := []demoUser{
		{email: "admin@simonpt.dev", password: "Admin123!", name: "Admin Demo", role: auth.RoleAdmin},
		{email: "user@simonpt.dev", password: "User123!", name: "Usuario Demo", role: auth.RoleUser},
	}

	for _, u := range demoUsers {
		_, _, err := authService.Register(u.email, u.password, u.name)
		if err != nil && err != auth.ErrEmailAlreadyExists {
			log.Printf("seed: no se pudo crear %s: %v", u.email, err)
			continue
		}
		if u.role == auth.RoleAdmin {
			if err := authService.SetRole(u.email, auth.RoleAdmin); err != nil {
				log.Printf("seed: no se pudo promover %s a admin: %v", u.email, err)
			}
		}
	}
}
