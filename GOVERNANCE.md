# Governance

## Model

MetaHub Registry is currently a **benevolent-dictator** project: Christian
Kniep owns the direction, the roadmap, and the release process, and is the
final decision-maker when consensus is not reached.

This is an honest description of a solo project, not a permanent structure.
The path to shared maintainership is described below.

## Decisions

- Day-to-day changes are decided by lazy consensus on the issue or pull
  request.
- When consensus is not reached, the maintainer decides and explains the
  reasoning in the thread.
- Changes to this document require a pull request and the maintainer's
  approval.

## Becoming a maintainer

Contributors can earn maintainer status. The process is deliberately
lightweight:

1. Pick an area: documentation, tests, the CLI, the UI, a storage backend, or
   a subsystem you already work on.
2. Contribute consistently: at least three merged pull requests in that area,
   with tests and reviews addressed.
3. Demonstrate judgement: security awareness, respect for the code of conduct,
   and willingness to say "this needs a design discussion".
4. Open an issue titled `maintainer: <name>` describing the area and the
   evidence. The maintainer responds within 30 days.
5. On approval, you receive write access to the relevant repository and are
   listed in this document.

Maintainers are expected to review pull requests in their area, triage issues,
and follow the security policy. Maintainer status lapses after 12 months of
inactivity, or can be removed for code-of-conduct violations. Removal is
recorded in the same issue tracker.

## Releases

The maintainer acts as release manager. Release policy:

- Semantic versioning per repository.
- A target of roughly monthly minor releases; no fixed dates are promised.
- The current and previous minor versions are supported.
- Security fixes are released as soon as possible.
- The 1.0 stability boundary is documented in the release notes: OCI
  endpoints, `/api/v1`, the profile schema, and CLI commands are stable;
  model repackaging, Waldur integration, inventory resolution, and storage
  backends are experimental.

## Licensing and contribution terms

The project is Apache-2.0. Contributions use the Developer Certificate of
Origin; there is **no CLA**. As a consequence, community contributions cannot
be relicensed or folded into a proprietary edition. This is a deliberate
choice: the project optimises for adoption, not for a future closed product.

A support or hosting offering may be provided by the maintainer; it sells
services, not a proprietary fork of the code.

## Trademark

The name and logo are governed by [TRADEMARK.md](TRADEMARK.md), not by the
source licence.

## Code of conduct

Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Report
concerns to `metahub@qnib.org`.

## Changes to governance

If the project gains multiple active maintainers, this document is expected to
move to a maintainer-council model. Any such change is made by pull request and
recorded here.
