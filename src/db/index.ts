import Database from "better-sqlite3";
import fs from "node:fs";
import path from "node:path";

const DATA_DIR = path.join(__dirname, "..", "..", "data");
const DB_PATH = process.env.DB_PATH || path.join(DATA_DIR, "app.db");

let db: Database.Database | null = null;

/**
 * Devuelve la conexión a SQLite, creándola (y su schema) si aún no existe.
 */
export function getDb(): Database.Database {
  if (!db) {
    fs.mkdirSync(path.dirname(DB_PATH), { recursive: true });
    db = new Database(DB_PATH);
    db.pragma("journal_mode = WAL");
    db.pragma("foreign_keys = ON");
    initSchema(db);
  }
  return db;
}

/**
 * Crea las tablas de la aplicación si no existen todavía.
 */
function initSchema(database: Database.Database): void {
  database.exec(`
    CREATE TABLE IF NOT EXISTS users (
      id            INTEGER PRIMARY KEY AUTOINCREMENT,
      email         TEXT NOT NULL UNIQUE,
      password_hash TEXT NOT NULL,
      name          TEXT NOT NULL,
      created_at    TEXT NOT NULL DEFAULT (datetime('now'))
    );
  `);
}

/**
 * Cierra la conexión activa (útil en tests / shutdown).
 */
export function closeDb(): void {
  if (db) {
    db.close();
    db = null;
  }
}

// Permite ejecutar `tsx src/db/index.ts` para inicializar la BD manualmente.
if (require.main === module) {
  getDb();
  console.log(`Base de datos inicializada en: ${DB_PATH}`);
}
