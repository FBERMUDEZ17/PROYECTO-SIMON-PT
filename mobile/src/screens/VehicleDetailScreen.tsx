// TAREA (mobile): "réplica del dashboard web" — detalle de un vehículo:
// mapa con la última posición (equivalente mobile de
// web/src/components/map/VehicleMap.tsx, con react-native-maps en vez de
// maplibre-gl) + historial reciente + alertas.
import { useEffect, useState } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import MapView, { Marker } from "react-native-maps";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";

import { getVehicle } from "@/api/vehicles";
import { colors, spacing } from "@/theme/theme";
import type { MainStackParamList } from "@/navigation/types";
import type { Vehicle } from "@/types/api";

type Props = NativeStackScreenProps<MainStackParamList, "VehicleDetail">;

export function VehicleDetailScreen({ route }: Props) {
  const { vehicleId } = route.params;
  const [vehicle, setVehicle] = useState<Vehicle | null>(null);

  useEffect(() => {
    getVehicle(vehicleId)
      .then(setVehicle)
      .catch(() => {
        // Offline o vehículo con id enmascarado no resoluble (ver
        // vehicles_handlers.go: List() ya viene con todo lo necesario para
        // no-admin) — el listado (Dashboard) es la fuente completa en ese
        // caso, esta pantalla puede quedar sin datos frescos.
      });
  }, [vehicleId]);

  if (!vehicle) {
    return (
      <View style={styles.center}>
        <Text style={styles.muted}>Cargando…</Text>
      </View>
    );
  }

  const reading = vehicle.latest_reading;

  return (
    <ScrollView style={styles.container} contentContainerStyle={{ padding: spacing(4) }}>
      <Text style={styles.title}>{vehicle.name}</Text>
      <Text style={styles.muted}>{vehicle.id}</Text>

      {reading && (
        <View style={styles.mapWrap}>
          <MapView
            style={styles.map}
            initialRegion={{
              latitude: reading.lat,
              longitude: reading.lon,
              latitudeDelta: 0.05,
              longitudeDelta: 0.05,
            }}
          >
            <Marker coordinate={{ latitude: reading.lat, longitude: reading.lon }} title={vehicle.name} />
          </MapView>
        </View>
      )}

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Alertas recientes</Text>
        {vehicle.recent_alerts.length === 0 && <Text style={styles.muted}>Sin alertas.</Text>}
        {vehicle.recent_alerts.map((a, i) => (
          <View key={i} style={styles.alertRow}>
            <Text style={styles.alertType}>{a.type}</Text>
            <Text style={styles.alertMessage}>{a.message}</Text>
          </View>
        ))}
      </View>

      {vehicle.recent_readings && vehicle.recent_readings.length > 0 && (
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Combustible (histórico reciente)</Text>
          <FuelSparkline values={vehicle.recent_readings.map((r) => r.fuel_level)} />
        </View>
      )}
    </ScrollView>
  );
}

// Sparkline minimalista sin dependencias de charting nativas (evita traer
// recharts/victory-native solo para esto en el scaffold inicial).
function FuelSparkline({ values }: { values: number[] }) {
  const max = Math.max(...values, 1);
  return (
    <View style={styles.sparkline}>
      {values.map((v, i) => (
        <View key={i} style={[styles.sparkBar, { height: Math.max(4, (v / max) * 60) }]} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.background },
  center: { flex: 1, backgroundColor: colors.background, justifyContent: "center", alignItems: "center" },
  title: { color: colors.text, fontSize: 22, fontWeight: "700" },
  muted: { color: colors.textMuted },
  mapWrap: { height: 220, borderRadius: 12, overflow: "hidden", marginVertical: spacing(4) },
  map: { flex: 1 },
  section: { marginTop: spacing(4) },
  sectionTitle: { color: colors.text, fontWeight: "600", marginBottom: spacing(2) },
  alertRow: {
    backgroundColor: colors.surface,
    borderRadius: 8,
    padding: spacing(3),
    marginBottom: spacing(2),
  },
  alertType: { color: colors.warning, fontWeight: "700", fontSize: 12 },
  alertMessage: { color: colors.text },
  sparkline: { flexDirection: "row", alignItems: "flex-end", height: 60, gap: 2 },
  sparkBar: { width: 4, backgroundColor: colors.primary, borderRadius: 2 },
});
