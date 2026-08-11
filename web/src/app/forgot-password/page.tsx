"use client";

// TAREA: "crear un restaurador de contraseña" — paso 1: pedir el token.
// No hay servicio de email configurado en el proyecto, así que el token
// se loguea en la consola del backend (dev); acá solo se confirma el
// envío (mensaje genérico, no revela si el email existe) y se linkea al
// paso 2 (/reset-password) para pegarlo manualmente.
import { useState } from "react";
import Link from "next/link";
import { useForgotPassword } from "@/hooks/useAuth";
import { ApiRequestError } from "@/lib/api";

export default function ForgotPasswordPage() {
  const forgotPassword = useForgotPassword();
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [sent, setSent] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await forgotPassword.mutateAsync(email);
      setSent(true);
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Error inesperado");
    }
  }

  return (
    <main className="flex flex-1 items-center justify-center px-6">
      <div className="flex w-full max-w-sm flex-col gap-4">
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
          Restablecer contraseña
        </h1>

        {sent ? (
          <div className="flex flex-col gap-3 text-sm text-slate-600 dark:text-slate-300">
            <p>
              Si el email está registrado, se generó un token de restablecimiento (revisá la
              consola del servidor backend — no hay envío de email configurado en este proyecto).
            </p>
            <Link href="/reset-password" className="font-medium underline">
              Ya tengo mi token →
            </Link>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <label className="flex flex-col gap-1 text-sm text-slate-600 dark:text-slate-300">
              Email
              <input
                required
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="rounded-md border border-slate-300 bg-white px-3 py-2 text-slate-900 outline-none focus:border-slate-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              />
            </label>

            {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

            <button
              type="submit"
              disabled={forgotPassword.isPending}
              className="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-700 disabled:opacity-50 dark:bg-slate-100 dark:text-slate-900"
            >
              {forgotPassword.isPending ? "Enviando…" : "Solicitar restablecimiento"}
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
