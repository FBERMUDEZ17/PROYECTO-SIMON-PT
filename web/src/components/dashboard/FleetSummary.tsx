// TAREA (mejora UX): resumen de flota de un vistazo — con más vehículos
// registrándose, el admin necesita un panorama general antes de bucear en
// la lista completa (total, cuántos tienen alerta activa, combustible
// promedio) en vez de tener que escanear cada fila a mano.
import type { Vehicle } from "@/types/api";
import { LOW_FUEL_THRESHOLD_HOURS, HIGH_SPEED_THRESHOLD_KMH } from "@/types/api";

function Stat({ label, value, tone = "default" }: { label: string; value: string; tone?: "default" | "warning" }) {
  return (
    <div className="flex flex-col gap-0.5 rounded-lg border border-slate-200 bg-white px-3 py-2 dark:border-slate-800 dark:bg-slate-900">
      <span className="text-xs text-slate-500 dark:text-slate-400">{label}</span>
      <span
        className={`text-lg font-semibold ${
          tone === "warning" ? "text-red-600 dark:text-red-400" : "text-slate-800 dark:text-slate-100"
        }`}
      >
        {value}
      </span>
    </div>
  );
}

export function FleetSummary({ vehicles }: { vehicles: Vehicle[] }) {
  const withReading = vehicles.filter((v) => v.latest_reading);

  const lowFuelCount = withReading.filter(
    (v) => v.latest_reading!.autonomy_hours !== undefined && v.latest_reading!.autonomy_hours! < LOW_FUEL_THRESHOLD_HOURS,
  ).length;
  const overSpeedCount = withReading.filter((v) => v.latest_reading!.speed_kmh > HIGH_SPEED_THRESHOLD_KMH).length;
  const avgFuel =
    withReading.length > 0
      ? withReading.reduce((sum, v) => sum + v.latest_reading!.fuel_level, 0) / withReading.length
      : null;

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
      <Stat label="Vehículos" value={String(vehicles.length)} />
      <Stat
        label="Combustible bajo"
        value={String(lowFuelCount)}
        tone={lowFuelCount > 0 ? "warning" : "default"}
      />
      <Stat
        label="Exceso de velocidad"
        value={String(overSpeedCount)}
        tone={overSpeedCount > 0 ? "warning" : "default"}
      />
      <Stat label="Combustible promedio" value={avgFuel !== null ? `${avgFuel.toFixed(0)}%` : "—"} />
    </div>
  );
}
