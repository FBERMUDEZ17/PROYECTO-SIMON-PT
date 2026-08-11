"use client";

// TAREA: página principal del "dashboard interactivo" — compone mapa,
// lista de vehículos, gráficos históricos y alertas (solo admin) en un
// único layout, todo alimentado por useLiveVehicles (fetch + caché offline
// + WebSocket en vivo).
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useCurrentUser } from "@/hooks/useAuth";
import { useLiveVehicles } from "@/hooks/useLiveVehicles";
import { Header } from "@/components/layout/Header";
import { VehicleMap } from "@/components/map/VehicleMap";
import { VehicleListPanel } from "@/components/dashboard/VehicleListPanel";
import { VehicleDetailPanel } from "@/components/dashboard/VehicleDetailPanel";
import { FleetSummary } from "@/components/dashboard/FleetSummary";
import { AlertsPanel } from "@/components/alerts/AlertsPanel";
import { Card } from "@/components/ui/Card";

export default function DashboardPage() {
  const router = useRouter();
  const { data: user, isPending: userLoading } = useCurrentUser();
  const { data: vehicles = [], isLoading, isError, wsStatus, online } = useLiveVehicles();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  // El proxy (src/proxy.ts) ya bloquea /dashboard sin cookie, pero no
  // valida el JWT (no tiene el secreto). Si el backend lo rechaza (p.ej.
  // expiró), GET /auth/me devuelve null y hay que mandar a /login.
  useEffect(() => {
    if (!userLoading && user === null) {
      router.replace("/login");
    }
  }, [user, userLoading, router]);

  const selectedVehicle = vehicles.find((v) => v.id === selectedId) ?? null;

  return (
    <div className="flex flex-1 flex-col">
      <Header online={online} wsStatus={wsStatus} />

      <main className="flex flex-1 flex-col gap-4 p-4">
        {/* TAREA (mejora UX): resumen de flota — panorama general antes de
            entrar al detalle vehículo por vehículo. */}
        {!isLoading && vehicles.length > 0 && <FleetSummary vehicles={vehicles} />}

        <div className="grid flex-1 gap-4 lg:items-start lg:grid-cols-[300px_1fr]">
          {/* TAREA (mejora UX): columna de vehículos/alertas fija (sticky)
              con su propio scroll — antes compartía el scroll de toda la
              página con los gráficos, y bajar a ver un gráfico hacía perder
              de vista la lista y el mapa. */}
          <div className="flex flex-col gap-4 lg:sticky lg:top-4 lg:max-h-[calc(100vh-2rem)] lg:overflow-y-auto">
            <Card
              title="Vehículos"
              action={
                // TAREA: "botón admin -> nueva ventana para agregar un
                // vehículo nuevo, asignado a un propietario". target="_blank"
                // para que el dashboard actual (con su WS/polling en vivo)
                // siga abierto mientras se completa el formulario.
                user?.role === "admin" && (
                  <Link
                    href="/dashboard/vehicles/new"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="rounded-md bg-slate-900 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-slate-700 dark:bg-slate-100 dark:text-slate-900"
                  >
                    + Agregar vehículo
                  </Link>
                )
              }
            >
              {isLoading && <p className="text-sm text-slate-500 dark:text-slate-400">Cargando…</p>}
              {isError && vehicles.length === 0 && (
                <p className="text-sm text-red-600 dark:text-red-400">
                  No se pudieron cargar los vehículos (sin red y sin datos en caché).
                </p>
              )}
              {!isLoading && <VehicleListPanel vehicles={vehicles} selectedId={selectedId} onSelect={setSelectedId} />}
            </Card>

            {/* TAREA: "sistema de alertas predictivas (solo admin visible)".
                Se acota al vehículo seleccionado (mapa o lista) cuando hay
                uno — antes siempre mostraba las alertas de toda la flota. */}
            {user?.role === "admin" && (
              <Card title="Alertas predictivas (admin)">
                <AlertsPanel vehicles={vehicles} filterVehicleId={selectedId} onClearFilter={() => setSelectedId(null)} />
              </Card>
            )}
          </div>

          <div className="flex flex-col gap-4">
            {/* Mapa fijo arriba de la columna derecha: el detalle/gráficos
                se desplazan debajo sin taparlo. */}
            <div className="lg:sticky lg:top-4 lg:z-10">
              <Card title="Mapa en tiempo real">
                <div className="h-[420px]">
                  <VehicleMap vehicles={vehicles} selectedId={selectedId} onSelect={setSelectedId} />
                </div>
              </Card>
            </div>

            <VehicleDetailPanel vehicle={selectedVehicle} />
          </div>
        </div>
      </main>
    </div>
  );
}
