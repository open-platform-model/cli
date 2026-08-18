#!/usr/bin/env bash
# Publish the repo's own test-fixture modules through `opm module publish` —
# the same pipeline, and the same gates, the official templates go through
# (publish-templates.sh). A fixture that violates a publish gate fails CI.
#
# Fixtures live on the testing domain (testing.opmodel.dev/modules/cli/*) and
# are published to GHCR, so every consumer — examples, e2e testdata, the kind
# dev cluster, a fresh clone — resolves them from a public registry with no
# local registry involved.
#
# For each tests/fixtures/modules/*/ tree carrying an identity package: read the
# declared coordinate from that package (grep-free — cue eval), check tag
# existence against GHCR, and invoke publish only for unpublished versions. The
# skip lives HERE, in the caller, because publish itself never skips: an already
# published tag is always a refusal (D15) — idempotency is the caller-side
# filter.
#
# With --dry-run (PR CI): run every gate without pushing. A dry run whose only
# refusal is already-published still passes — on a PR the committed version may
# legitimately be live; every other gate refusal fails the check.
set -euo pipefail

mode=publish
if [ "${1:-}" = "--dry-run" ]; then
  mode=dry-run
fi

opm=${OPM_BIN:-./bin/opm}
# Both domains must map: the fixtures are on testing.opmodel.dev, their deps
# (core, catalogs) on opmodel.dev.
#
# BOTH variables are required and they are read by different things. `cue eval`
# below reads CUE_REGISTRY; `opm` does NOT — it resolves --registry > OPM_REGISTRY
# > ~/.opm/config.cue (internal/config/resolver.go) and never looks at
# CUE_REGISTRY. Exporting only CUE_REGISTRY silently leaves opm on the caller's
# personal config, which reads as a clean GO against the wrong registry.
export CUE_REGISTRY=${CUE_REGISTRY:-testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works}
export OPM_REGISTRY=${OPM_REGISTRY:-$CUE_REGISTRY}

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

found=0
for dir in tests/fixtures/modules/*/; do
  [ -d "${dir}identity" ] || continue
  found=$((found + 1))
  f=$(basename "$dir")

  version=$(cd "$dir" && cue eval ./identity --out text -e Version)
  tag="v${version}"
  # Derive the GHCR repository from the declared path, not the directory name —
  # the two need not agree, and the path is the coordinate that is published.
  path=$(cd "$dir" && cue eval ./identity --out text -e ModulePath)
  repo="open-platform-model/${path%@*}"

  if [ "$mode" = publish ]; then
    if published "$repo" "$tag"; then
      echo "==> ${f}: ${tag} already published — skipped by the caller-side filter (D15)"
      continue
    fi
    echo "==> ${f}: publishing ${tag} to ${repo}"
    "$opm" module publish "./${dir}"
  else
    echo "==> ${f}: dry-run at ${tag} (${repo})"
    if out=$("$opm" module publish --dry-run "./${dir}" 2>&1); then
      echo "${out}"
      continue
    fi
    echo "${out}"
    if echo "${out}" | grep -q "already holds" && echo "${out}" | grep -q "1 refusal"; then
      echo "==> ${f}: only refusal is already-published — acceptable on a PR"
      continue
    fi
    echo "==> ${f}: publish gates refused"
    exit 1
  fi
done

if [ "$found" -eq 0 ]; then
  echo "no publishable fixture trees found under tests/fixtures/modules/" >&2
  exit 1
fi
