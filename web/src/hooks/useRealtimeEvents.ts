"use client";

// TAREA: "consumir WebSocket del backend" — conexión con reconexión por
// backoff exponencial.
import { useEffect, useRef, useState } from "react";
import { wsURL } from "@/lib/api";
import type { WsEvent } from "@/types/api";

export type ConnectionStatus = "connecting" | "open" | "closed";

/**
 * Conecta al WebSocket del backend (la cookie httpOnly de sesión viaja
 * automáticamente en el handshake) y reintenta con backoff exponencial
 * (1s, 2s, 4s, ... hasta 30s) si se corta. `enabled=false` cierra/pausa la
 * conexión (se usa junto con useOnlineStatus para no reintentar sin red).
 */
export function useRealtimeEvents(onEvent: (event: WsEvent) => void, enabled: boolean): ConnectionStatus {
  const [socketStatus, setSocketStatus] = useState<ConnectionStatus>("connecting");
  const onEventRef = useRef(onEvent);

  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    if (!enabled) return;

    let socket: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let unmounted = false;
    let attempt = 0;

    function connect() {
      setSocketStatus("connecting");
      socket = new WebSocket(wsURL());

      socket.onopen = () => {
        attempt = 0;
        setSocketStatus("open");
      };

      socket.onmessage = (event: MessageEvent<string>) => {
        try {
          const parsed = JSON.parse(event.data) as WsEvent;
          onEventRef.current(parsed);
        } catch {
          // Mensaje no-JSON (p.ej. ping de control): se ignora.
        }
      };

      socket.onclose = () => {
        setSocketStatus("closed");
        if (unmounted) return;
        const delay = Math.min(1000 * 2 ** attempt, 30_000);
        attempt += 1;
        reconnectTimer = setTimeout(connect, delay);
      };

      socket.onerror = () => {
        socket?.close();
      };
    }

    connect();

    return () => {
      unmounted = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [enabled]);

  // Estado derivado (no una segunda fuente de verdad en useState): evita
  // hacer setState síncrono dentro del efecto solo para reflejar `enabled`.
  return enabled ? socketStatus : "closed";
}
