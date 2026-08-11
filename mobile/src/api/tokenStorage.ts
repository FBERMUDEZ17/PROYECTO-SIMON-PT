// TAREA (mobile): persistencia del JWT. El backend Go es "sole owner" del
// JWT (ver CLAUDE.md); en web va en cookie httpOnly, en mobile no hay
// cookie jar equivalente así que lo guardamos en AsyncStorage y lo mandamos
// nosotros mismos como "Authorization: Bearer <token>" en cada request y en
// el query param ?token= del handshake WS (ambos soportados por el backend,
// ver internal/ws/handler.go:bearerToken).
//
// Nota de seguridad: AsyncStorage no está cifrado en disco. Para producción
// real, migrar a expo-secure-store (Keychain/Keystore) — se deja
// AsyncStorage aquí por simplicidad del scaffold y porque expo-secure-store
// no soporta bien Android backups grandes; es un cambio de una línea
// (mismo API get/set/remove) si se decide migrar.

import AsyncStorage from "@react-native-async-storage/async-storage";

const TOKEN_KEY = "@simonpt/auth_token";
const USER_KEY = "@simonpt/auth_user";

export async function saveToken(token: string): Promise<void> {
  await AsyncStorage.setItem(TOKEN_KEY, token);
}

export async function getToken(): Promise<string | null> {
  return AsyncStorage.getItem(TOKEN_KEY);
}

export async function clearToken(): Promise<void> {
  await AsyncStorage.multiRemove([TOKEN_KEY, USER_KEY]);
}

export async function saveCachedUser(userJson: string): Promise<void> {
  await AsyncStorage.setItem(USER_KEY, userJson);
}

export async function getCachedUser(): Promise<string | null> {
  return AsyncStorage.getItem(USER_KEY);
}
