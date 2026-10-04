#!/usr/bin/env bash
# cascade.sh: task deps:cascade. Moves the cli's upstream pins (library, the
# embedded opm-operator, and the opm catalog and core in the templates and test
# trees) to the newest published versions the release cascade allows, in the
# working tree only: no commit, no branch, no push.
#
# Design: workspace RELEASING.md, "The cascade" and "What each repo's task
# moves"; Phase 2 cascade contract §5.2 and §6.4; openspec change
# add-deps-cascade-task, design.md D3 to D9. The shared resolver
# ($CASCADE_RESOLVER) answers every version question.
#
# Exit: 0 when the working tree changed, 3 when there was nothing to do, any
# other code on error. Run it as `task -x deps:cascade`. Progress goes to
# stderr; warnings also go to $(git rev-parse --git-dir)/cascade/warnings.
set -euo pipefail

die() { printf 'cascade: %s\n' "$1" >&2; exit "${2:-1}"; }
say() { printf 'cascade: %s\n' "$*" >&2; }

[ -n "${CASCADE_RESOLVER:-}" ] || die "CASCADE_RESOLVER is not set; run task -x deps:cascade"
case "$CASCADE_RESOLVER" in /*) ;; *) die "CASCADE_RESOLVER must be absolute" ;; esac
RES=$CASCADE_RESOLVER

cd "$(git rev-parse --show-toplevel)"

LIB=github.com/open-platform-model/library
OP=github.com/open-platform-model/opm-operator
CAT=opmodel.dev/catalogs/opm@v4
CORE=opmodel.dev/core@v2
POD=testing.opmodel.dev/modules/cli/podinfo@v0
REP=templates/minimal # the representative file for the catalog (design.md D2)
KIND=hack/kind-platform.yaml
CUE_PIN_SOURCE=.github/workflows/pr.yml # design.md D8

# The cue.mod files the consistent set covers, in writing order (design.md D3).
# An explicit list, never a glob: .claude/worktrees/ and bin/ hold other modules.
CUE_DIRS=(
  templates/minimal templates/standard templates/advanced
  hack/platform
  tests/fixtures/modules/podinfo
  examples
  tests/e2e/testdata/operator-owned
  internal/instinit/testdata/initvalues
  internal/workflow/render/testdata/skip-unprovided
  tests/e2e/testdata/duplicate-identities
  tests/integration/module-apply/testdata
  tests/fixtures/valid/simple-module
  tests/fixtures/valid/module-with-debug-values
)
# The podinfo fixture's consumers (design.md D7).
CONSUMERS=(examples tests/e2e/testdata/operator-owned)
PODDIR=tests/fixtures/modules/podinfo
# Version-advance modules and their coordinates (design.md D6).
ADV_DIRS=(templates/minimal templates/standard templates/advanced "$PODDIR")
declare -A ADV_COORD=(
  [templates/minimal]=opmodel.dev/templates/minimal@v1
  [templates/standard]=opmodel.dev/templates/standard@v1
  [templates/advanced]=opmodel.dev/templates/advanced@v1
  [$PODDIR]=$POD
)

# ---------------------------------------------------------------------------
# Setup (contract §5.2 rules 1 to 4).

snapshot() {
  { git status --porcelain --untracked-files=all
    git diff HEAD --binary
    git ls-files -z --others --exclude-standard | xargs -0 -r sha256sum
  } | sha256sum
}
START=""
if [ "${CASCADE_ALLOW_DIRTY:-}" = 1 ]; then
  START=$(snapshot)
elif [ -n "$(git status --porcelain --untracked-files=all)" ]; then
  die "the working tree is not clean; commit or stash first, or set CASCADE_ALLOW_DIRTY=1"
fi

STATE="$(git rev-parse --absolute-git-dir)/cascade"
mkdir -p "$STATE"
: >"$STATE/warnings"
export CASCADE_WARNINGS="$STATE/warnings"

# warn KEY MESSAGE: a warning for the PR body (contract §2.2 format).
warn() {
  printf 'cascade: warning: %s\n' "$2" >&2
  printf '%s\t%s\n' "$1" "$2" >>"$CASCADE_WARNINGS"
}

REGISTRY='testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export CUE_REGISTRY="$REGISTRY" OPM_REGISTRY="$REGISTRY" GOWORK=off

# ---------------------------------------------------------------------------
# Helpers.

# r ARGS...: run the resolver. Sets OUT to its stdout and RC to 0 (move, yes)
# or 3 (stay, no); any other exit ends the task with that code, so a failure
# never turns into a fallback version.
r() {
  RC=0
  OUT=$("$RES" "$@") || RC=$?
  case "$RC" in
    0 | 3) ;;
    *) die "resolver \`$1\` failed (exit $RC): $*" "$RC" ;;
  esac
}

# newest KEY ARGS...: r newest ARGS..., adding --expect when CASCADE_EXPECT
# names KEY (contract §5.4). Sets NEW to the target, or empty to stay.
newest() {
  local key="$1" kv e="" words=()
  shift
  read -r -a words <<<"${CASCADE_EXPECT:-}"
  for kv in "${words[@]}"; do
    if [ "${kv%%=*}" = "$key" ]; then e=${kv#*=}; fi
  done
  if [ -n "$e" ]; then r newest "$@" --expect "$e"; else r newest "$@"; fi
  NEW=""
  if [ "$RC" = 0 ]; then NEW=$OUT; fi
}

# vcmp A B: sets CMP to -1, 0 or 1 by SemVer precedence. Equal strings need
# no resolver call; anything else asks the resolver, which owns the ordering.
vcmp() {
  if [ "$1" = "$2" ]; then
    CMP=0
  else
    r semver-cmp "$1" "$2"
    CMP=$OUT
  fi
  case "$CMP" in -1 | 0 | 1) ;; *) die "semver-cmp \`$1\` \`$2\` answered '$CMP'" ;; esac
}

# pin_of C: sets X to the core version catalog C pins (cached).
declare -A PINOF=()
pin_of() {
  if [ -z "${PINOF[$1]:-}" ]; then
    r pin-of "$CAT" "$1" "$CORE"
    [ "$RC" = 0 ] || die "catalog \`$1\` pins no \`$CORE\`"
    PINOF[$1]=$OUT
  fi
  X=${PINOF[$1]}
}

# frozen FILE KEY: true when .cascade-frozen freezes KEY for FILE (cached).
declare -A FROZEN=()
frozen() {
  local k="$1|$2"
  if [ -z "${FROZEN[$k]:-}" ]; then
    r is-frozen "$1" "$2" --repo-root .
    FROZEN[$k]=$RC
  fi
  [ "${FROZEN[$k]}" = 0 ]
}

# published COORD VERSION: true when the resolver says that exact cue
# module version is published (cached).
declare -A PUBLISHED=()
published() {
  local k="$1|$2"
  if [ -z "${PUBLISHED[$k]:-}" ]; then
    r published cue "$1" "$2"
    PUBLISHED[$k]=$RC
  fi
  [ "${PUBLISHED[$k]}" = 0 ]
}

# cue_dep_v FILE KEY: the v: of KEY in FILE's deps, or nothing.
cue_dep_v() {
  awk -v k="\"$2\": {" '
    index($0, k) { f = 1; next }
    f && /^[[:space:]]*v:/ { match($0, /"[^"]*"/); print substr($0, RSTART + 1, RLENGTH - 2); exit }
    f && /^[[:space:]]*}/ { exit }' "$1"
}

# cue_deps FILE: every dep in FILE as "key version" lines.
cue_deps() {
  awk '
    /^[[:space:]]*"[^"]+": \{/ { match($0, /"[^"]+"/); k = substr($0, RSTART + 1, RLENGTH - 2); next }
    k != "" && /^[[:space:]]*v:/ { match($0, /"[^"]*"/); print k, substr($0, RSTART + 1, RLENGTH - 2); k = "" }' "$1"
}

# rewrite FILE AFTER_PATTERN VERSION: rewrite the first quoted value on the
# first `v:` or `version:` line after the line containing AFTER_PATTERN, as
# text, and check the result.
rewrite() {
  local f="$1" tmp
  tmp=$(mktemp "$STATE/edit.XXXXXX")
  awk -v k="$2" -v nv="$3" '
    !done && index($0, k) { f = 1; print; next }
    f && /^[[:space:]]*(v|version):/ { sub(/"[^"]*"/, "\"" nv "\""); f = 0; done = 1 }
    { print }' "$f" >"$tmp"
  cat "$tmp" >"$f"
  rm -f "$tmp"
}
set_cue_dep_v() { # FILE KEY VERSION
  rewrite "$1" "\"$2\": {" "$3"
  [ "$(cue_dep_v "$1" "$2")" = "$3" ] || die "could not rewrite \`$2\` in \`$1\`"
}
kind_version() {
  awk -v k="$CAT:" '
    index($0, k) { f = 1; next }
    f && /^[[:space:]]*version:/ { match($0, /"[^"]*"/); print substr($0, RSTART + 1, RLENGTH - 2); exit }' "$KIND"
}

# lang_line FILE: the language.version line of a module.cue.
lang_line() { awk '/^language:/ { f = 1 } f && /version:/ { print; exit }' "$1"; }

# id_version: the bare ^Version: value of the identity file on stdin.
id_version() { sed -n 's/^Version: "\(.*\)"$/\1/p'; }

# f_changed M FDIR IFILE: exit 0 when FDIR differs from M in any path other than
# IFILE, or when IFILE differs from M outside its ^Version: line; exit 1 otherwise.
# Copied as is from Phase 2 cascade contract §5.2 rule 11.
f_changed() {
  local m="$1" d="$2" i="$3" p
  while IFS= read -r p; do
    [ "$p" = "$i" ] || return 0
  done < <({ git diff --name-only "$m" -- "$d"
             git ls-files --others --exclude-standard -- "$d"; } | sort -u)
  git cat-file -e "$m:$i" 2>/dev/null || return 0
  if diff -q <(git show "$m:$i" | grep -v '^Version:') \
             <(grep -v '^Version:' "$i") >/dev/null; then
    return 1
  fi
  return 0
}

# go_requires: every require line of go.mod as "path version".
go_requires() {
  awk '/^require \($/ { f = 1; next } f && /^\)/ { f = 0 } f && NF >= 2 { print $1, $2 }
       /^require [^(]/ { print $2, $3 }' go.mod
}

report() { # report DISPLAY FROM TO
  if [ -n "$3" ] && [ "$3" != "$2" ]; then say "$1 $2 -> $3"; else say "$1 $2 (current)"; fi
}

# ---------------------------------------------------------------------------
# Phase A: resolve. Every call that decides a target runs here, before any
# edit, so an error leaves the tree unchanged (contract §5.2 rule 5).

r check-files --repo-root .
[ "$RC" = 0 ] || die "check-files answered $RC"

LIB_CUR=$(awk -v m="$LIB" '$1 == m { print $2; exit }' go.mod)
[ -n "$LIB_CUR" ] || die "go.mod does not require \`$LIB\`"
newest "$LIB" go "$LIB" --current "$LIB_CUR" --repo-root .
LIB_T=$NEW
report library "$LIB_CUR" "$LIB_T"

OP_CUR=$(sed -n 's/^const PinnedOperatorVersion = "\(.*\)"$/\1/p' internal/operator/manifest.go)
[ -n "$OP_CUR" ] || die "internal/operator/manifest.go has no PinnedOperatorVersion"
newest "$OP" release opm-operator --asset install.yaml --current "$OP_CUR" --repo-root .
OP_T=$NEW
report opm-operator "$OP_CUR" "$OP_T"

CAT_REP=$(cue_dep_v "$REP/cue.mod/module.cue" "$CAT")
[ -n "$CAT_REP" ] || die "\`$REP/cue.mod/module.cue\` pins no \`$CAT\`"
newest "$CAT" cue "$CAT" --current "$CAT_REP" --repo-root .
T=${NEW:-$CAT_REP}
report "opm catalog" "$CAT_REP" "$T"

r hold "$CORE" --repo-root .
HOLD=""
if [ "$RC" = 0 ]; then HOLD=$OUT; fi

# The consistent set, per file (design.md D3; contract §5.2 rules 7 and 8).
declare -A CAT_CUR=() CAT_T=() CORE_T=() MOVES=() FROZEN_KEYS=()
plan_file() {
  local d="$1" m="$1/cue.mod/module.cue" cc kc cand kt held=0 k
  [ -f "$m" ] || die "\`$m\` does not exist"
  cc=$(cue_dep_v "$m" "$CAT")
  kc=$(cue_dep_v "$m" "$CORE")
  [ -n "$kc" ] || die "\`$m\` pins no \`$CORE\`"
  CAT_CUR[$d]=$cc

  # Catalog: up to T, never lowered, never past a freeze. A file without a
  # catalog takes the representative's catalog after its move.
  if [ -n "$cc" ]; then
    cand=$cc
    vcmp "$cc" "$T"
    if [ "$CMP" = -1 ] && ! frozen "$m" "$CAT"; then cand=$T; fi
  else
    cand=${CAT_T[$REP]}
  fi

  # Core follows the catalog's own pin, under an in-date hold (contract §9.11).
  pin_of "$cand"
  if [ -n "$HOLD" ]; then
    vcmp "$X" "$HOLD"
    if [ "$CMP" = 1 ] && [ -n "$cc" ] && [ "$cand" != "$cc" ]; then
      warn "$CAT" "catalog \`$cand\` needs core \`$X\`, above the hold \`$HOLD\`; catalog held too"
      cand=$cc
      pin_of "$cand"
      vcmp "$X" "$HOLD"
    fi
    if [ "$CMP" = 1 ]; then
      warn "$CORE" "core \`$X\` that catalog \`$cand\` pins is above the hold \`$HOLD\`; core held"
      held=1
    fi
  fi
  kt=$kc
  if [ "$held" = 0 ]; then
    vcmp "$X" "$kc"
    if [ "$CMP" = 1 ] && ! frozen "$m" "$CORE"; then kt=$X; fi
    if [ "$CMP" = -1 ]; then
      warn "$CORE" "core \`$kc\` is ahead of the core \`$X\` that catalog \`$cand\` pins"
    fi
  fi
  CAT_T[$d]=$cand
  CORE_T[$d]=$kt

  MOVES[$d]=""
  if [ -n "$cc" ] && [ "$cand" != "$cc" ]; then MOVES[$d]+=" ${CAT%@*}@$cand"; fi
  if [ "$kt" != "$kc" ]; then MOVES[$d]+=" ${CORE%@*}@$kt"; fi
  # Every key frozen for a file the task touches, for the check after tidy.
  if [ -n "${MOVES[$d]}" ]; then
    FROZEN_KEYS[$d]=""
    for k in $(cue_deps "$m" | awk '$1 ~ /^(testing\.)?opmodel\.dev\// { print $1 }'); do
      if frozen "$m" "$k"; then FROZEN_KEYS[$d]+=" $k"; fi
    done
  fi
}
for d in "${CUE_DIRS[@]}"; do plan_file "$d"; done

KIND_CUR=$(kind_version)
[ -n "$KIND_CUR" ] || die "\`$KIND\` has no version under \`$CAT\`"
# The kind file follows hack/platform's catalog up, never down (contract v1.1
# clarification C8): a kind version above it stays, with a warning.
KIND_T=""
vcmp "v$KIND_CUR" "${CAT_T[hack/platform]}"
if [ "$CMP" = -1 ] && ! frozen "$KIND" "$CAT"; then
  KIND_T=${CAT_T[hack/platform]#v}
elif [ "$CMP" = 1 ]; then
  warn "$CAT" "\`$KIND\` \`$KIND_CUR\` is ahead of \`hack/platform\` \`${CAT_T[hack/platform]}\`; left as is"
fi

# Version advances, once per PR (design.md D6; contract §5.2 rule 11).
BASE_REF=${CASCADE_BASE:-origin/main}
M=$(git merge-base "$BASE_REF" HEAD) || die "no merge base between \`$BASE_REF\` and HEAD"
declare -A ADV_B=() ADV_T=() ADV_FINAL=()
for d in "${ADV_DIRS[@]}"; do
  i="$d/identity/identity.cue"
  cur=$(id_version <"$i")
  B=$(git show "$M:$i" | id_version)
  [ -n "$cur" ] && [ -n "$B" ] || die "no Version in \`$i\` (worktree or merge base)"
  ADV_B[$d]=$B
  tgt=$B
  if f_changed "$M" "$d" "$i" || [ -n "${MOVES[$d]:-}" ]; then
    if published "${ADV_COORD[$d]}" "v$B"; then
      r next-patch "v$B"
      tgt=${OUT#v}
    fi
  fi
  ADV_FINAL[$d]=$cur
  vcmp "v$tgt" "v$cur"
  if [ "$CMP" = 1 ]; then
    ADV_T[$d]=$tgt
    ADV_FINAL[$d]=$tgt
  elif [ "$CMP" = -1 ]; then
    warn "${ADV_COORD[$d]}" "\`$i\` declares \`$cur\`, above the cascade target \`$tgt\`; left as is"
  fi
done

# The podinfo consumers (design.md D7).
POD_FINAL=v${ADV_FINAL[$PODDIR]}
declare -A SWAP=() REPIN=()
for c in "${CONSUMERS[@]}"; do
  m="$c/cue.mod/module.cue"
  pc=$(cue_dep_v "$m" "$POD")
  [ -n "$pc" ] || die "\`$m\` pins no \`$POD\`"
  swap=""
  # A consumer whose catalog or core moves runs get and tidy, which need its
  # podinfo published. On a branch that already advanced podinfo, point it at
  # the merge base's published podinfo for that step.
  if [ -n "${MOVES[$c]}" ] && ! published "$POD" "$pc"; then swap=v${ADV_B[$PODDIR]}; fi
  if [ -z "$swap" ] && [ "$pc" = "$POD_FINAL" ]; then continue; fi
  if frozen "$m" "$POD"; then continue; fi # a frozen pin is never edited, not even for a moment
  if [ -n "$swap" ]; then
    published "$POD" "$swap" ||
      die "\`$m\` pins the unpublished \`$POD\` \`$pc\`, and the merge base's \`$swap\` is not published either"
    SWAP[$c]=$swap
  fi
  REPIN[$c]=$POD_FINAL
done

# language.version of every moved CUE upstream against the cue that PR CI
# installs, never the cue on PATH (design.md D8; contract §5.2 rule 10).
CUE_PIN=$(awk 'match($0, /cuelang\.org\/go\/cmd\/cue@v[0-9]+\.[0-9]+\.[0-9]+/) {
  print substr($0, RSTART + 23, RLENGTH - 23); exit }' "$CUE_PIN_SOURCE")
declare -A LANG_DONE=()
lang_check() { # lang_check MODULE VERSION
  [ -z "${LANG_DONE[$1|$2]:-}" ] || return 0
  LANG_DONE[$1|$2]=1
  if [ -z "$CUE_PIN" ]; then
    warn - "no \`cuelang.org/go/cmd/cue\` version in \`$CUE_PIN_SOURCE\`; \`language.version\` not checked"
    return 0
  fi
  local lang
  r language-of "$1" "$2"
  [ "$RC" = 0 ] || return 0
  lang=$OUT
  vcmp "$lang" "$CUE_PIN"
  if [ "$CMP" = 1 ]; then
    warn "$1" "\`$1\` \`$2\` declares \`language.version\` \`$lang\`, newer than the cue \`$CUE_PIN\` that \`$CUE_PIN_SOURCE\` installs"
  fi
}
# Every pin that differs from the merge base once this run is done, not only
# this run's moves: the warnings file starts empty on every run, and the PR
# body lists every pin moved against the base (contract §6.4 step 6).
base_dep_v() { # base_dep_v FILE KEY: the v: of KEY in FILE at the merge base
  git show "$M:$1" 2>/dev/null | cue_dep_v /dev/stdin "$2" || true
}
for d in "${CUE_DIRS[@]}"; do
  m="$d/cue.mod/module.cue"
  if [ -n "${CAT_CUR[$d]}" ] && [ "${CAT_T[$d]}" != "$(base_dep_v "$m" "$CAT")" ]; then lang_check "$CAT" "${CAT_T[$d]}"; fi
  if [ "${CORE_T[$d]}" != "$(base_dep_v "$m" "$CORE")" ]; then lang_check "$CORE" "${CORE_T[$d]}"; fi
done

# Anything to do?
work=0
if [ -n "$LIB_T" ] || [ -n "$OP_T" ] || [ -n "$KIND_T" ]; then work=1; fi
for d in "${CUE_DIRS[@]}"; do if [ -n "${MOVES[$d]}" ]; then work=1; fi; done
if [ "${#ADV_T[@]}" -gt 0 ] || [ "${#REPIN[@]}" -gt 0 ]; then work=1; fi

finish() {
  if [ -n "$START" ]; then
    if [ "$(snapshot)" != "$START" ]; then exit 0; fi
    exit 3
  fi
  if [ -n "$(git status --porcelain --untracked-files=all)" ]; then exit 0; fi
  exit 3
}
if [ "$work" = 0 ]; then
  say "nothing to move"
  finish
fi

# ---------------------------------------------------------------------------
# Phase B: tools, from the unmodified tree, so a library move that breaks
# compilation cannot stop the task from producing its diff (contract rule 11).

command -v cue >/dev/null || die "cue is not on PATH"
mkdir -p "$STATE/bin"
go build -o "$STATE/bin/opm" ./cmd/opm

# ---------------------------------------------------------------------------
# Phase C: edit (contract §5.2 rule 12 and §6.4). A failure here exits
# non-zero and may leave a partly edited tree; callers discard it.

# 1. Shipped: library, then the operator embed.
if [ -n "$LIB_T" ]; then
  before=$(go_requires)
  go get "$LIB@$LIB_T"
  go mod tidy
  after=$(go_requires)
  while read -r p v; do
    [ "$p" != "$LIB" ] || continue
    old=$(awk -v p="$p" '$1 == p { print $2 }' <<<"$before")
    if [ "$old" != "$v" ]; then
      warn "$LIB" "\`go mod tidy\` moved \`$p\` from \`${old:-none}\` to \`$v\`"
    fi
  done <<<"$after"
fi
if [ -n "$OP_T" ]; then
  task -x operator:sync VERSION="$OP_T" >&2
fi

# 2. Catalog and core: templates first, then the test trees (CUE_DIRS order).
for d in "${CUE_DIRS[@]}"; do
  [ -n "${MOVES[$d]}" ] || continue
  m="$d/cue.mod/module.cue"
  if [ -n "${SWAP[$d]:-}" ]; then set_cue_dep_v "$m" "$POD" "${SWAP[$d]}"; fi
  before=$(cue_deps "$m")
  lang_before=$(lang_line "$m")
  say "$d:${MOVES[$d]}"
  # shellcheck disable=SC2086 # MOVES holds one word per moved key
  (cd "$d" && cue mod get ${MOVES[$d]} >&2 && cue mod tidy >&2)
  [ "$(lang_line "$m")" = "$lang_before" ] || die "\`cue mod\` changed \`language.version\` in \`$m\`"
  while read -r k v; do
    old=$(awk -v k="$k" '$1 == k { print $2 }' <<<"$before")
    [ "$old" != "$v" ] || continue
    case " ${FROZEN_KEYS[$d]} " in
      *" $k "*) die "\`cue mod tidy\` raised the frozen \`$k\` in \`$m\` from \`$old\` to \`$v\`; freeze the whole module, or hold the upstream" ;;
    esac
    case "$k" in
      "$CAT") [ "$v" = "${CAT_T[$d]}" ] || warn "$k" "\`cue mod tidy\` set \`$k\` in \`$m\` to \`$v\`, not the target \`${CAT_T[$d]}\`" ;;
      "$CORE") [ "$v" = "${CORE_T[$d]}" ] || warn "$k" "\`cue mod tidy\` set \`$k\` in \`$m\` to \`$v\`, not the target \`${CORE_T[$d]}\`" ;;
      *) warn "$k" "\`cue mod tidy\` moved \`$k\` in \`$m\` from \`${old:-none}\` to \`$v\`" ;;
    esac
  done < <(cue_deps "$m")
done
if [ -n "$KIND_T" ]; then
  say "$KIND: $KIND_CUR -> $KIND_T"
  rewrite "$KIND" "$CAT:" "$KIND_T"
  [ "$(kind_version)" = "$KIND_T" ] || die "could not rewrite the catalog version in \`$KIND\`"
fi

# 3. Version advances.
for d in "${ADV_DIRS[@]}"; do
  [ -n "${ADV_T[$d]:-}" ] || continue
  "$STATE/bin/opm" module version set "${ADV_T[$d]}" "$d" >&2
done

# 4. The podinfo consumers follow the fixture, as text, never followed by get
# or tidy: the new podinfo is published only when this PR merges.
for c in "${CONSUMERS[@]}"; do
  [ -n "${REPIN[$c]:-}" ] || continue
  set_cue_dep_v "$c/cue.mod/module.cue" "$POD" "${REPIN[$c]}"
done

# 5. Docs bundles: a warning, never a hold (design.md D9; contract §9.5). It
# runs when library or the operator differs from the merge base, not only when
# this run moved it, so a later run on the branch keeps the warning.
LIB_BASE=$(git show "$M:go.mod" | awk -v m="$LIB" '$1 == m { print $2; exit }')
OP_BASE=$(git show "$M:internal/operator/manifest.go" | sed -n 's/^const PinnedOperatorVersion = "\(.*\)"$/\1/p')
if [ "${LIB_T:-$LIB_CUR}" != "$LIB_BASE" ] || [ "${OP_T:-$OP_CUR}" != "$OP_BASE" ]; then
  if pins=$(go run ./hack/docskit-dump pins 2>"$STATE/docskit.err"); then
    # Outside a process substitution, so a changed output shape stops the task.
    entries=$(jq -er '.pins | to_entries[] | "\(.key)\t\(.value)"' <<<"$pins") ||
      die "\`hack/docskit-dump pins\` printed no \`.pins\` entries"
    while IFS=$'\t' read -r project pin; do
      [ -n "$project" ] || continue
      r published oci "open-platform-model/docs/$project" "$pin"
      if [ "$RC" = 3 ]; then
        case "$project" in
          library) key=$LIB ;;
          opm-operator) key=$OP ;;
          core) key=$CORE ;;
          *) key=- ;;
        esac
        warn "$key" "docs bundle for \`$project\` \`$pin\` is not published; G1 will fail the next release PR until it is"
      fi
    done <<<"$entries"
  else
    warn - "\`hack/docskit-dump pins\` did not build on the edited tree; docs bundles not checked"
  fi
fi

finish
