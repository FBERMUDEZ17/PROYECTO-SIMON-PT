# Simon PT — Mobile (React Native / Expo)

Réplica mobile del dashboard web (`web/`), consumiendo el mismo backend Go
(`internal/`, `cmd/server`). Ver `CLAUDE.md` en la raíz del repo para la
arquitectura completa del backend.

## Stack

- **Expo SDK 51** (React Native 0.74) + TypeScript
- **React Navigation** (stack de auth, tabs + stack de la app)
- **AsyncStorage** para el JWT (el backend emite el token también en el
  body de login/register, no solo en la cookie httpOnly que usa la web —
  ver `src/api/tokenStorage.ts`)
- **expo-sqlite** para cache offline de vehículos y cola de lecturas
  pendientes (`src/storage/`)
- **expo-notifications + Firebase Cloud Messaging** para push (`src/notifications/push.ts`)
- **react-native-maps** para el mapa de posición (equivalente a
  `maplibre-gl` en la web)

## Requisitos previos

```
cd mobile
npm install
```

Backend corriendo (desde la raíz del repo):

```
JWT_SECRET=devsecret go run ./cmd/server
```

## Configurar la URL del backend

Un dispositivo/emulador **no** puede usar `localhost` para llegar a tu
máquina de desarrollo. Configurá:

```
# mobile/.env (no versionado)
EXPO_PUBLIC_API_URL=http://<tu-ip-lan>:8080
EXPO_PUBLIC_WS_URL=ws://<tu-ip-lan>:8080/ws
```

O editá los defaults en `app.json` (`expo.extra`) / `src/config.ts`.
También hay que agregar ese origen a `FRONTEND_ORIGIN` del backend si
en algún momento se sirve algo vía navegador embebido; para llamadas
REST/WS puras desde la app nativa, CORS no aplica (solo afecta a
navegadores).

## Correr

```
npm run start      # abre Expo Dev Tools / Metro
npm run android
npm run ios
```

## Push notifications (Firebase Cloud Messaging)

1. Crear un proyecto en Firebase Console, agregar apps Android/iOS con los
   `package`/`bundleIdentifier` de `app.json` (`com.simonpt.mobile`).
2. Descargar `google-services.json` (Android) y `GoogleService-Info.plist`
   (iOS) a la raíz de `mobile/` (ya están en `.gitignore`, no se commitean).
3. `src/notifications/push.ts` obtiene el Expo push token (que envuelve el
   token nativo FCM/APNs) y lo registra contra
   `POST /devices/register` — **este endpoint todavía no existe en el
   backend Go**, es el próximo paso pendiente: agregar una tabla
   `device_tokens (user_id, push_token, platform)` y, en
   `internal/alerts` (o en `telemetry.Service.Ingest` donde se crean las
   alertas), disparar un push vía Firebase Admin SDK a los tokens del
   dueño del vehículo, además del broadcast WS existente.
4. Mientras tanto, con la app en foreground, las alertas WS (`type:
   "alert"`) se muestran igual como notificación local
   (`notifyLocalAlert`), así que el flujo se puede probar de punta a
   punta sin el endpoint de servidor.

## Sincronización offline

- `src/storage/db.ts` abre una SQLite local con dos tablas:
  `vehicles_cache` (último `GET /vehicles` bueno) y `pending_readings`
  (lecturas de sensor que no se pudieron enviar).
- `src/storage/offlineQueue.ts` reintenta drenar la cola cuando
  `NetInfo` detecta que volvió la conexión, y también en un poll cada 15s
  mientras el Dashboard está abierto (`DashboardScreen.tsx`).
- El listado de vehículos (`useVehicles`) cae a la cache SQLite si el
  fetch falla por falta de red (`NetworkError`), en vez de mostrar una
  pantalla vacía.

## Batería / datos móviles

- El socket WS (`useRealtimeSocket.ts`) se desconecta cuando la app pasa a
  background (`AppState`) y reconecta al volver a foreground — evita
  mantener el radio de red despierto sin necesidad.
- Reconexión con backoff exponencial (2s → 30s tope) en vez de reintento
  agresivo en loop.
- Mientras en background, las alertas llegan por push (no por WS).

## Enmascarado de device IDs

El backend ya enmascara `device_id`/nombre para usuarios no-admin, tanto
en REST (`vehicles_handlers.go`) como en WS (`ws/hub.go:renderForRole`) —
el mobile nunca recibe el id completo salvo que el usuario sea admin.
`src/utils/maskDeviceId.ts` es un port 1:1 de
`internal/vehicles/mask.go` usado como defensa en profundidad, no como
mecanismo principal.

## Pendiente / fuera de este scaffold

- Endpoint `POST /devices/register` en el backend Go + envío real de push
  vía Firebase Admin SDK.
- Refresh token / rotación de JWT (hoy el token dura 24h y no hay
  refresh; al vencer, `useAuth` trata el fallo de `/auth/me` como
  no-logueado).
- Gráficos más ricos (recharts/victory-native) — se usa un sparkline
  casero en `VehicleDetailScreen` para no traer una dependencia de
  charting nativo en el scaffold inicial.
