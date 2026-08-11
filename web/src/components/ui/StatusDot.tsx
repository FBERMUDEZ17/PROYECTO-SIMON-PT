// TAREA: indicador visual de estado (online/offline, WS) usado en el
// Header — apoya "sync automático cuando hay conexión".
const COLORS = {
  online: "bg-emerald-500",
  offline: "bg-red-500",
  warn: "bg-amber-500",
  connecting: "bg-amber-400 animate-pulse",
} as const;

export function StatusDot({
  status,
  label,
}: {
  status: keyof typeof COLORS;
  label: string;
}) {
  return (
    <span className="inline-flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400">
      <span className={`h-2 w-2 rounded-full ${COLORS[status]}`} aria-hidden />
      {label}
    </span>
  );
}
