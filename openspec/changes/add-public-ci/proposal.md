## Why

The repository is canonical on private GitHub and is about to become public.
Branch protection and required status checks are unavailable on private
repositories under the current plan, so the fast checks must exist, be green
and work from a fork **before** the visibility flip. Today all building happens
in GoCD, which contributors cannot see or run.

## What Changes

Add a public, fork-safe fast lane to `metahub-admin-ui`. The Go tests are
container-free, so the fast lane runs them directly; a separate integration
workflow remains as a slower release-time safety net.

- Go: `go build ./...`, `go vet ./...`, `golangci-lint`, `go test -short ./...`
- Node: `npm ci --ignore-scripts`, `npm run build`, `npm run lint`, `npm test`
- `npm audit --omit=dev --audit-level=high`
- `gitleaks` with the committed `.gitleaks.toml`
- DCO trailer check
- `go-licenses` plus `license-checker` allowlists and generated `THIRD_PARTY_NOTICES.md`
- `govulncheck` (advisory while the dependency baseline is cleared)
- outbound-host allowlist

An aggregator job named `fast` depends on every fast job and is the single
check to mark required at the visibility flip. Every workflow declares
`contents: read`, pins third-party actions by full commit SHA, uses no secrets
and avoids `pull_request_target`.

## Capabilities

### New Capabilities

- `public-ci`: the fast checks, fork-safety rules and aggregator for `metahub-admin-ui`.

### Modified Capabilities

None.

## Impact

- New `.github/workflows/`, `.github/scripts/`, allowlists and
  `THIRD_PARTY_NOTICES.md`; `.github/dependabot.yml` for dependency updates.
- Test-only changes where the container boundary had to be expressed with
  `testing.Short()`.
- No runtime behaviour, API, image or GoCD pipeline change.
- Part of the session lane `oss-public-ci`; the visibility flip is gated on this
  lane being green.
