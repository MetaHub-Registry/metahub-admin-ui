#!/usr/bin/env bash
# Print the npm third-party notices in Markdown. CI regenerates this file and
# fails when the committed copy differs.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT}"

cat <<'HEADER'
# Third-party notices

Generated from the installed npm tree with `license-checker`. Do not edit by
hand: run `.github/scripts/third-party-notices.sh > THIRD_PARTY_NOTICES.md`.

| Package | Licence | Source |
|---|---|---|
HEADER

npm exec --yes license-checker@25.0.1 -- --json --excludePrivatePackages 2>/dev/null |
  python3 -c '
import json
import sys

packages = json.load(sys.stdin)
for name in sorted(packages):
    meta = packages[name]
    source = meta.get("repository") or meta.get("url") or ""
    if isinstance(source, dict):
        source = source.get("url", "")
    source = source.removesuffix(".git")
    lic = meta.get("licenses", "")
    print("| `%s` | %s | %s |" % (name, lic, source))
'
