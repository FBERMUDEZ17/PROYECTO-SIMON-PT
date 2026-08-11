// Espejo de los DTOs de internal/httpapi/dto.go y vehicles_handlers.go.
// Mantener en sync manualmente con el backend Go (no hay codegen aquí).

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
  // El backend también setea una cookie httpOnly, pero en mobile no hay
  // cookie jar compartido con el navegador: usamos este token en el header
  // "Authorization: Bearer <token>" (mismo tokenFromRequest del backend).
  token: string;
}

export interface ApiErrorBody {
  error: string;
}

export interface SensorReading {
  lat: number;
  lon: number;
  fuel_level: number;
  temperature_c: number;
  speed_kmh: number;
  recorded_at: string;
  autonomy_hours?: number;
}

export interface VehicleAlert {
  type: string;
  message: string;
  created_at: string;
}

// Mismos umbrales que web/src/types/api.ts — usados para los filtros
// rápidos y los badges de "combustible bajo"/"exceso de velocidad".
export const LOW_FUEL_THRESHOLD_HOURS = 1;
export const HIGH_SPEED_THRESHOLD_KMH = 80;

export interface Vehicle {
  id: string; // enmascarado para no-admin (ej. "DEV-****-XC54")
  name: string;
  latest_reading?: SensorReading;
  recent_readings?: SensorReading[];
  recent_alerts: VehicleAlert[];
}

export interface SensorDataRequest {
  vehicle_id: string;
  lat: number;
  lon: number;
  fuel_level: number;
  temperature_c: number;
  speed_kmh: number;
  recorded_at?: string;
}

export interface SensorDataResponse {
  vehicle_id: string;
  recorded_at: string;
  autonomy_hours?: number;
  alert_triggered: boolean;
}

// Eventos del hub WS (internal/ws/hub.go): Type "sensor_reading" | "alert".
export interface WsEventBase {
  type: "sensor_reading" | "alert";
  vehicle_id: string;
}

export interface WsSensorReadingEvent extends WsEventBase {
  type: "sensor_reading";
  payload: SensorReading;
}

export interface WsAlertEvent extends WsEventBase {
  type: "alert";
  payload: {
    alert_type: string;
    message: string;
  };
}

export type WsEvent = WsSensorReadingEvent | WsAlertEvent;
