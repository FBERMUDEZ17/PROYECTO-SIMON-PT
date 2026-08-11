"use client";

// TAREA: barra superior del dashboard — muestra estado de conexión/WS
// (apoya "sync automático") y el botón de logout (auth JWT en cookie).
import { useCurrentUser, useLogout } from "@/hooks/useAuth";
import { StatusDot } from "@/components/ui/StatusDot";
import { RoleBadge } from "@/components/ui/RoleBadge";
import type { ConnectionStatus } from "@/hooks/useRealtimeEvents";

export function Header({ online, wsStatus }: { online: boolean; wsStatus: ConnectionStatus }) {
  const { data: user } = useCurrentUser();
  const logout = useLogout();

  return (
    <header className="flex items-center justify-between border-b border-slate-200 bg-white px-6 py-3 dark:border-slate-800 dark:bg-slate-900">
      <div>
        <h1 className="text-lg font-semibold text-slate-900 dark:text-slate-100">SIMON PT</h1>
        <div className="mt-1 flex gap-3">
          <StatusDot status={online ? "online" : "offline"} label={online ? "En línea" : "Sin conexión"} />
          <StatusDot
            status={wsStatus === "open" ? "online" : wsStatus === "connecting" ? "connecting" : "offline"}
            label={`WS: ${wsStatus}`}
          />
        </div>
      </div>
      {user && (
        <div className="flex items-center gap-3 text-sm">
          <span className="text-slate-600 dark:text-slate-300">{user.name}</span>
          <RoleBadge role={user.role} />
          <button
            type="button"
            onClick={() => logout.mutate()}
            className="rounded-md border border-slate-300 px-3 py-1.5 text-slate-700 transition hover:bg-slate-100 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800"
          >
            Salir
          </button>
        </div>
      )}
    </header>
  );
}
