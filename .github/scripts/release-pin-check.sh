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
    # go run adds its own "exit status N" line; the tool's lines say why.
    case "$line" in "" | "exit status "*) continue ;; esac
    fail "${line#operator-pin: }"
  done <<<"$pin_out"
fi

# 6. Every version the cli's docs bundle pins has a docs bundle (docs-kit gate
# G2-pins). opmodel.dev anchors a site version on the cli's bundle and pulls
# the library, core and opm-operator bundles of exactly the versions its
# manifest pins, refusing a pin with none (docs-kit C16), so a release whose
# pins lack bundles would break every site build. The pins come from the
# program docs-kit runs (hack/docskit-dump pins); each is looked up
# anonymously, as the site pulls it, at its release tag. A 401, 403 or 404
# counts as missing: GHCR answers 403 for a package that does not exist yet
# or is private.
docs_registry=ghcr.io
docs_repo=open-platform-model/docs
# docs_bundle_status PROJECT TAG: the HTTP status of an anonymous manifest
# HEAD of PROJECT's docs bundle at TAG (the token request's status when GHCR
# refuses an anonymous token), 000 when ghcr.io is unreachable.
docs_bundle_status() {
  local body code token
  body=$(curl -sS --retry 3 --retry-all-errors -w '\n%{http_code}' \
    "https://${docs_registry}/token?scope=repository:${docs_repo}/$1:pull" 2>/dev/null) || { echo 000; return; }
  code=${body##*$'\n'}
  if [ "$code" != 200 ]; then echo "$code"; return; fi
  token=$(jq -r '.token // empty' <<<"${body%$'\n'*}")
  curl -sS -o /dev/null -w '%{http_code}' -I --retry 3 --retry-all-errors \
    -H "Authorization: Bearer ${token}" \
    -H 'Accept: application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json' \
    "https://${docs_registry}/v2/${docs_repo}/$1/manifests/$2" 2>/dev/null || true
}
if ! docs_pins=$(go run ./hack/docskit-dump pins 2>&1); then
  fail "hack/docskit-dump pins failed: ${docs_pins}"
else
  while read -r project pin; do
    [ -n "${project:-}" ] || continue
    ref="${docs_registry}/${docs_repo}/${project}:${pin}"
    code=$(docs_bundle_status "$project" "$pin")
    code=${code:-000}
    case "$code" in
      200) ;;
      401|403|404) fail "docs bundle: the cli pins ${project} ${pin}, and ${ref} does not exist or is not public (publish it: run ${project}'s Docs workflow in release mode for its v${pin} release)" ;;
      *) fail "docs bundle: ${ref}: lookup failure, HTTP ${code} (re-run when ghcr.io is reachable)" ;;
    esac
  done < <(jq -r '.pins | to_entries[] | "\(.key) \(.value)"' <<<"$docs_pins")
fi

if [ "${#failures[@]}" -gt 0 ]; then
  printf '%s\n' "${failures[@]}" >&2
  echo "release-pin: ${#failures[@]} violation(s); see workspace RELEASING.md, section \"Gates\" (G1)" >&2
  exit 1
fi
echo "release-pin: ok (no replace, published OPM pins, no dev template pins, operator module pin served and consistent, docs bundles exist for every docs pin)"
