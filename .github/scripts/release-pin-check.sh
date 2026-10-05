#!/usr/bin/env bash
# Release-pin gate (G1): a release must never ship a local, unpublished or
# mismatched upstream pin. The rule is shared by every repo in the release
# cascade (workspace RELEASING.md, section "Gates"); the cli adds the checks that
# the operator module pin in internal/operator/pin.go is served and records the
# operator release that module deploys, and that every version its docs bundle
# pins has a docs bundle (docs-kit gate G2-pins).
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

# Read go.mod once; a failed read stops the gate instead of reading as "no pins".
gomod=$(go mod edit -json) || { echo "release-pin: go.mod: go mod edit -json failed; cannot check Go pins" >&2; exit 1; }

# 1. No replace directive in go.mod.
if ! jq -e '.Replace == null' >/dev/null <<<"$gomod"; then
  while IFS= read -r line; do
    fail "go.mod: replace directive ${line} (remove it; release against published modules)"
  done < <(jq -r '.Replace[] | "\(.Old.Path) => \(.New.Path)\(if .New.Version then "@" + .New.Version else "" end)"' <<<"$gomod")
fi

# 2. Every OPM Go pin is a tagged release, never a pseudo-version.
pseudo='([-.]0\.|-)[0-9]{14}-[0-9a-f]{12}$'
while read -r path version; do
  [ -n "${path:-}" ] || continue
  if grep -qE "$pseudo" <<<"$version"; then
    fail "go.mod: ${path} ${version} is a pseudo-version (pin a released tag: go get ${path}@<tag>)"
    continue
  fi
  # The repository is the first three path segments; a nested module
  # (github.com/open-platform-model/docs-kit/cobradump) is tagged
  # <subdir>/<version> in it, the form the Go module proxy reads. A major
  # suffix (/v2) is no directory and no part of the tag.
  repo=$(cut -d/ -f1-3 <<<"$path")
  sub=$(sed -E 's#(^|/)v[0-9]+$##' <<<"${path#"$repo"}")
  sub=${sub#/}
  tag=${sub:+${sub}/}${version}
  rc=0
  GIT_TERMINAL_PROMPT=0 git ls-remote --exit-code --tags "https://${repo}.git" "refs/tags/${tag}" </dev/null >/dev/null 2>&1 || rc=$?
  case "$rc" in
    0) ;;
    2) fail "go.mod: ${path} ${version} has no tag ${tag} in ${repo} (pin a published release)" ;;
    *) fail "go.mod: ${path} ${version}: lookup failure, git ls-remote exited ${rc} for https://${repo}.git (re-run when github.com is reachable)" ;;
  esac
done < <(jq -r '.Require[]? | select(.Path | startswith("github.com/open-platform-model/")) | "\(.Path) \(.Version)"' <<<"$gomod")

# 3. No development CUE pin in a shipped template.
compgen -G 'templates/*/cue.mod/module.cue' >/dev/null || fail "templates/*/cue.mod/module.cue: no template module found"
while IFS= read -r hit; do
  [ -n "$hit" ] || continue
  fail "${hit%%:*}: dev pin at line $(cut -d: -f2 <<<"$hit"): $(cut -d: -f3- <<<"$hit" | sed 's/^[[:space:]]*//') (pin a published release with cue mod get)"
done < <(grep -HnE 'v: "[^"]*-0\.dev\.' templates/*/cue.mod/module.cue || true)

# 4. No tracked local-module.cue anywhere.
while IFS= read -r file; do
  [ -n "$file" ] || continue
  fail "${file}: tracked local-module.cue (git rm --cached it; it redirects a module to a local path)"
done < <(git ls-files '*cue.mod/local-module.cue')

# 5. The operator module pin is served and consistent (0021:D11:R6): the
# registry serves PinnedModuleVersion, and PinnedOperatorVersion is the
# operator release that module's operator package states. hack/operator-pin
# reads it without a render; a lookup that fails is a failure, not a pass.
if ! pin_out=$(go run ./hack/operator-pin --check 2>&1); then
  while IFS= read -r line; do
    # go run adds its own "exit status N" line, and on a cold cache one
    # "go: downloading ..." line per module; the tool's lines say why.
    case "$line" in "" | "exit status "* | "go: "*) continue ;; esac
    fail "${line#operator-pin: }"
  done <<<"$pin_out"
fi

# 6. Every version the cli's docs bundle pins has a docs bundle (docs-kit gate
# G2-pins); the lookup lives in docs-pins-check.sh, which pr.yml also runs as
# a warning on every pull request that moves library or the operator. Each
# line it prints is a violation, and a failing exit with no line is one too,
# so a crash never reads as a pass.
docs_rc=0
docs_out=$(.github/scripts/docs-pins-check.sh) || docs_rc=$?
while IFS= read -r line; do
  [ -n "$line" ] || continue
  fail "$line"
done <<<"$docs_out"
if [ "$docs_rc" -ne 0 ] && [ -z "$docs_out" ]; then
  fail "docs bundle: .github/scripts/docs-pins-check.sh exited ${docs_rc} without naming a problem"
fi

if [ "${#failures[@]}" -gt 0 ]; then
  printf '%s\n' "${failures[@]}" >&2
  echo "release-pin: ${#failures[@]} violation(s); see workspace RELEASING.md, section \"Gates\" (G1)" >&2
  exit 1
fi
echo "release-pin: ok (no replace, published OPM pins, no dev template pins, operator module pin served and consistent, docs bundles exist for every docs pin)"
