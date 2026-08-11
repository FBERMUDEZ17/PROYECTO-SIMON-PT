import { getDb } from "./index";

/**
 * Imprime el schema SQL de todas las tablas de usuario existentes en la BD.
 */
function printSchema(): void {
  const db = getDb();
  const rows = db
    .prepare<[], { name: string; sql: string }>(
      `SELECT name, sql FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`
    )
    .all();

  if (rows.length === 0) {
    console.log("No hay tablas en la base de datos.");
    return;
  }

  for (const row of rows) {
    console.log(`-- Tabla: ${row.name}`);
    console.log(row.sql + ";\n");
  }
}

printSchema();
