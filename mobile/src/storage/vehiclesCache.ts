// Cache local de vehículos (última respuesta buena de GET /vehicles),
// para que el dashboard tenga algo que mostrar apenas se abre la app sin
// esperar la red, y como fallback si el fetch falla por estar offline.
import { getDb } from "./db";
import type { Vehicle } from "@/types/api";

export async function cacheVehicles(vehicles: Vehicle[]): Promise<void> {
  const db = await getDb();
  const now = new Date().toISOString();
  await db.withTransactionAsync(async () => {
    for (const v of vehicles) {
      await db.runAsync(
        `INSERT INTO vehicles_cache (id, name, payload_json, updated_at)
         VALUES (?, ?, ?, ?)
         ON CONFLICT(id) DO UPDATE SET name = excluded.name,
           payload_json = excluded.payload_json, updated_at = excluded.updated_at`,
        [v.id, v.name, JSON.stringify(v), now]
      );
    }
  });
}

export async function getCachedVehicles(): Promise<Vehicle[]> {
  const db = await getDb();
  const rows = await db.getAllAsync<{ payload_json: string }>(
    "SELECT payload_json FROM vehicles_cache ORDER BY updated_at DESC"
  );
  return rows.map((r) => JSON.parse(r.payload_json) as Vehicle);
}
