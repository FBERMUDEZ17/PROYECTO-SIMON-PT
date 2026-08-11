// Cliente HTTP central, equivalente mobile de web/src/lib/api.ts.
// Diferencia clave: web usa credentials:"include" (cookie httpOnly); acá no
// hay cookie jar de navegador, así que adjuntamos el Bearer token guardado
// en AsyncStorage (ver tokenStorage.ts) en cada request.
import { getApiUrl } from "@/config";
import { getToken } from "./tokenStorage";
import type { ApiErrorBody } from "@/types/api";

export class ApiRequestError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
  }
}

// Error de red (sin conexión, timeout, DNS, etc.) — distinto de un error
// HTTP de la API. El código offline-first lo usa para decidir si debe
// encolar la petición en vez de reportar el fallo al usuario.
export class NetworkError extends Error {
  constructor(cause: unknown) {
    super("network request failed");
    this.name = "NetworkError";
    this.cause = cause;
  }
}

interface RequestOptions extends RequestInit {
  auth?: boolean; // default true: adjunta Authorization si hay token
}

async function request<T>(path: string, init: RequestOptions = {}): Promise<T> {
  const { auth = true, headers, ...rest } = init;

  const finalHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    ...(headers as Record<string, string> | undefined),
  };

  if (auth) {
    const token = await getToken();
    if (token) {
      finalHeaders.Authorization = `Bearer ${token}`;
    }
  }

  let res: Response;
  try {
    res = await fetch(`${getApiUrl()}${path}`, { ...rest, headers: finalHeaders });
  } catch (err) {
    throw new NetworkError(err);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  const data = await res.json().catch(() => null);

  if (!res.ok) {
    const message = (data as ApiErrorBody | null)?.error ?? res.statusText;
    throw new ApiRequestError(res.status, message);
  }

  return data as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: "GET" }),
  post: <T>(path: string, body?: unknown, opts: RequestOptions = {}) =>
    request<T>(path, {
      ...opts,
      method: "POST",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),
};
