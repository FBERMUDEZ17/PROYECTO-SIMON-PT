// TAREA (mobile): capa de auth — register/login/logout/me contra el mismo
// backend Go (internal/httpapi/handlers.go). Persiste el token devuelto en
// el body (authResponse.token) ya que en mobile no hay cookie httpOnly.
import { api } from "./client";
import { clearToken, saveToken } from "./tokenStorage";
import type { AuthResponse, User } from "@/types/api";

export async function register(email: string, password: string, name: string): Promise<AuthResponse> {
  const res = await api.post<AuthResponse>("/auth/register", { email, password, name }, { auth: false });
  await saveToken(res.token);
  return res;
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  const res = await api.post<AuthResponse>("/auth/login", { email, password }, { auth: false });
  await saveToken(res.token);
  return res;
}

export async function logout(): Promise<void> {
  try {
    await api.post("/auth/logout");
  } finally {
    await clearToken();
  }
}

export async function me(): Promise<User> {
  return api.get<User>("/auth/me");
}

export async function forgotPassword(email: string): Promise<void> {
  await api.post("/auth/forgot-password", { email }, { auth: false });
}

export async function resetPassword(token: string, newPassword: string): Promise<void> {
  await api.post("/auth/reset-password", { token, new_password: newPassword }, { auth: false });
}
