import type { Role } from "@/types/api";

// TAREA: "al ingresar un usuario no se identifica si es admin o user" —
// badge de color prominente (antes era un texto gris chico), visible en
// el Header apenas se loguea.
const STYLES: Record<Role, string> = {
  admin:
    "bg-purple-100 text-purple-800 ring-1 ring-purple-300 dark:bg-purple-950 dark:text-purple-200 dark:ring-purple-800",
  user: "bg-sky-100 text-sky-800 ring-1 ring-sky-300 dark:bg-sky-950 dark:text-sky-200 dark:ring-sky-800",
};

export function RoleBadge({ role }: { role: Role }) {
  return (
    <span className={`rounded-full px-2.5 py-1 text-xs font-bold uppercase tracking-wide ${STYLES[role]}`}>
      {role === "admin" ? "Admin" : "Usuario"}
    </span>
  );
}
