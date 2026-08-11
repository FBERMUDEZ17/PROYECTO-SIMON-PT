// Equivalente mobile de web/src/hooks/useVehicles.ts + useLiveVehicles.ts
// combinados: fetch inicial (con fallback a cache SQLite si falla por
// estar offline), y aplica los eventos WS entrantes en memoria para que la
// UI se actualice en vivo sin refetch.
import { useCallback, useEffect, useState } from "react";

import { listVehicles } from "@/api/vehicles";
import { NetworkError } from "@/api/client";
import { cacheVehicles, getCachedVehicles } from "@/storage/vehiclesCache";
import { useRealtimeSocket } from "./useRealtimeSocket";
import type { Vehicle, WsEvent } from "@/types/api";

export function useVehicles() {
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [loading, setLoading] = useState(true);
  const [isOffline, setIsOffline] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const fresh = await listVehicles();
      setVehicles(fresh);
      setIsOffline(false);
      setError(null);
      await cacheVehicles(fresh);
    } catch (err) {
      if (err instanceof NetworkError) {
        // Sin red: servimos lo último visto en vez de una pantalla vacía.
        const cached = await getCachedVehicles();
        setVehicles(cached);
        setIsOffline(true);
      } else {
        setError(err instanceof Error ? err.message : "error desconocido");
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const applyEvent = useCallback((event: WsEvent) => {
    setVehicles((prev) => {
      const idx = prev.findIndex((v) => v.id === event.vehicle_id);
      if (idx === -1) return prev; // vehículo aún no visto en el listado; se recogerá en el próximo refresh
      const next = [...prev];
      const vehicle = { ...next[idx] };

      if (event.type === "sensor_reading") {
        vehicle.latest_reading = event.payload;
        vehicle.recent_readings = [...(vehicle.recent_readings ?? []), event.payload].slice(-50);
      } else if (event.type === "alert") {
        vehicle.recent_alerts = [
          { type: event.payload.alert_type, message: event.payload.message, created_at: new Date().toISOString() },
          ...vehicle.recent_alerts,
        ].slice(0, 10);
      }

      next[idx] = vehicle;
      return next;
    });
  }, []);

  const { status } = useRealtimeSocket(applyEvent);

  return { vehicles, loading, error, isOffline, refresh, wsStatus: status };
}
