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
  node -e '
const fs = require("fs");
const packages = JSON.parse(fs.readFileSync(0, "utf8"));
for (const name of Object.keys(packages).sort()) {
  const meta = packages[name];
  let source = meta.repository || meta.url || "";
  if (typeof source === "object") source = source.url || "";
  source = String(source).replace(/\.git$/, "");
  const lic = meta.licenses || "";
  console.log("| `" + name + "` | " + lic + " | " + source + " |");
}
'
