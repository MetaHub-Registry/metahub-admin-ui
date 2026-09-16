# Agent Instructions

## Status

New repo (session `hostname-profile-resolution`). **No codebase knowledge
graph yet** (`.understand-anything/` is absent) — the codebase is small
enough to read directly. Once material code exists, generate it with the
global `/understand` tool and commit the artifacts, as in metahub-api/ui.

## Layout & Conventions

- **Go module at repo root** — `main.go` → `internal/cmd` (cobra root +
  `version` command) → `internal/server` (chi router, logrus). House style
  copied from metahub-api; see `openspec/changes/add-inventory-service/design.md`.
- **SPA at repo root** — `index.html`, `src/`, `vite.config.ts` (Vite +
  React + Tailwind v4 + react-query + react-router; metahub-ui conventions).
- **Embedding** — `assets.go` (package main) declares `//go:embed all:dist`
  and hands the `fs.FS` to `internal/cmd.Execute`. The committed
  `dist/index.html` is a **placeholder**; `npm run build` overwrites it
  (shows as a git diff — restore it or ignore, see README). `test.sh`
  recreates it if `dist/` is gone.
- **Version** — single source `.bumpversion.cfg`, which rewrites
  `internal/cmd/version.go` (Go) and `package.json` (SPA) in lockstep.
- **Surface contract** — `/api/v1/` is additive-only; the resolve path never
  responds 5xx; 404 `{}` means no-match. Delta spec:
  `openspec/changes/add-inventory-service/specs/inventory-lookup/spec.md`.

## Workflows

- Tests + coverage: `./test.sh` (Go always; vitest when node_modules exists).
- Sonar: `source .env && ./misc/bin/sonar.sh`, then verify the quality gate
  via the SonarQube MCP server (project key `metahub-admin-ui`).
- Commit after each completed OpenSpec task; push only at session end.

## No `.env` yet

Sonar needs `SONAR_QUBE_URL` / `SONAR_QUBE_TOKEN` in `.env` (gitignored) —
copy the pattern from metahub-api's `.env` when first running `sonar.sh`.
