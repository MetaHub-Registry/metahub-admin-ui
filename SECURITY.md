# Security policy

## Supported versions

The latest two minor releases receive security fixes. Older releases may be
patched at the maintainer's discretion; upgrade to a supported release before
reporting.

## Reporting a vulnerability

**Do not open a public issue.**

Use either channel:

1. **GitHub private vulnerability reporting** on the affected repository
   (Security → Report a vulnerability). This is preferred because it gives us
   a structured thread and a coordinated disclosure path.
2. **Email:** `security@metahub-registry.org`.

Please include:

- affected repository and version or commit,
- a description of the issue and its impact,
- reproduction steps or a proof of concept,
- any suggested fix or mitigation,
- whether you intend to publish, and on what timeline.

## What to expect

- Acknowledgement within **5 business days**.
- An initial assessment and a proposed timeline within **10 business days**.
- A default **90-day embargo** from acknowledgement before public disclosure,
  shortened by agreement if a fix ships sooner. We will credit reporters who
  want credit.
- A GitHub Security Advisory and a release note when a fix ships.

These are targets, not a contractual SLA. The project is maintained by one
person on a best-effort basis.

## Scope

In scope:

- the registry API and OCI distribution endpoints,
- authentication and token handling,
- profile resolution and manifest manipulation,
- the CLI and the web UIs,
- the documented deployment paths (compose, Helm).

Out of scope:

- vulnerabilities in third-party dependencies — report them upstream (we will
  still take a dependency bump),
- issues that require a compromised host or cluster admin,
- findings from automated scanners without a demonstrated impact.

## No bug bounty

There is no paid bounty. We will credit reporters in the advisory unless you
ask us not to.
