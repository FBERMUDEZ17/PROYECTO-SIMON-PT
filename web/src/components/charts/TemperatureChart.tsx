"use client";

// TAREA: variable de temperatura en grados centígrados — visualización
// histórica en el frontend (el backend ya la captura/valida en Celsius,
// ver internal/sensors/validate.go: ErrInvalidTemp, rango -50..150°C).
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from "recharts";
import type { SensorReading } from "@/types/api";
import { formatChartTick, formatChartTooltipLabel } from "@/lib/chartTime";

/** Historial de temperatura (°C) de un vehículo. */
export function TemperatureChart({ readings }: { readings: SensorReading[] }) {
  const data = readings.map((r) => ({
    ts: new Date(r.recorded_at).getTime(),
    temperature: r.temperature_c,
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
        <YAxis fontSize={12} unit="°C" width={48} />
        <Tooltip
          labelFormatter={(ts) => formatChartTooltipLabel(Number(ts))}
          formatter={(value) => [`${Number(value).toFixed(1)}°C`, "Temperatura"]}
        />
        <Line
          type="linear"
          dataKey="temperature"
          stroke="#a855f7"
          strokeWidth={2}
          dot={false}
          isAnimationActive={false}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}
