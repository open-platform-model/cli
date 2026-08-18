#!/usr/bin/env bash
# Publish the official template modules (enhancement 0011 D25) through
# `opm module publish` — dogfooding the pipeline, so a template that violates
# any publish gate fails the cli release.
#
# For each templates/*/ tree: resolve the declared version from the identity
# package (grep-free — cue eval), check tag existence against GHCR, and invoke
# publish only for unpublished versions. The skip lives HERE, in the caller,
# because publish itself never skips: an already-published tag is always a
# refusal (D15) — idempotency is the caller-side filter.
#
# With --dry-run (PR CI): run every gate without pushing. A dry run whose only
# refusal is already-published still passes — on a PR the committed version
# may legitimately be live; every other gate refusal fails the check.
set -euo pipefail

mode=publish
if [ "${1:-}" = "--dry-run" ]; then
  mode=dry-run
fi

opm=${OPM_BIN:-./bin/opm}
export CUE_REGISTRY=${CUE_REGISTRY:-opmodel.dev=ghcr.io/open-platform-model}

# published <repo> <tag> — 0 when the manifest exists. GHCR_AUTH
# ("user:token") authenticates the token exchange; without it the anonymous
# pull token serves public packages. A 404 is the normal first-publish state.
published() {
  local repo=$1 tag=$2 token code
  token=$(curl -sf ${GHCR_AUTH:+-u "$GHCR_AUTH"} \
    "https://ghcr.io/token?scope=repository:${repo}:pull" | jq -r .token) || return 1
  code=$(curl -s -o /dev/null -w '%{http_code}' \
    -H "Authorization: Bearer ${token}" \
    -H "Accept: application/vnd.oci.image.manifest.v1+json" \
    "https://ghcr.io/v2/${repo}/manifests/${tag}")
  [ "$code" = "200" ]
}

for dir in templates/*/; do
  t=$(basename "$dir")
  version=$(cd "$dir" && cue eval ./identity --out text -e Version)
  tag="v${version}"
  repo="open-platform-model/opmodel.dev/templates/${t}"

  if [ "$mode" = publish ]; then
    if published "$repo" "$tag"; then
      echo "==> ${t}: ${tag} already published — skipped by the caller-side filter (D15)"
      continue
    fi
    echo "==> ${t}: publishing ${tag}"
    "$opm" module publish "./${dir}"
  else
    echo "==> ${t}: dry-run at ${tag}"
    if out=$("$opm" module publish --dry-run "./${dir}" 2>&1); then
      echo "${out}"
      continue
    fi
    echo "${out}"
    if echo "${out}" | grep -q "already holds" && echo "${out}" | grep -q "1 refusal"; then
      echo "==> ${t}: only refusal is already-published — acceptable on a PR"
      continue
    fi
    echo "==> ${t}: publish gates refused"
    exit 1
  fi
done
