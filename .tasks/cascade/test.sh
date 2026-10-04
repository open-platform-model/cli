#!/usr/bin/env bash
# test.sh: the scenario test of task deps:cascade (Phase 2 cascade contract §8;
# openspec change add-deps-cascade-task, design.md D10). Every scenario runs in
# a sandbox copy of the tree under a temporary directory, against the stub
# resolver in testdata/, and never touches this checkout.
#
#   CASCADE_TEST_SET=offline  pre-checks plus the scenarios that need no GHCR,
#                             Go proxy or GitHub access
#   CASCADE_TEST_SET=all      (default) every scenario
#   CASCADE_RESOLVER_REAL     the absolute path of the real cascade-resolve.sh;
#                             the title and body scenario runs only when set
#
# Prints PASS, FAIL or SKIP per check; exits 0 when nothing failed, else 1.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
HERE="$ROOT/.tasks/cascade"
STUB="$HERE/testdata/stub-resolve.sh"
STUB_SHA=970130f7d55c07f5b86d4f5b6f392330427ff923eb34f93553656bcd4b893d9c
SET="${CASCADE_TEST_SET:-all}"
case "$SET" in
  offline | all) ;;
  *) printf 'test.sh: CASCADE_TEST_SET must be offline or all, not %s\n' "$SET" >&2; exit 1 ;;
esac
# The test needs no resolver beyond its stub. A CASCADE_RESOLVER_REAL that is
# set must name an executable by absolute path, so a missing checkout fails
# here instead of turning S5 into a quiet failure.
if [ -n "${CASCADE_RESOLVER_REAL:-}" ]; then
  case "$CASCADE_RESOLVER_REAL" in /*) ;; *) printf 'test.sh: CASCADE_RESOLVER_REAL must be absolute\n' >&2; exit 1 ;; esac
  [ -x "$CASCADE_RESOLVER_REAL" ] ||
    { printf 'test.sh: CASCADE_RESOLVER_REAL is not executable: %s\n' "$CASCADE_RESOLVER_REAL" >&2; exit 1; }
fi

FAILED=0
pass() { printf 'PASS %s\n' "$1"; }
fail() { printf 'FAIL %s: %s\n' "$1" "$2"; FAILED=1; }
skip() { printf 'SKIP %s: %s\n' "$1" "$2"; }

TMP=$(mktemp -d)
trap 'chmod -R u+w "$TMP" 2>/dev/null || true; rm -rf "$TMP"' EXIT
CHECKOUT_BEFORE=$(git -C "$ROOT" status --porcelain --untracked-files=all)

# The scenarios run hermetically: only the variables each one sets reach the task.
unset CASCADE_EXPECT CASCADE_ALLOW_DIRTY CASCADE_BASE CASCADE_STUB_TABLE CASCADE_STUB_LOG \
  CASCADE_WARNINGS CASCADE_SOURCE CASCADE_TAGS CASCADE_NOTES_FILE
export CASCADE_TODAY=2026-10-03

# g DIR ARGS...: git in a sandbox, with a fixed identity and no hooks or signing.
# Auto gc is off: the first commit holds thousands of loose objects, and a
# detached gc would still be writing packs when the trap removes the sandbox.
g() {
  local d="$1"; shift
  git -C "$d" -c user.name=cascade-test -c user.email=cascade-test@example.invalid \
    -c commit.gpgsign=false -c core.hooksPath=/dev/null -c gc.auto=0 -c maintenance.auto=false "$@"
}

# sandbox NAME: copy the tree (tracked and unignored files, as on disk) into
# $TMP/NAME/r, commit it as "base", and print the directory.
sandbox() {
  local d="$TMP/$1/r"
  mkdir -p "$d"
  (cd "$ROOT" && git ls-files -z --cached --others --exclude-standard | tar --null -T - -cf -) |
    tar -xf - -C "$d"
  g "$d" init -q
  g "$d" add -A
  g "$d" commit -q -m base
  printf '%s\n' "$d"
}

# commit_setup DIR: commit the scenario's setup edits and point CASCADE_BASE
# at that commit, by SHA, for every run in the scenario (contract §8 step 4).
commit_setup() {
  g "$1" add -A
  g "$1" commit -q --allow-empty -m setup
  CASCADE_BASE=$(g "$1" rev-parse HEAD)
  export CASCADE_BASE
}

LIB=github.com/open-platform-model/library
OP=github.com/open-platform-model/opm-operator
CAT=opmodel.dev/catalogs/opm@v4
CORE=opmodel.dev/core@v2
POD=testing.opmodel.dev/modules/cli/podinfo@v0
PODDIR=tests/fixtures/modules/podinfo
ADV_DIRS=(templates/minimal templates/standard templates/advanced "$PODDIR")
declare -A ADV_COORD=(
  [templates/minimal]=opmodel.dev/templates/minimal@v1
  [templates/standard]=opmodel.dev/templates/standard@v1
  [templates/advanced]=opmodel.dev/templates/advanced@v1
  [$PODDIR]=$POD
)

# id_version FILE: the bare Version of an identity file.
id_version() { sed -n 's/^Version: "\(.*\)"$/\1/p' "$1"; }
# cue_dep_v FILE KEY: the v: of KEY in a module.cue's deps.
cue_dep_v() {
  awk -v k="\"$2\": {" '
    index($0, k) { f = 1; next }
    f && /^[[:space:]]*v:/ { match($0, /"[^"]*"/); print substr($0, RSTART + 1, RLENGTH - 2); exit }
    f && /^[[:space:]]*}/ { exit }' "$1"
}

# current_rows DIR: the stub rows for the tree in DIR as it is (contract §8):
# a newest row per pin, the pin-of row for its catalog, and a published row
# per version-advance module at its declared version. Read before any setup.
current_rows() {
  local d="$1" key v cat="" core="" a
  while IFS=$'\t' read -r key _ _ v _; do
    case "$key" in
      "$LIB") printf 'newest\tgo\t%s\t%s\n' "$key" "$v" ;;
      "$OP") printf 'newest\trelease\topm-operator\t%s\n' "$v" ;;
      *) printf 'newest\tcue\t%s\t%s\n' "$key" "$v" ;;
    esac
    case "$key" in "$CAT") cat=$v ;; "$CORE") core=$v ;; esac
  done < <(cd "$d" && .tasks/cascade/pins.sh WORKTREE)
  printf 'pin-of\t%s\t%s\t%s\t%s\n' "$CAT" "$cat" "$CORE" "$core"
  for a in "${ADV_DIRS[@]}"; do
    printf 'published\tcue\t%s\tv%s\n' "${ADV_COORD[$a]}" "$(id_version "$d/$a/identity/identity.cue")"
  done
}

# run DIR TABLE LOG: task -x deps:cascade in DIR against the stub; sets RUN_RC.
# Its output goes to DIR/../out.N; the last run's file is in RUN_OUT.
RUNS=0
run() {
  RUNS=$((RUNS + 1))
  RUN_OUT="$1/../out.$RUNS"
  RUN_RC=0
  (cd "$1" && CASCADE_RESOLVER="$STUB" CASCADE_STUB_TABLE="$2" CASCADE_STUB_LOG="$3" \
    task -x deps:cascade) >"$RUN_OUT" 2>&1 || RUN_RC=$?
}
# normalized LOG: the stub log with every version replaced by V, sorted.
normalized() { sed -E 's/v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?/V/g' "$1" | LC_ALL=C sort; }
# clean DIR: the sandbox has no change against its last commit.
clean() { [ -z "$(g "$1" status --porcelain --untracked-files=all)" ]; }
# warned DIR TEXT: the sandbox's warnings file has a line containing TEXT.
warned() { grep -qF -- "$2" "$1/.git/cascade/warnings" 2>/dev/null; }
# why: the tail of the last run's output, for a FAIL line.
why() { tail -n 3 "$RUN_OUT" | tr '\n' ' '; }

# ---------------------------------------------------------------------------
# Pre-checks (both sets).

if [ "$(sha256sum "$STUB" | cut -d' ' -f1)" = "$STUB_SHA" ]; then
  pass "stub checksum"
else
  fail "stub checksum" "$STUB differs from Phase 2 cascade contract §7 (sha256 $STUB_SHA)"
fi

pre=$(sandbox pins)
if (cd "$pre" && diff <(.tasks/cascade/pins.sh WORKTREE) <(.tasks/cascade/pins.sh HEAD) >/dev/null); then
  if [ "$(cd "$pre" && .tasks/cascade/pins.sh WORKTREE | wc -l)" -eq 4 ]; then
    pass "pins.sh WORKTREE equals HEAD"
  else
    fail "pins.sh WORKTREE equals HEAD" "pins.sh does not print four pins"
  fi
else
  fail "pins.sh WORKTREE equals HEAD" "the two reads differ on a clean copy"
fi

# ---------------------------------------------------------------------------
# Offline scenarios (both sets): no GHCR, Go proxy or GitHub access.

# S1 no-op: the tree as it is, current rows only.
d=$(sandbox s1)
current_rows "$d" >"$TMP/s1/table"
commit_setup "$d"
run "$d" "$TMP/s1/table" "$TMP/s1/log"
if [ "$RUN_RC" != 3 ]; then
  fail "S1 no-op" "exit $RUN_RC, want 3: $(why)"
elif ! clean "$d"; then
  fail "S1 no-op" "the tree changed"
elif ! diff <(normalized "$TMP/s1/log") "$HERE/testdata/s1-calls.txt" >"$TMP/s1/diff"; then
  fail "S1 no-op" "the resolver calls differ from testdata/s1-calls.txt: $(tr '\n' ' ' <"$TMP/s1/diff")"
elif grep '^newest ' "$TMP/s1/log" | grep -v -e '--current ' >/dev/null ||
  grep '^newest ' "$TMP/s1/log" | grep -v -e '--repo-root ' >/dev/null; then
  fail "S1 no-op" "a newest call lacks --current or --repo-root"
else
  pass "S1 no-op"
fi

# S3 error: the first pin the task resolves (library) answers with an error.
d=$(sandbox s3)
{ printf 'newest\tgo\t%s\tERROR\n' "$LIB"; current_rows "$d"; } >"$TMP/s3/table"
commit_setup "$d"
run "$d" "$TMP/s3/table" "$TMP/s3/log"
if [ "$RUN_RC" = 0 ] || [ "$RUN_RC" = 3 ]; then
  fail "S3 error" "exit $RUN_RC, want neither 0 nor 3"
elif ! clean "$d"; then
  fail "S3 error" "the tree changed"
elif [ "$(tail -n 1 "$TMP/s3/log" | cut -d' ' -f1-3)" != "newest go $LIB" ]; then
  fail "S3 error" "the task went on after the failing call: $(tail -n 1 "$TMP/s3/log")"
elif [ -e "$d/.git/cascade/bin/opm" ]; then
  fail "S3 error" "phase B ran after a phase A error"
else
  pass "S3 error"
fi

# S6 dirty tree: an untracked file, CASCADE_ALLOW_DIRTY unset.
d=$(sandbox s6)
current_rows "$d" >"$TMP/s6/table"
commit_setup "$d"
printf 'x\n' >"$d/untracked-file"
run "$d" "$TMP/s6/table" "$TMP/s6/log"
if [ "$RUN_RC" != 1 ]; then
  fail "S6 dirty tree" "exit $RUN_RC, want 1"
elif [ "$(g "$d" status --porcelain --untracked-files=all)" != "?? untracked-file" ]; then
  fail "S6 dirty tree" "the tree changed beyond the untracked file"
else
  pass "S6 dirty tree"
fi

# S7 core ahead: one file's core above the core its catalog pins stays, with a
# warning (contract §9.10).
d=$(sandbox s7)
current_rows "$d" >"$TMP/s7/table"
f="$d/tests/integration/module-apply/testdata/cue.mod/module.cue"
perl -0pi -e 's/("opmodel\.dev\/core\@v2": \{\n\s*v:\s*)"[^"]+"/$1"v2.0.0-beta.99"/' "$f"
commit_setup "$d"
run "$d" "$TMP/s7/table" "$TMP/s7/log"
if [ "$(cue_dep_v "$f" "$CORE")" != v2.0.0-beta.99 ]; then
  fail "S7 core ahead" "the setup edit did not apply"
elif [ "$RUN_RC" != 3 ]; then
  fail "S7 core ahead" "exit $RUN_RC, want 3: $(why)"
elif ! clean "$d"; then
  fail "S7 core ahead" "the tree changed"
elif ! warned "$d" "core \`v2.0.0-beta.99\` is ahead of the core"; then
  fail "S7 core ahead" "no core-ahead warning"
else
  pass "S7 core ahead"
fi

# S8 a hold on core holds the catalog: the newest catalog pins a core above an
# in-date hold at the tree's core (contract §9.11).
d=$(sandbox s8)
tree_core=$(cue_dep_v "$d/templates/minimal/cue.mod/module.cue" "$CORE")
{ printf 'newest\tcue\t%s\tv4.99.0\n' "$CAT"
  printf 'pin-of\t%s\tv4.99.0\t%s\tv2.0.0-beta.99\n' "$CAT" "$CORE"
  current_rows "$d"; } >"$TMP/s8/table"
printf 'holds:\n  - pin: "%s"\n    max: "%s"\n    reason: "test hold"\n    expires: "2026-12-31"\n' \
  "$CORE" "$tree_core" >"$d/.cascade-hold"
commit_setup "$d"
run "$d" "$TMP/s8/table" "$TMP/s8/log"
if [ "$RUN_RC" != 3 ]; then
  fail "S8 core hold" "exit $RUN_RC, want 3: $(why)"
elif ! clean "$d"; then
  fail "S8 core hold" "the tree changed"
elif ! warned "$d" "catalog \`v4.99.0\` needs core \`v2.0.0-beta.99\`, above the hold"; then
  fail "S8 core hold" "no catalog-held warning"
else
  pass "S8 core hold"
fi

# S10 allowed dirty tree and expect hint: with CASCADE_ALLOW_DIRTY=1 an
# untracked file is kept and judged by snapshot, so nothing to move is exit 3;
# CASCADE_EXPECT reaches the library's newest call as --expect.
d=$(sandbox s10)
current_rows "$d" >"$TMP/s10/table"
commit_setup "$d"
printf 'x\n' >"$d/untracked-file"
export CASCADE_ALLOW_DIRTY=1 CASCADE_EXPECT="$LIB=v9.9.9 unrelated.example/x=v1.0.0"
run "$d" "$TMP/s10/table" "$TMP/s10/log"
unset CASCADE_ALLOW_DIRTY CASCADE_EXPECT
if [ "$RUN_RC" != 3 ]; then
  fail "S10 allowed dirty tree" "exit $RUN_RC, want 3: $(why)"
elif [ "$(g "$d" status --porcelain --untracked-files=all)" != "?? untracked-file" ]; then
  fail "S10 allowed dirty tree" "the tree changed beyond the untracked file"
elif ! grep -q "^newest go $LIB .*--expect v9.9.9" "$TMP/s10/log"; then
  fail "S10 allowed dirty tree" "the library newest call lacks --expect v9.9.9"
elif [ "$(grep -c -e '--expect' "$TMP/s10/log")" != 1 ]; then
  fail "S10 allowed dirty tree" "--expect reached a pin CASCADE_EXPECT does not name"
else
  pass "S10 allowed dirty tree"
fi

# S11 never lower: a template version a human set above the cascade target
# stays, with a warning (design.md D6).
d=$(sandbox s11)
current_rows "$d" >"$TMP/s11/table"
commit_setup "$d"
sed -i 's/^Version: ".*"$/Version: "1.99.0"/' "$d/templates/minimal/identity/identity.cue"
g "$d" commit -q -am "a human sets the version"
run "$d" "$TMP/s11/table" "$TMP/s11/log"
if [ "$RUN_RC" != 3 ]; then
  fail "S11 never lower" "exit $RUN_RC, want 3: $(why)"
elif ! clean "$d" || [ "$(id_version "$d/templates/minimal/identity/identity.cue")" != 1.99.0 ]; then
  fail "S11 never lower" "the human version was changed"
elif ! warned "$d" "declares \`1.99.0\`, above the cascade target"; then
  fail "S11 never lower" "no warning"
else
  pass "S11 never lower"
fi

# S12 kind ahead: hack/kind-platform.yaml above hack/platform's catalog stays,
# with a warning; it is never moved down (contract v1.1 clarification C8).
d=$(sandbox s12)
current_rows "$d" >"$TMP/s12/table"
perl -pi -e 's/^(\s*version:\s*)"[^"]+"/$1"4.99.0"/ if $seen; $seen = 1 if /opmodel\.dev\/catalogs\/opm\@v4:/' \
  "$d/hack/kind-platform.yaml"
setup_ok=0
if grep -q '"4.99.0"' "$d/hack/kind-platform.yaml"; then setup_ok=1; fi
commit_setup "$d"
run "$d" "$TMP/s12/table" "$TMP/s12/log"
if [ "$setup_ok" != 1 ]; then
  fail "S12 kind ahead" "the setup edit did not apply"
elif [ "$RUN_RC" != 3 ]; then
  fail "S12 kind ahead" "exit $RUN_RC, want 3: $(why)"
elif ! clean "$d"; then
  fail "S12 kind ahead" "the tree changed"
elif ! warned "$d" "\`hack/kind-platform.yaml\` \`4.99.0\` is ahead of"; then
  fail "S12 kind ahead" "no kind-ahead warning"
else
  pass "S12 kind ahead"
fi

# S13 warning against the merge base: a human commit on the branch already
# moved one file's core, and this run moves nothing; the language.version
# warning for that core still reaches the warnings file (contract §6.4 step 6).
d=$(sandbox s13)
f="$d/tests/integration/module-apply/testdata/cue.mod/module.cue"
tree_core=$(cue_dep_v "$f" "$CORE")
{ printf 'language-of\t%s\t%s\tv0.99.0\n' "$CORE" "$tree_core"; current_rows "$d"; } >"$TMP/s13/table"
cp "$f" "$TMP/s13/module.cue"
perl -0pi -e 's/("opmodel\.dev\/core\@v2": \{\n\s*v:\s*)"[^"]+"/$1"v2.0.0-alpha.1"/' "$f"
setup_ok=0
if [ "$(cue_dep_v "$f" "$CORE")" = v2.0.0-alpha.1 ]; then setup_ok=1; fi
commit_setup "$d"
cp "$TMP/s13/module.cue" "$f"
g "$d" commit -q -am "a human moves core"
run "$d" "$TMP/s13/table" "$TMP/s13/log"
if [ "$setup_ok" != 1 ]; then
  fail "S13 warning against the base" "the setup edit did not apply"
elif [ "$RUN_RC" != 3 ]; then
  fail "S13 warning against the base" "exit $RUN_RC, want 3: $(why)"
elif ! clean "$d"; then
  fail "S13 warning against the base" "the tree changed"
elif ! warned "$d" "\`$CORE\` \`$tree_core\` declares \`language.version\` \`v0.99.0\`"; then
  fail "S13 warning against the base" "no language.version warning"
else
  pass "S13 warning against the base"
fi

# S14 frozen unpublished podinfo: a consumer whose catalog must move freezes
# its podinfo at an unpublished version; the task stops in phase A with a clear
# message instead of failing inside cue mod get.
d=$(sandbox s14)
tree_core=$(cue_dep_v "$d/templates/minimal/cue.mod/module.cue" "$CORE")
{ printf 'newest\tcue\t%s\tv4.99.0\n' "$CAT"
  printf 'pin-of\t%s\tv4.99.0\t%s\t%s\n' "$CAT" "$CORE" "$tree_core"
  current_rows "$d"; } >"$TMP/s14/table"
f="$d/examples/cue.mod/module.cue"
K="$POD" perl -0pi -e 's/("\Q$ENV{K}\E": \{\n\s*v:\s*)"[^"]+"/$1"v0.1.99"/' "$f"
setup_ok=0
if [ "$(cue_dep_v "$f" "$POD")" = v0.1.99 ]; then setup_ok=1; fi
printf '  - path: examples/cue.mod/module.cue\n    pins: ["%s"]\n    reason: "test freeze"\n' "$POD" >>"$d/.cascade-frozen"
commit_setup "$d"
run "$d" "$TMP/s14/table" "$TMP/s14/log"
if [ "$setup_ok" != 1 ]; then
  fail "S14 frozen unpublished podinfo" "the setup edit did not apply"
elif [ "$RUN_RC" = 0 ] || [ "$RUN_RC" = 3 ]; then
  fail "S14 frozen unpublished podinfo" "exit $RUN_RC, want neither 0 nor 3"
elif ! clean "$d"; then
  fail "S14 frozen unpublished podinfo" "the tree changed"
elif ! grep -qF "freezes \`$POD\` at the unpublished \`v0.1.99\`" "$RUN_OUT"; then
  fail "S14 frozen unpublished podinfo" "no clear message: $(why)"
else
  pass "S14 frozen unpublished podinfo"
fi

# ---------------------------------------------------------------------------
# Network scenarios (CASCADE_TEST_SET=all): the older versions are real, so
# go get, operator:sync and cue mod get resolve them from the Go proxy, GitHub
# releases and GHCR, or from a warm cache.

OLDER="$HERE/testdata/older.tsv"
CUE_DIRS=(
  templates/minimal templates/standard templates/advanced hack/platform "$PODDIR"
  examples tests/e2e/testdata/operator-owned
  internal/instinit/testdata/initvalues internal/workflow/render/testdata/skip-unprovided
  tests/e2e/testdata/duplicate-identities tests/integration/module-apply/testdata
  tests/fixtures/valid/simple-module tests/fixtures/valid/module-with-debug-values
)
CONSUMERS=(examples tests/e2e/testdata/operator-owned)
# GOLDEN: what the first run leaves changed against the original tree once every
# pin is back at the tree's value (design.md D10; spike 1.4).
GOLDEN=$(printf '%s\n' templates/minimal/identity/identity.cue templates/standard/identity/identity.cue \
  templates/advanced/identity/identity.cue "$PODDIR/identity/identity.cue" \
  examples/cue.mod/module.cue tests/e2e/testdata/operator-owned/cue.mod/module.cue | LC_ALL=C sort)

older() { awk -F'\t' -v r="$1" -v k="$2" '$1 == r && $2 == k { print $3; exit }' "$OLDER"; }
next_patch() { awk -F. -v OFS=. '{ $NF = $NF + 1; print }' <<<"$1"; }
# older_than A B: true when A ranks below B (the stub's semver-cmp).
older_than() { [ "$(CASCADE_STUB_TABLE=/dev/null "$STUB" semver-cmp "$1" "$2")" = -1 ]; }
# set_v FILE KEY VERSION: rewrite KEY's v: in a module.cue as text, untidy on purpose.
set_v() {
  K="$2" V="$3" perl -0pi -e 's/("\Q$ENV{K}\E": \{\n\s*v:\s*)"[^"]+"/$1"$ENV{V}"/' "$1"
  [ "$(cue_dep_v "$1" "$2")" = "$3" ]
}
# setup_older DIR CATALOG CORE [PODINFO]: move every pin the task moves back to
# the given versions (library and operator to their older rows).
setup_older() {
  local d="$1" c m
  (cd "$d" && GOWORK=off go get "$LIB@$(older older "$LIB")" && GOWORK=off go mod tidy) >/dev/null 2>&1 &&
    (cd "$d" && task -x operator:sync VERSION="$(older older "$OP")") >/dev/null 2>&1 || return 1
  for c in "${CUE_DIRS[@]}"; do
    m="$d/$c/cue.mod/module.cue"
    if [ -n "$(cue_dep_v "$m" "$CAT")" ]; then set_v "$m" "$CAT" "$2" || return 1; fi
    set_v "$m" "$CORE" "$3" || return 1
  done
  perl -pi -e 's/^(\s*version:\s*)"[^"]+"/$1"'"${2#v}"'"/ if $seen; $seen = 1 if /opmodel\.dev\/catalogs\/opm\@v4:/' \
    "$d/hack/kind-platform.yaml"
  if [ -n "${4:-}" ]; then
    for c in "${CONSUMERS[@]}"; do set_v "$d/$c/cue.mod/module.cue" "$POD" "$4" || return 1; done
  fi
}
# golden DIR: the tree differs from the original (the root commit "base") in
# exactly the GOLDEN paths: each identity one patch up, and the consumers'
# podinfo pin at the fixture's new version. Prints the reason on failure.
golden() {
  local d="$1" base a want pod c
  base=$(g "$d" rev-list --max-parents=0 HEAD)
  if [ -n "$(g "$d" ls-files --others --exclude-standard)" ]; then echo "untracked files"; return 1; fi
  if [ "$(g "$d" diff --name-only "$base" | LC_ALL=C sort)" != "$GOLDEN" ]; then
    echo "changed paths: $(g "$d" diff --name-only "$base" | tr '\n' ' ')"; return 1
  fi
  for a in "${ADV_DIRS[@]}"; do
    want=$(next_patch "$(g "$d" show "$base:$a/identity/identity.cue" | sed -n 's/^Version: "\(.*\)"$/\1/p')")
    if [ "$(id_version "$d/$a/identity/identity.cue")" != "$want" ]; then
      echo "$a declares $(id_version "$d/$a/identity/identity.cue"), want $want"; return 1
    fi
  done
  pod=v$(id_version "$d/$PODDIR/identity/identity.cue")
  for c in "${CONSUMERS[@]}"; do
    if ! diff <(g "$d" show "$base:$c/cue.mod/module.cue" |
      K="$POD" V="$pod" perl -0pe 's/("\Q$ENV{K}\E": \{\n\s*v:\s*)"[^"]+"/$1"$ENV{V}"/') \
      "$d/$c/cue.mod/module.cue" >/dev/null; then
      echo "$c differs from the original beyond its podinfo pin $pod"; return 1
    fi
  done
}
commit_run() { g "$1" add -A; g "$1" commit -q -m "run"; }

if [ "$SET" = all ]; then
  # The older rows must stay older than the tree (contract §8).
  d=$(sandbox older)
  ok=1
  while IFS=$'\t' read -r key _ _ v _; do
    o=$(older older "$key")
    if [ -z "$o" ] || ! older_than "$o" "$v"; then
      fail "older.tsv" "\`older.tsv\` \`$key\` \`${o:-missing}\` is not older than the tree's \`$v\`; pick an older published version"
      ok=0
    fi
  done < <(cd "$d" && .tasks/cascade/pins.sh WORKTREE)
  for key in "$CAT" "$CORE"; do
    if ! older_than "$(older oldest "$key")" "$(older older "$key")"; then
      fail "older.tsv" "the oldest \`$key\` is not older than its older row"; ok=0
    fi
  done
  if ! older_than "$(older oldest "$POD")" "v$(id_version "$d/$PODDIR/identity/identity.cue")"; then
    fail "older.tsv" "the oldest podinfo is not older than the tree's fixture"; ok=0
  fi
  if [ "$ok" = 1 ]; then pass "older.tsv"; fi

  OLD_CAT=$(older older "$CAT")
  OLD_CORE=$(older older "$CORE")

  # S2 older pins: every pin older; the first run moves them back and advances
  # each version once; a second run against the same base changes nothing.
  d=$(sandbox s2)
  # Plus a language-of row newer than the cue PR CI installs, and a resolver
  # warning on the library lookup, both of which must reach the warnings file.
  { printf 'language-of\t%s\t%s\tv0.99.0\n' "$CAT" "$(cue_dep_v "$d/templates/minimal/cue.mod/module.cue" "$CAT")"
    printf 'warn\tgo\t%s\tnew major available: %sv2.0.0%s\n' "$LIB" '`' '`'
    current_rows "$d"; grep '^pin-of' "$OLDER"; } >"$TMP/s2/table"
  if ! setup_older "$d" "$OLD_CAT" "$OLD_CORE"; then
    fail "S2 older pins" "the setup did not apply"
  else
    commit_setup "$d"
    run "$d" "$TMP/s2/table" "$TMP/s2/log"
    if [ "$RUN_RC" != 0 ]; then
      fail "S2 older pins" "first run exit $RUN_RC, want 0: $(why)"
    elif ! reason=$(golden "$d"); then
      fail "S2 older pins" "first run: $reason"
    elif ! warned "$d" "docs bundle for \`library\`"; then
      fail "S2 older pins" "no docs-bundle warning"
    elif ! warned "$d" "declares \`language.version\` \`v0.99.0\`, newer than the cue"; then
      fail "S2 older pins" "no language.version warning"
    elif ! grep -qF "$LIB	new major available" "$d/.git/cascade/warnings"; then
      fail "S2 older pins" "the resolver's warning did not reach the warnings file"
    else
      # S5 title and body, on the first run's tree, against the real resolver.
      if [ -z "${CASCADE_RESOLVER_REAL:-}" ]; then
        skip "S5 title and body" "CASCADE_RESOLVER_REAL is not set"
      else
        title=$(cd "$d" && CASCADE_RESOLVER="$CASCADE_RESOLVER_REAL" task -x deps:cascade:title 2>"$TMP/s5.err") || true
        body=$(cd "$d" && CASCADE_RESOLVER="$CASCADE_RESOLVER_REAL" task -x deps:cascade:body 2>>"$TMP/s5.err") || true
        rows=$(grep -c '^| .* (`' <<<"$body" || true)
        if [ "$title" != "fix(deps): bump 4 upstream pins" ]; then
          fail "S5 title and body" "title '$title': $(tail -n 2 "$TMP/s5.err" | tr '\n' ' ')"
        elif ! grep -q '^<!-- cascade-title: ' <<<"$body" || ! grep -q '^<!-- cascade-labels: ' <<<"$body"; then
          fail "S5 title and body" "the body lacks a marker"
        elif [ "$rows" != 4 ]; then
          fail "S5 title and body" "$rows moved-pin rows, want 4"
        elif [ "$(grep '^## ' <<<"$body" | tail -n 1)" != "## Notes" ]; then
          fail "S5 title and body" "## Notes is not the last section"
        elif grep -q need-human-review <<<"$body"; then
          fail "S5 title and body" "the body carries need-human-review"
        else
          pass "S5 title and body"
        fi
      fi
      # The path-class map, through the real resolver's classify.
      if [ -n "${CASCADE_RESOLVER_REAL:-}" ]; then
        want=$(printf '%s\n' \
          "test	hack/platform/cue.mod/module.cue" "test	hack/kind-platform.yaml" \
          "test	examples/cue.mod/module.cue" "test	tests/fixtures/modules/podinfo/identity/identity.cue" \
          "test	internal/instinit/testdata/initvalues/cue.mod/module.cue" \
          "test	internal/cmd/platform/check_test.go" "shipped	templates/minimal/cue.mod/module.cue" \
          "shipped	internal/operator/dist/install.yaml" "shipped	go.mod")
        got=$(cut -f2 <<<"$want" | (cd "$d" && "$CASCADE_RESOLVER_REAL" classify --classes .tasks/cascade/classes)) || got="classify failed"
        if [ "$got" = "$want" ]; then
          pass "S5 classes"
        else
          fail "S5 classes" "classify printed: $(tr '\n\t' '; ' <<<"$got")"
        fi
      fi
      commit_run "$d"
      run "$d" "$TMP/s2/table" "$TMP/s2/log2"
      if [ "$RUN_RC" != 3 ]; then
        fail "S2 older pins" "second run exit $RUN_RC, want 3: $(why)"
      elif ! clean "$d"; then
        fail "S2 older pins" "the second run changed the tree"
      else
        pass "S2 older pins"
      fi
    fi
  fi

  # S4 frozen: the S2 setup, plus a freeze on one moved test tree for both keys.
  d=$(sandbox s4)
  { current_rows "$d"; grep '^pin-of' "$OLDER"; } >"$TMP/s4/table"
  frozen_file=tests/integration/module-apply/testdata/cue.mod/module.cue
  sibling=tests/e2e/testdata/duplicate-identities/cue.mod/module.cue
  if ! setup_older "$d" "$OLD_CAT" "$OLD_CORE"; then
    fail "S4 frozen" "the setup did not apply"
  else
    [ -f "$d/.cascade-frozen" ] || printf 'frozen:\n' >"$d/.cascade-frozen"
    printf '  - path: %s\n    pins: ["%s", "%s"]\n    reason: "test freeze"\n' \
      "$frozen_file" "$CAT" "$CORE" >>"$d/.cascade-frozen"
    commit_setup "$d"
    run "$d" "$TMP/s4/table" "$TMP/s4/log"
    if [ "$RUN_RC" != 0 ]; then
      fail "S4 frozen" "exit $RUN_RC, want 0: $(why)"
    elif ! g "$d" diff --quiet HEAD -- "$frozen_file"; then
      fail "S4 frozen" "$frozen_file changed"
    elif g "$d" diff --quiet HEAD -- "$sibling"; then
      fail "S4 frozen" "$sibling did not move"
    else
      pass "S4 frozen"
    fi
  fi

  # S9 a second move on the branch: after a first run advanced podinfo and its
  # consumers, a second catalog move must still get and tidy the consumers,
  # whose podinfo pin is not published yet (design.md D7).
  d=$(sandbox s9)
  current_rows "$d" >"$TMP/s9/current"
  { printf 'newest\tcue\t%s\t%s\n' "$CAT" "$OLD_CAT"
    printf 'published\tcue\t%s\t%s\n' "$POD" "$(older oldest "$POD")"
    grep '^pin-of' "$OLDER"; cat "$TMP/s9/current"; } >"$TMP/s9/table1"
  { grep '^pin-of' "$OLDER"; cat "$TMP/s9/current"; } >"$TMP/s9/table2"
  next_pod=v$(next_patch "$(id_version "$d/$PODDIR/identity/identity.cue")")
  if ! setup_older "$d" "$(older oldest "$CAT")" "$(older oldest "$CORE")" "$(older oldest "$POD")"; then
    fail "S9 second move" "the setup did not apply"
  else
    commit_setup "$d"
    run "$d" "$TMP/s9/table1" "$TMP/s9/log1"
    if [ "$RUN_RC" != 0 ]; then
      fail "S9 second move" "first run exit $RUN_RC, want 0: $(why)"
    elif [ "$(cue_dep_v "$d/examples/cue.mod/module.cue" "$CAT")" != "$OLD_CAT" ] ||
      [ "$(cue_dep_v "$d/examples/cue.mod/module.cue" "$POD")" != "$next_pod" ]; then
      fail "S9 second move" "after the first run, examples does not pin catalog $OLD_CAT and podinfo $next_pod"
    else
      commit_run "$d"
      run "$d" "$TMP/s9/table2" "$TMP/s9/log2"
      if [ "$RUN_RC" != 0 ]; then
        fail "S9 second move" "second run exit $RUN_RC, want 0: $(why)"
      elif ! reason=$(golden "$d"); then
        fail "S9 second move" "second run: $reason"
      else
        commit_run "$d"
        run "$d" "$TMP/s9/table2" "$TMP/s9/log3"
        if [ "$RUN_RC" != 3 ] || ! clean "$d"; then
          fail "S9 second move" "third run exit $RUN_RC, want 3 and no change"
        else
          pass "S9 second move"
        fi
      fi
    fi
  fi
else
  skip "network scenarios" "CASCADE_TEST_SET=offline"
fi

# ---------------------------------------------------------------------------

if [ "$(git -C "$ROOT" status --porcelain --untracked-files=all)" = "$CHECKOUT_BEFORE" ]; then
  pass "checkout untouched"
else
  fail "checkout untouched" "git status in $ROOT changed during the test"
fi

exit "$FAILED"
