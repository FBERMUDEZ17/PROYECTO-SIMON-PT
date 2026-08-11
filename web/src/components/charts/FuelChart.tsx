"use client";

// TAREA: "gráficos históricos (fuel) con Recharts" — dato real del backend
// (fuel_level de cada lectura).
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from "recharts";
import type { SensorReading } from "@/types/api";
import { formatChartTick, formatChartTooltipLabel } from "@/lib/chartTime";

/** Historial de nivel de combustible (%) de un vehículo. */
export function FuelChart({ readings }: { readings: SensorReading[] }) {
  const data = readings.map((r) => ({
    ts: new Date(r.recorded_at).getTime(),
    fuel: r.fuel_level,
  }));

  if (data.length === 0) {
    return <p className="text-sm text-slate-500 dark:text-slate-400">Sin datos históricos todavía.</p>;
  }

  return (
    <ResponsiveContainer width="100%" height={220}>
      <LineChart data={data}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-slate-200 dark:stroke-slate-700" />
        <XAxis
          dataKey="ts"
          type="number"
          domain={["dataMin", "dataMax"]}
          tickFormatter={formatChartTick}
          fontSize={12}
          tickMargin={8}
        />
        <YAxis domain={[0, 100]} fontSize={12} unit="%" width={40} />
        <Tooltip
          labelFormatter={(ts) => formatChartTooltipLabel(Number(ts))}
          formatter={(value) => [`${Number(value).toFixed(1)}%`, "Combustible"]}
        />
        <Line type="linear" dataKey="fuel" stroke="#0ea5e9" strokeWidth={2} dot={false} isAnimationActive={false} />
      </LineChart>
    </ResponsiveContainer>
  );
}
