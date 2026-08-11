// TAREA: página de login/registro (?mode=register alterna el formulario).
import Link from "next/link";
import { AuthForm } from "@/components/auth/AuthForm";

export default async function LoginPage(props: PageProps<"/login">) {
  const searchParams = await props.searchParams;
  const mode = searchParams.mode === "register" ? "register" : "login";

  return (
    <main className="flex flex-1 items-center justify-center px-6">
      <div className="flex flex-col items-center gap-4">
        <AuthForm mode={mode} />
        <p className="text-sm text-slate-500 dark:text-slate-400">
          {mode === "login" ? (
            <>
              ¿No tenés cuenta?{" "}
              <Link href="/login?mode=register" className="underline">
                Registrate
              </Link>
            </>
          ) : (
            <>
              ¿Ya tenés cuenta?{" "}
              <Link href="/login" className="underline">
                Iniciá sesión
              </Link>
            </>
          )}
        </p>
      </div>
    </main>
  );
}
