#!/usr/bin/env bash
# Publish the official template modules (0011:D25) through `opm module
# publish`, dogfooding the pipeline. Two phases, so no template publishes
# unless every template passed every gate in the same run:
#
#   gate     every templates/*/ tree, in both modes, reporting every failing
#            template rather than the first:
#            - identity: Version is a stable SemVer, and ModulePath is
#              opmodel.dev/templates/<dir>@v<major>;
#            - layout: no symbolic link, special file or nested cue.mod;
#            - `opm module tidy --check`;
#            - `opm module vet`, which renders the template's debugValues
#              against a platform generated from its own pins, as
#              `opm module build` does, so a template that does not render
#              never publishes;
#            - the tree equals its module zip: the zip `cue mod publish --out`
#              builds holds exactly the tree's files, so what vet read is what
#              publishes;
#            - the publish gates dry-run: GO, or the only refusal is that the
#              version is already published;
#            - versions, as `opm module init` resolves them (the highest
#              published stable version of the major): a GO version is above
#              that highest version; an already-published version is that
#              highest version, and its published artifact holds exactly the
#              tree's files, because published versions are immutable.
#   publish  (release only) acts on each template's gate verdict: GO is
#            published, already published and identical is skipped (the
#            caller-side filter of 0011:D15; publish itself never skips). GHCR
#            is not probed a second time, so a version another run published
#            in between makes `opm module publish` refuse instead of being
#            skipped.
#
# Changed implies bumped, by content: the check reads no git history, so it
# holds however a change reached main and however many releases were cut on top
# of it. It fetches the published module zip anonymously (pull scope), verifies
# it against its manifest digest, and compares it with the tree's zip file by
# file, paths and bytes, comments included. Zip order, timestamps, permissions
# and compression are not module content (modzip ignores them) and depend on
# the toolchain that built the publisher, so the zip bytes are not compared.
# Any fetch, listing, build or unpack error fails the template; it never counts
# as "not published".
#
# Why the tree must equal its zip: `opm module publish` zips the directory with
# modzip.CreateFromDir, which silently leaves out symbolic links, special files,
# nested modules and VCS files. Vet and tidy read the tree, so a file the zip
# omits is one vet read and no user ever receives. `cue mod publish --out` runs
# the same modzip for `source: kind: "self"`, which the publish gates require,
# and the workflows install the cue that the cli's go.mod requires. --out
# writes a local OCI image layout and pushes nothing.
#
# With --dry-run (PR CI) the script stops after the gate phase.
set -euo pipefail

mode=publish
if [ "${1:-}" = "--dry-run" ]; then
  mode=dry-run
fi

opm=${OPM_BIN:-./bin/opm}
# BOTH are required: `cue eval` reads CUE_REGISTRY, `opm` reads OPM_REGISTRY
# (--registry > OPM_REGISTRY > ~/.opm/config.cue, never CUE_REGISTRY). With only
# CUE_REGISTRY set, opm silently falls through to the caller's personal config;
# it happens to work on a CI runner that has none, and misleads everywhere else.
export CUE_REGISTRY=${CUE_REGISTRY:-opmodel.dev=ghcr.io/open-platform-model}
export OPM_REGISTRY=${OPM_REGISTRY:-$CUE_REGISTRY}
# Byte order for every sort and comparison.
export LC_ALL=C

for tool in cue curl jq unzip sha256sum diff find sort grep sed head tail cut mktemp cp rm basename; do
  command -v "$tool" >/dev/null || {
    echo "==> missing tool: ${tool}"
    exit 1
  }
done

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

ghcr=https://ghcr.io
stable_re='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'

# token <repo> — an anonymous pull token, never GHCR_AUTH: the templates are
# public, because `opm module init` fetches them anonymously.
token() {
  curl -sSf "${ghcr}/token?scope=repository:$1:pull" | jq -er .token
}

# stable_versions <repo> <major> <dir> — the published stable versions of one
# major, bare and ascending, one per line. GHCR answers 403 for the token of a
# repository it does not know (a template never published), and cue's
# modregistry, which the dry-run and `opm module init` list versions with,
# reads 403 and 404 as "no such module". So does this function, so its list
# agrees with the dry-run's; any other answer is an error. Follows the
# tags/list pagination (GHCR pages at 100 tags by default).
stable_versions() {
  local repo=$1 major=$2 d=$3 tok url code pages=0
  : >"$d/tags"
  code=$(curl -sS -o "$d/token.json" -w '%{http_code}' \
    "${ghcr}/token?scope=repository:${repo}:pull") || return 1
  case $code in
    200) tok=$(jq -er .token "$d/token.json") || return 1 ;;
    403) return 0 ;;
    *) return 1 ;;
  esac
  url="/v2/${repo}/tags/list?n=1000"
  while [ -n "$url" ]; do
    pages=$((pages + 1))
    [ "$pages" -le 20 ] || return 1
    code=$(curl -sS -D "$d/tags.headers" -o "$d/tags.json" -w '%{http_code}' \
      -H "Authorization: Bearer ${tok}" "${ghcr}${url}") || return 1
    case $code in
      200)
        jq -r '(.tags // [])[]' "$d/tags.json" >>"$d/tags" || return 1
        ;;
      403 | 404)
        [ "$pages" -eq 1 ] || return 1
        break
        ;;
      *) return 1 ;;
    esac
    url=$(sed -En 's/^[Ll]ink: *<([^>]*)>; *rel="next".*$/\1/p' "$d/tags.headers") || return 1
  done
  sed -n "s/^v\(${major}\.[0-9]*\.[0-9]*\)$/\1/p" "$d/tags" | { grep -E "$stable_re" || true; } |
    sort -t. -k1,1n -k2,2n -k3,3n
}

# above <a> <b> — 0 when stable version a is above stable version b.
above() {
  [ "$1" != "$2" ] &&
    [ "$(printf '%s\n%s\n' "$1" "$2" | sort -t. -k1,1n -k2,2n -k3,3n | tail -n 1)" = "$1" ]
}

# zip_layer <manifest-file> — the digest of the manifest's one module zip layer.
zip_layer() {
  jq -er '[.layers[] | select(.mediaType == "application/zip")]
    | if length == 1 then .[0].digest
      else error("want exactly one application/zip layer, found \(length)") end' "$1"
}

# published_zip <repo> <tag> <out> — fetch the published module zip and verify
# it against its digest.
published_zip() {
  local repo=$1 tag=$2 out=$3 tok digest
  tok=$(token "$repo") || return 1
  curl -sSf -o "$out.manifest" \
    -H "Authorization: Bearer ${tok}" \
    -H "Accept: application/vnd.oci.image.manifest.v1+json" \
    "${ghcr}/v2/${repo}/manifests/${tag}" || return 1
  digest=$(zip_layer "$out.manifest") || return 1
  [[ $digest =~ ^sha256:[0-9a-f]{64}$ ]] || {
    echo "unexpected layer digest '${digest}'"
    return 1
  }
  curl -sSfL -o "$out" -H "Authorization: Bearer ${tok}" \
    "${ghcr}/v2/${repo}/blobs/${digest}" || return 1
  [ "sha256:$(sha256sum "$out" | cut -d' ' -f1)" = "$digest" ] || {
    echo "downloaded blob does not match ${digest}"
    return 1
  }
}

# tree_zip <dir> <version> <out> — the zip `opm module publish` would push.
tree_zip() {
  local dir=$1 version=$2 out=$3 oci manifest digest
  oci="$out.oci"
  (cd "$dir" && cue mod publish "v${version}" --out "$oci") >/dev/null || return 1
  manifest=$(jq -er '.manifests | if length == 1 then .[0].digest
    else error("want one manifest, found \(length)") end' "$oci/index.json") || return 1
  digest=$(zip_layer "$oci/blobs/sha256/${manifest#sha256:}") || return 1
  cp "$oci/blobs/sha256/${digest#sha256:}" "$out" || return 1
}

# layout <template> <dir> — 0 when the tree holds only regular files and
# directories and no nested module. The module zip silently omits symbolic
# links, special files and nested modules, so vet would read files that never
# publish; name them instead.
layout() {
  local t=$1 dir=$2 found
  found=$(find "$dir" -type l -printf '    %P\n') || return 1
  if [ -n "$found" ]; then
    echo "==> ${t}: symbolic links never publish (a module zip omits them); replace them with files:"
    echo "$found"
    return 1
  fi
  found=$(find "$dir" ! -type d ! -type f -printf '    %P\n') || return 1
  if [ -n "$found" ]; then
    echo "==> ${t}: special files never publish; remove them:"
    echo "$found"
    return 1
  fi
  found=$(find "$dir" -mindepth 2 -iname cue.mod -printf '    %P\n') || return 1
  if [ -n "$found" ]; then
    echo "==> ${t}: a nested cue.mod makes a nested module, which never publishes; remove it:"
    echo "$found"
    return 1
  fi
}

# zip_holds_tree <template> <dir> <zip> — 0 when the module zip holds exactly
# the tree's files, so the published module is the vetted one. Directories are
# not module content: a module zip holds no directory entries.
zip_holds_tree() {
  local t=$1 dir=$2 zip=$3 rc=0
  find "$dir" ! -type d -printf '%P\n' | sort >"$zip.tree-files" || return 1
  unzip -Z1 "$zip" | sort >"$zip.zip-files" || return 1
  diff "$zip.zip-files" "$zip.tree-files" >"$zip.files-diff" || rc=$?
  case $rc in
    0) return 0 ;;
    1)
      echo "==> ${t}: the module zip does not hold exactly the tree's files, so the published module would not be the vetted one:"
      sed -En 's/^> /    only in the tree: /p; s/^< /    only in the zip: /p' "$zip.files-diff" | head -n 60
      return 1
      ;;
    *)
      echo "==> ${t}: comparing the tree with its module zip failed"
      return 1
      ;;
  esac
}

# same_as_published <template> <dir> <version> <work-dir> — 0 when GHCR's
# v<version> holds exactly the files of the tree's zip; otherwise prints why.
same_as_published() {
  local t=$1 dir=$2 version=$3 d=$4 rc=0
  if ! published_zip "open-platform-model/opmodel.dev/templates/${t}" "v${version}" "$d/published.zip"; then
    echo "==> ${t}: cannot fetch the published v${version} from GHCR; a fetch error never counts as unpublished"
    return 1
  fi
  if ! { unzip -q "$d/published.zip" -d "$d/published" && unzip -q "$d/tree.zip" -d "$d/tree"; }; then
    echo "==> ${t}: cannot unpack a module zip"
    return 1
  fi
  (cd "$d" && diff -ru published tree) >"$d/diff" 2>&1 || rc=$?
  case $rc in
    0)
      echo "==> ${t}: v${version} already published and identical to the tree"
      return 0
      ;;
    1)
      echo "==> ${t}: the tree differs from the published v${version}:"
      # Zip entries carry no mtime; drop diff's meaningless header timestamps.
      sed -E 's/^((---|\+\+\+) [^[:space:]]+)[[:space:]].*$/\1/' "$d/diff" | head -n 60
      echo "    Published versions are immutable: run 'opm module version set <semver> ./${dir}'."
      echo "    If this branch is behind main, update it first: main may already publish that version."
      return 1
      ;;
    *)
      cat "$d/diff"
      echo "==> ${t}: comparing the tree with the published v${version} failed"
      return 1
      ;;
  esac
}

# gate <template> <dir> — 0 when every gate passes. Sets version to the
# declared version and verdict to publish (GO) or skip (already published and
# identical).
# Runs as an `if` condition, where errexit is off, so every step checks its
# own status. The identity is read here, so a broken one fails only its own
# template and never hides another template's failure.
gate() {
  local t=$1 dir=$2 path major want d out held listed versions highest
  version="" verdict=""
  if ! version=$(cd "$dir" && cue eval ./identity --out text -e Version) ||
    ! path=$(cd "$dir" && cue eval ./identity --out text -e ModulePath); then
    echo "==> ${t}: cannot read Version and ModulePath from ./${dir}identity"
    return 1
  fi
  echo "==> ${t}: gates at v${version}"
  if ! [[ $version =~ $stable_re ]]; then
    echo "==> ${t}: v${version} is not a stable version; 'opm module init' resolves only stable template versions"
    return 1
  fi
  major=${version%%.*}
  want="opmodel.dev/templates/${t}@v${major}"
  if [ "$path" != "$want" ]; then
    echo "==> ${t}: ModulePath is ${path}; the template in templates/${t}/ at v${version} must be ${want}"
    return 1
  fi
  layout "$t" "$dir" || return 1
  if ! "$opm" module tidy --check "./$dir"; then
    echo "==> ${t}: cue.mod/module.cue is not tidy; run 'opm module tidy ./$dir'"
    return 1
  fi
  if ! "$opm" module vet "./$dir"; then
    echo "==> ${t}: 'opm module vet' failed; a template must render its own debugValues"
    return 1
  fi
  d=$(mktemp -d "$work/${t}.XXXXXX") || return 1
  if ! tree_zip "$dir" "$version" "$d/tree.zip"; then
    echo "==> ${t}: cannot build the tree's module zip with 'cue mod publish --out'"
    return 1
  fi
  zip_holds_tree "$t" "$dir" "$d/tree.zip" || return 1
  held=no
  if out=$("$opm" module publish --dry-run "./$dir" 2>&1); then
    echo "$out"
  else
    echo "$out"
    # Anchored: exactly one refusal, and it names this template's own path and
    # version, so "11 refusals" or another module's refusal never matches.
    if ! { grep -Eq ' 1 refusal$' <<<"$out" && grep -Fq "${want} already holds v${version}" <<<"$out"; }; then
      echo "==> ${t}: publish gates refused"
      return 1
    fi
    held=yes
  fi
  # The version `opm module init` resolves is the highest published stable
  # version of the major; it must be this tree, or this tree must be above it.
  if ! versions=$(stable_versions "open-platform-model/opmodel.dev/templates/${t}" "$major" "$d"); then
    echo "==> ${t}: cannot list the published versions on GHCR; a fetch error never counts as unpublished"
    return 1
  fi
  highest=$(tail -n 1 <<<"$versions")
  listed=no
  if grep -Fqx "$version" <<<"$versions"; then listed=yes; fi
  if [ "$held" = no ]; then
    if [ "$listed" = yes ]; then
      echo "==> ${t}: GHCR lists v${version}, but the publish dry-run found it unpublished"
      return 1
    fi
    if [ -n "$highest" ] && ! above "$version" "$highest"; then
      echo "==> ${t}: v${version} is not above the highest published v${highest}; run 'opm module version set <semver> ./${dir}' with a higher version."
      echo "    If this branch is behind main, update it first: main may already publish a higher version."
      return 1
    fi
    verdict=publish
    return 0
  fi
  if [ "$listed" = no ]; then
    echo "==> ${t}: the publish dry-run found v${version} published, but GHCR does not list it"
    return 1
  fi
  if [ "$version" != "$highest" ]; then
    echo "==> ${t}: v${version} is published, but 'opm module init' fetches the highest published v${highest}; the tree must be at v${highest}, or above it with a bump."
    echo "    If this branch is behind main, update it first."
    return 1
  fi
  same_as_published "$t" "$dir" "$version" "$d" || return 1
  verdict=skip
}

failed=()
verdicts=()
for dir in templates/*/; do
  t=$(basename "$dir")
  if gate "$t" "$dir"; then
    verdicts+=("${t} ${verdict} ${version}")
    [ "$verdict" = skip ] || echo "==> ${t}: v${version} passes every gate and is not published yet"
  else
    failed+=("$t")
  fi
done
if [ "${#failed[@]}" -gt 0 ]; then
  echo "==> gates failed for: ${failed[*]}; nothing published"
  exit 1
fi
[ "$mode" = publish ] || exit 0

for row in "${verdicts[@]}"; do
  read -r t verdict version <<<"$row"
  if [ "$verdict" = skip ]; then
    echo "==> ${t}: v${version} already published; skipped"
    continue
  fi
  echo "==> ${t}: publishing v${version}"
  if ! "$opm" module publish "./templates/${t}/"; then
    echo "==> ${t}: publishing v${version} failed; nothing after it was published, and a re-run skips what this run pushed"
    exit 1
  fi
done
