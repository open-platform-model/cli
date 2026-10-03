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

if [ "$(git -C "$ROOT" status --porcelain --untracked-files=all)" = "$CHECKOUT_BEFORE" ]; then
  pass "checkout untouched"
else
  fail "checkout untouched" "git status in $ROOT changed during the test"
fi

exit "$FAILED"
