// Espejo mobile de web/(auth)/forgot-password: el backend nunca revela si
// el email existe (siempre 200 genérico) y, en dev, loggea el token en el
// servidor — no hay email real configurado (ver CLAUDE.md). Por eso este
// flujo también pide pegar el token manualmente, igual que la web.
import { useState } from "react";
import { ActivityIndicator, Alert, StyleSheet, Text, TextInput, TouchableOpacity, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";

import { forgotPassword, resetPassword } from "@/api/auth";
import { colors, spacing } from "@/theme/theme";
import type { AuthStackParamList } from "@/navigation/types";

type Props = NativeStackScreenProps<AuthStackParamList, "ForgotPassword">;

export function ForgotPasswordScreen({ navigation }: Props) {
  const [email, setEmail] = useState("");
  const [token, setToken] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [requested, setRequested] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const requestReset = async () => {
    setSubmitting(true);
    try {
      await forgotPassword(email.trim());
      setRequested(true);
      Alert.alert("Listo", "Si el email existe, se generó un token de reseteo (revisá el log del servidor en dev).");
    } finally {
      setSubmitting(false);
    }
  };

  const confirmReset = async () => {
    setSubmitting(true);
    try {
      await resetPassword(token.trim(), newPassword);
      Alert.alert("Contraseña actualizada", "Ya podés iniciar sesión.", [
        { text: "OK", onPress: () => navigation.navigate("Login") },
      ]);
    } catch (err) {
      Alert.alert("No se pudo resetear", err instanceof Error ? err.message : "Error desconocido");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Recuperar contraseña</Text>

      <TextInput
        style={styles.input}
        placeholder="Email"
        placeholderTextColor={colors.textMuted}
        autoCapitalize="none"
        keyboardType="email-address"
        value={email}
        onChangeText={setEmail}
      />
      <TouchableOpacity style={styles.button} onPress={requestReset} disabled={submitting}>
        {submitting && !requested ? <ActivityIndicator color={colors.background} /> : <Text style={styles.buttonText}>Enviar token</Text>}
      </TouchableOpacity>

      {requested && (
        <>
          <TextInput
            style={styles.input}
            placeholder="Token (pegado del log del servidor)"
            placeholderTextColor={colors.textMuted}
            autoCapitalize="none"
            value={token}
            onChangeText={setToken}
          />
          <TextInput
            style={styles.input}
            placeholder="Nueva contraseña"
            placeholderTextColor={colors.textMuted}
            secureTextEntry
            value={newPassword}
            onChangeText={setNewPassword}
          />
          <TouchableOpacity style={styles.button} onPress={confirmReset} disabled={submitting}>
            <Text style={styles.buttonText}>Confirmar nueva contraseña</Text>
          </TouchableOpacity>
        </>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.background, justifyContent: "center", padding: spacing(6) },
  title: { color: colors.text, fontSize: 22, fontWeight: "700", textAlign: "center", marginBottom: spacing(8) },
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
    marginVertical: spacing(2),
  },
  buttonText: { color: colors.background, fontWeight: "700" },
});
