"use client";

// TAREA: "caché offline con localStorage/IndexedDB" + "sync automático
// cuando hay conexión" — fetch real con fallback a IndexedDB si falla, e
// invalidación automática de la query al volver online.
import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api, ApiRequestError } from "@/lib/api";
import { cacheVehicles, getCachedVehicles } from "@/lib/indexeddb";
import type { Vehicle } from "@/types/api";
import { useOnlineStatus } from "./useOnlineStatus";

export const VEHICLES_QUERY_KEY = ["vehicles"] as const;

/**
 * Lista completa de vehículos (con historial + predicción de autonomía).
 * Cachea cada respuesta exitosa en IndexedDB y, si el fetch falla (sin
 * red), sirve el último snapshot cacheado en su lugar — esa es la "caché
 * offline". Al recuperar conexión, se invalida y refetchea automáticamente
 * ("sync automático").
 */
export function useVehicles() {
  const online = useOnlineStatus();
  const queryClient = useQueryClient();

  const query = useQuery<Vehicle[], Error, Vehicle[]>({
    queryKey: VEHICLES_QUERY_KEY,
    queryFn: async () => {
      try {
        const data = await api.get<Vehicle[]>("/vehicles");
        void cacheVehicles(data); // TAREA: cachea el snapshot exitoso en IndexedDB.
        return data;
      } catch (err) {
        // El backend respondió (sesión vencida/cerrada, sin permiso, etc.):
        // no es un problema de red, así que no hay que taparlo con el
        // snapshot cacheado — eso dejaba el dashboard mostrando vehículos
        // de otra sesión después de un logout. Solo se cae a la caché
        // offline cuando el fetch ni siquiera llegó al backend.
        if (err instanceof ApiRequestError) throw err;

        // TAREA: sin red -> sirve el último snapshot cacheado (caché offline).
        const cached = await getCachedVehicles();
        if (cached.length > 0) return cached;
        throw err;
      }
    },
    staleTime: 15_000,
    refetchInterval: online ? 15_000 : false,
  });

  // TAREA: "sync automático cuando hay conexión".
  useEffect(() => {
    if (online) {
      void queryClient.invalidateQueries({ queryKey: VEHICLES_QUERY_KEY });
    }
  }, [online, queryClient]);

  return query;
}
