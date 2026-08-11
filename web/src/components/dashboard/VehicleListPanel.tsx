// TAREA: "dashboard interactivo" — lista lateral de vehículos con badge de
// combustible/autonomía; el id que se ve acá ya viene enmascarado por el
// backend para no-admin ("enmascarar device IDs en UI").
import { useMemo, useState } from "react";
import type { Vehicle } from "@/types/api";
import { LOW_FUEL_THRESHOLD_HOURS, HIGH_SPEED_THRESHOLD_KMH } from "@/types/api";

function isLowFuel(v: Vehicle) {
  const r = v.latest_reading;
  return !!r && r.autonomy_hours !== undefined && r.autonomy_hours < LOW_FUEL_THRESHOLD_HOURS;
}

function isOverSpeed(v: Vehicle) {
  const r = v.latest_reading;
  return !!r && r.speed_kmh > HIGH_SPEED_THRESHOLD_KMH;
}

type ListFilter = "all" | "low_fuel" | "over_speed";

function fuelBadge(v: Vehicle) {
  const reading = v.latest_reading;
  if (!reading) {
    return <span className="text-xs text-slate-400">Sin lecturas</span>;
  }
  const lowFuel = reading.autonomy_hours !== undefined && reading.autonomy_hours < LOW_FUEL_THRESHOLD_HOURS;
  return (
    <span
      className={`rounded-full px-2 py-0.5 text-xs font-medium ${
        lowFuel
          ? "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300"
          : "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300"
      }`}
    >
      {reading.fuel_level.toFixed(0)}%
      {reading.autonomy_hours !== undefined && ` · ${reading.autonomy_hours.toFixed(1)}h`}
    </span>
  );
}

// TAREA: variable de temperatura en grados centígrados — valor actual
// visible en la lista, además del histórico en VehicleDetailPanel.
function temperatureBadge(v: Vehicle) {
  const reading = v.latest_reading;
  if (!reading) return null;
  return (
    <span className="rounded-full bg-purple-100 px-2 py-0.5 text-xs font-medium text-purple-700 dark:bg-purple-950 dark:text-purple-300">
      {reading.temperature_c.toFixed(0)}°C
    </span>
  );
}

// TAREA: variable de velocidad (speed_kmh) — valor actual visible en la
// lista, en rojo si supera el umbral de la alerta predictiva (>80 km/h).
function speedBadge(v: Vehicle) {
  const reading = v.latest_reading;
  if (!reading) return null;
  const overLimit = reading.speed_kmh > HIGH_SPEED_THRESHOLD_KMH;
  return (
    <span
      className={`rounded-full px-2 py-0.5 text-xs font-medium ${
        overLimit
          ? "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300"
          : "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300"
      }`}
    >
      {reading.speed_kmh.toFixed(0)} km/h
    </span>
  );
}

const FILTERS: { key: ListFilter; label: string }[] = [
  { key: "all", label: "Todos" },
  { key: "low_fuel", label: "Combustible bajo" },
  { key: "over_speed", label: "Exceso de velocidad" },
];

export function VehicleListPanel({
  vehicles,
  selectedId,
  onSelect,
}: {
  vehicles: Vehicle[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<ListFilter>("all");

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return vehicles.filter((v) => {
      if (q && !v.name.toLowerCase().includes(q) && !v.id.toLowerCase().includes(q)) return false;
      if (filter === "low_fuel" && !isLowFuel(v)) return false;
      if (filter === "over_speed" && !isOverSpeed(v)) return false;
      return true;
    });
  }, [vehicles, query, filter]);

  if (vehicles.length === 0) {
    return <p className="text-sm text-slate-500 dark:text-slate-400">Todavía no llegaron datos de sensores.</p>;
  }

  return (
    <div className="flex flex-col gap-2">
      {/* TAREA (mejora UX): buscador + filtros rápidos — con una flota
          creciendo, desplazarse a mano por toda la lista deja de ser
          práctico, así que se puede buscar por nombre/id y acotar a los
          vehículos con alertas activas. */}
      <input
        type="search"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Buscar vehículo por nombre o ID…"
        className="w-full rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-sm text-slate-800 placeholder:text-slate-400 focus:border-slate-400 focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
      />
      <div className="flex flex-wrap gap-1.5">
        {FILTERS.map((f) => (
          <button
            key={f.key}
            type="button"
            onClick={() => setFilter(f.key)}
            className={`rounded-full px-2.5 py-1 text-xs font-medium transition ${
              filter === f.key
                ? "bg-slate-800 text-white dark:bg-slate-100 dark:text-slate-900"
                : "bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700"
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>
      <p className="text-xs text-slate-400">
        {filtered.length} de {vehicles.length} vehículos
      </p>

      {filtered.length === 0 ? (
        <p className="py-2 text-sm text-slate-500 dark:text-slate-400">Ningún vehículo coincide con la búsqueda.</p>
      ) : (
        <ul className="flex max-h-[50vh] flex-col divide-y divide-slate-100 overflow-y-auto dark:divide-slate-800">
          {filtered.map((v) => (
            <li key={v.id}>
              <button
                type="button"
                onClick={() => onSelect(v.id)}
                className={`flex w-full items-center justify-between gap-2 px-1 py-2 text-left transition ${
                  v.id === selectedId ? "bg-slate-100 dark:bg-slate-800" : "hover:bg-slate-50 dark:hover:bg-slate-800/50"
                }`}
              >
                <div>
                  <p className="text-sm font-medium text-slate-800 dark:text-slate-100">{v.name}</p>
                  <p className="font-mono text-xs text-slate-400">{v.id}</p>
                </div>
                <div className="flex items-center gap-1.5">
                  {temperatureBadge(v)}
                  {speedBadge(v)}
                  {fuelBadge(v)}
                </div>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
