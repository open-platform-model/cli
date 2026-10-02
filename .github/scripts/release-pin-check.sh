#!/usr/bin/env bash
# Release-pin gate (G1): a release must never ship a local, unpublished or
# mismatched upstream pin. The rule is shared by every repo in the release
# cascade (workspace RELEASING.md, section "Gates"); the cli adds the check that
# the embedded opm-operator manifest matches PinnedOperatorVersion.
#
# CI runs this in the lint job of pr.yml and ci.yml on release-please branches
# only; `task deps:release-check` runs it locally on any branch.
#
# Every violation is collected and printed before exiting, so one run names
# them all. Exit 0: no violation. Exit 1: at least one violation or a lookup
# that could not be completed (never treated as a pass).
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

failures=()
fail() { failures+=("release-pin: $1"); }

# 1. No replace directive in go.mod.
if ! go mod edit -json | jq -e '.Replace == null' >/dev/null; then
  while IFS= read -r line; do
    fail "go.mod: replace directive ${line} (remove it; release against published modules)"
  done < <(go mod edit -json | jq -r '.Replace[] | "\(.Old.Path) => \(.New.Path)\(if .New.Version then "@" + .New.Version else "" end)"')
fi

# 2. Every OPM Go pin is a tagged release, never a pseudo-version.
pseudo='([-.]0\.|-)[0-9]{14}-[0-9a-f]{12}$'
while read -r path version; do
  [ -n "${path:-}" ] || continue
  if grep -qE "$pseudo" <<<"$version"; then
    fail "go.mod: ${path} ${version} is a pseudo-version (pin a released tag: go get ${path}@<tag>)"
    continue
  fi
  repo=$(sed -E 's#/v[0-9]+$##' <<<"$path")
  rc=0
  git ls-remote --exit-code --tags "https://${repo}.git" "refs/tags/${version}" </dev/null >/dev/null 2>&1 || rc=$?
  case "$rc" in
    0) ;;
    2) fail "go.mod: ${path} ${version} has no tag ${version} in ${repo} (pin a published release)" ;;
    *) fail "go.mod: ${path} ${version}: lookup failure, git ls-remote exited ${rc} for https://${repo}.git (re-run when github.com is reachable)" ;;
  esac
done < <(go mod edit -json | jq -r '.Require[]? | select(.Path | startswith("github.com/open-platform-model/")) | "\(.Path) \(.Version)"')

# 3. No development CUE pin in a shipped template.
while IFS= read -r hit; do
  [ -n "$hit" ] || continue
  fail "${hit%%:*}: dev pin at line $(cut -d: -f2 <<<"$hit"): $(cut -d: -f3- <<<"$hit" | sed 's/^[[:space:]]*//') (pin a published release with cue mod get)"
done < <(grep -HnE 'v: "[^"]*-0\.dev\.' templates/*/cue.mod/module.cue || true)

# 4. No tracked local-module.cue anywhere.
while IFS= read -r file; do
  [ -n "$file" ] || continue
  fail "${file}: tracked local-module.cue (git rm --cached it; it redirects a module to a local path)"
done < <(git ls-files '*cue.mod/local-module.cue')

# 5. The embedded operator manifest matches PinnedOperatorVersion.
manifest=internal/operator/manifest.go
installyaml=internal/operator/dist/install.yaml
pinned=$(sed -n 's/^const PinnedOperatorVersion = "\(.*\)"$/\1/p' "$manifest")
images=$(grep -E '^\s*image: ghcr\.io/open-platform-model/opm-operator:' "$installyaml" || true)
count=$(grep -c . <<<"$images" || true)
if [ -z "$pinned" ]; then
  fail "${manifest}: cannot read PinnedOperatorVersion (task operator:sync VERSION=<tag>)"
elif [ "$count" -ne 1 ]; then
  fail "${installyaml}: expected exactly one opm-operator image line, found ${count} (task operator:sync VERSION=${pinned})"
else
  tag=$(sed -E 's#.*opm-operator:([^@[:space:]]+).*#\1#' <<<"$images")
  if [ "$tag" != "$pinned" ]; then
    fail "${installyaml}: operator image tag ${tag} differs from PinnedOperatorVersion ${pinned} in ${manifest} (task operator:sync VERSION=<tag>)"
  fi
fi

if [ "${#failures[@]}" -gt 0 ]; then
  printf '%s\n' "${failures[@]}" >&2
  echo "release-pin: ${#failures[@]} violation(s); see workspace RELEASING.md, section \"Gates\" (G1)" >&2
  exit 1
fi
echo "release-pin: ok (no replace, published OPM pins, no dev template pins, operator embed matches ${pinned})"
