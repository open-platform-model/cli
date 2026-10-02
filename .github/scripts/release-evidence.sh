#!/usr/bin/env bash
# Operator-embed evidence (G4), interim until the cluster-backed e2e CI job
# (cli change add-embedded-operator-e2e-job) replaces it. Workspace
# RELEASING.md, section "Gates".
#
# On a release-please PR whose PinnedOperatorVersion moved since the last cli
# release, the PR must carry the label e2e-verified: a human ran
# `task test:e2e` against the new embed. Every other PR passes.
#
# Inputs (environment, set by .github/workflows/release-evidence.yml):
#   HEAD_REF  PR head branch          REF_NAME  fallback when HEAD_REF is empty
#   BASE_REF  PR base branch          LABELS    JSON array of the PR's label names
# Needs the base branch at origin/$BASE_REF and the release tags fetched.
set -euo pipefail

fail() {
  echo "G4: $1" >&2
  exit 1
}

pin_re='s/^const PinnedOperatorVersion = "\(.*\)"$/\1/p'
manifest=internal/operator/manifest.go

branch="${HEAD_REF:-${REF_NAME:-}}"
case "$branch" in
  release-please--*) ;;
  *)
    echo "G4: applies to release-please PRs only (branch '${branch}'); nothing to check"
    exit 0
    ;;
esac

base="${BASE_REF:?BASE_REF is required}"
version=$(git show "origin/${base}:.release-please-manifest.json" 2>/dev/null | jq -er '."."' 2>/dev/null) ||
  fail "cannot read the last cli version from .release-please-manifest.json on origin/${base}"
tag="v${version}"
git rev-parse -q --verify "refs/tags/${tag}" >/dev/null ||
  fail "last cli tag ${tag} not found (fetch tags; the release must be tagged before G4 can compare)"

old=$(git show "${tag}:${manifest}" 2>/dev/null | sed -n "$pin_re") || old=""
new=$(sed -n "$pin_re" "$manifest")
[ -n "$old" ] || fail "cannot read PinnedOperatorVersion at ${tag}"
[ -n "$new" ] || fail "cannot read PinnedOperatorVersion in ${manifest}"

if [ "$old" = "$new" ]; then
  echo "G4: embedded operator ${new} unchanged since ${tag}; no evidence needed"
  exit 0
fi

if jq -e 'index("e2e-verified")' <<<"${LABELS:-[]}" >/dev/null; then
  echo "G4: embedded operator moved ${old} -> ${new} since ${tag}; label e2e-verified present"
  exit 0
fi

fail "embedded operator moved ${old} -> ${new} since ${tag}. Run task test:e2e against this PR's head, then add the label e2e-verified (remove it if the pin moves again). Interim until add-embedded-operator-e2e-job runs the e2e suite in CI."
