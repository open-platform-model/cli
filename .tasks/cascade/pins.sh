#!/usr/bin/env bash
# pins.sh WORKTREE|<git ref>: print the cli's upstream pins, one TSV row each:
#   <pin-key> <display> <class> <v-version> <labels>
# Phase 2 cascade contract §4.1 and §6.4; workspace RELEASING.md, "The cascade".
# The shared resolver's title and body read these rows to name the moved pins.
# A pin missing at the ref is omitted. Template and fixture versions are this
# repo's own, not upstream pins, so they are not listed.
set -euo pipefail

die() { printf 'pins.sh: %s\n' "$1" >&2; exit 1; }

[ $# -eq 1 ] && [ -n "$1" ] || die "usage: pins.sh WORKTREE|<git ref>"
ref="$1"
cd "$(git rev-parse --show-toplevel)"
if [ "$ref" != WORKTREE ]; then
  git rev-parse --verify --quiet "$ref^{commit}" >/dev/null || die "not a commit: $ref"
fi

# has PATH: the file exists at the ref.
has() {
  if [ "$ref" = WORKTREE ]; then [ -f "$1" ]; else git cat-file -e "$ref:$1" 2>/dev/null; fi
}
# read PATH: the file's content at the ref.
read_file() {
  if [ "$ref" = WORKTREE ]; then cat "$1"; else git show "$ref:$1"; fi
}
# cue_dep_v KEY: the v: of KEY's block in the module.cue on stdin.
cue_dep_v() {
  awk -v k="\"$1\": {" '
    index($0, k) { f = 1; next }
    f && /^[[:space:]]*v:/ { match($0, /"[^"]*"/); print substr($0, RSTART + 1, RLENGTH - 2); exit }
    f && /^[[:space:]]*}/ { exit }'
}
row() { # row KEY DISPLAY VERSION
  [ -n "$3" ] || return 0
  [[ "$3" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]] || die "$1: not a v-prefixed version: $3"
  printf '%s\t%s\tshipped\t%s\t\n' "$1" "$2" "$3"
}

lib=github.com/open-platform-model/library
if has go.mod; then
  row "$lib" library "$(read_file go.mod | awk -v m="$lib" '$1 == m {print $2; exit}')"
fi

# The operator module pin and the operator release it deploys, both from
# internal/operator/pin.go: a module release that deploys the same operator
# still shows as a moved pin (0021:D11:R3).
pin=internal/operator/pin.go
if has "$pin"; then
  row github.com/open-platform-model/opm-operator opm-operator \
    "$(read_file "$pin" | sed -n 's/^const PinnedOperatorVersion = "\(.*\)"$/\1/p')"
  mv=$(read_file "$pin" | sed -n 's/^const PinnedModuleVersion = "\(.*\)"$/\1/p')
  row opmodel.dev/modules/opm_operator@v0 "opm-operator module" "${mv:+v$mv}"
fi

rep=templates/minimal/cue.mod/module.cue
if has "$rep"; then
  row opmodel.dev/catalogs/opm@v4 "opm catalog" "$(read_file "$rep" | cue_dep_v opmodel.dev/catalogs/opm@v4)"
  row opmodel.dev/core@v2 core "$(read_file "$rep" | cue_dep_v opmodel.dev/core@v2)"
fi
