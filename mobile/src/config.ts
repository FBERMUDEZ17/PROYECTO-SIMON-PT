// Config centralizada: URL del backend Go.
//
// TAREA (mobile): el APK standalone (ver eas.json, build "preview") se
// distribuye para que lo instale y pruebe gente fuera de esta red — no
// tiene sentido hornear una IP de LAN fija en el build. Por eso, además
// del default de compilación (app.json `expo.extra` / EXPO_PUBLIC_API_URL
// en dev), la URL del backend es reconfigurable en tiempo de ejecución
// desde la pantalla de login ("Configurar servidor") y se persiste en
// AsyncStorage — cualquiera que corra el backend Go en su propia máquina
// puede apuntar la app ahí sin necesitar un rebuild.
import Constants from "expo-constants";
import AsyncStorage from "@react-native-async-storage/async-storage";

const extra = (Constants.expoConfig?.extra ?? {}) as {
  apiUrl?: string;
  wsUrl?: string;
};

// DEFAULT_API_URL es el valor horneado en el build (dev: LAN/env var;
// producción: el default de app.json, "http://localhost:8080" — sirve tal
// cual solo si el backend corre en la misma máquina que un emulador
// Android con port-forwarding, o como placeholder a reemplazar desde la
// pantalla de login).
export const DEFAULT_API_URL = process.env.EXPO_PUBLIC_API_URL ?? extra.apiUrl ?? "http://localhost:8080";

const BACKEND_URL_STORAGE_KEY = "@simonpt/backend_api_url";

let currentApiUrl = DEFAULT_API_URL;
let listeners: Array<() => void> = [];

export function getApiUrl(): string {
  return currentApiUrl;
}

// getWsUrl deriva la URL del WS del mismo host/puerto que la API REST
// (mismo backend, mismo puerto, ver internal/httpapi/router.go: /ws vive
// en el mismo router que el resto de las rutas).
export function getWsUrl(): string {
  try {
    const u = new URL(currentApiUrl);
    const wsProtocol = u.protocol === "https:" ? "wss:" : "ws:";
    return `${wsProtocol}//${u.host}/ws`;
  } catch {
    return DEFAULT_API_URL.replace(/^http/, "ws") + "/ws";
  }
}

// setApiUrl actualiza la URL en memoria (efecto inmediato en los próximos
// fetch/WS) y la persiste para que sobreviva a un reinicio de la app.
export async function setApiUrl(rawUrl: string): Promise<void> {
  const trimmed = rawUrl.trim().replace(/\/+$/, "");
  currentApiUrl = trimmed;
  await AsyncStorage.setItem(BACKEND_URL_STORAGE_KEY, trimmed);
  listeners.forEach((cb) => cb());
}

// loadApiUrlOverride debe llamarse una vez al arrancar la app (antes de
// cualquier request), para recuperar la URL guardada en una sesión previa.
export async function loadApiUrlOverride(): Promise<void> {
  const stored = await AsyncStorage.getItem(BACKEND_URL_STORAGE_KEY);
  if (stored) {
    currentApiUrl = stored;
    listeners.forEach((cb) => cb());
  }
}

export function onApiUrlChange(cb: () => void): () => void {
  listeners.push(cb);
  return () => {
    listeners = listeners.filter((l) => l !== cb);
  };
}

// TAREA (mobile): umbral usado para decidir cuándo reintentar la
// sincronización offline / reconexión del WS con backoff.
export const RECONNECT_BASE_DELAY_MS = 2000;
export const RECONNECT_MAX_DELAY_MS = 30000;
