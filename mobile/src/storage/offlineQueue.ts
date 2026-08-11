// TAREA (mobile): cola de sincronización offline. Si el dispositivo reporta
// una lectura de sensor (o la app la genera localmente, p.ej. desde un
// simulador de telemetría a bordo) sin conectividad, se guarda acá en vez
// de perderse; drainQueue() la reenvía cuando vuelve la red.
//
// Nota: a diferencia de la ingesta real "en vivo" del vehículo (que en este
// proyecto normalmente hace el propio hardware/backend), este módulo es
// para el caso en que la app mobile actúa como origen de datos (ej. modo
// operador/conductor) y necesita resiliencia ante cortes de datos móviles.
import NetInfo from "@react-native-community/netinfo";
import { getDb } from "./db";
import { sendSensorData } from "@/api/vehicles";
import { NetworkError } from "@/api/client";
import type { SensorDataRequest } from "@/types/api";

const MAX_ATTEMPTS = 8;

export async function enqueueReading(reading: SensorDataRequest): Promise<void> {
  const db = await getDb();
  await db.runAsync(
    "INSERT INTO pending_readings (payload_json, created_at, attempts) VALUES (?, ?, 0)",
    [JSON.stringify(reading), new Date().toISOString()]
  );
}

export async function pendingCount(): Promise<number> {
  const db = await getDb();
  const row = await db.getFirstAsync<{ n: number }>(
    "SELECT COUNT(*) as n FROM pending_readings"
  );
  return row?.n ?? 0;
}

/**
 * Intenta enviar las lecturas encoladas en orden. Se detiene apenas
 * detecta que seguimos sin red (NetworkError) para no quemar reintentos en
 * ráfaga; sí descarta entradas que agotaron MAX_ATTEMPTS (probable payload
 * inválido — reintentarlo para siempre no serviría de nada).
 */
export async function drainQueue(): Promise<{ sent: number; remaining: number }> {
  const state = await NetInfo.fetch();
  if (!state.isConnected || state.isInternetReachable === false) {
    return { sent: 0, remaining: await pendingCount() };
  }

  const db = await getDb();
  const rows = await db.getAllAsync<{
    id: number;
    payload_json: string;
    attempts: number;
  }>("SELECT id, payload_json, attempts FROM pending_readings ORDER BY id ASC");

  let sent = 0;
  for (const row of rows) {
    const reading = JSON.parse(row.payload_json) as SensorDataRequest;
    try {
      await sendSensorData(reading);
      await db.runAsync("DELETE FROM pending_readings WHERE id = ?", [row.id]);
      sent += 1;
    } catch (err) {
      if (err instanceof NetworkError) {
        // Sin conexión de nuevo: paramos el drenado, el resto queda para
        // el próximo intento (evento de red o el poll periódico).
        break;
      }
      const attempts = row.attempts + 1;
      if (attempts >= MAX_ATTEMPTS) {
        await db.runAsync("DELETE FROM pending_readings WHERE id = ?", [row.id]);
      } else {
        await db.runAsync(
          "UPDATE pending_readings SET attempts = ?, last_error = ? WHERE id = ?",
          [attempts, String(err), row.id]
        );
      }
    }
  }

  return { sent, remaining: await pendingCount() };
}

// Se suscribe a cambios de conectividad y drena la cola automáticamente
// apenas vuelve la red. Llamar una vez desde la raíz de la app.
export function watchConnectivityAndDrain(): () => void {
  const unsubscribe = NetInfo.addEventListener((state) => {
    if (state.isConnected && state.isInternetReachable !== false) {
      drainQueue().catch(() => {
        // best-effort: un fallo de drenado no debe tumbar el listener.
      });
    }
  });
  return unsubscribe;
}
