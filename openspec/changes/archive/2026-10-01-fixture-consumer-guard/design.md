## Context

Fixtures (`tests/fixtures/modules/*`) are published CUE modules on `testing.opmodel.dev/modules/cli/*`. Two in-repo CUE modules consume them by version: `examples/` and `tests/e2e/testdata/operator-owned/`. PR CI resolves the tree's fixture version from a job-local registry (`hack/fixtures.sh seed`), core and the catalogs from GHCR.

No Go code, command, flag or exit code of `opm` changes. The config rules about command syntax, flags and example output apply to the script subcommand below instead.

## Goals / Non-Goals

**Goals:**

- A consumer whose core or catalog pin is lower than what CUE resolves for its fixture pins fails PR CI, naming the consumer and showing the diff.
- The same code protects opm-operator's modulepackages.
- A manual fix path that is literally `cue mod get` plus `cue mod tidy`.

**Non-Goals:**

- Writing consumer pins from the workspace task (workspace PR).

## Decisions

### D1. The check lives in `hack/fixtures.sh` as `consumers`

`hack/fixtures.sh consumers <dir>...`. Each repo passes its own consumer dirs (cli: `examples tests/e2e/testdata/operator-owned`; opm-operator: every `test/fixtures/modulepackages/*`), so the file stays byte-identical and `task fixtures:lint` stays green.

Per consumer, in a scratch copy:

```bash
for dep in <testing.opmodel.dev dep keys of module.cue>; do
  cue mod get "${dep%@*}@<pinned v>"   # same version: forces the requirement walk
done
cue mod tidy                            # drops what get added and nothing needs
diff -u <committed module.cue> <resolved module.cue>
```

Output and exit status:

| Case | Line | Exit |
| --- | --- | --- |
| identical | `    ok` | 0 |
| differs | unified diff, then `FAIL <dir>: module.cue differs from what CUE resolves for its fixture pins` plus a two-line hint (re-run `deps:pins:fixtures`, apply the diff, or `FIX=1`) | 1 |
| `cue mod get` fails (version neither published nor seeded) | CUE's error, then `FAIL <dir>: cue mod get <path>@<v> failed (is that version published, or seeded into <host>?)` | 1 |
| `cue mod tidy` fails | CUE's error, then `FAIL <dir>: cue mod tidy failed` | 1 |
| no `cue.mod/module.cue`, no fixture pin, no `v:` | `FAIL <dir>: <reason>` | 1 |
| tracked `cue.mod` outside `FIXTURES_DIR` pins a fixture, not listed | `FAIL <file>: pins a testing.opmodel.dev fixture but is not a listed consumer` | 1 |
| `FIX=1` and differs | the diff, then `    fixed: wrote the resolved module.cue` | 0 |

Every consumer is checked before the exit; one failure does not hide the next. The scratch dir is removed by an EXIT trap on every path, `die` included. `CUE_REGISTRY` defaults to GHCR; PR CI passes the mixed mapping. Only `cue` is required (no `opm`). Parsing uses `sed`/`awk`, not `grep -P`, so it runs on macOS.

### D2. Wired after the seed in the `fixtures` job and `task test:fixtures`

It needs the tree's fixture version, which only the seeded registry holds at PR time. The `fixtures` job already has the registry, `cue` v0.17.1, the mixed mapping and a fresh `CUE_CACHE_DIR`. It runs in `pr.yml` only: every change reaches `main` through a PR, so a push-time run would re-check what the PR already checked.

## Research & Decisions

All runs in scratch copies (`git archive`), a throwaway `registry:2` on `127.0.0.1:5593` (removed), empty `HOME` and `DOCKER_CONFIG`, and an `opm` shim that refuses a publish whose `testing.opmodel.dev` mapping is not the throwaway registry.

### Why `cue mod tidy --check` is not the check

**Context**: tidy is the obvious drift gate.
**Explored**: the `dd23e01` operator-owned file (podinfo `v0.1.11`, core alpha.6, catalogs/opm `v4.0.1`), fresh cache, canonical GHCR mapping.
**Options considered**:
1. `cue mod tidy --check`: exits 0 on the stale file; `cue export` fetches core alpha.6 next to podinfo `v0.1.11`.
2. `cue mod get <fixture>@<same version>` then `cue mod tidy`: raises core to beta.1 and catalogs/opm to `v4.4.4` (and adds `cue.dev/x/k8s.io`, which tidy drops again). Byte-identical to the hand fix in `a3d8b01`.
**Decision**: option 2.
**Rationale**: tidy does not apply MVS over a dep the consumer already lists.

### Proof of the subcommand

| Run | Result |
| --- | --- |
| cli `main`, GHCR | `examples ok`, `operator-owned ok`, rc 0 |
| operator-owned on core alpha.6 / catalogs `v4.0.1` | diff `v4.0.1 -> v4.4.4`, `alpha.6 -> beta.1`, FAIL, rc 1 |
| operator-owned pinned to unpublished `v0.1.99`, listed before `examples` | CUE "module not found", FAIL line, `examples` still checked and ok, rc 1, scratch `TMPDIR` empty afterwards |
| only `examples` listed | FAIL naming `tests/e2e/testdata/operator-owned/cue.mod/module.cue` as unlisted, rc 1 |
| fixture bumped to `0.1.900`, seeded, consumers text-re-pinned with operator-owned on alpha.6 | FAIL with the alpha.6 diff, rc 1 |
| same, `FIX=1` | operator-owned fixed; the only remaining diff against `main` is the version line |
| opm-operator `main` modulepackages, GHCR | four `ok`, rc 0 |
| `shellcheck hack/fixtures.sh` | clean |

### Alternatives

1. A cli-only `hack/fixture-consumers.sh` (the first draft): leaves opm-operator unguarded and is a second copy of the same idea. Rejected.
2. An offline Go test comparing pins by semver: reimplements MVS, misses deps a consumer lacks, and the unit job tests only `./internal/...`. Rejected.
3. Run it in the `unit` or `e2e` job: they seed too, but `fixtures` is the job named for fixture gates. Rejected.

## Risks / Trade-offs

- [GHCR outage] the step fails with the rest of the job.
- [Copy drift] `hack/fixtures.sh` changes in both repos; mitigated by `task fixtures:lint` and back-to-back merges.
- [Future CUE applies MVS over listed deps] the check keeps working: it compares against whatever CUE resolves.

## Appendix: reference implementation of `consumers`

The exact diff proved in scratch (applies to `hack/fixtures.sh` at cli `437bebd` and opm-operator `c3e4232`, which are byte-identical). Section 1 applies it as is.

```diff
--- a/hack/fixtures.sh
+++ b/hack/fixtures.sh
@@ -33,6 +33,18 @@
 #            maps testing.opmodel.dev to (refuses a ghcr.io mapping)
 #   publish  publish the tree's fixtures to CUE_REGISTRY (default: GHCR);
 #            honours SINCE=<git-ref> and PRERELEASE=<id>
+#   consumers <dir>...
+#            check that each consumer (a dir holding a cue.mod that pins a
+#            fixture) pins exactly what CUE resolves for the fixture versions
+#            it names: in a scratch copy, `cue mod get <fixture>@<pinned>` per
+#            testing.opmodel.dev pin, then `cue mod tidy`, diffed against the
+#            committed module.cue. Needed because CUE keeps a dep the consumer
+#            already lists at its listed version: a consumer on a new fixture
+#            but a stale core passes `cue mod tidy --check` and evaluates
+#            against the stale core. Also fails on a tracked cue.mod outside
+#            FIXTURES_DIR that pins a fixture but is not listed. Resolves
+#            through CUE_REGISTRY (default: GHCR; the seeded mapping in PR CI).
+#            FIX=1 writes the resolved module.cue back instead of failing.
 #
 # Environment
 #   FIXTURES_DIR       fixture root; auto-detected (tests/fixtures/modules or
@@ -49,6 +61,7 @@
 #   SINCE              publish: skip fixtures unchanged since this git ref
 #   PRERELEASE         publish: append a SemVer pre-release segment to the tag
 #                      (e.g. e2e.gabc1234) so it never claims the release version
+#   FIX                consumers: 1 rewrites a drifted module.cue in place
 set -euo pipefail
 
 GHCR_REGISTRY='testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
@@ -83,6 +96,7 @@
 BASE_REF=${BASE_REF:-origin/main}
 SINCE=${SINCE:-}
 PRERELEASE=${PRERELEASE:-}
+FIX=${FIX:-}
 
 require_tools() {
   command -v cue >/dev/null || die "cue not on PATH"
@@ -257,11 +271,111 @@
   publish_all
 }
 
+# fixture_deps <module.cue>: the testing.opmodel.dev dep keys a cue.mod pins
+# (dep keys only, never the `module:` line).
+fixture_deps() {
+  sed -n 's/^[[:space:]]*"\(testing\.opmodel\.dev\/[^"]*\)":[[:space:]]*{.*$/\1/p' "$1"
+}
+
+# dep_version <module.cue> <dep>: the v: pinned under <dep> (empty when absent).
+dep_version() {
+  awk -v p="\"$2\"" 'index($0, p) {f = 1} f && /v: "/ {match($0, /"[^"]+"/); print substr($0, RSTART + 1, RLENGTH - 2); exit}' "$1"
+}
+
+consumer_fail() {
+  echo "FAIL $1: $2" >&2
+}
+
+cmd_consumers() {
+  command -v cue >/dev/null || die "cue not on PATH"
+  [ "$#" -gt 0 ] || die "consumers: name the consumer dirs (each holds a cue.mod)"
+  export CUE_REGISTRY=${CUE_REGISTRY:-$GHCR_REGISTRY}
+  local scratch dir mod work deps dep ver out rc=0 ok listed f n=0
+  scratch=$(mktemp -d)
+  # shellcheck disable=SC2064 # expand now: $scratch is local to this function
+  trap "rm -rf '$scratch'" EXIT
+  echo "fixture consumers against $(testing_host)"
+  listed=" "
+  for dir in "$@"; do
+    dir=${dir%/}
+    dir=${dir#./}
+    n=$((n + 1))
+    mod="$dir/cue.mod/module.cue"
+    listed="${listed}${mod} "
+    echo "==> ${dir}"
+    if [ ! -f "$mod" ]; then
+      consumer_fail "$dir" "no cue.mod/module.cue"
+      rc=1
+      continue
+    fi
+    work="$scratch/$n"
+    mkdir -p "$work"
+    cp -R "$dir/." "$work/"
+    ok=1
+    deps=$(fixture_deps "$mod")
+    if [ -z "$deps" ]; then
+      consumer_fail "$dir" "pins no testing.opmodel.dev fixture; not a consumer"
+      rc=1
+      continue
+    fi
+    for dep in $deps; do
+      ver=$(dep_version "$mod" "$dep")
+      if [ -z "$ver" ]; then
+        consumer_fail "$dir" "no v: under \"$dep\""
+        ok=0
+        break
+      fi
+      # Same version on purpose: it forces CUE to walk the fixture's own
+      # requirements and raise every shared dep to at least the fixture's pin.
+      if ! out=$(cd "$work" && cue mod get "${dep%@*}@${ver}" 2>&1); then
+        echo "$out" >&2
+        consumer_fail "$dir" "cue mod get ${dep%@*}@${ver} failed (is that version published, or seeded into $(testing_host)?)"
+        ok=0
+        break
+      fi
+    done
+    if [ "$ok" -eq 1 ] && ! out=$(cd "$work" && cue mod tidy 2>&1); then
+      echo "$out" >&2
+      consumer_fail "$dir" "cue mod tidy failed"
+      ok=0
+    fi
+    if [ "$ok" -eq 0 ]; then
+      rc=1
+      continue
+    fi
+    if diff -u --label "$mod (committed)" --label "$mod (resolved)" "$mod" "$work/cue.mod/module.cue"; then
+      echo "    ok"
+    elif [ "$FIX" = "1" ]; then
+      cp "$work/cue.mod/module.cue" "$mod"
+      echo "    fixed: wrote the resolved module.cue"
+    else
+      consumer_fail "$dir" "module.cue differs from what CUE resolves for its fixture pins"
+      echo "     Re-run the workspace root task deps:pins:fixtures, apply the diff above," >&2
+      echo "     or re-run this with FIX=1 against a registry that holds the pinned fixture versions." >&2
+      rc=1
+    fi
+  done
+  # A consumer nobody listed is a consumer nobody checks.
+  if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
+    while IFS= read -r f; do
+      case "$f" in "$FIXTURES_DIR"/*) continue ;; esac
+      [ -n "$(fixture_deps "$f")" ] || continue
+      case "$listed" in *" $f "*) continue ;; esac
+      consumer_fail "$f" "pins a testing.opmodel.dev fixture but is not a listed consumer"
+      rc=1
+    done < <(git ls-files -- '*cue.mod/module.cue')
+  else
+    echo "    (not a git work tree: skipped the unlisted-consumer check)"
+  fi
+  return $rc
+}
+
 case "$cmd" in
   pins) cmd_pins ;;
   check) cmd_check ;;
   seed) cmd_seed ;;
   publish) cmd_publish ;;
+  consumers) cmd_consumers "$@" ;;
   -h|--help|help) usage 0 ;;
-  *) die "unknown subcommand '$cmd' (pins|check|seed|publish)" ;;
+  *) die "unknown subcommand '$cmd' (pins|check|seed|publish|consumers)" ;;
 esac
```
