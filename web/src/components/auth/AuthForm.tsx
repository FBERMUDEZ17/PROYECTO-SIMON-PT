"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useLogin, useRegister } from "@/hooks/useAuth";
import { ApiRequestError } from "@/lib/api";

type Mode = "login" | "register";

// TAREA: "componentes reutilizables" + "autenticación con JWT" — un solo
// formulario para login y registro (llama a useLogin/useRegister, que a su
// vez disparan el Set-Cookie del backend).
/** Formulario reutilizable de login/registro. Comparte validación básica
 * y manejo de estado/errores entre ambos modos. */
export function AuthForm({ mode }: { mode: Mode }) {
  const router = useRouter();
  const login = useLogin();
  const register = useRegister();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);

  const pending = mode === "login" ? login.isPending : register.isPending;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      if (mode === "login") {
        await login.mutateAsync({ email, password });
      } else {
        await register.mutateAsync({ email, password, name });
      }
      router.push("/dashboard");
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Error inesperado");
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex w-full max-w-sm flex-col gap-4">
      <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
        {mode === "login" ? "Iniciar sesión" : "Crear cuenta"}
      </h1>

      {mode === "register" && (
        <label className="flex flex-col gap-1 text-sm text-slate-600 dark:text-slate-300">
          Nombre
          <input
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="rounded-md border border-slate-300 bg-white px-3 py-2 text-slate-900 outline-none focus:border-slate-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
          />
        </label>
      )}

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

      <label className="flex flex-col gap-1 text-sm text-slate-600 dark:text-slate-300">
        Password
        <input
          required
          type="password"
          minLength={8}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="rounded-md border border-slate-300 bg-white px-3 py-2 text-slate-900 outline-none focus:border-slate-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
        />
      </label>

      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

      <button
        type="submit"
        disabled={pending}
        className="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-700 disabled:opacity-50 dark:bg-slate-100 dark:text-slate-900"
      >
        {pending ? "Enviando…" : mode === "login" ? "Entrar" : "Registrarme"}
      </button>

      {mode === "login" && (
        <Link href="/forgot-password" className="text-sm text-slate-500 underline dark:text-slate-400">
          ¿Olvidaste tu contraseña?
        </Link>
      )}
    </form>
  );
}
