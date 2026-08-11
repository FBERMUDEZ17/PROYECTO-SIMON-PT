// TAREA (mobile): "enmascarar device IDs en UI".
// El backend YA enmascara el device_id/nombre para usuarios no-admin en
// las respuestas REST (vehicles_handlers.go) y en los eventos WS
// (ws/hub.go:renderForRole) — el mobile nunca ve el id real de un
// no-admin, así que esta función es defensiva/redundante por diseño:
// asegura que si algún dato llegara sin enmascarar, la UI igual nunca
// muestre el id completo.
//
// Port 1:1 de internal/vehicles/mask.go (MaskDeviceID) — mantener en sync
// si cambia el original.
export function maskDeviceId(id: string): string {
  const parts = id.split("-");

  if (parts.length >= 3) {
    const masked = [...parts];
    for (let i = 1; i < parts.length - 1; i++) {
      masked[i] = "****";
    }
    return masked.join("-");
  }
  if (parts.length === 2) {
    return `${parts[0]}-****`;
  }
  if (id.length <= 4) {
    return "*".repeat(id.length);
  }
  return id[0] + "*".repeat(id.length - 2) + id[id.length - 1];
}
