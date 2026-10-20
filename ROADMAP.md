# Roadmap

This is a direction, not a commitment. There are no dates. Items move between
buckets as reality changes.

## Now

- 1.0 hardening: tests, error paths, upgrade notes.
- Documentation: quickstart, architecture, trust model, `/api/v1` reference.
- Offline first-run demo.
- Minimal Helm chart.
- Natural-key identifiers in the API (replacing storage element IDs).
- Neo4j 5 LTS as the supported reference backend.

## Next

- Storage boundary: domain repository interfaces instead of the current
  213-method `Database` interface.
- An embedded, permissively licensed graph backend (candidate: Kuzu, MIT) as
  the default for the quickstart.
- OpenAPI 3.1 generated from the handlers.
- A design for signature-based verification of derived artefacts.
- Finish `metahub-admin-ui` (inventory resolver engine, UI, seed content).

## Later

- Postgres + Apache AGE as a scale-out backend.
- A read-only public demo instance, if traffic justifies it.
- An operator / CRD for Kubernetes.
- Waldur billing adapter, once it works against a supported core.
- A support offering.

## Not planned

- Changing OCI tag-to-digest semantics for standard clients.
- A proprietary edition containing community code (no CLA).
- Telemetry or phone-home.
