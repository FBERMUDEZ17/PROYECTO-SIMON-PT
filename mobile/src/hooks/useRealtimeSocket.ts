// TAREA (mobile): "consumir WebSocket del backend" + "manejo de batería y
// datos móviles optimizado". Equivalente mobile de
// web/src/hooks/useRealtimeEvents.ts, con diferencias deliberadas:
//   - Autenticación por query param ?token= (no hay cookie jar en RN),
//     soportado por internal/ws/handler.go:bearerToken.
//   - Se desconecta cuando la app pasa a background (AppState) para no
//     mantener el socket vivo (y el radio de red despierto) mientras el
//     usuario no está mirando el dashboard; reconecta al volver a
//     foreground. Las notificaciones de alertas mientras está en
//     background llegan vía push (ver notifications/), no vía este socket.
//   - Backoff exponencial en reconexión para no reintentar en loop
//     agresivo con mala señal (ahorro de batería/datos).
import { useCallback, useEffect, useRef, useState } from "react";
import { AppState, type AppStateStatus } from "react-native";
import NetInfo from "@react-native-community/netinfo";

import { getWsUrl, RECONNECT_BASE_DELAY_MS, RECONNECT_MAX_DELAY_MS } from "@/config";
import { getToken } from "@/api/tokenStorage";
import type { WsEvent } from "@/types/api";

type ConnectionStatus = "idle" | "connecting" | "open" | "closed";

export function useRealtimeSocket(onEvent: (event: WsEvent) => void) {
  const [status, setStatus] = useState<ConnectionStatus>("idle");
  const wsRef = useRef<WebSocket | null>(null);
  const attemptRef = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const shouldRunRef = useRef(true);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  const clearReconnectTimer = () => {
    if (reconnectTimer.current) {
      clearTimeout(reconnectTimer.current);
      reconnectTimer.current = null;
    }
  };

  const connect = useCallback(async () => {
    if (!shouldRunRef.current) return;

    const net = await NetInfo.fetch();
    if (!net.isConnected) {
      // Sin red: no tiene sentido intentar, esperamos el evento de NetInfo.
      setStatus("closed");
      return;
    }

    const token = await getToken();
    if (!token) {
      setStatus("closed");
      return;
    }

    setStatus("connecting");
    const url = `${getWsUrl()}?token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      attemptRef.current = 0;
      setStatus("open");
    };

    ws.onmessage = (msg) => {
      try {
        const event = JSON.parse(msg.data as string) as WsEvent;
        onEventRef.current(event);
      } catch {
        // payload no-JSON inesperado: lo ignoramos en vez de tumbar el socket.
      }
    };

    const scheduleReconnect = () => {
      if (!shouldRunRef.current) return;
      setStatus("closed");
      clearReconnectTimer();
      const delay = Math.min(
        RECONNECT_BASE_DELAY_MS * 2 ** attemptRef.current,
        RECONNECT_MAX_DELAY_MS
      );
      attemptRef.current += 1;
      reconnectTimer.current = setTimeout(() => {
        connect();
      }, delay);
    };

    ws.onerror = scheduleReconnect;
    ws.onclose = scheduleReconnect;
  }, []);

  const disconnect = useCallback(() => {
    clearReconnectTimer();
    wsRef.current?.close();
    wsRef.current = null;
    setStatus("closed");
  }, []);

  useEffect(() => {
    shouldRunRef.current = true;
    connect();

    const onAppStateChange = (next: AppStateStatus) => {
      if (next === "active") {
        shouldRunRef.current = true;
        connect();
      } else {
        // background/inactive: cerramos para ahorrar batería/datos; las
        // alertas se siguen recibiendo vía push mientras tanto.
        shouldRunRef.current = false;
        disconnect();
      }
    };
    const appStateSub = AppState.addEventListener("change", onAppStateChange);

    const netSub = NetInfo.addEventListener((state) => {
      if (state.isConnected && AppState.currentState === "active" && status !== "open") {
        attemptRef.current = 0;
        shouldRunRef.current = true;
        connect();
      }
    });

    return () => {
      shouldRunRef.current = false;
      appStateSub.remove();
      netSub();
      disconnect();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return { status };
}
