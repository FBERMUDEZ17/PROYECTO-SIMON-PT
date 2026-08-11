package vehicles

// TAREA: "enmascarar device IDs (DEV-****-XC54) para non-admin" — usado
// por los handlers HTTP de /vehicles y por el hub de WebSockets, ambos
// consultando claims.IsAdmin() para decidir si aplicar la máscara.

import "strings"

// MaskDeviceID enmascara segmentos internos de un device id para usuarios
// no-admin, ej. "DEV-1234-XC54" -> "DEV-****-XC54". Conserva el primer y
// último segmento (separados por "-") para que el id siga siendo
// reconocible sin exponer el identificador completo del dispositivo.
func MaskDeviceID(id string) string {
	parts := strings.Split(id, "-")

	switch {
	case len(parts) >= 3:
		masked := make([]string, len(parts))
		masked[0] = parts[0]
		for i := 1; i < len(parts)-1; i++ {
			masked[i] = "****"
		}
		masked[len(parts)-1] = parts[len(parts)-1]
		return strings.Join(masked, "-")
	case len(parts) == 2:
		return parts[0] + "-****"
	case len(id) <= 4:
		return strings.Repeat("*", len(id))
	default:
		return id[:1] + strings.Repeat("*", len(id)-2) + id[len(id)-1:]
	}
}
