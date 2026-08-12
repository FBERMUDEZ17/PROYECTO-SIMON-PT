import { Text, View } from "react-native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";

import { DashboardScreen } from "@/screens/DashboardScreen";
import { AlertsScreen } from "@/screens/AlertsScreen";
import { SettingsScreen } from "@/screens/SettingsScreen";
import { VehicleDetailScreen } from "@/screens/VehicleDetailScreen";
import { VehiclesProvider, useVehicles } from "@/hooks/useVehicles";
import { useAuth } from "@/hooks/useAuth";
import { LOW_FUEL_THRESHOLD_HOURS, HIGH_SPEED_THRESHOLD_KMH } from "@/types/api";
import { colors } from "@/theme/theme";
import type { MainStackParamList, MainTabParamList } from "./types";

const Tab = createBottomTabNavigator<MainTabParamList>();
const Stack = createNativeStackNavigator<MainStackParamList>();

const screenOptions = {
  headerStyle: { backgroundColor: colors.surface },
  headerTintColor: colors.text,
  tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border, height: 62, paddingTop: 6 },
  tabBarActiveTintColor: colors.primary,
  tabBarInactiveTintColor: colors.textMuted,
  tabBarLabelStyle: { fontSize: 11, fontWeight: "600" as const },
};

// Icono simple (glyph + halo circular cuando la pestaña está activa) sin
// depender de @expo/vector-icons, que no está entre las dependencias del
// proyecto — sólo react-native-svg/react-native puro.
function TabIcon({ glyph, focused }: { glyph: string; focused: boolean }) {
  return (
    <View
      style={{
        width: 30,
        height: 30,
        borderRadius: 15,
        alignItems: "center",
        justifyContent: "center",
        backgroundColor: focused ? colors.primary + "26" : "transparent",
        transform: [{ scale: focused ? 1.08 : 1 }],
      }}
    >
      <Text style={{ fontSize: focused ? 17 : 15 }}>{glyph}</Text>
    </View>
  );
}

// Badge numérico reutilizable para pestañas (combustible bajo / alertas
// activas) — react-navigation soporta tabBarBadge nativamente, pero acá
// necesitamos lógica propia (contar sólo alertas recientes) así que se
// pinta a mano sobre el ícono.
function TabBadge({ count }: { count: number }) {
  if (count <= 0) return null;
  return (
    <View
      style={{
        position: "absolute",
        top: -2,
        right: -6,
        minWidth: 16,
        height: 16,
        borderRadius: 8,
        paddingHorizontal: 3,
        backgroundColor: colors.danger,
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      <Text style={{ color: colors.background, fontSize: 10, fontWeight: "700" }}>{count > 9 ? "9+" : count}</Text>
    </View>
  );
}

function Tabs() {
  const { vehicles } = useVehicles();
  // TAREA: "sistema de alertas predictivas (solo admin visible)" — igual
  // que web/src/app/dashboard/page.tsx (user?.role === "admin"), la
  // pestaña de Alertas solo se muestra a usuarios admin. Los datos ya
  // vienen filtrados/no filtrados según el rol desde el backend, pero acá
  // además ocultamos la navegación a la pantalla para no-admins.
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const alertCount = vehicles.reduce((sum, v) => sum + v.recent_alerts.length, 0);
  const attentionCount = vehicles.filter((v) => {
    const r = v.latest_reading;
    if (!r) return false;
    const lowFuel = r.autonomy_hours !== undefined && r.autonomy_hours < LOW_FUEL_THRESHOLD_HOURS;
    const overSpeed = r.speed_kmh > HIGH_SPEED_THRESHOLD_KMH;
    return lowFuel || overSpeed;
  }).length;

  return (
    <Tab.Navigator screenOptions={screenOptions}>
      <Tab.Screen
        name="Dashboard"
        component={DashboardScreen}
        options={{
          title: "Flota",
          tabBarIcon: ({ focused }) => (
            <View>
              <TabIcon glyph="🚚" focused={focused} />
              <TabBadge count={attentionCount} />
            </View>
          ),
        }}
      />
      {isAdmin && (
        <Tab.Screen
          name="Alerts"
          component={AlertsScreen}
          options={{
            title: "Alertas",
            tabBarIcon: ({ focused }) => (
              <View>
                <TabIcon glyph="🔔" focused={focused} />
                <TabBadge count={alertCount} />
              </View>
            ),
          }}
        />
      )}
      <Tab.Screen
        name="Settings"
        component={SettingsScreen}
        options={{
          title: "Ajustes",
          tabBarIcon: ({ focused }) => <TabIcon glyph="⚙️" focused={focused} />,
        }}
      />
    </Tab.Navigator>
  );
}

export function MainNavigator() {
  return (
    <VehiclesProvider>
      <Stack.Navigator screenOptions={screenOptions}>
        <Stack.Screen name="Tabs" component={Tabs} options={{ headerShown: false }} />
        <Stack.Screen name="VehicleDetail" component={VehicleDetailScreen} options={{ title: "Detalle del vehículo" }} />
      </Stack.Navigator>
    </VehiclesProvider>
  );
}
