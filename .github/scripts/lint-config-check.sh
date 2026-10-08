#!/usr/bin/env bash
# Linter configuration check, offline. It replaces the configuration check
# of golangci/golangci-lint-action, which downloads the JSON schema from
# golangci-lint.run on every run and failed the required Lint job when that
# download timed out. The same command, golangci-lint config verify, runs
# here against the schema committed under .github/golangci-lint/.
#
# One file, .golangci-lint-version, names the linter version CI installs.
# This check refuses a tree where that file, the committed schema, its
# checksum and the workflows' use of the action disagree:
#
#   - .golangci-lint-version is vX.Y.Z;
#   - .github/golangci-lint/golangci.vX.Y.jsonschema.json exists, is the
#     only schema there, and matches .github/golangci-lint/SHA256SUMS;
#   - every use of the action in .github/workflows/ is pinned to one commit
#     SHA, reads the version file (version-file, never version) and has its
#     own configuration check off (verify: false), and its workflow runs
#     this script;
#   - go.mod has no golangci-lint line (the action reads one before the
#     version file);
#   - the installed golangci-lint is of the same X.Y (a patch difference is
#     allowed: the schema is per minor line);
#   - golangci-lint config verify --schema <file> passes for .golangci.yml.
#
# The linter runs with its proxy variables set to a closed local port, so a
# run that reaches for the network fails at once.
#
# The schema is jsonschema/golangci.jsonschema.json of
# github.com/golangci/golangci-lint at the tag the version file names (the
# file golangci-lint.run serves as golangci.vX.Y.jsonschema.json). To move
# the linter: AGENTS.md, "Moving the golangci-lint version".
#
# Usage: lint-config-check.sh [--root DIR]
#
#   --root DIR  check the tree at DIR (default: this repository). The
#               scenario test (lint-config-check-test.sh) uses it.
#
# Exit: 0 ok; 1 a refusal, one line each on stderr; 2 usage.
set -euo pipefail

root=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --root)
      [ "$#" -ge 2 ] && [ -n "$2" ] || { echo "lint-config: --root needs a directory" >&2; exit 2; }
      root=$2; shift 2 ;;
    *) echo "lint-config: unknown argument: $1" >&2; exit 2 ;;
  esac
done
if [ -z "$root" ]; then
  root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
fi
cd "$root"

version_file=.golangci-lint-version
schema_dir=.github/golangci-lint
action=golangci/golangci-lint-action
script=.github/scripts/lint-config-check.sh

problems=0
fail() {
  echo "lint-config: $*" >&2
  problems=$((problems + 1))
}

# The version file.
minor=""
if [ ! -f "$version_file" ]; then
  fail "$version_file is missing"
else
  version=$(tr -d '[:space:]' <"$version_file")
  if [[ "$version" =~ ^v([0-9]+\.[0-9]+)\.[0-9]+$ ]]; then
    minor=${BASH_REMATCH[1]}
  else
    fail "$version_file holds '$version'; want vX.Y.Z"
  fi
fi

# The committed schema and its checksum.
schema=""
if [ -n "$minor" ]; then
  schema=$schema_dir/golangci.v$minor.jsonschema.json
  if [ ! -f "$schema" ]; then
    fail "$schema is missing: $version_file names $version, so commit the schema of the v$minor line"
    schema=""
  fi
  for f in "$schema_dir"/*.jsonschema.json; do
    [ -e "$f" ] || continue
    [ "$f" = "$schema_dir/golangci.v$minor.jsonschema.json" ] || fail "$f is not the schema of $version; remove it"
  done
  if [ -n "$schema" ]; then
    want=$(basename "$schema")
    if [ ! -f "$schema_dir/SHA256SUMS" ]; then
      fail "$schema_dir/SHA256SUMS is missing"
    elif [ "$(awk 'NF' "$schema_dir/SHA256SUMS" | wc -l)" -ne 1 ] ||
      [ "$(awk 'NF {print $2}' "$schema_dir/SHA256SUMS")" != "$want" ]; then
      fail "$schema_dir/SHA256SUMS must hold one line, for $want"
    elif ! (cd "$schema_dir" && sha256sum --check --status SHA256SUMS); then
      fail "$schema does not match $schema_dir/SHA256SUMS"
    fi
  fi
fi

# The workflows. A step is the "uses:" line and the lines after it, up to the
# next list item at the same or a lower indent. The action name is matched in
# any letter case, quoted or not, as the forge resolves it.
shas=""
uses_total=0
for wf in .github/workflows/*.yml .github/workflows/*.yaml; do
  [ -e "$wf" ] || continue
  grep -Eiq "^[[:space:]]*(-[[:space:]]+)?uses:[[:space:]]*[\"']?${action}@" "$wf" || continue
  while IFS=$'\t' read -r line ref verify vfile ver; do
    uses_total=$((uses_total + 1))
    if [[ "$ref" =~ ^[0-9a-f]{40}$ ]]; then
      shas+="$ref $wf:$line"$'\n'
    else
      fail "$wf:$line: $action is not pinned to a commit SHA ('$ref')"
    fi
    [ "$verify" = "false" ] ||
      fail "$wf:$line: $action needs 'verify: false'; its own configuration check downloads the schema"
    [ "$vfile" = "$version_file" ] ||
      fail "$wf:$line: $action needs 'version-file: $version_file'"
    [ "$ver" = "-" ] ||
      fail "$wf:$line: $action sets 'version: $ver'; only $version_file names the version"
  done < <(awk -v action="$action" '
    function strip(s) {
      sub(/[[:space:]]+#.*$/, "", s)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", s)
      gsub(/^["\047]|["\047]$/, "", s)
      return s
    }
    function flush() {
      if (inblock) printf "%d\t%s\t%s\t%s\t%s\n", start, ref, verify, vfile, ver
      inblock = 0
    }
    {
      line = $0
      match(line, /^[[:space:]]*/)
      indent = RLENGTH
      body = substr(line, indent + 1)
      if (inblock && body ~ /^- / && indent <= item_indent) flush()
      if (inblock && body != "" && body !~ /^#/ && indent < item_indent) flush()
      if (body ~ /^(- +)?uses:/ && index(tolower(body), action "@")) {
        flush()
        inblock = 1; start = NR
        item_indent = (body ~ /^- /) ? indent : indent - 2
        ref = body; sub(/^.*@/, "", ref); ref = strip(ref)
        verify = "-"; vfile = "-"; ver = "-"
        next
      }
      if (!inblock) next
      if (body ~ /^verify:/)       { v = body; sub(/^verify:/, "", v);       verify = strip(v) }
      if (body ~ /^version-file:/) { v = body; sub(/^version-file:/, "", v); vfile = strip(v) }
      if (body ~ /^version:/)      { v = body; sub(/^version:/, "", v);      ver = strip(v) }
    }
    END { flush() }
  ' "$wf")
  grep -Eq "^[[:space:]]*(-[[:space:]]+)?run:[[:space:]]*bash[[:space:]]+${script}[[:space:]]*$" "$wf" ||
    fail "$wf uses $action but has no step 'run: bash $script'"
done
if [ "$uses_total" -eq 0 ]; then
  fail "no workflow uses $action; this check has nothing to keep in step"
elif [ "$(printf '%s' "$shas" | awk 'NF {print $1}' | sort -u | wc -l)" -gt 1 ]; then
  fail "$action is pinned to more than one commit: $(printf '%s' "$shas" | awk 'NF {printf "%s%s (%s)", sep, $1, $2; sep = ", "}')"
fi

# go.mod: the action takes a golangci-lint version from it before it reads
# the version file.
if [ -f go.mod ] && grep -q 'github.com/golangci/golangci-lint' go.mod; then
  fail "go.mod names golangci-lint; the action would take its version from there, not from $version_file"
fi

# The installed linter, then the configuration itself.
if ! command -v golangci-lint >/dev/null; then
  fail "golangci-lint not found on PATH; install $(cat "$version_file" 2>/dev/null || echo "the version in $version_file")"
elif [ -n "$minor" ]; then
  have=$(golangci-lint version --short 2>/dev/null || true)
  if [[ ! "$have" =~ ^v?${minor//./\\.}\. ]]; then
    fail "installed golangci-lint is '$have' but $version_file names $version; install a v$minor release"
  elif [ -n "$schema" ]; then
    HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 \
      https_proxy=http://127.0.0.1:9 http_proxy=http://127.0.0.1:9 NO_PROXY='' no_proxy='' \
      golangci-lint config verify --schema "$schema" ||
      fail ".golangci.yml does not pass golangci-lint config verify against $schema"
  fi
fi

if [ "$problems" -gt 0 ]; then
  echo "lint-config: $problems problem(s)" >&2
  exit 1
fi
echo "lint-config: ok ($version, $schema, no network)"
