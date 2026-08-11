import { FlatList, StyleSheet, Text, View } from "react-native";

import { useVehicles } from "@/hooks/useVehicles";
import { colors, spacing } from "@/theme/theme";
import type { Vehicle, VehicleAlert } from "@/types/api";

interface FlatAlert extends VehicleAlert {
  vehicleName: string;
  vehicleId: string;
}

function flattenAlerts(vehicles: Vehicle[]): FlatAlert[] {
  return vehicles
    .flatMap((v) => v.recent_alerts.map((a) => ({ ...a, vehicleName: v.name, vehicleId: v.id })))
    .sort((a, b) => (a.created_at < b.created_at ? 1 : -1));
}

export function AlertsScreen() {
  const { vehicles, loading } = useVehicles();
  const alerts = flattenAlerts(vehicles);

  return (
    <View style={styles.container}>
      <FlatList
        contentContainerStyle={styles.list}
        data={alerts}
        keyExtractor={(a, i) => `${a.vehicleId}-${a.created_at}-${i}`}
        ListEmptyComponent={!loading ? <Text style={styles.empty}>Sin alertas recientes.</Text> : null}
        renderItem={({ item }) => (
          <View style={styles.row}>
            <Text style={styles.type}>{item.type}</Text>
            <Text style={styles.message}>{item.message}</Text>
            <Text style={styles.meta}>
              {item.vehicleName} · {new Date(item.created_at).toLocaleString()}
            </Text>
          </View>
        )}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.background },
  list: { padding: spacing(4) },
  empty: { color: colors.textMuted, textAlign: "center", marginTop: spacing(10) },
  row: {
    backgroundColor: colors.surface,
    borderRadius: 10,
    padding: spacing(3),
    marginBottom: spacing(2),
    borderLeftWidth: 3,
    borderLeftColor: colors.danger,
  },
  type: { color: colors.warning, fontWeight: "700", fontSize: 12 },
  message: { color: colors.text, marginTop: 2 },
  meta: { color: colors.textMuted, fontSize: 11, marginTop: spacing(1) },
});
