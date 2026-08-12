// TAREA (mobile): "réplica del dashboard web" — detalle de un vehículo:
// última posición + historial reciente (combustible/temperatura/velocidad,
// equivalente de web/src/components/charts/*) + alertas.
//
// La posición se muestra como link a Google/Apple Maps en vez de un mapa
// embebido (react-native-maps): el mapa nativo requiere una API key de
// Google Maps configurada en app.json para builds standalone de Android —
// sin ella, el MapView revienta nativo al montarse y tumba la app entera
// (no es un error de JS capturable con error boundary). Hasta tener una
// API key real, este link evita el crash sin perder la funcionalidad.
import { useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Linking,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  useWindowDimensions,
  View,
} from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";

import { useVehicles } from "@/hooks/useVehicles";
import { useAuth } from "@/hooks/useAuth";
import { colors, spacing } from "@/theme/theme";
import type { MainStackParamList } from "@/navigation/types";
import type { SensorReading } from "@/types/api";

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
  // TAREA: "sistema de alertas predictivas (solo admin visible)" — igual
  // que web (que ni siquiera muestra esta sección a no-admins).
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";

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

          <TouchableOpacity style={styles.mapWrap} onPress={() => openInMaps(reading.lat, reading.lon, vehicle.name)}>
            <Text style={styles.mapLinkTitle}>📍 Última posición conocida</Text>
            <Text style={styles.muted}>
              {reading.lat.toFixed(5)}, {reading.lon.toFixed(5)}
            </Text>
            <Text style={styles.mapLinkCta}>Ver en Maps →</Text>
          </TouchableOpacity>
          <Text style={styles.mapHint}>
            Se abre la app de Maps de tu teléfono. Para volver a Simon PT, usa el botón &quot;atrás&quot; de tu
            celular.
          </Text>
        </>
      ) : (
        <Text style={[styles.muted, { marginTop: spacing(4) }]}>Todavía no llegaron lecturas de sensores.</Text>
      )}

      {isAdmin && (
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
      )}

      {readings.length > 0 && (
        <>
          <ChartSection
            title="Combustible (histórico reciente)"
            readings={readings}
            getValue={(r) => r.fuel_level}
            color={colors.primary}
            unit="%"
          />
          <ChartSection
            title="Temperatura (histórico reciente)"
            readings={readings}
            getValue={(r) => r.temperature_c}
            color={colors.warning}
            unit="°C"
          />
          <ChartSection
            title="Velocidad (histórico reciente)"
            readings={readings}
            getValue={(r) => r.speed_kmh}
            color={colors.success}
            unit=" km/h"
          />
        </>
      )}
    </ScrollView>
  );
}

// openInMaps delega la ubicación a la app de mapas nativa del teléfono
// (Google Maps en Android, Apple Maps en iOS) vía un intent/URL scheme
// estándar del sistema operativo — no depende de ningún módulo nativo
// propio, así que no puede tumbar la app.
function openInMaps(lat: number, lon: number, label: string) {
  const query = encodeURIComponent(label);
  const url =
    Platform.OS === "ios" ? `maps:0,0?q=${query}@${lat},${lon}` : `geo:${lat},${lon}?q=${lat},${lon}(${query})`;
  Linking.openURL(url).catch(() => {
    // Fallback universal si no hay app de mapas nativa instalada.
    Linking.openURL(`https://www.google.com/maps/search/?api=1&query=${lat},${lon}`);
  });
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
  readings,
  getValue,
  color,
  unit,
}: {
  title: string;
  readings: SensorReading[];
  getValue: (r: SensorReading) => number;
  color: string;
  unit: string;
}) {
  const values = useMemo(() => readings.map(getValue), [readings, getValue]);
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
      <Sparkline readings={readings} values={values} color={color} unit={unit} />
    </View>
  );
}

// Sparkline minimalista sin dependencias de charting nativas (evita traer
// recharts/victory-native solo para esto en el scaffold inicial), pero
// "explorable": tocar una barra la selecciona y muestra su valor + hora
// exacta arriba del gráfico. Las barras usan flex:1 (en vez de un ancho
// fijo en px) para repartirse todo el ancho disponible del contenedor —
// así el gráfico se adapta solo si el usuario rota el celular a horizontal
// (más ancho disponible) sin dejar espacio vacío ni recortarse.
function Sparkline({
  readings,
  values,
  color,
  unit,
}: {
  readings: SensorReading[];
  values: number[];
  color: string;
  unit: string;
}) {
  const [selected, setSelected] = useState<number | null>(null);
  // Re-render en rotación: el ancho de las barras (flex) ya se recalcula
  // solo vía layout, pero necesitamos que el componente vuelva a montar el
  // gesto de selección con las dimensiones nuevas (evita un índice
  // seleccionado que quede "pegado" visualmente al rotar).
  const { width } = useWindowDimensions();
  useEffect(() => setSelected(null), [width]);

  const max = Math.max(...values, 1);
  const min = Math.min(...values, 0);
  const range = Math.max(max - min, 1);

  const active = selected !== null ? selected : values.length - 1;
  const activeReading = readings[active];
  const activeValue = values[active];

  return (
    <View>
      {activeReading && activeValue !== undefined && (
        <Text style={styles.sparkTooltip}>
          {activeValue.toFixed(1)}
          {unit} · {new Date(activeReading.recorded_at).toLocaleTimeString()}
        </Text>
      )}
      <View style={styles.sparkline}>
        {values.map((v, i) => (
          <Pressable key={i} style={styles.sparkCol} onPress={() => setSelected(i)}>
            <View
              style={[
                styles.sparkBar,
                {
                  height: Math.max(4, ((v - min) / range) * 60),
                  backgroundColor: color,
                  opacity: i === active ? 1 : 0.55,
                },
              ]}
            />
          </Pressable>
        ))}
      </View>
      <Text style={styles.sparkHint}>Toca una barra para ver el valor exacto de ese momento.</Text>
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
  mapWrap: {
    borderRadius: 12,
    marginTop: spacing(4),
    padding: spacing(4),
    backgroundColor: colors.surface,
  },
  mapLinkTitle: { color: colors.text, fontWeight: "600", marginBottom: spacing(1) },
  mapLinkCta: { color: colors.primary, fontWeight: "600", marginTop: spacing(2) },
  mapHint: { color: colors.textMuted, fontSize: 11, marginTop: spacing(1), textAlign: "center" },
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
  sparkline: { flexDirection: "row", alignItems: "flex-end", height: 60, width: "100%" },
  sparkCol: { flex: 1, alignItems: "center", justifyContent: "flex-end", paddingHorizontal: 1 },
  sparkBar: { width: "100%", borderRadius: 2, minWidth: 2 },
  sparkTooltip: { color: colors.text, fontSize: 12, fontWeight: "700", marginBottom: spacing(1) },
  sparkHint: { color: colors.textMuted, fontSize: 10, marginTop: spacing(1), textAlign: "center" },
});
