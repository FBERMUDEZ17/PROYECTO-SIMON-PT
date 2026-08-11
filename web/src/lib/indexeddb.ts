// TAREA: "caché offline con localStorage/IndexedDB" + "sync automático
// cuando hay conexión" (el sync lo dispara useVehicles al recuperar
// conexión, invalidando la query para refetchear).
import { openDB, type DBSchema, type IDBPDatabase } from "idb";
import type { Vehicle } from "@/types/api";

interface FleetDB extends DBSchema {
  vehicles: {
    key: string; // vehicle.id (ya enmascarado si aplica)
    value: Vehicle & { cachedAt: string };
  };
  meta: {
    key: string;
    value: { key: string; value: string };
  };
}

const DB_NAME = "fleet-dashboard";
const DB_VERSION = 1;

let dbPromise: Promise<IDBPDatabase<FleetDB>> | null = null;

/** Solo hay IndexedDB en el navegador; en SSR esto no debe invocarse. */
function getDB(): Promise<IDBPDatabase<FleetDB>> {
  if (typeof window === "undefined") {
    throw new Error("IndexedDB no está disponible en el servidor");
  }
  if (!dbPromise) {
    dbPromise = openDB<FleetDB>(DB_NAME, DB_VERSION, {
      upgrade(db) {
        if (!db.objectStoreNames.contains("vehicles")) {
          db.createObjectStore("vehicles", { keyPath: "id" });
        }
        if (!db.objectStoreNames.contains("meta")) {
          db.createObjectStore("meta", { keyPath: "key" });
        }
      },
    });
  }
  return dbPromise;
}

/** Cachea el snapshot completo de vehículos (reemplaza lo anterior). */
export async function cacheVehicles(vehicles: Vehicle[]): Promise<void> {
  const db = await getDB();
  const tx = db.transaction("vehicles", "readwrite");
  await tx.store.clear();
  const cachedAt = new Date().toISOString();
  await Promise.all(vehicles.map((v) => tx.store.put({ ...v, cachedAt })));
  await tx.done;

  const metaTx = db.transaction("meta", "readwrite");
  await metaTx.store.put({ key: "lastSync", value: cachedAt });
  await metaTx.done;
}

export async function getCachedVehicles(): Promise<Vehicle[]> {
  const db = await getDB();
  // El campo extra `cachedAt` queda en el valor pero no en el tipo Vehicle;
  // structural typing de TS lo permite sin necesidad de destructurar.
  return await db.getAll("vehicles");
}

export async function getLastSyncTime(): Promise<string | null> {
  const db = await getDB();
  const row = await db.get("meta", "lastSync");
  return row?.value ?? null;
}
