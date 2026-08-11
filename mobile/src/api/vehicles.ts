import { api } from "./client";
import type { SensorDataRequest, SensorDataResponse, Vehicle } from "@/types/api";

export async function listVehicles(): Promise<Vehicle[]> {
  return api.get<Vehicle[]>("/vehicles");
}

export async function getVehicle(id: string): Promise<Vehicle> {
  return api.get<Vehicle>(`/vehicles/${encodeURIComponent(id)}`);
}

// Usado tanto por el flujo online (envío directo) como por el drenado de
// la cola offline (ver storage/offlineQueue.ts) — misma forma de request.
export async function sendSensorData(data: SensorDataRequest): Promise<SensorDataResponse> {
  return api.post<SensorDataResponse>("/sensors/data", data);
}
