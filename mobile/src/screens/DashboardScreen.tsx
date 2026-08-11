import { useEffect, useState } from "react";
import { FlatList, RefreshControl, StyleSheet, Text, View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";

import { useVehicles } from "@/hooks/useVehicles";
import { VehicleCard } from "@/components/VehicleCard";
import { ConnectionBanner } from "@/components/ConnectionBanner";
import { pendingCount, drainQueue } from "@/storage/offlineQueue";
import { colors, spacing } from "@/theme/theme";
import type { MainStackParamList, MainTabParamList } from "@/navigation/types";

// Composite porque este tab vive dentro del stack de MainNavigator y
// necesita poder navegar a "VehicleDetail", que no es parte del propio
// tab navigator sino de su stack padre.
type Props = CompositeScreenProps<
  BottomTabScreenProps<MainTabParamList, "Dashboard">,
  NativeStackScreenProps<MainStackParamList>
>;

export function DashboardScreen({ navigation }: Props) {
  const { vehicles, loading, isOffline, refresh, wsStatus } = useVehicles();
  const [pending, setPending] = useState(0);

  useEffect(() => {
    const refreshPending = () => pendingCount().then(setPending);
    refreshPending();
    const interval = setInterval(async () => {
      await drainQueue();
      refreshPending();
    }, 15000);
    return () => clearInterval(interval);
  }, []);

  return (
    <View style={styles.container}>
      <ConnectionBanner isOffline={isOffline} wsStatus={wsStatus} pendingCount={pending} />
      <FlatList
        contentContainerStyle={styles.list}
        data={vehicles}
        keyExtractor={(v) => v.id}
        refreshControl={<RefreshControl refreshing={loading} onRefresh={refresh} tintColor={colors.primary} />}
        ListEmptyComponent={
          !loading ? (
            <Text style={styles.empty}>Sin vehículos para mostrar todavía.</Text>
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
  list: { padding: spacing(4) },
  empty: { color: colors.textMuted, textAlign: "center", marginTop: spacing(10) },
});
