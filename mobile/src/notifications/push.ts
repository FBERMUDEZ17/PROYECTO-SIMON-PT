// TAREA (mobile): "Notificaciones push para alertas".
//
// Usa expo-notifications como capa de presentación/permisos (funciona con
// APNs en iOS y FCM en Android sin código nativo extra). El flujo
// end-to-end, ya implementado en ambos lados:
//   1. La app pide permiso y obtiene un Expo push token (envuelve el
//      token nativo FCM/APNs — no requiere que el usuario tenga su propio
//      proyecto Firebase configurado para probar).
//   2. Se registra ese token contra el backend vía registerDeviceToken
//      (POST /devices/register, internal/httpapi/devices_handlers.go +
//      internal/devices.Store).
//   3. El backend, al crear una alerta (internal/telemetry, vía
//      broadcastIngestResult en sensors_handlers.go), además de
//      difundirla por WS, dispara un push a los tokens registrados del
//      dueño del vehículo usando la Expo Push API (internal/push) — Expo
//      entrega el push final a FCM/APNs por su cuenta.
//
// App.tsx llama a registerForPushNotificationsAsync + registerDeviceToken
// automáticamente apenas hay sesión iniciada (ver useEffect en Root()),
// así que no depende de que el usuario abra Ajustes.
import * as Device from "expo-device";
import * as Notifications from "expo-notifications";
import { Platform } from "react-native";

import { api } from "@/api/client";

Notifications.setNotificationHandler({
  handleNotification: async () => ({
    // shouldShowAlert es el campo legado (SDK <51) que el tipo todavía
    // exige; shouldShowBanner/shouldShowList son los nuevos separados
    // por foreground/notification-center. Se ponen ambos en true.
    shouldShowAlert: true,
    shouldShowBanner: true,
    shouldShowList: true,
    shouldPlaySound: true,
    shouldSetBadge: false,
  }),
});

const ANDROID_CHANNEL_ALERTS = "low-fuel-alerts";

export async function setupNotificationChannel(): Promise<void> {
  if (Platform.OS === "android") {
    await Notifications.setNotificationChannelAsync(ANDROID_CHANNEL_ALERTS, {
      name: "Alertas de combustible/telemetría",
      importance: Notifications.AndroidImportance.HIGH,
      vibrationPattern: [0, 250, 250, 250],
    });
  }
}

/**
 * Pide permiso y devuelve el Expo push token del dispositivo, o null si el
 * usuario lo negó o estamos en un simulador (los simuladores no reciben
 * push reales).
 */
export async function registerForPushNotificationsAsync(): Promise<string | null> {
  if (!Device.isDevice) {
    console.warn("push: los simuladores no soportan notificaciones push remotas");
    return null;
  }

  await setupNotificationChannel();

  const { status: existing } = await Notifications.getPermissionsAsync();
  let finalStatus = existing;
  if (existing !== "granted") {
    const { status } = await Notifications.requestPermissionsAsync();
    finalStatus = status;
  }
  if (finalStatus !== "granted") {
    return null;
  }

  const { data: expoPushToken } = await Notifications.getExpoPushTokenAsync();
  return expoPushToken;
}

/**
 * Registra el Expo push token contra POST /devices/register (protegido
 * por RequireAuth — requiere estar logueado), para que el backend sepa a
 * qué tokens mandarle push cuando se cree una alerta de un vehículo de
 * este usuario (ver internal/httpapi/devices_handlers.go).
 */
export async function registerDeviceToken(pushToken: string): Promise<void> {
  try {
    await api.post("/devices/register", { push_token: pushToken, platform: Platform.OS });
  } catch (err) {
    console.warn("push: no se pudo registrar el token contra el backend", err);
  }
}

export function addNotificationReceivedListener(
  handler: (notification: Notifications.Notification) => void
) {
  return Notifications.addNotificationReceivedListener(handler);
}

export function addNotificationResponseListener(
  handler: (response: Notifications.NotificationResponse) => void
) {
  return Notifications.addNotificationResponseReceivedListener(handler);
}

/**
 * Dispara una notificación local inmediata. Se usa como fallback: si el
 * socket WS recibe un evento "alert" mientras la app está en foreground,
 * mostramos un aviso local en vez de depender del push remoto (que es
 * para cuando la app está en background/cerrada).
 */
export async function notifyLocalAlert(vehicleId: string, message: string): Promise<void> {
  await Notifications.scheduleNotificationAsync({
    content: {
      title: `Alerta: ${vehicleId}`,
      body: message,
      sound: true,
    },
    trigger: null,
  });
}
