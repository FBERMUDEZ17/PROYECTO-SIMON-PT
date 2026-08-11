import { StyleSheet, Text, View } from "react-native";

import { colors, spacing } from "@/theme/theme";
import type { Vehicle } from "@/types/api";

function fuelColor(level: number | undefined): string {
  if (level === undefined) return colors.textMuted;
  if (level < 15) return colors.danger;
  if (level < 35) return colors.warning;
  return colors.success;
}

export function VehicleCard({ vehicle, onPress }: { vehicle: Vehicle; onPress?: () => void }) {
  const reading = vehicle.latest_reading;
  const hasAlert = vehicle.recent_alerts.length > 0;

  return (
    <View style={styles.card} accessible accessibilityRole="button" onTouchEnd={onPress}>
      <View style={styles.headerRow}>
        <Text style={styles.name}>{vehicle.name}</Text>
        {hasAlert && <View style={styles.alertDot} />}
      </View>
      <Text style={styles.id}>{vehicle.id}</Text>

      {reading ? (
        <View style={styles.metricsRow}>
          <Metric label="Combustible" value={`${reading.fuel_level.toFixed(0)}%`} color={fuelColor(reading.fuel_level)} />
          <Metric label="Temp." value={`${reading.temperature_c.toFixed(1)}°C`} />
          <Metric label="Velocidad" value={`${reading.speed_kmh.toFixed(0)} km/h`} />
          {reading.autonomy_hours !== undefined && (
            <Metric label="Autonomía" value={`${reading.autonomy_hours.toFixed(1)}h`} />
          )}
        </View>
      ) : (
        <Text style={styles.noData}>Sin lecturas recientes</Text>
      )}
    </View>
  );
}

function Metric({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <View style={styles.metric}>
      <Text style={[styles.metricValue, color ? { color } : null]}>{value}</Text>
      <Text style={styles.metricLabel}>{label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.surface,
    borderRadius: 12,
    padding: spacing(4),
    marginBottom: spacing(3),
    borderWidth: 1,
    borderColor: colors.border,
  },
  headerRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  name: { color: colors.text, fontSize: 16, fontWeight: "600" },
  id: { color: colors.textMuted, fontSize: 12, marginTop: 2, marginBottom: spacing(2) },
  alertDot: { width: 10, height: 10, borderRadius: 5, backgroundColor: colors.danger },
  metricsRow: { flexDirection: "row", flexWrap: "wrap", gap: spacing(4) },
  metric: { minWidth: 70 },
  metricValue: { color: colors.text, fontSize: 15, fontWeight: "600" },
  metricLabel: { color: colors.textMuted, fontSize: 11 },
  noData: { color: colors.textMuted, fontStyle: "italic" },
});
