"use client";

// TAREA: "autenticación con JWT (guardar en httpOnly cookie)" — hooks de
// login/register/logout y de sesión actual (GET /auth/me), con estado
// manejado por TanStack Query.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, ApiRequestError } from "@/lib/api";
import type { AuthResponse, User } from "@/types/api";

const ME_QUERY_KEY = ["auth", "me"] as const;

interface LoginInput {
  email: string;
  password: string;
}

interface RegisterInput extends LoginInput {
  name: string;
}

/**
 * El JWT vive en una cookie httpOnly (no accesible desde JS), así que la
 * única forma de saber quién está logueado -y su rol- es preguntarle al
 * backend vía GET /auth/me.
 */
export function useCurrentUser() {
  return useQuery<User | null>({
    queryKey: ME_QUERY_KEY,
    queryFn: async () => {
      try {
        return await api.get<User>("/auth/me");
      } catch (err) {
        if (err instanceof ApiRequestError && err.status === 401) {
          return null;
        }
        throw err;
      }
    },
    staleTime: 60_000,
    retry: false,
  });
}

export function useLogin() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: LoginInput) => api.post<AuthResponse>("/auth/login", input),
    onSuccess: (data) => {
      queryClient.setQueryData(ME_QUERY_KEY, data.user);
    },
  });
}

export function useRegister() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: RegisterInput) => api.post<AuthResponse>("/auth/register", input),
    onSuccess: (data) => {
      queryClient.setQueryData(ME_QUERY_KEY, data.user);
    },
  });
}

// TAREA: "crear un restaurador de contraseña" — pide el token (el backend
// lo loguea en su consola, no hay email configurado) y luego lo canjea.
export function useForgotPassword() {
  return useMutation({
    mutationFn: (email: string) => api.post<{ message: string }>("/auth/forgot-password", { email }),
  });
}

export function useResetPassword() {
  return useMutation({
    mutationFn: (input: { token: string; newPassword: string }) =>
      api.post<{ message: string }>("/auth/reset-password", {
        token: input.token,
        new_password: input.newPassword,
      }),
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.post<void>("/auth/logout"),
    onSuccess: () => {
      queryClient.setQueryData(ME_QUERY_KEY, null);
      queryClient.clear();
    },
  });
}
