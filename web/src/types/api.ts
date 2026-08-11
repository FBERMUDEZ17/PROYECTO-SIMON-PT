// TAREA (frontend Next.js): tipos compartidos que reflejan las respuestas
// JSON del backend Go (ver internal/httpapi/dto.go y vehicles_handlers.go).

export type Role = "user" | "admin";

export interface User {
  id: number;
  email: string;
  name: string;
  role: Role;
  created_at: string;
}

export interface AuthResponse {
  user: User;
  token: string; // No lo usamos desde el navegador (viaja en cookie httpOnly);
  // se incluye por compatibilidad con clientes API/tests.
}

export interface ApiError {
  error: string;
}

export interface SensorReading {
  lat: number;
  lon: number;
  fuel_level: number; // porcentaje 0-100
  temperature_c: number;
  speed_kmh: number; // velocidad instantánea reportada por el sensor GPS
  recorded_at: string; // RFC3339
  autonomy_hours?: number; // ausente si no hay predicción válida (nivel estable/subiendo o datos insuficientes)
}

export interface Alert {
  type: string;
  message: string;
  created_at: string;
}

/** device id ya viene enmascarado por el backend si el usuario no es admin. */
export interface Vehicle {
  id: string;
  name: string;
  latest_reading?: SensorReading;
  /** Solo presente en GET /vehicles/{id} (no en el listado), para gráficos. */
  recent_readings?: SensorReading[];
  recent_alerts: Alert[];
}

// TAREA: "consumir WebSocket del backend" — forma de los eventos
// (ver internal/ws/hub.go).
export interface SensorReadingEvent {
  type: "sensor_reading";
  vehicle_id: string;
  payload: SensorReading;
}

export interface AlertEvent {
  type: "alert";
  vehicle_id: string;
  payload: {
    alert_type: string;
    message: string;
  };
}

export type WsEvent = SensorReadingEvent | AlertEvent;

// TAREA: "admin crea un vehículo nuevo asignado a un propietario" — ver
// internal/httpapi/admin_handlers.go (GET /admin/users, POST /admin/vehicles).
export interface AdminUserSummary {
  id: number;
  email: string;
  name: string;
  role: Role;
}

export interface CreateVehicleInput {
  vehicle_id: string;
  owner_user_id: number;
  lat: number;
  lon: number;
  fuel_level: number;
  temperature_c: number;
  speed_kmh: number;
}

export interface CreateVehicleResponse {
  vehicle_id: string;
  owner_user_id: number;
  recorded_at: string;
}

export const LOW_FUEL_THRESHOLD_HOURS = 1;
// TAREA: alerta predictiva de exceso de velocidad (ver
// internal/telemetry/service.go: HighSpeedThresholdKmh, solo admin).
export const HIGH_SPEED_THRESHOLD_KMH = 80;
