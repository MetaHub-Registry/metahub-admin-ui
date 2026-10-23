#!/usr/bin/env bash
# Verify that every commit in a pull request carries a Signed-off-by trailer
# matching the commit author. Intended for the pull_request event.
set -euo pipefail

if [ "${GITHUB_EVENT_NAME:-}" != "pull_request" ]; then
  echo "not a pull request; skipping DCO"
  exit 0
fi

base="${DCO_BASE_SHA:?DCO_BASE_SHA is required}"
head="${DCO_HEAD_SHA:?DCO_HEAD_SHA is required}"

status=0
for commit in $(git rev-list --no-merges "${base}..${head}"); do
  author="$(git log -1 --format='%ae' "${commit}")"
  if ! git log -1 --format='%B' "${commit}" | grep -qiE "^Signed-off-by: .+ <${author}>$"; then
    subject="$(git log -1 --format='%s' "${commit}")"
    echo "::error::${commit} (${subject}) is missing a Signed-off-by trailer for <${author}>"
    status=1
  fi
done
exit "${status}"
