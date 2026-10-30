# Tasks — public CI for metahub-admin-ui

## 1. Fast lane

- [x] 1.1 Add `.github/workflows/ci.yml` with `push`/`pull_request` triggers,
      `permissions: contents: read`, concurrency cancellation and a `fast`
      aggregator.
- [x] 1.2 Add the build, static-analysis, lint and container-free test jobs.
- [x] 1.3 Add gitleaks (pinned CLI, committed config), DCO and the outbound-host
      allowlist. Add the Go and npm licence allowlists and generated
      `THIRD_PARTY_NOTICES.md`.
- [x] 1.4 Add `.github/dependabot.yml`.

## 2. Integration lane

- [x] 2.1 Add `integration.yml` for the full Go suite on release tags and on
      demand; no container runtime is needed.

## 3. Follow-ups (open-sourcing gate)

- [ ] 3.1 Clear the `govulncheck` baseline and make the job blocking.
      Scan on 2026-10-30: **23 reachable findings, all in the Go standard
      library** (no module findings); they close with the Go patch that
      `1.26.x` resolves to in CI, while local `go1.26.0` still reports them.
      Re-scan on the CI Go patch before making the job blocking.
- [x] 3.2 `npm audit --omit=dev --audit-level=high` green — verified
      2026-10-30: 0 vulnerabilities. Keep it green as dependencies move.
- [ ] 3.3 Align the module floor with CI: `go.mod` declares `go 1.24.0` while
      the fast lane pins `go 1.26.x`, which violates `go-toolchain-policy`
      (major.minor must match). Raise the directive to `go 1.26.0` (preferred,
      matching the other services) or pin the lane to `1.24.x`.
- [ ] 3.4 Run the fast lane from a fork pull request before the visibility flip
      and record the run URL here.
