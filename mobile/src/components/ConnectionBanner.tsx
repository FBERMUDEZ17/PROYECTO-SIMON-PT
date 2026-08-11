import { StyleSheet, Text, View } from "react-native";
import { colors, spacing } from "@/theme/theme";

export function ConnectionBanner({
  isOffline,
  wsStatus,
  pendingCount,
}: {
  isOffline: boolean;
  wsStatus: string;
  pendingCount: number;
}) {
  if (!isOffline && wsStatus === "open" && pendingCount === 0) return null;

  return (
    <View style={styles.banner}>
      <Text style={styles.text}>
        {isOffline ? "Sin conexión — mostrando datos guardados. " : ""}
        {wsStatus !== "open" ? "Tiempo real desconectado. " : ""}
        {pendingCount > 0 ? `${pendingCount} lectura(s) pendiente(s) de sincronizar.` : ""}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  banner: {
    backgroundColor: colors.surfaceAlt,
    paddingVertical: spacing(2),
    paddingHorizontal: spacing(4),
  },
  text: { color: colors.warning, fontSize: 12 },
});
