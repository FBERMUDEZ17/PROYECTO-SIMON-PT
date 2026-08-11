# SETUP.md — Guía de despliegue local

Requisitos previos: **Go 1.26+**, **Node.js 20+** (usado con 24.x), **npm**. No hace falta Docker, Postgres ni ninguna infraestructura externa — el backend usa SQLite embebido.

## 1. Backend (Go)

Desde la raíz del repo:

```bash
JWT_SECRET=un-secreto-cualquiera go run ./cmd/server
```

`JWT_SECRET` es **obligatorio** (el servidor no arranca sin él, no tiene default por seguridad). Por defecto:

| Variable | Default | Qué hace |
|---|---|---|
| `ADDR` | `:8080` | Puerto de escucha |
| `DB_PATH` | `data/app.db` | Archivo SQLite (usar `:memory:` para pruebas descartables) |
| `FRONTEND_ORIGIN` | `http://localhost:3000` | Orígenes permitidos por CORS (coma-separado) |
| `COOKIE_SECURE` | `false` | Poner `true` solo detrás de HTTPS |

Verificación rápida (con el server corriendo):

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"correct-pw","name":"Demo"}'
```

Debería devolver `{"user": {...}, "token": "..."}`.

### Correr los tests del backend

```bash
go build ./...        # compila todo
go test ./...          # corre los 95 tests (unit + integración)
go test ./internal/sensors/... -v    # ej.: solo la predicción de combustible
go test ./internal/auth/... -v       # ej.: solo autenticación
```

Todos usan SQLite `:memory:` — no dejan archivos ni requieren limpieza.

## 2. Frontend web (Next.js)

Con el backend ya corriendo en otra terminal:

```bash
cd web
npm install
cp .env.local.example .env.local   # si no existe, crear con la línea de abajo
# .env.local debe tener: NEXT_PUBLIC_API_URL=http://localhost:8080
npm run dev
```

Abrir `http://localhost:3000`. El login/registro que hagas ahí pega directo contra el backend Go (cookie httpOnly, `credentials: "include"`).

## 3. App mobile (React Native / Expo)

### Opción A — Instalar el APK ya compilado (recomendado para evaluar rápido)

No requiere Node, Android Studio ni nada instalado. Bajar e instalar directamente en un Android:

**📦 [Descargar APK](https://expo.dev/artifacts/eas/qqr9G7K0Zz4jISvIwE50tIQonjxbQ49xtX1-U1-3-gM.apk)**

Al abrir la app por primera vez, tocar **"⚙ Servidor"** en la pantalla de login y poner la URL donde corre el backend:
- Mismo Wi-Fi que tu backend: `http://<IP-LAN-de-tu-PC>:8080` (ej. `http://192.168.1.56:8080`)
- Emulador Android en la misma PC que el backend: `http://10.0.2.2:8080`

> El APK **no** funciona con `localhost:8080` porque "localhost" en el teléfono es el teléfono mismo, no tu PC.

### Opción B — Correr desde código fuente (Expo Go)

Expo Go **no soporta `react-native-maps`** (requiere módulos nativos que Expo Go no trae) — el mapa de detalle del vehículo no va a cargar ahí. Sirve igual para revisar el resto de la app (login, listado, alertas, notificaciones locales).

```bash
cd mobile
npm install
# opcional: crear .env con EXPO_PUBLIC_API_URL / EXPO_PUBLIC_WS_URL apuntando a tu backend
npx expo start
```

Escanear el QR con la app **Expo Go** (Android/iOS), en la misma red Wi-Fi que la PC.

### Opción C — Generar tu propio APK (EAS Build)

```bash
cd mobile
npx eas-cli login          # cuenta gratuita en expo.dev
npx eas-cli build --platform android --profile preview
```

El perfil `preview` (`mobile/eas.json`) genera un `.apk` instalable directo (no un `.aab` de Play Store). El `.easignore` en la raíz del repo excluye `node_modules` del archivo subido — sin eso, la subida sube de ~1MB a ~400MB+.

## 4. Verificación rápida end-to-end

1. Backend corriendo (`go run ./cmd/server`).
2. `POST /sensors/data` con un token de un usuario logueado → crea un vehículo nuevo y su primera lectura.
3. Abrir el dashboard web (o la app mobile) logueado con ese mismo usuario → el vehículo debe aparecer con la lectura, y actualizarse en vivo por WebSocket si mandás una segunda lectura.
4. Mandar una lectura con `fuel_level` bajo y una lectura previa reciente → debería dispararse una alerta de combustible bajo, visible en el panel de alertas.

## Problemas comunes

| Síntoma | Causa | Solución |
|---|---|---|
| `JWT_SECRET debe estar definido` al arrancar | Falta la env var | `JWT_SECRET=algo go run ./cmd/server` |
| Web no puede loguear (CORS) | `FRONTEND_ORIGIN` no incluye tu origen | Setear `FRONTEND_ORIGIN=http://localhost:3000` (o el que uses) |
| Mobile no conecta al backend | Apuntando a `localhost` desde un dispositivo físico/emulador | Usar la IP LAN de la PC (dispositivo físico) o `10.0.2.2` (emulador Android) |
| Expo Go dice "Something went wrong" | `react-native-maps` no soportado en Expo Go | Usar el APK standalone (Opción A) |
| Expo Go dice "incompatible SDK version" | Tu Expo Go instalado no coincide con la SDK del proyecto (`mobile/app.json` → `expo.sdkVersion` implícito por la versión de `expo` en `package.json`, hoy 54) | Actualizar Expo Go desde la store, o bajar la SDK del proyecto para que coincida |
