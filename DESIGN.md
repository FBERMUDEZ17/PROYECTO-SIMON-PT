# DESIGN.md — Decisiones de arquitectura y stack

Este documento explica **qué se eligió y por qué**, y los trade-offs conscientes que se tomaron dado el alcance de la prueba. Para "cómo levantarlo", ver [SETUP.md](./SETUP.md).

## Resumen del sistema

Plataforma de telemetría vehicular (IoT) con tres piezas:

| Componente | Ubicación | Stack |
|---|---|---|
| Backend / API | `cmd/`, `internal/` | Go, chi, SQLite (modernc.org/sqlite, sin cgo), WebSockets (gorilla) |
| Dashboard web | `web/` | Next.js (App Router), TanStack Query, maplibre-gl, recharts |
| App mobile | `mobile/` | React Native + Expo, React Navigation, SQLite local, Expo Notifications |

Los tres consumen la **misma API Go** — no hay lógica de negocio duplicada entre frontend web y mobile; ambos son clientes delgados sobre el mismo backend.

## Por qué Go para el backend

- **Concurrencia nativa** para el hub de WebSockets (una goroutine por conexión, canal con buffer para no bloquear la ingesta de sensores si un cliente está lento — ver `internal/ws/hub.go`) sin la complejidad de un runtime de eventos externo.
- **Sin runtime pesado**: un solo binario estático, arranque instantáneo, ideal para un servicio que recibe telemetría de alta frecuencia.
- **Tipado fuerte + compilación** atrapa errores de contrato (forma de los DTOs, nil handling) antes de producción, algo valioso en un sistema con muchos endpoints y roles (admin/user).

**Trade-off aceptado**: menos velocidad de desarrollo inicial que un stack todo-JS (Node/Express), pero se prioriza corrección y rendimiento en la capa que más importa (ingesta + tiempo real).

## Por qué SQLite (modernc.org/sqlite, sin cgo)

- Cero infraestructura externa: no hay que levantar Postgres/MySQL para correr o evaluar el proyecto — `go run` ya deja todo funcionando.
- `modernc.org/sqlite` es una implementación **pura Go** (sin cgo), lo que evita depender de un compilador C y simplifica el build multiplataforma (Windows incluido, que es donde se desarrolló esto).
- El schema (`internal/db/db.go`) es un único bloque `CREATE TABLE IF NOT EXISTS` — no hay herramienta de migraciones porque el proyecto no la necesita a esta escala; los cambios de schema posteriores (ej. `owner_user_id`, `device_tokens`) se aplican con `ALTER TABLE` idempotentes en `migrate()`.

**Trade-off aceptado**: SQLite no escala a múltiples instancias de escritura concurrente (un solo archivo, locking a nivel de archivo). Para producción real con más de un nodo de backend, esto se reemplazaría por Postgres — el código de acceso a datos ya está aislado en `internal/*/store.go` por paquete, así que el cambio no tocaría la lógica de negocio.

## Autenticación: JWT + cookie httpOnly (con fallback a Bearer)

- El backend es el **único dueño del JWT**: lo emite en login/register y lo setea como cookie `httpOnly` (`internal/httpapi/cookie.go`) — el JS del frontend web nunca toca el token directamente, mitigando XSS de robo de sesión.
- `tokenFromRequest` (middleware) acepta **tanto** el header `Authorization: Bearer` **como** la cookie — mismo código de verificación sirve para el navegador (cookie automática) y para clientes API/mobile (que no tienen cookie jar de navegador y mandan el token explícito, ver `mobile/src/api/client.ts`).
- El endpoint de login/register también devuelve el token en el **body** de la respuesta (no solo en la cookie) — es lo que permite que mobile lo persista en `AsyncStorage` sin necesitar lógica de cookie.

**Trade-off aceptado**: no hay refresh token — el JWT dura 24h fijas. Para una prueba técnica es suficiente; en producción se agregaría rotación.

## WebSockets para tiempo real (no polling)

Se eligió push real (WS) sobre polling porque el caso de uso es telemetría de alta frecuencia con múltiples clientes mirando el mismo dashboard — polling cada N segundos generaría carga innecesaria y latencia perceptible en las alertas de combustible bajo. El hub (`internal/ws/hub.go`) hace **fan-out** de cada evento a todos los clientes conectados, aplicando el enmascarado de `device_id` por rol en el momento del envío (`renderForRole`), así el mismo evento se ve distinto para admin vs usuario común sin duplicar lógica de negocio.

## Enmascarado de `device_id` por rol

Requisito: los usuarios no-admin no deben ver el identificador completo del dispositivo (`DEV-1234-XC54` → `DEV-****-XC54`). Se implementó en **dos puntos independientes** que deben mantenerse en sync manualmente (documentado en `CLAUDE.md`): las respuestas REST (`vehicles_handlers.go`) y los eventos WS (`ws/hub.go`). Se consideró centralizar en un único punto de serialización, pero REST y WS tienen ciclos de vida distintos (request/response vs conexión persistente con fan-out) — la duplicación consciente de una función pura (`vehicles.MaskDeviceID`, cubierta por test) fue más simple que introducir una capa de serialización compartida para dos casos.

## Predicción de autonomía de combustible

`internal/sensors/fuel_prediction.go` usa **regresión lineal por mínimos cuadrados** sobre las últimas N lecturas de combustible para estimar la tasa de consumo (%/hora) y de ahí la autonomía restante. Se eligió regresión simple (no un modelo de ML) porque:
- Es determinística y auditable — importante cuando dispara alertas.
- Con pocas lecturas (a veces 2-3) un modelo más complejo sobreajustaría sin aportar precisión real.
- Es trivial de testear exhaustivamente (`fuel_prediction_test.go` cubre datos insuficientes, timestamps iguales, consumo estable/creciente, ruido).

## Mobile: Expo managed + Expo Push API (no Firebase Admin SDK directo)

- **Expo** (no React Native CLI puro) para evitar mantener proyectos nativos iOS/Android a mano en un repo que ya es multi-stack; `expo-notifications` da push funcional sin código nativo propio.
- Para el **envío** de push desde el backend (`internal/push/expo.go`) se usa la **Expo Push API** en vez de integrar el Firebase Admin SDK directamente: el cliente ya obtiene un Expo push token (que envuelve FCM/APNs), y Expo entrega el mensaje final — esto evita que el backend necesite credenciales de un proyecto Firebase propio para que la funcionalidad sea probable de punta a punta.
- **SQLite local** (`expo-sqlite`) para cache de vehículos + cola de lecturas pendientes, replicando offline-first lo que `web/src/lib/indexeddb.ts` hace en el navegador con IndexedDB — mismo patrón, tecnología nativa de cada plataforma.
- Para la **entrega** (para que se pueda instalar directo sin Expo Go, sin depender de una API key de Google Maps ni otros módulos nativos custom) se generó un build standalone real vía **EAS Build** (`mobile/eas.json`, perfil `preview` → APK instalable), documentado en SETUP.md. La posición del vehículo se resuelve delegando a la app de mapas del sistema (`Linking.openURL` con esquema `geo:`) en vez de embeber un mapa nativo — evita el módulo nativo `react-native-maps`, que requiere una API key de Google Maps para no crashear en un build standalone.

## Testing: qué se cubrió y qué no

- **Backend**: 95 tests (`go test ./...`), incluyendo unit tests de la lógica más sensible a errores silenciosos — predicción de combustible, validación de entrada, enmascarado de IDs — y tests de integración end-to-end contra un router real con SQLite en memoria (auth, ownership de vehículos, WebSockets, password reset, registro de push tokens).
- **Web/mobile**: sin suite de tests automatizados propia (fuera del typecheck de TypeScript). Dado el alcance y tiempo de la prueba, se priorizó la cobertura del backend porque ahí vive toda la lógica de negocio (validación, predicción, autorización, deduplicación de alertas) — los frontends son consumidores delgados de esa API ya testeada.

## Qué se dejaría distinto con más tiempo

- Migraciones de schema versionadas (hoy es un único `CREATE TABLE IF NOT EXISTS` + `ALTER TABLE` ad-hoc en `migrate()`).
- Refresh tokens para el JWT.
- Tests de integración E2E para mobile (Detox) y componentes web (Testing Library).
- `.easignore` ya corregido para no subir `node_modules` en cada build de EAS (ver `SETUP.md`), pero un pipeline de CI con cache de dependencias reduciría aún más los tiempos de build.
