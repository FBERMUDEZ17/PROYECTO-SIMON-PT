"use client";

// TAREA: "consumir WebSocket del backend" — junta useVehicles (fetch +
// caché offline) con useRealtimeEvents (WS), parchando la caché de
// TanStack Query en vivo cuando llega un evento.
import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useVehicles, VEHICLES_QUERY_KEY } from "./useVehicles";
import { useRealtimeEvents } from "./useRealtimeEvents";
import { useOnlineStatus } from "./useOnlineStatus";
import type { Vehicle, WsEvent } from "@/types/api";

const MAX_HISTORY_POINTS = 200;
const MAX_ALERTS = 10;

function applyEvent(vehicle: Vehicle, event: WsEvent): Vehicle {
  if (event.type === "sensor_reading") {
    return {
      ...vehicle,
      latest_reading: event.payload,
      recent_readings: [...(vehicle.recent_readings ?? []), event.payload].slice(-MAX_HISTORY_POINTS),
    };
  }
  return {
    ...vehicle,
    recent_alerts: [
      { type: event.payload.alert_type, message: event.payload.message, created_at: new Date().toISOString() },
      ...vehicle.recent_alerts,
    ].slice(0, MAX_ALERTS),
  };
}

/**
 * useVehicles() + WebSocket: los eventos en tiempo real parchean
 * directamente la caché de TanStack Query (sin esperar al próximo poll).
 */
export function useLiveVehicles() {
  const online = useOnlineStatus();
  const queryClient = useQueryClient();
  const vehiclesQuery = useVehicles();

  const handleEvent = useCallback(
    (event: WsEvent) => {
      queryClient.setQueryData<Vehicle[]>(VEHICLES_QUERY_KEY, (old) => {
        if (!old) return old;
        // Si el vehículo del evento todavía no está en la lista (primera
        // lectura de un device id nuevo), se ignora hasta el próximo
        // refetch: no tenemos suficiente info (name) para insertarlo bien.
        return old.map((v) => (v.id === event.vehicle_id ? applyEvent(v, event) : v));
      });
    },
    [queryClient],
  );

  const wsStatus = useRealtimeEvents(handleEvent, online);

  return { ...vehiclesQuery, wsStatus, online };
}
