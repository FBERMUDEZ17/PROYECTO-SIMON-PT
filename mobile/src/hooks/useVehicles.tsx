// Equivalente mobile de web/src/hooks/useVehicles.ts + useLiveVehicles.ts
// combinados: fetch inicial (con fallback a cache SQLite si falla por
// estar offline), y aplica los eventos WS entrantes en memoria para que la
// UI se actualice en vivo sin refetch.
//
// Convertido a Context (mismo patrón que useAuth.tsx) para que Dashboard,
// Alertas, el detalle de vehículo y los badges del tab bar comparten una
// única suscripción WS/fetch en vez de que cada pantalla abra su propio
// socket — además de ser más eficiente en batería/datos, es lo que
// permite que VehicleDetailScreen lea el vehículo ya cargado por el
// listado en vez de volver a pedirlo por id (ver comentario en
// VehicleDetailScreen.tsx: para no-admin el id ya viene enmascarado y
// GET /vehicles/{id} con ese id 404 siempre).
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";

import { listVehicles } from "@/api/vehicles";
import { NetworkError } from "@/api/client";
import { cacheVehicles, getCachedVehicles } from "@/storage/vehiclesCache";
import { useRealtimeSocket } from "./useRealtimeSocket";
import type { Vehicle, WsEvent } from "@/types/api";

interface VehiclesContextValue {
  vehicles: Vehicle[];
  loading: boolean;
  error: string | null;
  isOffline: boolean;
  refresh: () => Promise<void>;
  wsStatus: "idle" | "connecting" | "open" | "closed";
}

const VehiclesContext = createContext<VehiclesContextValue | null>(null);

export function VehiclesProvider({ children }: { children: ReactNode }) {
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

  const value = useMemo(
    () => ({ vehicles, loading, error, isOffline, refresh, wsStatus: status }),
    [vehicles, loading, error, isOffline, refresh, status]
  );

  return <VehiclesContext.Provider value={value}>{children}</VehiclesContext.Provider>;
}

export function useVehicles(): VehiclesContextValue {
  const ctx = useContext(VehiclesContext);
  if (!ctx) throw new Error("useVehicles debe usarse dentro de <VehiclesProvider>");
  return ctx;
}
