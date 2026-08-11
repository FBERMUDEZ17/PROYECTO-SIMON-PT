"use client";

// TAREA: "botón admin -> nueva ventana para agregar un vehículo nuevo,
// asignado a un propietario" (ver internal/httpapi/admin_handlers.go).
// Reutiliza los mismos campos que ya existen para una lectura de sensor
// (lat/lon/fuel_level/temperature_c/speed_kmh) — no se inventan
// características nuevas, solo se agrega el selector de propietario.
import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCurrentUser } from "@/hooks/useAuth";
import { useAdminUsers, useCreateVehicle } from "@/hooks/useAdmin";
import { ApiRequestError } from "@/lib/api";

const inputClass =
  "rounded-md border border-slate-300 bg-white px-3 py-2 text-slate-900 outline-none focus:border-slate-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100";
const labelClass = "flex flex-col gap-1 text-sm text-slate-600 dark:text-slate-300";

// TAREA (mejora UX): lat/lon por defecto — sin esto el campo queda en
// blanco y hay que salir a buscar coordenadas a mano para cada alta. Se
// usa Bogotá (centro del sistema de referencia geodésico nacional,
// MAGNA-SIRGAS) porque es donde ya está la mayoría de la flota de prueba;
// el admin lo pisa con la ubicación real del vehículo si corresponde.
const DEFAULT_LAT = "4.71100";
const DEFAULT_LON = "-74.07210";

export default function NewVehiclePage() {
  const router = useRouter();
  const { data: user, isPending: userLoading } = useCurrentUser();
  const usersQuery = useAdminUsers();
  const createVehicle = useCreateVehicle();

  // Solo-admin: mismo patrón que /dashboard (redirige apenas se sabe que
  // no corresponde). El backend igual re-valida el rol en cada request.
  useEffect(() => {
    if (userLoading) return;
    if (user === null) {
      router.replace("/login?next=/dashboard/vehicles/new");
    } else if (user && user.role !== "admin") {
      router.replace("/dashboard");
    }
  }, [user, userLoading, router]);

  const [vehicleId, setVehicleId] = useState("");
  const [ownerUserId, setOwnerUserId] = useState("");
  const [lat, setLat] = useState(DEFAULT_LAT);
  const [lon, setLon] = useState(DEFAULT_LON);
  const [fuelLevel, setFuelLevel] = useState("100");
  const [temperatureC, setTemperatureC] = useState("25");
  const [speedKmh, setSpeedKmh] = useState("0");
  const [error, setError] = useState<string | null>(null);
  const [created, setCreated] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      const result = await createVehicle.mutateAsync({
        vehicle_id: vehicleId.trim(),
        owner_user_id: Number(ownerUserId),
        lat: Number(lat),
        lon: Number(lon),
        fuel_level: Number(fuelLevel),
        temperature_c: Number(temperatureC),
        speed_kmh: Number(speedKmh),
      });
      setCreated(result.vehicle_id);
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Error inesperado");
    }
  }

  if (userLoading || !user || user.role !== "admin") {
    return null; // evita parpadeo mientras se resuelve la sesión/rol o se redirige
  }

  return (
    <main className="mx-auto flex w-full max-w-lg flex-1 flex-col gap-4 px-6 py-10">
      <div>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">Agregar vehículo</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          Crea un vehículo nuevo con su primera lectura de sensor, asignado al propietario que elijas.
        </p>
      </div>

      {created ? (
        <div className="flex flex-col gap-3 rounded-md border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-200">
          <p>
            Vehículo <span className="font-mono font-medium">{created}</span> creado correctamente.
          </p>
          <div className="flex gap-3">
            <Link href="/dashboard" className="font-medium underline">
              Ir al dashboard →
            </Link>
            <button
              type="button"
              onClick={() => {
                setCreated(null);
                setVehicleId("");
                setLat(DEFAULT_LAT);
                setLon(DEFAULT_LON);
              }}
              className="font-medium underline"
            >
              Agregar otro
            </button>
          </div>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <label className={labelClass}>
            ID del vehículo (device id)
            <input
              required
              value={vehicleId}
              onChange={(e) => setVehicleId(e.target.value)}
              placeholder="DEV-1234-XC54"
              pattern="[A-Za-z0-9]+(-[A-Za-z0-9]+)*"
              title="Alfanumérico, segmentos separados por guion (ej. DEV-1234-XC54)"
              className={`${inputClass} font-mono`}
            />
          </label>

          <label className={labelClass}>
            Propietario
            {usersQuery.isLoading && <p className="text-xs text-slate-400">Cargando usuarios…</p>}
            {usersQuery.isError && (
              <p className="text-xs text-red-600 dark:text-red-400">No se pudo cargar la lista de usuarios.</p>
            )}
            {usersQuery.data && (
              <select
                required
                value={ownerUserId}
                onChange={(e) => setOwnerUserId(e.target.value)}
                className={inputClass}
              >
                <option value="" disabled>
                  Seleccioná un usuario…
                </option>
                {usersQuery.data.map((u) => (
                  <option key={u.id} value={u.id}>
                    {u.name} ({u.email}){u.role === "admin" ? " — admin" : ""}
                  </option>
                ))}
              </select>
            )}
          </label>

          <div className="grid grid-cols-2 gap-4">
            <label className={labelClass}>
              Latitud
              <input
                required
                type="number"
                step="any"
                min={-90}
                max={90}
                value={lat}
                onChange={(e) => setLat(e.target.value)}
                className={inputClass}
              />
            </label>
            <label className={labelClass}>
              Longitud
              <input
                required
                type="number"
                step="any"
                min={-180}
                max={180}
                value={lon}
                onChange={(e) => setLon(e.target.value)}
                className={inputClass}
              />
            </label>
          </div>
          <p className="-mt-2 text-xs text-slate-400">
            Por defecto: Bogotá ({DEFAULT_LAT}, {DEFAULT_LON}). Reemplazá con la ubicación real del vehículo si la
            conocés.
          </p>

          <div className="grid grid-cols-3 gap-4">
            <label className={labelClass}>
              Combustible (%)
              <input
                required
                type="number"
                step="any"
                min={0}
                max={100}
                value={fuelLevel}
                onChange={(e) => setFuelLevel(e.target.value)}
                className={inputClass}
              />
            </label>
            <label className={labelClass}>
              Temperatura (°C)
              <input
                required
                type="number"
                step="any"
                value={temperatureC}
                onChange={(e) => setTemperatureC(e.target.value)}
                className={inputClass}
              />
            </label>
            <label className={labelClass}>
              Velocidad (km/h)
              <input
                required
                type="number"
                step="any"
                min={0}
                max={300}
                value={speedKmh}
                onChange={(e) => setSpeedKmh(e.target.value)}
                className={inputClass}
              />
            </label>
          </div>

          {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

          <button
            type="submit"
            disabled={createVehicle.isPending || !usersQuery.data}
            className="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-700 disabled:opacity-50 dark:bg-slate-100 dark:text-slate-900"
          >
            {createVehicle.isPending ? "Creando…" : "Crear vehículo"}
          </button>
        </form>
      )}

      <Link href="/dashboard" className="text-sm text-slate-500 underline dark:text-slate-400">
        ← Volver al dashboard
      </Link>
    </main>
  );
}
