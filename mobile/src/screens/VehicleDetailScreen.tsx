// TAREA (mobile): "réplica del dashboard web" — detalle de un vehículo:
// mapa con la última posición (equivalente mobile de
// web/src/components/map/VehicleMap.tsx, con react-native-maps en vez de
// maplibre-gl) + historial reciente (combustible/temperatura/velocidad,
// equivalente de web/src/components/charts/*) + alertas.
import { useMemo } from "react";
import { ActivityIndicator, ScrollView, StyleSheet, Text, View } from "react-native";
import MapView, { Marker } from "react-native-maps";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";

import { useVehicles } from "@/hooks/useVehicles";
import { colors, spacing } from "@/theme/theme";
import type { MainStackParamList } from "@/navigation/types";

type Props = NativeStackScreenProps<MainStackParamList, "VehicleDetail">;

export function VehicleDetailScreen({ route }: Props) {
  const { vehicleId } = route.params;
  // Bug corregido: esta pantalla hacía un segundo fetch por id
  // (GET /vehicles/{id}), pero para usuarios no-admin el id que llega por
  // navegación ya viene enmascarado por el backend ("DEV-****-XC54") — ese
  // id enmascarado nunca resuelve en GetByID (404, ver
  // internal/httpapi/vehicles_handlers.go), así que la pantalla se quedaba
  // en "Cargando…" para siempre y ni el mapa ni los gráficos llegaban a
  // pintarse. La corrección (igual que el frontend web,
  // VehicleDetailPanel.tsx): usar el mismo listado ya cargado por
  // useVehicles (compartido vía VehiclesProvider), que es la fuente
  // completa de datos y además se actualiza en vivo por WS.
  const { vehicles, loading, refresh } = useVehicles();
  const vehicle = useMemo(() => vehicles.find((v) => v.id === vehicleId), [vehicles, vehicleId]);

  if (!vehicle && loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color={colors.primary} size="large" />
        <Text style={[styles.muted, { marginTop: spacing(2) }]}>Cargando…</Text>
      </View>
    );
  }

  if (!vehicle) {
    return (
      <View style={styles.center}>
        <Text style={styles.muted}>No se encontró información de este vehículo.</Text>
        <Text style={styles.retry} onPress={refresh}>
          Reintentar
        </Text>
      </View>
    );
  }

  const reading = vehicle.latest_reading;
  const readings = vehicle.recent_readings ?? [];

  return (
    <ScrollView style={styles.container} contentContainerStyle={{ padding: spacing(4) }}>
      <Text style={styles.title}>{vehicle.name}</Text>
      <Text style={styles.muted}>{vehicle.id}</Text>

      {reading ? (
        <>
          <View style={styles.statsRow}>
            <Stat label="Combustible" value={`${reading.fuel_level.toFixed(0)}%`} color={colors.primary} />
            <Stat label="Temperatura" value={`${reading.temperature_c.toFixed(1)}°C`} color={colors.warning} />
            <Stat label="Velocidad" value={`${reading.speed_kmh.toFixed(0)} km/h`} color={colors.success} />
          </View>

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
        </>
      ) : (
        <Text style={[styles.muted, { marginTop: spacing(4) }]}>Todavía no llegaron lecturas de sensores.</Text>
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

      {readings.length > 0 && (
        <>
          <ChartSection
            title="Combustible (histórico reciente)"
            values={readings.map((r) => r.fuel_level)}
            color={colors.primary}
            unit="%"
          />
          <ChartSection
            title="Temperatura (histórico reciente)"
            values={readings.map((r) => r.temperature_c)}
            color={colors.warning}
            unit="°C"
          />
          <ChartSection
            title="Velocidad (histórico reciente)"
            values={readings.map((r) => r.speed_kmh)}
            color={colors.success}
            unit=" km/h"
          />
        </>
      )}
    </ScrollView>
  );
}

function Stat({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <View style={styles.stat}>
      <Text style={[styles.statValue, { color }]}>{value}</Text>
      <Text style={styles.statLabel}>{label}</Text>
    </View>
  );
}

function ChartSection({
  title,
  values,
  color,
  unit,
}: {
  title: string;
  values: number[];
  color: string;
  unit: string;
}) {
  const last = values[values.length - 1];
  return (
    <View style={styles.section}>
      <View style={styles.chartHeader}>
        <Text style={styles.sectionTitle}>{title}</Text>
        {last !== undefined && (
          <Text style={[styles.chartLast, { color }]}>
            {last.toFixed(1)}
            {unit}
          </Text>
        )}
      </View>
      <Sparkline values={values} color={color} />
    </View>
  );
}

// Sparkline minimalista sin dependencias de charting nativas (evita traer
// recharts/victory-native solo para esto en el scaffold inicial).
function Sparkline({ values, color }: { values: number[]; color: string }) {
  const max = Math.max(...values, 1);
  const min = Math.min(...values, 0);
  const range = Math.max(max - min, 1);
  return (
    <View style={styles.sparkline}>
      {values.map((v, i) => (
        <View
          key={i}
          style={[styles.sparkBar, { height: Math.max(4, ((v - min) / range) * 60), backgroundColor: color }]}
        />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.background },
  center: {
    flex: 1,
    backgroundColor: colors.background,
    justifyContent: "center",
    alignItems: "center",
    padding: spacing(4),
  },
  title: { color: colors.text, fontSize: 22, fontWeight: "700" },
  muted: { color: colors.textMuted, textAlign: "center" },
  retry: { color: colors.primary, fontWeight: "600", marginTop: spacing(3) },
  statsRow: { flexDirection: "row", gap: spacing(3), marginTop: spacing(4) },
  stat: {
    flex: 1,
    backgroundColor: colors.surface,
    borderRadius: 10,
    paddingVertical: spacing(3),
    alignItems: "center",
  },
  statValue: { fontSize: 16, fontWeight: "700" },
  statLabel: { color: colors.textMuted, fontSize: 11, marginTop: 2 },
  mapWrap: { height: 220, borderRadius: 12, overflow: "hidden", marginTop: spacing(4) },
  map: { flex: 1 },
  section: { marginTop: spacing(5) },
  chartHeader: { flexDirection: "row", justifyContent: "space-between", alignItems: "baseline" },
  chartLast: { fontSize: 13, fontWeight: "700" },
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
  sparkBar: { width: 4, borderRadius: 2 },
});
