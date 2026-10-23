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
# Only check commits introduced by the PR, not the ones main brought along.
base="$(git merge-base "${base}" "${head}" 2>/dev/null || echo "${base}")"

status=0
for commit in $(git rev-list --no-merges "${base}..${head}"); do
  author_name="$(git log -1 --format='%an' "${commit}")"
  author="$(git log -1 --format='%ae' "${commit}")"
  # Bots cannot sign off; Dependabot/Renovate/GitHub Actions commits are exempt.
  case "${author_name} ${author}" in
    *"[bot]"*) echo "skipping bot commit ${commit} (${author_name})"; continue ;;
  esac
  if ! git log -1 --format='%B' "${commit}" | grep -qiE "^Signed-off-by: .+ <${author}>$"; then
    subject="$(git log -1 --format='%s' "${commit}")"
    echo "::error::${commit} (${subject}) is missing a Signed-off-by trailer for <${author}>"
    status=1
  fi
done
exit "${status}"
