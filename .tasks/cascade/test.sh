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

FAILED=0
pass() { printf 'PASS %s\n' "$1"; }
fail() { printf 'FAIL %s: %s\n' "$1" "$2"; FAILED=1; }

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

# ---------------------------------------------------------------------------

if [ "$(git -C "$ROOT" status --porcelain --untracked-files=all)" = "$CHECKOUT_BEFORE" ]; then
  pass "checkout untouched"
else
  fail "checkout untouched" "git status in $ROOT changed during the test"
fi

exit "$FAILED"
