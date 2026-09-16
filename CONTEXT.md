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
- Task 3.2 (seed example inventory config for the dev cluster) — config
  volume/Secret wiring for the flux deployment still open.

## Deployment wiring (2026-09-16)

- **flux/metahub**: `dev/metahub-admin-ui/` (deployment :8080 + ClusterIP
  service), Flux Kustomization `metahub-infra-dev-metahub-admin-ui`
  (root kustomization Wave 3-dev), HTTPRoute
  `adminui.dev.example.com → metahub-admin-ui:80`.
- **stack**: `tmpl/metahub-admin-ui.tpl` — Test (docker ui+go targets) →
  Image (**amd64/arm64 fan-out** on arch-matched agents via the repo's
  `misc/gocd/bin/build-image.sh`) → Manifest (`manifest-image.sh`, plain tag)
  → Deploy/GitSync (patches the flux deployment.yaml, reads
  `metahub.image.full`) → AwaitRollout. **Deployed & verified**: pod 1/1 on
  arm64 node, health + resolve 404 checked in-cluster.
- **Dockerfile gotcha (fixed)**: the go stage is `$BUILDPLATFORM`-pinned, so
  `ARG TARGETOS TARGETARCH` → `GOOS/GOARCH` is MANDATORY — without it both
  manifest variants contain an amd64 binary (exec format error on arm64).
  House rule: multi-arch via per-arch job fan-out, not single buildx jobs.
