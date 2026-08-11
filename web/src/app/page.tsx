import Link from "next/link";

export default function Home() {
  return (
    <main className="flex flex-1 flex-col items-center justify-center gap-6 px-6 text-center">
      <h1 className="text-3xl font-semibold text-slate-900 dark:text-slate-100">Fleet Dashboard</h1>
      <p className="max-w-md text-slate-600 dark:text-slate-400">
        Monitoreo de flota en tiempo real: ubicación GPS, combustible, temperatura y alertas
        predictivas.
      </p>
      <div className="flex gap-3">
        <Link
          href="/login"
          className="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-700 dark:bg-slate-100 dark:text-slate-900"
        >
          Iniciar sesión
        </Link>
        <Link
          href="/login?mode=register"
          className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-100 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-900"
        >
          Crear cuenta
        </Link>
      </div>
    </main>
  );
}
