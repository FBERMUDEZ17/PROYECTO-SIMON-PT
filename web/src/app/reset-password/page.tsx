"use client";

// TAREA: "crear un restaurador de contraseña" — paso 2: canjear el token
// (pegado manualmente, ya que no hay email real) por un password nuevo.
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useResetPassword } from "@/hooks/useAuth";
import { ApiRequestError } from "@/lib/api";

export default function ResetPasswordPage() {
  const router = useRouter();
  const resetPassword = useResetPassword();

  const [token, setToken] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await resetPassword.mutateAsync({ token, newPassword });
      setDone(true);
      setTimeout(() => router.push("/login"), 1500);
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Error inesperado");
    }
  }

  return (
    <main className="flex flex-1 items-center justify-center px-6">
      <div className="flex w-full max-w-sm flex-col gap-4">
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
          Elegir nueva contraseña
        </h1>

        {done ? (
          <p className="text-sm text-emerald-600 dark:text-emerald-400">
            Password actualizado. Redirigiendo a login…
          </p>
        ) : (
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <label className="flex flex-col gap-1 text-sm text-slate-600 dark:text-slate-300">
              Token de restablecimiento
              <input
                required
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="pegá acá el token del log del servidor"
                className="rounded-md border border-slate-300 bg-white px-3 py-2 font-mono text-xs text-slate-900 outline-none focus:border-slate-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              />
            </label>

            <label className="flex flex-col gap-1 text-sm text-slate-600 dark:text-slate-300">
              Password nuevo
              <input
                required
                type="password"
                minLength={8}
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="rounded-md border border-slate-300 bg-white px-3 py-2 text-slate-900 outline-none focus:border-slate-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              />
            </label>

            {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

            <button
              type="submit"
              disabled={resetPassword.isPending}
              className="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-700 disabled:opacity-50 dark:bg-slate-100 dark:text-slate-900"
            >
              {resetPassword.isPending ? "Guardando…" : "Actualizar contraseña"}
            </button>
          </form>
        )}

        <Link href="/login" className="text-sm text-slate-500 underline dark:text-slate-400">
          ← Volver a iniciar sesión
        </Link>
      </div>
    </main>
  );
}
