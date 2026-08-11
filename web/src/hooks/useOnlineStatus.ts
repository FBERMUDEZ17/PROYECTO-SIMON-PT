"use client";

// TAREA: base de "sync automático cuando hay conexión" — lo consumen
// useVehicles (refetch) y useLiveVehicles/useRealtimeEvents (WS on/off).
import { useSyncExternalStore } from "react";

function subscribe(callback: () => void) {
  window.addEventListener("online", callback);
  window.addEventListener("offline", callback);
  return () => {
    window.removeEventListener("online", callback);
    window.removeEventListener("offline", callback);
  };
}

function getSnapshot() {
  return navigator.onLine;
}

// SSR no tiene `navigator`: useSyncExternalStore usa este snapshot en el
// render de servidor y en la primera pasada de hidratación del cliente
// (evita el hydration mismatch), y recién después re-renderiza con el
// valor real de navigator.onLine.
function getServerSnapshot() {
  return true;
}

/** Refleja navigator.onLine y se actualiza con los eventos online/offline. */
export function useOnlineStatus(): boolean {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
