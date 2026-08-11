// TAREA: "sistema de alertas predictivas (solo admin visible)".
import type { Vehicle } from "@/types/api";

/**
 * Panel de alertas predictivas. La visibilidad "solo admin" es
 * responsabilidad de quien lo usa (ver app/dashboard/page.tsx): este
 * componente no valida el rol por sí mismo porque los datos que recibe
 * (recent_alerts) ya vienen del backend, que es quien realmente decide qué
 * exponer según el JWT de la sesión.
 */
export function AlertsPanel({ vehicles }: { vehicles: Vehicle[] }) {
  const alerts = vehicles
    .flatMap((v) => v.recent_alerts.map((a) => ({ ...a, vehicleId: v.id, vehicleName: v.name })))
    .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());

  if (alerts.length === 0) {
    return <p className="text-sm text-slate-500 dark:text-slate-400">Sin alertas activas.</p>;
  }

  // TAREA (mejora UX): agrupar por vehículo — con una flota más grande, el
  // mismo vehículo genera varias alertas seguidas (velocidad, combustible)
  // y listarlas todas sueltas hace que un solo vehículo problemático tape
  // el resto. Se muestra la más reciente de cada vehículo, con el conteo
  // de las demás.
  const byVehicle = new Map<string, { vehicleName: string; latest: (typeof alerts)[number]; count: number }>();
  for (const a of alerts) {
    const existing = byVehicle.get(a.vehicleId);
    if (existing) {
      existing.count += 1;
    } else {
      byVehicle.set(a.vehicleId, { vehicleName: a.vehicleName, latest: a, count: 1 });
    }
  }
  const grouped = [...byVehicle.values()].sort(
    (a, b) => new Date(b.latest.created_at).getTime() - new Date(a.latest.created_at).getTime(),
  );

  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs text-slate-400">
        {alerts.length} alerta{alerts.length === 1 ? "" : "s"} en {grouped.length} vehículo
        {grouped.length === 1 ? "" : "s"}
      </p>
      <ul className="flex max-h-[50vh] flex-col gap-2 overflow-y-auto">
        {grouped.map((g) => (
          <li
            key={g.latest.vehicleId}
            className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm dark:border-amber-900 dark:bg-amber-950"
          >
            <div className="flex items-center justify-between gap-2">
              <p className="font-medium text-amber-800 dark:text-amber-200">{g.vehicleName}</p>
              {g.count > 1 && (
                <span className="shrink-0 rounded-full bg-amber-200 px-2 py-0.5 text-xs font-semibold text-amber-800 dark:bg-amber-900 dark:text-amber-200">
                  ×{g.count}
                </span>
              )}
            </div>
            <p className="text-amber-700 dark:text-amber-300">{g.latest.message}</p>
            <p className="text-xs text-amber-600/70 dark:text-amber-400/70">
              {new Date(g.latest.created_at).toLocaleString()}
            </p>
          </li>
        ))}
      </ul>
    </div>
  );
}
