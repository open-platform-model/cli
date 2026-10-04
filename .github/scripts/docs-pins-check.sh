#!/usr/bin/env bash
# Docs-bundle pins (docs-kit gate G2-pins): every version the cli's docs
# bundle pins must have a docs bundle. opmodel.dev anchors a site version on
# the cli's bundle and pulls the library, core and opm-operator bundles of
# exactly the versions its manifest pins, refusing a pin with none (docs-kit
# C16). The pins come from the program docs-kit runs (hack/docskit-dump pins);
# each is looked up anonymously, as the site pulls it, at its release tag. A
# 401, 403 or 404 counts as missing: GHCR answers 403 for a package that does
# not exist yet or is private.
#
# Usage: docs-pins-check.sh [--warn] [--moved-from REV]
#
#   (default)        print one line per problem on stdout; exit 1 if any.
#                    G1 (release-pin-check.sh) runs this mode.
#   --moved-from REV check only when the library version in go.mod or
#                    PinnedOperatorVersion (internal/operator/pin.go; at a REV
#                    from before the operator module pin, manifest.go) differs
#                    between REV and the work tree; otherwise say so and exit
#                    0. A module pin move that deploys the same operator
#                    changes no docs pin, so it checks nothing.
#   --warn           every problem, a failed build or lookup included, is a
#                    GitHub warning annotation (and a job-summary line when
#                    GITHUB_STEP_SUMMARY is set); exit 0. pr.yml's Lint job
#                    runs this mode with --moved-from the pull request's base.
#
# This check used to run inside task deps:cascade; it runs in the pull
# request's own CI so the cascade's compute never runs code from a newly
# pinned library (security pass 2026-10-04, finding CAS-R2).
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

warn_mode=false
moved_from=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --warn) warn_mode=true; shift ;;
    --moved-from)
      [ "$#" -ge 2 ] && [ -n "$2" ] || { echo "docs-pins: --moved-from needs a revision" >&2; exit 2; }
      moved_from=$2; shift 2 ;;
    *) echo "docs-pins: usage: docs-pins-check.sh [--warn] [--moved-from REV]" >&2; exit 2 ;;
  esac
done

LIB=github.com/open-platform-model/library
PIN=internal/operator/pin.go
# Where PinnedOperatorVersion lived before the operator module pin replaced
# the embedded manifest; only a base revision can still have it.
OLD_PIN=internal/operator/manifest.go

problems=0
problem() {
  problems=$((problems + 1))
  if [ "$warn_mode" = true ]; then
    echo "::warning title=Docs bundle::$1"
    if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then printf -- '- %s\n' "$1" >>"$GITHUB_STEP_SUMMARY"; fi
  else
    echo "$1"
  fi
}
finish() {
  if [ "$problems" -gt 0 ] && [ "$warn_mode" = false ]; then exit 1; fi
  exit 0
}

lib_of() { awk -v m="$LIB" '$1 == m { print $2; exit }'; }
op_of() { sed -n 's/^const PinnedOperatorVersion = "\(.*\)"$/\1/p'; }

if [ -n "$moved_from" ]; then
  base_pin=$PIN
  if ! git cat-file -e "$moved_from:$PIN" 2>/dev/null; then base_pin=$OLD_PIN; fi
  if ! base_gomod=$(git show "$moved_from:go.mod" 2>/dev/null) ||
    ! base_pinfile=$(git show "$moved_from:$base_pin" 2>/dev/null); then
    problem "docs bundles not checked: cannot read go.mod and $base_pin at $moved_from"
    finish
  fi
  lib_base=$(lib_of <<<"$base_gomod")
  op_base=$(op_of <<<"$base_pinfile")
  lib_tree=$(lib_of <go.mod)
  op_tree=""
  if [ -f "$PIN" ]; then op_tree=$(op_of <"$PIN"); fi
  # An empty read means go.mod or the pin file changed shape; two empty reads
  # would compare equal and skip the check without a word.
  if [ -z "$lib_base" ] || [ -z "$lib_tree" ] || [ -z "$op_base" ] || [ -z "$op_tree" ]; then
    problem "docs bundles not checked: cannot read the $LIB version in go.mod or PinnedOperatorVersion in $PIN ($base_pin at the base) (base ${lib_base:-?}/${op_base:-?}, tree ${lib_tree:-?}/${op_tree:-?})"
    finish
  fi
  if [ "$lib_tree" = "$lib_base" ] && [ "$op_tree" = "$op_base" ]; then
    echo "docs-pins: library ${lib_tree} and opm-operator ${op_tree} unchanged against ${moved_from}; nothing to check" >&2
    finish
  fi
fi

docs_registry=ghcr.io
docs_repo=open-platform-model/docs
# docs_bundle_status PROJECT TAG: the HTTP status of an anonymous manifest
# HEAD of PROJECT's docs bundle at TAG (the token request's status when GHCR
# refuses an anonymous token), 000 when ghcr.io is unreachable.
docs_bundle_status() {
  local body code token
  body=$(curl -sS --connect-timeout 10 --max-time 30 --retry 3 --retry-all-errors -w '\n%{http_code}' \
    "https://${docs_registry}/token?scope=repository:${docs_repo}/$1:pull" 2>/dev/null) || { echo 000; return; }
  code=${body##*$'\n'}
  if [ "$code" != 200 ]; then echo "$code"; return; fi
  token=$(jq -r '.token // empty' <<<"${body%$'\n'*}")
  curl -sS -o /dev/null -w '%{http_code}' -I --connect-timeout 10 --max-time 30 --retry 3 --retry-all-errors \
    -H "Authorization: Bearer ${token}" \
    -H 'Accept: application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json' \
    "https://${docs_registry}/v2/${docs_repo}/$1/manifests/$2" 2>/dev/null || true
}

if ! docs_pins=$(go run ./hack/docskit-dump pins 2>&1); then
  problem "hack/docskit-dump pins failed: $(tr '\n' ' ' <<<"$docs_pins")"
  finish
fi
if ! entries=$(jq -er '.pins | to_entries[] | "\(.key) \(.value)"' <<<"$docs_pins"); then
  problem "hack/docskit-dump pins printed no .pins entries"
  finish
fi
checked=""
while read -r project pin; do
  [ -n "${project:-}" ] || continue
  ref="${docs_registry}/${docs_repo}/${project}:${pin}"
  code=$(docs_bundle_status "$project" "$pin")
  code=${code:-000}
  case "$code" in
    200) checked+=" ${project} ${pin}," ;;
    401|403|404) problem "docs bundle: the cli pins ${project} ${pin}, and ${ref} does not exist or is not public (publish it: run ${project}'s Docs workflow in release mode for its v${pin} release)" ;;
    *) problem "docs bundle: ${ref}: lookup failure, HTTP ${code} (re-run when ghcr.io is reachable)" ;;
  esac
done <<<"$entries"
if [ "$problems" = 0 ]; then echo "docs-pins: ok, a docs bundle exists for${checked%,}" >&2; fi
finish
