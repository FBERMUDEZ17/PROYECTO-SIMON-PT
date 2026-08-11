// TAREA: helper compartido por los gráficos históricos (fuel/speed/temp) —
// el eje X debe ser un valor numérico (epoch ms), no una etiqueta de texto
// "HH:mm" repetida entre días. Con datasets de varios días (lecturas cada
// 2h), esa etiqueta se repite (ej. "08:02" aparece un día sí y otro
// también); Recharts trata el eje X categórico por el valor de la
// etiqueta, así que con etiquetas duplicadas el tooltip termina
// resolviendo al índice equivocado al pasar el mouse — el bug reportado
// ("la barra vertical no coincide con el valor puntual"). Usar el
// timestamp real como dataKey numérico evita la ambigüedad.

/** Etiqueta corta para los ticks del eje X (fecha + hora). */
export function formatChartTick(ms: number): string {
  return new Date(ms).toLocaleString([], {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/** Etiqueta completa para el tooltip (fecha + hora, más legible que el tick). */
export function formatChartTooltipLabel(ms: number): string {
  return new Date(ms).toLocaleString([], {
    weekday: "short",
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}
