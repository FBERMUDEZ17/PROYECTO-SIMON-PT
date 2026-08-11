import { useMemo, useState } from "react";
import { FlatList, ScrollView, StyleSheet, Text, TouchableOpacity, View } from "react-native";

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

// TAREA (bug corregido): esta pantalla mostraba siempre las alertas de
// TODA la flota, sin forma de acotarlas a un vehículo puntual (mismo bug
// que tenía el AlertsPanel del frontend web). Se agrega un selector de
// vehículo (chips, igual criterio que los filtros de DashboardScreen) que
// filtra la lista — "Todos" mantiene el comportamiento anterior.
export function AlertsScreen() {
  const { vehicles, loading } = useVehicles();
  const [selectedVehicleId, setSelectedVehicleId] = useState<string | null>(null);

  const vehiclesWithAlerts = useMemo(() => vehicles.filter((v) => v.recent_alerts.length > 0), [vehicles]);

  const alerts = useMemo(() => {
    const all = flattenAlerts(vehicles);
    return selectedVehicleId ? all.filter((a) => a.vehicleId === selectedVehicleId) : all;
  }, [vehicles, selectedVehicleId]);

  return (
    <View style={styles.container}>
      {vehiclesWithAlerts.length > 0 && (
        <ScrollView horizontal showsHorizontalScrollIndicator={false} style={styles.chipsRow} contentContainerStyle={styles.chipsContent}>
          <TouchableOpacity
            onPress={() => setSelectedVehicleId(null)}
            style={[styles.chip, selectedVehicleId === null && styles.chipActive]}
          >
            <Text style={[styles.chipText, selectedVehicleId === null && styles.chipTextActive]}>Todos</Text>
          </TouchableOpacity>
          {vehiclesWithAlerts.map((v) => (
            <TouchableOpacity
              key={v.id}
              onPress={() => setSelectedVehicleId(v.id)}
              style={[styles.chip, selectedVehicleId === v.id && styles.chipActive]}
            >
              <Text style={[styles.chipText, selectedVehicleId === v.id && styles.chipTextActive]}>{v.name}</Text>
            </TouchableOpacity>
          ))}
        </ScrollView>
      )}

      <FlatList
        contentContainerStyle={styles.list}
        data={alerts}
        keyExtractor={(a, i) => `${a.vehicleId}-${a.created_at}-${i}`}
        ListEmptyComponent={
          !loading ? (
            <Text style={styles.empty}>
              {selectedVehicleId ? "Este vehículo no tiene alertas recientes." : "Sin alertas recientes."}
            </Text>
          ) : null
        }
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
  chipsRow: { flexGrow: 0, paddingTop: spacing(3) },
  chipsContent: { paddingHorizontal: spacing(4), gap: spacing(2) },
  chip: {
    backgroundColor: colors.surface,
    borderRadius: 999,
    paddingHorizontal: spacing(3),
    paddingVertical: spacing(1.5),
    borderWidth: 1,
    borderColor: colors.border,
    marginRight: spacing(2),
  },
  chipActive: { backgroundColor: colors.primary, borderColor: colors.primary },
  chipText: { color: colors.textMuted, fontSize: 12, fontWeight: "600" },
  chipTextActive: { color: colors.background },
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
