// TAREA: agrupa los gráficos históricos (fuel + velocidad + temperatura)
// del vehículo seleccionado en el dashboard.
import type { Vehicle } from "@/types/api";
import { Card } from "@/components/ui/Card";
import { FuelChart } from "@/components/charts/FuelChart";
import { SpeedChart } from "@/components/charts/SpeedChart";
import { TemperatureChart } from "@/components/charts/TemperatureChart";

export function VehicleDetailPanel({ vehicle }: { vehicle: Vehicle | null }) {
  if (!vehicle) {
    return (
      <Card title="Detalle del vehículo">
        <p className="text-sm text-slate-500 dark:text-slate-400">
          Seleccioná un vehículo de la lista o el mapa para ver su historial.
        </p>
      </Card>
    );
  }

  const readings = vehicle.recent_readings ?? [];
  const position = vehicle.latest_reading;

  return (
    <div className="flex flex-col gap-4">
      <Card title={`${vehicle.name} — Ubicación`}>
        {position ? (
          <p className="font-mono text-sm text-slate-700 dark:text-slate-300">
            Lat: {position.lat.toFixed(5)} · Lon: {position.lon.toFixed(5)}
          </p>
        ) : (
          <p className="text-sm text-slate-500 dark:text-slate-400">Sin lecturas GPS todavía.</p>
        )}
      </Card>
      <Card title={`${vehicle.name} — Combustible`}>
        <FuelChart readings={readings} />
      </Card>
      <Card title={`${vehicle.name} — Velocidad (km/h)`}>
        <SpeedChart readings={readings} />
      </Card>
      <Card title={`${vehicle.name} — Temperatura (°C)`}>
        <TemperatureChart readings={readings} />
      </Card>
    </div>
  );
}
