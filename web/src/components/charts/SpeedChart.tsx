"use client";

// TAREA: variable de velocidad (speed_kmh) — dato real reportado por el
// sensor GPS del backend (ver internal/sensors/models.go), no estimado.
// ReferenceLine marca el umbral de la alerta predictiva de exceso de
// velocidad (ver internal/telemetry/service.go: HighSpeedThresholdKmh).
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, ReferenceLine } from "recharts";
import type { SensorReading } from "@/types/api";
import { HIGH_SPEED_THRESHOLD_KMH } from "@/types/api";
import { formatChartTick, formatChartTooltipLabel } from "@/lib/chartTime";

/** Historial de velocidad (km/h) de un vehículo. */
export function SpeedChart({ readings }: { readings: SensorReading[] }) {
  const data = readings.map((r) => ({
    ts: new Date(r.recorded_at).getTime(),
    speed: r.speed_kmh,
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
        <YAxis fontSize={12} unit=" km/h" width={60} />
        <Tooltip
          labelFormatter={(ts) => formatChartTooltipLabel(Number(ts))}
          formatter={(value) => [`${Number(value).toFixed(0)} km/h`, "Velocidad"]}
        />
        <ReferenceLine
          y={HIGH_SPEED_THRESHOLD_KMH}
          stroke="#ef4444"
          strokeDasharray="4 4"
          label={{ value: `${HIGH_SPEED_THRESHOLD_KMH} km/h`, position: "insideTopRight", fontSize: 11, fill: "#ef4444" }}
        />
        <Line type="linear" dataKey="speed" stroke="#f97316" strokeWidth={2} dot={false} isAnimationActive={false} />
      </LineChart>
    </ResponsiveContainer>
  );
}
