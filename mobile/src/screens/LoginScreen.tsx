import { useState } from "react";
import { ActivityIndicator, Alert, StyleSheet, Text, TextInput, TouchableOpacity, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";

import { useAuth } from "@/hooks/useAuth";
import { getApiUrl, setApiUrl } from "@/config";
import { colors, spacing } from "@/theme/theme";
import type { AuthStackParamList } from "@/navigation/types";

type Props = NativeStackScreenProps<AuthStackParamList, "Login">;

export function LoginScreen({ navigation }: Props) {
  const { login } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);

  // TAREA (mobile): el APK standalone (build vía EAS, ver mobile/eas.json)
  // se distribuye para probarse en cualquier máquina/red, no solo en la
  // que lo compiló — por eso la URL del backend es editable acá en vez de
  // estar fija en el build (ver src/config.ts).
  const [showServerConfig, setShowServerConfig] = useState(false);
  const [serverUrl, setServerUrl] = useState(getApiUrl());

  const onSubmit = async () => {
    setSubmitting(true);
    try {
      await login(email.trim(), password);
    } catch (err) {
      Alert.alert("No se pudo iniciar sesión", err instanceof Error ? err.message : "Error desconocido");
    } finally {
      setSubmitting(false);
    }
  };

  const onSaveServerUrl = async () => {
    await setApiUrl(serverUrl);
    setShowServerConfig(false);
    Alert.alert("Servidor actualizado", `La app va a usar:\n${getApiUrl()}`);
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Simon PT</Text>
      <Text style={styles.subtitle}>Monitoreo de flota en tiempo real</Text>

      <TextInput
        style={styles.input}
        placeholder="Email"
        placeholderTextColor={colors.textMuted}
        autoCapitalize="none"
        keyboardType="email-address"
        value={email}
        onChangeText={setEmail}
      />
      <TextInput
        style={styles.input}
        placeholder="Contraseña"
        placeholderTextColor={colors.textMuted}
        secureTextEntry
        value={password}
        onChangeText={setPassword}
      />

      <TouchableOpacity style={styles.button} onPress={onSubmit} disabled={submitting}>
        {submitting ? <ActivityIndicator color={colors.background} /> : <Text style={styles.buttonText}>Ingresar</Text>}
      </TouchableOpacity>

      <TouchableOpacity onPress={() => navigation.navigate("Register")}>
        <Text style={styles.link}>¿No tenés cuenta? Registrate</Text>
      </TouchableOpacity>
      <TouchableOpacity onPress={() => navigation.navigate("ForgotPassword")}>
        <Text style={styles.link}>Olvidé mi contraseña</Text>
      </TouchableOpacity>

      <TouchableOpacity onPress={() => setShowServerConfig((v) => !v)} style={styles.serverToggle}>
        <Text style={styles.serverToggleText}>⚙ Servidor: {getApiUrl()}</Text>
      </TouchableOpacity>

      {showServerConfig && (
        <View style={styles.serverConfigBox}>
          <Text style={styles.serverConfigLabel}>URL del backend (Go)</Text>
          <TextInput
            style={styles.input}
            placeholder="http://192.168.1.10:8080"
            placeholderTextColor={colors.textMuted}
            autoCapitalize="none"
            autoCorrect={false}
            keyboardType="url"
            value={serverUrl}
            onChangeText={setServerUrl}
          />
          <TouchableOpacity style={styles.buttonSecondary} onPress={onSaveServerUrl}>
            <Text style={styles.buttonText}>Guardar</Text>
          </TouchableOpacity>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.background, justifyContent: "center", padding: spacing(6) },
  title: { color: colors.text, fontSize: 28, fontWeight: "700", textAlign: "center" },
  subtitle: { color: colors.textMuted, textAlign: "center", marginBottom: spacing(8) },
  input: {
    backgroundColor: colors.surface,
    borderColor: colors.border,
    borderWidth: 1,
    borderRadius: 8,
    padding: spacing(3),
    color: colors.text,
    marginBottom: spacing(3),
  },
  button: {
    backgroundColor: colors.primary,
    borderRadius: 8,
    padding: spacing(3),
    alignItems: "center",
    marginTop: spacing(2),
  },
  buttonSecondary: {
    backgroundColor: colors.surfaceAlt,
    borderRadius: 8,
    padding: spacing(3),
    alignItems: "center",
  },
  buttonText: { color: colors.background, fontWeight: "700" },
  link: { color: colors.primary, textAlign: "center", marginTop: spacing(4) },
  serverToggle: { marginTop: spacing(8), alignItems: "center" },
  serverToggleText: { color: colors.textMuted, fontSize: 12 },
  serverConfigBox: { marginTop: spacing(4) },
  serverConfigLabel: { color: colors.textMuted, fontSize: 12, marginBottom: spacing(2) },
});
