## ADDED Requirements

### Requirement: Fast checks run on every contribution

`metahub-admin-ui` SHALL run a fast check suite on every pull request and every push to
the default branch. The suite SHALL NOT require a container runtime, a database
or any project secret.

The suite SHALL include:

- Go: `go build ./...`, `go vet ./...`, `golangci-lint`, `go test -short ./...`
- Node: `npm ci --ignore-scripts`, `npm run build`, `npm run lint`, `npm test`
- `npm audit --omit=dev --audit-level=high`
- `gitleaks` with the committed `.gitleaks.toml`
- DCO trailer check
- `go-licenses` plus `license-checker` allowlists and generated `THIRD_PARTY_NOTICES.md`
- `govulncheck` (advisory while the dependency baseline is cleared)
- outbound-host allowlist

#### Scenario: Pull request from a fork

- **WHEN** a pull request is opened from a fork
- **THEN** the fast suite runs with a read-only token
- **AND** every job completes without reading a secret
- **AND** the result is reported on the pull request

#### Scenario: Push to the default branch

- **WHEN** a commit lands on the default branch
- **THEN** the same fast suite runs and the `fast` aggregator reports it

### Requirement: A single aggregator check represents the fast lane

The repository SHALL expose an aggregator job named `fast` that depends on every
fast job and succeeds only if all of them succeed. Only this aggregator SHALL be
marked as a required status check at the visibility flip.

#### Scenario: One fast job fails

- **WHEN** any fast job fails or is cancelled
- **THEN** the `fast` aggregator fails

### Requirement: Public workflows are fork-safe by construction

Pull-request-triggered workflows SHALL NOT reference secrets, SHALL declare
`permissions` explicitly with `contents: read` at most, SHALL NOT use
`pull_request_target`, and SHALL pin every third-party action to a full commit
SHA.

#### Scenario: A check needs a private resource

- **WHEN** a fast check would need a project secret or special repository access
- **THEN** it is redesigned, moved to the integration lane, or dropped

### Requirement: The full suite also runs at release time

The repository currently has no container-backed tests, so the fast lane runs
the whole Go suite with `go test -short`. A slower integration workflow SHALL
run the full suite on release tags and on `workflow_dispatch`, with read-only
permissions and no secrets. A future test that starts containers SHALL follow
the same rule and exit early under `testing.Short()`.

#### Scenario: Release tag is pushed

- **WHEN** a release tag is pushed
- **THEN** the integration workflow runs the full Go suite; no container
  runtime is required today

### Requirement: Secret scanning and DCO are enforced

Secret scanning SHALL use the committed `.gitleaks.toml`; an unreviewed finding
SHALL fail the job. Every commit in a pull request SHALL carry a
`Signed-off-by` trailer and the DCO job SHALL fail when one is missing.

#### Scenario: New secret in a change

- **WHEN** a change introduces a value that gitleaks reports and the path is not
  allowlisted
- **THEN** the secret-scanning job fails

#### Scenario: Commit without sign-off

- **WHEN** a pull request contains a commit without `Signed-off-by`
- **THEN** the DCO job fails and names the commit

### Requirement: Dependency licences and vulnerabilities are checked

The repository SHALL commit a licence allowlist and a generated
`THIRD_PARTY_NOTICES.md`; an unlisted licence or notices drift SHALL fail the
job. Repositories with npm dependencies SHALL run `npm audit --omit=dev` at the
committed severity threshold. Go repositories SHALL run `govulncheck`
(advisory until the current dependency baseline is cleared).

#### Scenario: Unlisted licence

- **WHEN** a dependency under a licence outside the allowlist is added
- **THEN** the licence job fails and names the dependency

### Requirement: Outbound hosts are allowlisted

Non-test source SHALL reference only hosts on `.github/outbound-allowlist.txt`.
A reference to any other host SHALL fail the job.

#### Scenario: Analytics endpoint added

- **WHEN** a change adds code that contacts a host not on the allowlist
- **THEN** the outbound-host job fails and names the host
