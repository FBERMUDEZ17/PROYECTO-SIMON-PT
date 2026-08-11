# PROYECTO-SIMON-PT
Backend Go que recibe telemetría de vehículos (GPS, combustible, temperatura, velocidad), predice autonomía de combustible, dispara alertas, y transmite todo en vivo por WebSocket a un dashboard  web (Next.js) y una app mobile (React Native / Expo).

# Simon PT — Plataforma de telemetría vehicular (IoT)

Backend Go que recibe telemetría de vehículos (GPS, combustible, temperatura, velocidad), predice autonomía de combustible, dispara alertas, y transmite todo en vivo por WebSocket a un dashboard web (Next.js) y una app mobile (React Native / Expo).

- 📄 **[DESIGN.md](./DESIGN.md)** — stack elegido y trade-offs técnicos.
- 🛠️ **[SETUP.md](./SETUP.md)** — cómo levantar todo localmente (backend, web, mobile).
- 📱 **[Descargar APK Android](https://expo.dev/artifacts/eas/qqr9G7K0Zz4jISvIwE50tIQonjxbQ49xtX1-U1-3-gM.apk)** — instalable directo, sin Expo Go.

## Estructura del repo

```
cmd/server/       # entrypoint del backend Go
internal/         # lógica de negocio del backend (auth, sensores, alertas, ws, etc.)
web/               # dashboard Next.js
mobile/            # app React Native / Expo
DESIGN.md          # decisiones de arquitectura y trade-offs
SETUP.md           # guía de despliegue local
```

## Quickstart

```bash
JWT_SECRET=devsecret go run ./cmd/server   # backend en :8080
go test ./...                                # 95 tests, backend
```

Ver [SETUP.md](./SETUP.md) para el resto (web, mobile, verificación end-to-end).

## Tests

```bash
go test ./...
```

95 tests automatizados en el backend: unit tests de la lógica crítica (predicción de autonomía de combustible, validación de sensores, enmascarado de device IDs, autenticación/JWT) y tests de integración end-to-end (auth, ownership de vehículos, WebSockets, password reset, registro de push tokens) contra un router real con SQLite en memoria.
