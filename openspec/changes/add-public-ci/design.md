# Design — public CI for metahub-admin-ui

## Stack and boundary

Go 1.24.0 declared in `go.mod` (embedded SPA) and TypeScript/Vite. The fast
lane pins `go 1.26.x`, so closing task 3.3 must align the directive with the
lane.

Fast checks:

- Go: `go build ./...`, `go vet ./...`, `golangci-lint`, `go test -short ./...`
- Node: `npm ci --ignore-scripts`, `npm run build`, `npm run lint`, `npm test`
- `npm audit --omit=dev --audit-level=high`
- `gitleaks` with the committed `.gitleaks.toml`
- DCO trailer check
- `go-licenses` plus `license-checker` allowlists and generated `THIRD_PARTY_NOTICES.md`
- `govulncheck` (advisory while the dependency baseline is cleared)
- outbound-host allowlist

Container-backed checks: The Go tests do not start containers, so the fast lane runs the whole Go suite with `-short`.

The committed `dist/index.html` placeholder keeps `go:embed all:dist` compiling on a fresh clone; CI's `npm run build` overwrites it inside the runner only.

## Decisions

### Self-contained, fork-safe workflows

The workflows live in this repository. A cross-repository reusable workflow
would need either a public `.github` repository before the launch gate or
private access that a fork does not receive. Every PR-triggered workflow is
therefore self-contained, declares `permissions: contents: read`, pins
third-party actions by SHA and references no secret.

### One aggregator

`fast` depends on every fast job and fails when any of them fails or is
cancelled. Marking only `fast` as a required check at the visibility flip keeps
the branch-protection configuration stable as checks are added.

### No container-backed tests

No test starts a container, so the fast lane runs the whole suite with
`-short` and no service dependency. `integration.yml` still runs the full Go
suite on release tags and on `workflow_dispatch` as a slower safety net, with
no secrets.

### Dependencies and licences

The licence check fails on a dependency outside the allowlist and the notices
file is regenerated in CI; a drift fails the job. `npm audit --omit=dev
--audit-level=high` is blocking for the production graph and returned
0 vulnerabilities on 2026-10-30. `govulncheck` is advisory
(`continue-on-error: true`): the 2026-10-30 scan found 23 reachable findings,
all in the Go standard library, with no module findings. They close with the
Go patch `1.26.x` resolves to in CI; the job still runs on every pull request.

### Outbound hosts

Non-test source may reference only hosts on `.github/outbound-allowlist.txt`.
A new host fails the job and must be added deliberately in review. This is a
review gate, not a network sandbox.

## Risks

| Risk | Mitigation |
|---|---|
| A workflow needs a secret when run from a fork | it belongs in the integration lane or is dropped |
| Action supply chain | every action is pinned to a full commit SHA |
| Dependency advisories | Dependabot opens weekly Go and npm PRs; `npm audit --omit=dev` is blocking, `govulncheck` is advisory until the stdlib findings clear |
