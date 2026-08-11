"use client";

// TAREA: "admin crea un vehículo nuevo asignado a un propietario" — hooks
// para el selector de usuarios y la creación del vehículo (ver
// internal/httpapi/admin_handlers.go).
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { VEHICLES_QUERY_KEY } from "./useVehicles";
import type { AdminUserSummary, CreateVehicleInput, CreateVehicleResponse } from "@/types/api";

const ADMIN_USERS_QUERY_KEY = ["admin", "users"] as const;

/** Lista de usuarios registrados, para elegir el propietario del vehículo nuevo. */
export function useAdminUsers() {
  return useQuery<AdminUserSummary[]>({
    queryKey: ADMIN_USERS_QUERY_KEY,
    queryFn: () => api.get<AdminUserSummary[]>("/admin/users"),
    staleTime: 30_000,
  });
}

export function useCreateVehicle() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateVehicleInput) => api.post<CreateVehicleResponse>("/admin/vehicles", input),
    onSuccess: () => {
      // El nuevo vehículo también llega por WebSocket a otras pestañas
      // abiertas, pero esta pestaña (la del formulario) no está suscrita
      // al hub del dashboard: invalidar fuerza el refetch si el admin
      // vuelve a /dashboard.
      void queryClient.invalidateQueries({ queryKey: VEHICLES_QUERY_KEY });
    },
  });
}
