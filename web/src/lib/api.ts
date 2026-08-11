// TAREA: "autenticación con JWT (guardar en httpOnly cookie)" — cliente
// HTTP central; credentials:"include" es lo que hace funcionar la cookie.
import type { ApiError } from "@/types/api";

/**
 * Base URL del backend Go. El navegador habla directo con él (no hay proxy
 * BFF en Next.js): la cookie httpOnly `auth_token` la emite y lee el
 * backend, y credentials:"include" hace que el navegador la adjunte
 * automáticamente en cada request.
 */
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export class ApiRequestError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: "include", // imprescindible para enviar/recibir la cookie httpOnly
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });

  if (res.status === 204) {
    return undefined as T;
  }

  const data = await res.json().catch(() => null);

  if (!res.ok) {
    const message = (data as ApiError | null)?.error ?? res.statusText;
    throw new ApiRequestError(res.status, message);
  }

  return data as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: "GET" }),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: "POST",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),
};

// TAREA: "consumir WebSocket del backend".
/** URL del WebSocket, misma cookie de sesión (se adjunta automáticamente
 * en el handshake porque backend y navegador comparten dominio/sitio). */
export function wsURL(): string {
  const httpUrl = new URL(API_BASE_URL);
  const wsProtocol = httpUrl.protocol === "https:" ? "wss:" : "ws:";
  return `${wsProtocol}//${httpUrl.host}/ws`;
}
