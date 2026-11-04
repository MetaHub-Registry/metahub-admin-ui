# metahub-admin-ui

The **ops-facing admin service** of the MetaHub stack. metahub-ui is
end-user-facing and metahub-admin is CLI-only — this repo hosts the small
long-running services operators need, starting with the **inventory service**:
the single defined surface behind which the entire profile-resolution chain
lives (exact-IP → hostname regex via DNS PTR → CIDR longest-prefix).

metahub-api consumes this surface at token-mint time
(`GET /api/v1/resolve?ip=`) and fail-opens to `generic` on any non-200 —
this service **never 5xxes the resolve path**.

Architecture: one static Go binary serving the versioned JSON API
(`/api/v1/*`, chi + logrus) **and** the embedded Vite SPA
(`go:embed all:dist`). v0 is read-only from YAML config; growth path:
SQLite store → edit UI → Netbox sync — each step invisible to consumers.

```
┌────────────┐  GET /auth (no profile suffix)   ┌──────────────────────┐
│ metahub-api│ ──── GET /api/v1/resolve?ip=X ──► │  metahub-admin-ui    │
│ (consumer) │ ◄── 200 {profile} / 404 {} ────── │  resolve chain + SPA │
└────────────┘        fail-open ⇒ generic          └──────────────────────┘
```

## Getting started

```bash
# SPA
npm ci
npm run build        # writes dist/ — overwrites the committed placeholder

# Go service
go run .             # serves SPA + API on :8080 (METAHUB_ADMIN_UI_PORT)
go run . version

# Dev UI (proxies /api to :8080)
npm run dev
```

> **Gotcha:** `npm run build` overwrites the committed
> `dist/index.html` placeholder (kept so `go:embed` compiles on a fresh
> clone). `git restore dist/index.html` if it shows up as modified.
> Tests: `./test.sh` recreates the placeholder when `dist/` is empty.

## Tests & quality

```bash
./test.sh            # Go tests + coverage (gocover-cobertura/gocoverstats)
                     # + vitest coverage when node_modules exists
```

## Versioning

`.bumpversion.cfg` bumps `internal/cmd/version.go` **and** `package.json`
in lockstep.
