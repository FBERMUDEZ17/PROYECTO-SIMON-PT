import { useEffect, useMemo, useState } from "react";
import { FlatList, RefreshControl, StyleSheet, Text, TextInput, TouchableOpacity, View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";

import { useVehicles } from "@/hooks/useVehicles";
import { VehicleCard } from "@/components/VehicleCard";
import { ConnectionBanner } from "@/components/ConnectionBanner";
import { pendingCount, drainQueue } from "@/storage/offlineQueue";
import { colors, spacing } from "@/theme/theme";
import { LOW_FUEL_THRESHOLD_HOURS, HIGH_SPEED_THRESHOLD_KMH } from "@/types/api";
import type { Vehicle } from "@/types/api";
import type { MainStackParamList, MainTabParamList } from "@/navigation/types";

// Composite porque este tab vive dentro del stack de MainNavigator y
// necesita poder navegar a "VehicleDetail", que no es parte del propio
// tab navigator sino de su stack padre.
type Props = CompositeScreenProps<
  BottomTabScreenProps<MainTabParamList, "Dashboard">,
  NativeStackScreenProps<MainStackParamList>
>;

// TAREA (mejora UX): mismo criterio que web/src/components/dashboard/
// VehicleListPanel.tsx — buscador + filtros rápidos para no tener que
// desplazarse a mano por toda la flota.
type ListFilter = "all" | "low_fuel" | "over_speed";

function isLowFuel(v: Vehicle) {
  const r = v.latest_reading;
  return !!r && r.autonomy_hours !== undefined && r.autonomy_hours < LOW_FUEL_THRESHOLD_HOURS;
}

function isOverSpeed(v: Vehicle) {
  const r = v.latest_reading;
  return !!r && r.speed_kmh > HIGH_SPEED_THRESHOLD_KMH;
}

const FILTERS: { key: ListFilter; label: string }[] = [
  { key: "all", label: "Todos" },
  { key: "low_fuel", label: "Combustible bajo" },
  { key: "over_speed", label: "Exceso de velocidad" },
];

export function DashboardScreen({ navigation }: Props) {
  const { vehicles, loading, isOffline, refresh, wsStatus } = useVehicles();
  const [pending, setPending] = useState(0);
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<ListFilter>("all");

  useEffect(() => {
    const refreshPending = () => pendingCount().then(setPending);
    refreshPending();
    const interval = setInterval(async () => {
      await drainQueue();
      refreshPending();
    }, 15000);
    return () => clearInterval(interval);
  }, []);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return vehicles.filter((v) => {
      if (q && !v.name.toLowerCase().includes(q) && !v.id.toLowerCase().includes(q)) return false;
      if (filter === "low_fuel" && !isLowFuel(v)) return false;
      if (filter === "over_speed" && !isOverSpeed(v)) return false;
      return true;
    });
  }, [vehicles, query, filter]);

  return (
    <View style={styles.container}>
      <ConnectionBanner isOffline={isOffline} wsStatus={wsStatus} pendingCount={pending} />

      {vehicles.length > 0 && (
        <View style={styles.filters}>
          <TextInput
            value={query}
            onChangeText={setQuery}
            placeholder="Buscar vehículo por nombre o ID…"
            placeholderTextColor={colors.textMuted}
            style={styles.search}
          />
          <View style={styles.chipsRow}>
            {FILTERS.map((f) => (
              <TouchableOpacity
                key={f.key}
                onPress={() => setFilter(f.key)}
                style={[styles.chip, filter === f.key && styles.chipActive]}
              >
                <Text style={[styles.chipText, filter === f.key && styles.chipTextActive]}>{f.label}</Text>
              </TouchableOpacity>
            ))}
          </View>
          <Text style={styles.count}>
            {filtered.length} de {vehicles.length} vehículos
          </Text>
        </View>
      )}

      <FlatList
        contentContainerStyle={styles.list}
        data={filtered}
        keyExtractor={(v) => v.id}
        refreshControl={<RefreshControl refreshing={loading} onRefresh={refresh} tintColor={colors.primary} />}
        ListEmptyComponent={
          !loading ? (
            <Text style={styles.empty}>
              {vehicles.length === 0
                ? "Sin vehículos para mostrar todavía."
                : "Ningún vehículo coincide con la búsqueda o el filtro."}
            </Text>
          ) : null
        }
        renderItem={({ item }) => (
          <VehicleCard vehicle={item} onPress={() => navigation.navigate("VehicleDetail", { vehicleId: item.id })} />
        )}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.background },
  list: { padding: spacing(4), paddingTop: 0 },
  empty: { color: colors.textMuted, textAlign: "center", marginTop: spacing(10) },
  filters: { paddingHorizontal: spacing(4), paddingTop: spacing(3), paddingBottom: spacing(1) },
  search: {
    backgroundColor: colors.surface,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: colors.border,
    color: colors.text,
    paddingHorizontal: spacing(3),
    paddingVertical: spacing(2),
    fontSize: 14,
  },
  chipsRow: { flexDirection: "row", flexWrap: "wrap", gap: spacing(2), marginTop: spacing(2) },
  chip: {
    backgroundColor: colors.surface,
    borderRadius: 999,
    paddingHorizontal: spacing(3),
    paddingVertical: spacing(1.5),
    borderWidth: 1,
    borderColor: colors.border,
  },
  chipActive: { backgroundColor: colors.primary, borderColor: colors.primary },
  chipText: { color: colors.textMuted, fontSize: 12, fontWeight: "600" },
  chipTextActive: { color: colors.background },
  count: { color: colors.textMuted, fontSize: 11, marginTop: spacing(2) },
});
