## Context

The CLI's Go module is `github.com/opmodel/cli`. `library` and `opm-operator` already live under `github.com/open-platform-model/*`. Enhancement 0006 (D15) requires this rename to land before any slice adds a `library` import, so the cross-repo edge is written once, under the final name.

The rename touches four distinct surfaces with different risk profiles:

1. **Go source** (~114 files) — import statements. Compiler-checked; the build fails loudly if any are missed or malformed.
2. **Build tooling** (`Taskfile.yml`, `.goreleaser.yml`) — the import path appears as a **string literal** inside `-X importpath.Var=value` linker flags. Not part of Go's import graph, not checked by `go build`/`go vet`/`go test`. If missed, the build succeeds and `opm version` silently reports an empty version string.
3. **Repo docs** (`CLAUDE.md`) — one prose reference to the `oerrors` import alias.
4. **Active OpenSpec specs** (9 files under `openspec/specs/`) — `WHEN`/`THEN` requirement text that names the import path directly. Archived changes under `openspec/changes/archive/` are excluded — they are historical record of a change as it was proposed and implemented at the time, not living documentation.

Because this is a Go **module** rename (the `module` line itself changes), the Go-source surface cannot be migrated incrementally file-by-file: any `.go` file still importing `github.com/opmodel/cli/...` after the `module` line flips would be resolving an import path that is no longer this module and no longer exists anywhere else, so the build breaks until every internal import is updated. This is the one surface where Constitution Principle VIII's "small batch sizes" doesn't decompose further — the Go-source rewrite is atomic by the nature of what a module rename is, whatever else may be sequenced around it.

## Goals / Non-Goals

**Goals:**

- Every `github.com/opmodel/cli` reference outside `openspec/changes/archive/` becomes `github.com/open-platform-model/cli`, verified by a repo-wide grep with zero remaining hits in that scope.
- The build-tooling ldflags are updated as an explicit, separately-verified step — not assumed to be swept by a `.go`-only rewrite.
- The 9 active spec files' requirement text stays accurate to the actual import path.
- No behavior change: this is an identifier rename, not a logic change.

**Non-Goals:**

- No CI workflow changes — `.github/` currently has no workflows referencing the module path, so there is nothing to update there. (If workflows are added later, they inherit the new path from day one.)
- No permanent CI guard against re-introducing the old path. The verification grep guard here is a one-time completion gate for this change, not a standing lint rule — a repo with no CI workflows today isn't the place to introduce one as a side effect of a rename.
- No touching `openspec/changes/archive/` — historical record stays as it was written.
- No touching `openspec/changes/cue-binary-integration/` — out of scope, called out in the proposal.
- No release cut / version tag as part of this change.

## Decisions

### Rewrite order: `go.mod` → Go-source (atomic) → ldflags → docs → specs → verify

`go.mod`'s `module` line changes first (plus the adjacent `go 1.26.0`/`1.25.0` doc-drift fix, corrected the same time `go.mod` is touched). Then every internal import statement is rewritten in one pass — this must land as a single unit since a half-renamed Go-source tree does not compile. The build-tooling ldflags come immediately after, treated as a build-correctness fix, not folded into the later "docs" pass, because an unnoticed miss here produces a silently-broken `opm version`, not a compile error. Docs (`CLAUDE.md`) and the 9 spec files follow, since they carry no build risk if briefly stale mid-change. The grep-guard verification runs last, as the completion gate.

**Alternatives considered:**
- Rewrite package-by-package, verifying each compiles before moving to the next. Rejected: doesn't work for a module-line rename — every file importing the module's own old path breaks simultaneously the moment `go.mod`'s `module` line changes, regardless of order.
- Treat ldflags as part of the general docs/tooling cleanup pass. Rejected: it's not a docs issue, it's a silent build-correctness issue (no compiler signal on miss) — it deserves its own explicit, checked step rather than being bundled with lower-risk prose edits.

### Rewrite mechanism: scripted substitution + `goimports` re-sort, not a raw find/sed alone

A plain `sed 's#github.com/opmodel/cli#github.com/open-platform-model/cli#g'` across `*.go` files correctly rewrites every import path string, but doesn't re-sort import blocks. `opmodel` and `open-platform-model` don't sort identically within a `github.com/...` import group (`open-platform-model` sorts before `opmodel` alphabetically), so a handful of files may end up with the self-import in the wrong position relative to other `github.com/...` third-party imports in the same block after a naive substitution. The rewrite step is therefore: scripted path substitution across `.go` files, followed by `task fmt` (`goimports -w .`) to re-sort import groups, so the diff is exactly "import path changed, plus whatever re-sorting that implies" and nothing else.

**Alternatives considered:**
- Manual per-file editing. Rejected: 114 files, purely mechanical, manual editing is slower and more error-prone than scripted substitution for this shape of change.
- Skip the `goimports` re-sort pass and accept whatever ordering the substitution leaves. Rejected: leaves a small number of files with import blocks that don't match repo convention (`goimports`-clean), which `task lint`/`task fmt` would flag anyway on the next unrelated touch to those files — cheaper to fix once, now, as part of this change.

### Verification: a repo-wide grep guard, scoped to exclude `openspec/changes/archive/`, as the completion gate

Before the change is considered done, `grep -rl "opmodel/cli" --exclude-dir=archive .` (or equivalent) must return nothing. This is the single check that catches every surface at once — Go source, ldflags, docs, active specs — rather than trusting that each individual step was complete.

**Alternatives considered:**
- Rely on `go build`/`task test` passing as sufficient verification. Rejected: this is exactly the gap that lets the ldflags miss ship silently — compilation success says nothing about the string-literal surface.

## Risks / Trade-offs

- **Ldflags miss ships a silently-broken `opm version`.** [Risk] The build-tooling string literals aren't compiler-checked, so a rewrite that only touches `.go` files would compile and test clean while quietly breaking version reporting. → **Mitigation:** treated as its own explicit step (not folded into general docs cleanup); caught regardless by the final grep-guard verification, which scans `Taskfile.yml`/`.goreleaser.yml` along with everything else.
- **Import re-sort produces a larger diff than a pure string substitution would.** [Risk] Running `goimports` after the rewrite may touch import ordering in files beyond just the self-import line, making the diff slightly noisier to review. → **Mitigation:** acceptable — the alternative (skipping re-sort) leaves non-conventional import ordering that surfaces as unrelated churn on the next actual change to those files; better to absorb it once, now, while the diff is already large and mechanical.
- **A future contributor or AI assistant reintroduces the old path from muscle memory or stale training data.** [Risk] Nothing structurally prevents `github.com/opmodel/cli` from reappearing in a later PR. → **Mitigation:** none built into this change (no CI workflows exist to host a standing guard today — see Non-Goals); the one-time grep-guard here only proves the state at completion. Worth a future note if `.github/` workflows are ever added.
- **`cue-binary-integration`'s eventual revival will need to rebase past this rename.** [Risk] That change touches `internal/cue/` and `opm version` output, both adjacent to this change's ldflags/version surface. → **Mitigation:** out of scope here per the proposal; whoever revives that change inherits the renamed module the same as any other future work would.
