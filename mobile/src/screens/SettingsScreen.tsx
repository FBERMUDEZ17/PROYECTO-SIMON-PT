import { useEffect, useState } from "react";
import { Alert, StyleSheet, Switch, Text, TouchableOpacity, View } from "react-native";

import { useAuth } from "@/hooks/useAuth";
import { registerForPushNotificationsAsync, registerDeviceToken } from "@/notifications/push";
import { colors, spacing } from "@/theme/theme";

export function SettingsScreen() {
  const { user, logout } = useAuth();
  const [pushEnabled, setPushEnabled] = useState(false);

  useEffect(() => {
    registerForPushNotificationsAsync().then((token) => setPushEnabled(!!token));
  }, []);

  const togglePush = async (value: boolean) => {
    if (value) {
      const token = await registerForPushNotificationsAsync();
      if (token) {
        await registerDeviceToken(token);
        setPushEnabled(true);
      } else {
        Alert.alert("Permiso denegado", "Habilitá las notificaciones en Ajustes del sistema para recibir alertas.");
      }
    } else {
      // expo-notifications no tiene "des-registro" del lado cliente; el
      // backend simplemente dejaría de usar el token si el usuario lo pide
      // (fuera de alcance de este toggle visual).
      setPushEnabled(false);
    }
  };

  return (
    <View style={styles.container}>
      <View style={styles.section}>
        <Text style={styles.label}>Usuario</Text>
        <Text style={styles.value}>{user?.name}</Text>
        <Text style={styles.value}>{user?.email}</Text>
        <Text style={styles.role}>{user?.role}</Text>
      </View>

      <View style={styles.rowSection}>
        <Text style={styles.label}>Notificaciones push</Text>
        <Switch value={pushEnabled} onValueChange={togglePush} />
      </View>

      <TouchableOpacity style={styles.logoutButton} onPress={() => logout()}>
        <Text style={styles.logoutText}>Cerrar sesión</Text>
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.background, padding: spacing(4) },
  section: { backgroundColor: colors.surface, borderRadius: 12, padding: spacing(4), marginBottom: spacing(4) },
  rowSection: {
    backgroundColor: colors.surface,
    borderRadius: 12,
    padding: spacing(4),
    marginBottom: spacing(4),
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
  },
  label: { color: colors.textMuted, fontSize: 12 },
  value: { color: colors.text, fontSize: 16, fontWeight: "600" },
  role: { color: colors.primary, marginTop: spacing(1), textTransform: "uppercase", fontSize: 11 },
  logoutButton: { backgroundColor: colors.danger, borderRadius: 8, padding: spacing(3), alignItems: "center" },
  logoutText: { color: colors.background, fontWeight: "700" },
});
