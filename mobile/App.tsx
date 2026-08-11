// TAREA (mobile): punto de entrada — replica el gate de auth del frontend
// web (proxy.ts sólo mira presencia de cookie; acá miramos presencia +
// validez de token vía /auth/me, ver useAuth.tsx) y arranca los
// subsistemas de fondo: DB local (SQLite), cola offline, y notificaciones.
import { useEffect, useRef, useState } from "react";
import { ActivityIndicator, StatusBar, View } from "react-native";
import { NavigationContainer } from "@react-navigation/native";

import { AuthProvider, useAuth } from "@/hooks/useAuth";
import { AuthNavigator } from "@/navigation/AuthNavigator";
import { MainNavigator } from "@/navigation/MainNavigator";
import { loadApiUrlOverride } from "@/config";
import { initDb } from "@/storage/db";
import { drainQueue, watchConnectivityAndDrain } from "@/storage/offlineQueue";
import {
  addNotificationResponseListener,
  registerDeviceToken,
  registerForPushNotificationsAsync,
  setupNotificationChannel,
} from "@/notifications/push";
import { colors } from "@/theme/theme";

function Root() {
  const { user, loading } = useAuth();

  // TAREA: manejo de lifecycle — listener de notificación tocada, para en
  // el futuro poder deep-link a VehicleDetail desde el push (dejado
  // preparado; requiere pasar vehicleId en notification.data desde el
  // backend cuando envíe el push).
  useEffect(() => {
    const sub = addNotificationResponseListener((response) => {
      const vehicleId = (response.notification.request.content.data as { vehicleId?: string } | undefined)?.vehicleId;
      if (vehicleId) {
        // La navegación real requeriría un navigationRef global; se deja
        // como TODO explícito para no adivinar la UX deseada.
        console.log("push tocado para vehículo", vehicleId);
      }
    });
    return () => sub.remove();
  }, []);

  // Apenas hay sesión iniciada, pide permiso de push y registra el token
  // contra el backend — así el usuario no tiene que ir manualmente a
  // Ajustes para que las alertas de SUS vehículos le lleguen por push.
  useEffect(() => {
    if (!user) return;
    registerForPushNotificationsAsync()
      .then((expoToken) => {
        if (expoToken) return registerDeviceToken(expoToken);
      })
      .catch((err) => console.warn("push: fallo el registro automático", err));
  }, [user]);

  if (loading) {
    return (
      <View style={{ flex: 1, backgroundColor: colors.background, justifyContent: "center", alignItems: "center" }}>
        <ActivityIndicator color={colors.primary} size="large" />
      </View>
    );
  }

  return user ? <MainNavigator /> : <AuthNavigator />;
}

export default function App() {
  const bootstrapped = useRef(false);
  // Gatea el render hasta que loadApiUrlOverride() resuelva: si no,
  // useAuth().me() (dentro de AuthProvider) podría dispararse contra la
  // URL default en vez de la que el usuario configuró en una sesión
  // anterior (ver src/config.ts).
  const [configReady, setConfigReady] = useState(false);

  useEffect(() => {
    if (bootstrapped.current) return;
    bootstrapped.current = true;

    loadApiUrlOverride()
      .catch((err) => console.warn("config: fallo cargando URL de backend guardada", err))
      .finally(() => setConfigReady(true));

    initDb().catch((err) => console.warn("db: fallo iniciando SQLite local", err));
    setupNotificationChannel().catch(() => {});
    drainQueue().catch(() => {}); // intenta vaciar lo pendiente de la sesión anterior
    const unwatch = watchConnectivityAndDrain();
    return unwatch;
  }, []);

  if (!configReady) {
    return (
      <View style={{ flex: 1, backgroundColor: colors.background, justifyContent: "center", alignItems: "center" }}>
        <ActivityIndicator color={colors.primary} size="large" />
      </View>
    );
  }

  return (
    <AuthProvider>
      <StatusBar barStyle="light-content" backgroundColor={colors.background} />
      <NavigationContainer>
        <Root />
      </NavigationContainer>
    </AuthProvider>
  );
}
