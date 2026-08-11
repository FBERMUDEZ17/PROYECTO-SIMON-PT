// TAREA: "autenticación con JWT (guardar en httpOnly cookie)" — protección
// de rutas en el borde (antes conocido como middleware.js; renombrado a
// proxy.js en Next.js 16).
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Debe coincidir con httpapi.AuthCookieName en el backend Go.
const AUTH_COOKIE = "auth_token";

/**
 * Redirección de UX en el borde: solo mira si la cookie de sesión existe,
 * no valida su firma ni expiración (no tiene el secreto JWT, y no debería
 * tenerlo). La autorización real ocurre en cada request al backend Go vía
 * RequireAuth — esto únicamente evita el parpadeo de mostrar /dashboard
 * para luego redirigir tras el primer fetch fallido.
 *
 * Nota dev: la cookie la emite el backend Go (ej. localhost:8080) sin
 * atributo Domain, por lo que queda asociada al host "localhost" sin
 * distinguir puerto (así especifica RFC 6265: el matching de cookies
 * ignora el puerto) — por eso también es visible en las requests que
 * llegan a este proxy (localhost:3000). En producción, si frontend y
 * backend viven en dominios registrables distintos, este check dejaría de
 * ver la cookie; ahí este proxy pasaría a ser un no-op y la protección
 * seguiría dependiendo, correctamente, del backend.
 */
export function proxy(request: NextRequest) {
  const hasSession = request.cookies.has(AUTH_COOKIE);
  const { pathname } = request.nextUrl;

  if (pathname.startsWith("/dashboard") && !hasSession) {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("next", pathname);
    return NextResponse.redirect(loginUrl);
  }

  if (pathname === "/login" && hasSession) {
    return NextResponse.redirect(new URL("/dashboard", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/dashboard/:path*", "/login"],
};
