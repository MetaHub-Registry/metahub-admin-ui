# Context: metahub-admin-ui (worktree)

## What changed

Session change `add-admin-ui-inventory-service` — **skeleton phase**
(per-repo change `add-inventory-service`, tasks 1.1–1.3):

- Go module scaffolded (`main.go`, `internal/cmd` cobra, `internal/server`
  chi) with `GET /api/v1/health` and a stub `GET /api/v1/resolve`
  (spec-compliant `404 {}` no-match; 400 on missing/invalid ip).
- SPA scaffolded at repo root (Vite + React 19 + Tailwind 4 + react-query +
  react-router; vitest/eslint configs mirrored from metahub-ui), placeholder
  page, smoke test, embedded via `//go:embed all:dist`.
- Repo infra: `test.sh` (Go + UI coverage), `sonar-project.properties`,
  `misc/bin/sonar.sh`, `.bumpversion.cfg` (bumps Go version + package.json),
  multi-stage `Dockerfile` (node → go, `CGO_ENABLED=0`, no SSH mount),
  `docker-compose.yaml` (host 8083 → container 8080), `.dockerignore`,
  `.gitignore`, README/AGENTS.

## Key files

- `internal/server/server.go` — routes: `/api/v1/health`, `/api/v1/resolve`
  (stub), SPA mount. Contract: resolve path never 5xx.
- `assets.go` — the `//go:embed all:dist` seam between Go and SPA.
- `openspec/changes/add-inventory-service/` — tasks 2.x (engine, config,
  endpoint wiring) and 3.x (deployment) still open.

## Gotchas

- `npm run build` overwrites the committed `dist/index.html` placeholder →
  shows as git diff; `git restore dist/index.html` (or ignore).
- `.env` (sonar credentials) not yet created — copy pattern from metahub-api.
- chi-middleware/logrus-logger v0.2.0 API is
  `logger.Logger(name string, log *logrus.Entry)` (name first, Entry, not
  *logrus.Logger) — easy to trip over.
- `dist/` must exist at `go build` time (embed); test.sh guards it.

## Open questions / next

- Task 2.1 config loading (viper? house style `CFG_FILE`?) — decide when
  implementing the `inventory` block.
- flux/stack render line + compose-on-internal-network wiring (task 3.1)
  needs the session-level `task render` slot `ADMINUI` (already reserved in
  the session Taskfile).
