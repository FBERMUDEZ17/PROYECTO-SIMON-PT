// TAREA (mobile): "sincronización offline con SQLite local".
// Base de datos local (expo-sqlite, API async nueva de SDK 51) con dos
// responsabilidades:
//   1. Cache de lecturas/vehículos ya vistos, para poder pintar el
//      dashboard sin red (equivalente mobile de web/src/lib/indexeddb.ts).
//   2. Cola de escrituras pendientes (sensor_data que no se pudo enviar por
//      falta de conexión) — ver offlineQueue.ts, que opera sobre esta DB.
import * as SQLite from "expo-sqlite";

let dbPromise: Promise<SQLite.SQLiteDatabase> | null = null;

function open(): Promise<SQLite.SQLiteDatabase> {
  if (!dbPromise) {
    dbPromise = SQLite.openDatabaseAsync("simonpt.db");
  }
  return dbPromise;
}

export async function initDb(): Promise<void> {
  const db = await open();
  await db.execAsync(`
    PRAGMA journal_mode = WAL;

    CREATE TABLE IF NOT EXISTS vehicles_cache (
      id TEXT PRIMARY KEY,
      name TEXT NOT NULL,
      payload_json TEXT NOT NULL,
      updated_at TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS pending_readings (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      payload_json TEXT NOT NULL,
      created_at TEXT NOT NULL,
      attempts INTEGER NOT NULL DEFAULT 0,
      last_error TEXT
    );
  `);
}

export async function getDb(): Promise<SQLite.SQLiteDatabase> {
  await initDb();
  return open();
}
