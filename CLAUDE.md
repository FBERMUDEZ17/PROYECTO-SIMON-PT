# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A vehicle telemetry/IoT dashboard: a Go backend ingests sensor readings (GPS, fuel, temperature) from vehicles, predicts fuel autonomy, raises low-fuel alerts, and streams updates over WebSockets to a Next.js frontend. Auth is JWT-based, issued by the Go backend and stored in an httpOnly cookie.

There is also a **legacy/parallel TS prototype** at the repo root (`src/db/`, root `package.json`) using `better-sqlite3` — a `users`-only schema, likely superseded by the Go backend's `internal/db`. Don't assume it's wired into anything; the real backend is the Go one under `cmd/` and `internal/`.

Comments throughout the Go code are tagged `// TAREA: ...` — these mark which piece of functionality (from the original build prompts) a file/function implements. Useful for tracing "why does this exist."

## Commands

### Backend (Go, from repo root)
```
go build ./...                        # compile
go test ./...                         # run all tests
go test ./internal/auth/...           # single package
go test ./internal/httpapi/ -run TestName   # single test
JWT_SECRET=devsecret go run ./cmd/server   # run the API server (JWT_SECRET is required, no default — server exits if unset)
```
Server defaults: listens on `:8080` (override `ADDR`), SQLite file at `data/app.db` (override `DB_PATH`, or use `:memory:`), CORS allows `http://localhost:3000` (override `FRONTEND_ORIGIN`, comma-separated), cookie `Secure` flag off unless `COOKIE_SECURE=true`.

### Frontend (Next.js, from `web/`)
```
npm run dev      # dev server (localhost:3000), expects backend on localhost:8080 (NEXT_PUBLIC_API_URL in web/.env.local)
npm run build
npm run lint
```
No frontend test suite is configured.

### Legacy root TS prototype (from repo root)
```
npm run db:init     # tsx src/db/index.ts — creates data/app.db with just a `users` table
npm run db:schema   # tsx src/db/print-schema.ts
npm run build        # tsc
```

## Architecture

### Backend layering (`internal/`)
Each domain is its own package with a `Store`/`Service` constructed in `cmd/server/main.go` and wired together via `httpapi.Deps` in `internal/httpapi/router.go`:

- `db` — opens SQLite (`modernc.org/sqlite`, pure Go, no cgo) and owns the full schema (`users`, `vehicles`, `sensor_readings`, `alerts`) as one `CREATE TABLE IF NOT EXISTS` block run on every `Open`.
- `auth` — `Service.Register`/`Login` (bcrypt hashing, email normalized to lowercase to dedupe), `GenerateToken`/`VerifyToken` for JWTs (24h TTL). Roles are `user`/`admin` (`auth.RoleUser`/`RoleAdmin`); `SetRole` exists only as a Go-level escape hatch, no HTTP endpoint promotes users to admin.
- `sensors` — `Store` (persistence for readings), `Validate` (input validation), `fuel_prediction.go` (least-squares linear regression over recent fuel-level readings to estimate hours of autonomy; `LowFuelAlert()` fires under `LowFuelThresholdHours` = 1h).
- `vehicles` — `Store` (vehicle catalog, auto-created via `EnsureExists` on first reading, owned by whichever user submitted that first reading — `owner_user_id`) and `MaskDeviceID`, which masks the middle segments of a device id (`DEV-1234-XC54` → `DEV-****-XC54`) for non-admin users. Non-admin users only see/operate on their own vehicles (`Store.List` filters by owner; `Vehicle.IsOwnedBy`); admin sees all. A non-owner posting sensor data for someone else's vehicle gets `ErrForbidden` (403); fetching someone else's vehicle by ID returns 404, not 403 (doesn't confirm existence).
- `alerts` — `Store`, dedupes alerts of the same type/vehicle within a 15-minute window.
- `telemetry` — orchestration layer (`Service.Ingest`): validates → persists reading → ensures vehicle exists → recomputes fuel prediction over the last 10 readings → conditionally creates a deduped low-fuel alert. Called from `POST /sensors/data`, then its result is broadcast over the WS hub.
- `ws` — `Hub` runs its own goroutine (started in `main.go`), fans out `Event{Type, VehicleID, Payload}` to connected clients over a buffered channel (drops events rather than blocking ingestion if a client is slow). `renderForRole` applies `vehicles.MaskDeviceID` per-connection based on the client's role, so admin and non-admin sockets see different payloads for the *same* broadcast.
- `httpapi` — chi router (`router.go`) wires public routes (`POST /auth/register|login|logout`, `POST /auth/forgot-password|reset-password`, `GET /ws`) and a protected group behind `RequireAuth` middleware (`GET /auth/me`, `POST /sensors/data`, `GET /vehicles`, `GET /vehicles/{id}`).

### Password reset
`auth.Service.RequestPasswordReset`/`ResetPassword` (`internal/auth/reset.go`) implement a two-step flow backed by the `password_resets` table (random 32-byte hex token, 30min TTL, single-use). No email provider is configured: `POST /auth/forgot-password` always responds with a generic 200 (doesn't reveal whether the email exists) and, if the user exists, `log.Printf`s the token server-side — that's how you retrieve it in dev. `POST /auth/reset-password` exchanges `{token, new_password}` for an updated `password_hash`. Frontend: `/forgot-password` and `/reset-password` pages, the second requiring the token to be pasted in manually (copied from the server log).

### Auth flow (cross-cutting)
The backend is the sole owner of the JWT: it's issued on register/login and set as an httpOnly cookie (`auth_token`, `internal/httpapi/cookie.go`), never handled by frontend JS directly. `tokenFromRequest` accepts either an `Authorization: Bearer` header (tests/API clients) or the cookie (browser) — same middleware path for both. `/ws` does its own auth (not behind the `RequireAuth` group) since WebSocket handshakes use header/query-token/cookie interchangeably.

On the frontend, `web/src/proxy.ts` (Next 16's renamed `middleware.ts`) only checks *cookie presence* at the edge to avoid a dashboard flash-then-redirect — it cannot validate the JWT (no secret) and isn't the real gate. Actual authorization always happens per-request against the Go backend. `web/src/lib/api.ts` is the single fetch wrapper; every call uses `credentials: "include"` so the cookie round-trips automatically between `localhost:3000` and `localhost:8080`.

### Role-based masking
Device IDs are masked for non-admin users in *two* independent places that must stay in sync: `internal/httpapi/vehicles_handlers.go` (REST responses) and `internal/ws/hub.go` (`renderForRole`, WS broadcasts) — both call `vehicles.MaskDeviceID` keyed off `claims.IsAdmin()`/`client.role`.

### Frontend structure (`web/src/`)
Next.js App Router. `hooks/useLiveVehicles.ts` + `useRealtimeEvents.ts` consume the WS stream (via `lib/api.ts`'s `wsURL()`) to keep vehicle state live; `hooks/useVehicles.ts`/`useAuth.ts` wrap TanStack Query around the REST endpoints. `lib/indexeddb.ts` persists data client-side (via `idb`) for offline/resilience. `components/map/VehicleMap.tsx` uses `maplibre-gl`; `components/charts/*` use `recharts`.

### Testing conventions (Go)
Integration tests (`internal/httpapi/*_integration_test.go`) spin up the full router against an in-memory SQLite DB via the shared `newTestDeps` helper in `testhelpers_test.go` — reuse that helper rather than re-deriving the wiring from `main.go`. Unit tests sit next to their package (`*_test.go`).
