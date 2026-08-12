# Simon PT — Plataforma de telemetría vehicular (IoT)

Simon PT recibe telemetría en vivo de una flota de vehículos (GPS, nivel de combustible, temperatura, velocidad), predice cuántas horas de autonomía le quedan a cada uno, dispara alertas cuando el combustible está por agotarse, y transmite todo en tiempo real por WebSocket a un **dashboard web** (Next.js) y una **app mobile** (React Native / Expo) — ambos consumen la misma API, sin lógica de negocio duplicada.

Incluye autenticación con roles (`admin` / `user`): un admin ve la flota completa, crea vehículos y ve las alertas predictivas; un usuario común solo ve sus propios vehículos, con el identificador de dispositivo parcialmente enmascarado.

- 📄 **[DESIGN.md](./DESIGN.md)** — decisiones de arquitectura y trade-offs técnicos.
- 🛠️ **[SETUP.md](./SETUP.md)** — guía paso a paso para levantar todo localmente (backend, web, mobile) y solución de problemas comunes.
- 📱 **[Descargar APK Android](https://raw.githubusercontent.com/FBERMUDEZ17/PROYECTO-SIMON-PT/main/mobile/builds/simon-pt-preview.apk)** — instalable directo, sin necesitar Expo Go.

## Cuentas de prueba

El backend crea automáticamente estas dos cuentas la primera vez que arranca (`seedDemoUsers` en `cmd/server/main.go`, idempotente — no duplica nada en corridas siguientes), para poder evaluar el rol admin sin tener que registrarse ni editar la base de datos a mano:

| Rol | Email | Contraseña |
|---|---|---|
| **Admin** | `admin@simonpt.dev` | `Admin123!` |
| **Usuario** | `user@simonpt.dev` | `User123!` |

Sirven tanto en el dashboard web como en la app mobile (misma API). Para desactivar este sembrado (p.ej. si se apunta el binario a una base de datos real, no de demo), arrancar con `SEED_DEMO_USERS=false`.

## Funcionalidades principales

- **Ingesta de telemetría**: `POST /sensors/data` valida y persiste cada lectura (GPS, combustible, temperatura, velocidad).
- **Predicción de autonomía**: regresión lineal por mínimos cuadrados sobre las últimas lecturas de combustible para estimar horas restantes.
- **Alertas de combustible bajo**: se disparan automáticamente por debajo de un umbral, deduplicadas en una ventana de 15 minutos.
- **Tiempo real por WebSocket**: cada lectura/alerta nueva se transmite en vivo a todos los clientes conectados (web y mobile), sin polling.
- **Autenticación JWT** (cookie `httpOnly` en web, token persistido en el dispositivo para mobile) con roles `admin`/`user`.
- **Recuperación de contraseña** (flujo de dos pasos, token de un solo uso).
- **Enmascarado de `device_id`** para usuarios no-admin (`DEV-1234-XC54` → `DEV-****-XC54`), tanto en REST como en WebSocket.
- **Dashboard web**: mapa en vivo, lista/búsqueda de vehículos, gráficos históricos, panel de alertas (solo admin).
- **App mobile**: mismas funcionalidades adaptadas a móvil — gráficas de línea de tendencia explorables (arrastrar el dedo muestra valor + fecha/hora, con marcadores de cambio de día), ubicación vía la app de mapas del sistema, notificaciones push, cache offline-first.

## Stack tecnológico

| Componente | Ubicación | Stack |
|---|---|---|
| **Backend / API** | `cmd/`, `internal/` | Go 1.26, [chi](https://github.com/go-chi/chi) (router), SQLite embebido vía `modernc.org/sqlite` (puro Go, sin cgo), WebSockets ([gorilla/websocket](https://github.com/gorilla/websocket)), JWT + bcrypt (`golang.org/x/crypto`) |
| **Dashboard web** | `web/` | Next.js 16 (App Router), React 19, TanStack Query, Tailwind CSS 4, maplibre-gl (mapa), Recharts (gráficos), `idb` (IndexedDB, cache offline) |
| **App mobile** | `mobile/` | React Native 0.81 + Expo SDK 54, React Navigation, `react-native-svg` (gráficas de línea táctiles), `expo-sqlite` (cache offline), `expo-notifications` (push), `@react-native-async-storage/async-storage` |
| **Tests** | — | `go test` (95 tests: unit + integración end-to-end contra SQLite en memoria) |
| **Distribución mobile** | `mobile/eas.json` | EAS Build (perfil `preview`) → APK standalone instalable sin Expo Go |

Ver [DESIGN.md](./DESIGN.md) para el razonamiento detrás de cada elección (por qué Go, por qué SQLite embebido, por qué WebSockets en vez de polling, etc.).

## Estructura del repo

```
cmd/server/        # entrypoint del backend Go
internal/          # lógica de negocio del backend (auth, sensores, alertas, vehículos, ws, etc.)
web/                # dashboard Next.js
mobile/             # app React Native / Expo (incluye mobile/builds/ con el APK ya compilado)
DESIGN.md           # decisiones de arquitectura y trade-offs
SETUP.md            # guía de despliegue local paso a paso
```

## Requisitos de instalación

- **Go 1.26+**
- **Node.js 20+** (probado con 24.x) y **npm**
- Nada de infraestructura externa: sin Docker, sin Postgres — el backend usa SQLite embebido (un archivo local).
- Para compilar tu propio APK mobile (opcional, ya hay uno compilado en `mobile/builds/`): cuenta gratuita en [expo.dev](https://expo.dev) para usar EAS Build.

### Instalación rápida (backend)

```bash
git clone https://github.com/FBERMUDEZ17/PROYECTO-SIMON-PT.git
cd PROYECTO-SIMON-PT
JWT_SECRET=un-secreto-cualquiera go run ./cmd/server   # backend en :8080
```

Para el dashboard web y la app mobile, ver la guía completa (con todas las variables de entorno y las tres formas de correr mobile) en **[SETUP.md](./SETUP.md)**.

## Instrucciones de uso

Con el backend corriendo (`go run ./cmd/server`, puerto `8080` por defecto):

1. **Iniciar sesión** con una de las [cuentas de prueba](#cuentas-de-prueba) ya sembradas, o registrar una nueva:
   ```bash
   curl -X POST http://localhost:8080/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"admin@simonpt.dev","password":"Admin123!"}'
   ```
   o desde el dashboard web (`web/`, ver SETUP.md) / la app mobile (APK o `npx expo start`).

2. **Enviar una lectura de sensor** (crea el vehículo automáticamente si es la primera lectura para ese `vehicle_id`):
   ```bash
   curl -X POST http://localhost:8080/sensors/data \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer <token del paso 1>" \
     -d '{"vehicle_id":"DEV-1234-XC54","lat":-33.45,"lon":-70.66,"fuel_level":42.5,"temperature_c":88.2,"speed_kmh":75}'
   ```

3. **Ver el vehículo en vivo** — en el dashboard web (`http://localhost:3000` con el mismo usuario logueado) o en la app mobile: aparece con la última lectura y se actualiza en tiempo real por WebSocket al mandar lecturas nuevas.

4. **Provocar una alerta de combustible bajo** — mandar una lectura con `fuel_level` bajo tras una lectura previa reciente; la alerta aparece en el panel de alertas (visible solo para usuarios `admin`).

5. **Roles**: un usuario registrado por su cuenta se crea como `user` (ve solo sus propios vehículos, con el `device_id` parcialmente enmascarado). No hay endpoint HTTP para autopromoverse a `admin` — para eso está la cuenta admin ya sembrada (`admin@simonpt.dev`, ver [Cuentas de prueba](#cuentas-de-prueba)); promover otra cuenta es un cambio a nivel de base de datos/código (`auth.Service.SetRole`), documentado en SETUP.md/DESIGN.md.

Verificación end-to-end completa y solución de problemas: ver **[SETUP.md](./SETUP.md)**.

## Tests

```bash
go test ./...
```

95 tests automatizados en el backend: unit tests de la lógica crítica (predicción de autonomía de combustible, validación de sensores, enmascarado de device IDs, autenticación/JWT) y tests de integración end-to-end (auth, ownership de vehículos, WebSockets, password reset, registro de push tokens) contra un router real con SQLite en memoria.
