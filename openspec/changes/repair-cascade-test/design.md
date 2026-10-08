## Context

`setup_older` (`.tasks/cascade/test.sh`) prepares S2, S4 and S9: it moves every pin back, commits that as the merge base, and the task must move the pins forward again. For the library it ran `go get $LIB@<older row> && go mod tidy` with all output discarded.

Two steps need the cli source to compile against the lowered library: `go mod tidy` in the setup, and `build_base ./cmd/opm` in `cascade.sh` phase B, which builds `opm` from an export of the merge base (the setup commit).

Measured on `origin/main` d2f4ee3e, in a scratch copy: with `v1.0.0-beta.2` and with `v1.0.0-beta.5`, `go get` exits 0 and `go mod tidy` exits 1 ("does not contain package .../opm/k8s/inventory"); `go build ./cmd/opm` fails the same way. The Go proxy lists nothing between `v1.0.0-beta.5` and the tree's `v1.0.0-beta.6`.

## Goals / Non-Goals

**Goals:**

- The full set passes on `main` for a reason that holds after the next library bump.
- A failed setup says which step failed and why.
- A stale `older.tsv` row fails the required `Lint` job, not only the path-filtered network job.

**Non-Goals:**

- Any change to `cascade.sh`, to a scenario assertion, to the stub or to a workflow.
- Any pin bump.

## Research & Decisions

### Where the older library comes from

**Context**: no published library below the tree's version compiles with the tree, and after every API-adopting bump none will.

**Explored**: the two failing commands above; a prototype of option 3 in a scratch copy (`go get`, `go mod tidy` and `go build ./cmd/opm` exit 0; moving back to the tree version restores `go.mod` and `go.sum` byte for byte).

**Options considered**:

1. Move the row to another published version. Not possible today (none compiles), and it breaks again at the next bump that adopts new API.
2. Leave the library at the tree's version in the setup. S2 then no longer moves the library, and S5's title and three-row assertions change. That weakens the test.
3. Make the older version from the tree's own library: copy its source from the module cache, zip it under a lower version name, and serve it from a `file://` proxy in `$TMP` for the setup's `go get` and `go mod tidy` only.

**Decision**: option 3.

**Rationale**: the source is the tree's library, so the tree always compiles against it; nothing in the setup can go stale. The task run keeps its environment: its `go get $LIB@<tree version>` and `go mod tidy` use the normal proxy and checksum database, and the build from the merge base finds the made-up version in the module cache with its hash in the setup commit's `go.sum`. The cost is about 25 lines of shell and a `zip` dependency (Principle VII: simpler options 1 and 2 do not hold).

### The version name

The name MUST sort below the tree's version in SemVer, in the stub's `semver-cmp` and in Go. `v0.0.0-0.cascade.<tree version without v>` does (`v0.0.0-0.cascade.1.0.0-beta.6`): a `v0.0.0` prerelease sorts below every version a module path without a major suffix can pin, Go pseudo-versions such as `v1.0.1-0.<time>-<hash>` included. The suffix is the tree's version, so one name always holds one content in the module cache. The test asserts the order with the stub before it uses the name, with its own FAIL text. A library on a `/v2` path would need a `v2.0.0-0...` name; the module path constant in `test.sh` changes then anyway.

### Environment of the setup

`go get` and `go mod tidy` in `setup_older` run with:

- `GOPROXY=file://$TMP/goproxy,<the user's GOPROXY>`: the file proxy answers only the made-up version; a missing file falls through.
- `GONOSUMDB=$LIB`: the checksum database cannot know the made-up version. Scoped to the setup; the task run still verifies the real library.
- `GONOPROXY=none`, so a developer's `GOPRIVATE` setting cannot route around the file proxy; `GOWORK=off` as before.

### Deviation from the cascade contract

**Context**: the shared cascade contract (`open-platform-model/.github`, `openspec/changes/archive/2026-10-04-add-cascade-resolver/contract.md`, section 8) says `older.tsv` lists "per pin key, an older real published version" and that in S2 "the older versions are real, so `cue mod get` and `go get` resolve". Every repo's `test.sh` follows that shape.

**Decision**: the cli departs from section 8 for one pin key, the library. It has no `older.tsv` row; its older version is made up from the tree's library. Catalog, core and podinfo keep section 8 as written.

**Rationale**: section 8 assumes an older published version that the tree compiles against. For a Go library pin that holds only while the consumer uses no API newer than the row. Today no such library version exists, so S2, S4 and S9 cannot hold under the letter of section 8.

**Open**: the contract text belongs to `.github` and this change does not touch it. Whether the contract gets a clarification, and whether sibling repos with a Go pin take the same setup, is the owner's decision. Until then `test.sh` names the deviation in a comment where it happens.

### Failure text

```text
FAIL S2 older pins: the setup did not apply: go mod tidy with library v1.0.0-0.cascade.beta.6: <last lines>
```

`setup_older` sets `SETUP_WHY`; the three callers print it.

### The older.tsv check in both sets

The check reads `pins.sh` and the stub's `semver-cmp` only. It moves above the offline scenarios unchanged, minus the library (whose made-up version is checked instead).

## Risks / Trade-offs

- [The made-up version stays in the developer's Go module cache] → a few megabytes, a name no real release can take, content fixed per name. The Go caches are tool caches; the checkout stays untouched.
- [`zip` is missing] → the scenario fails with a FAIL line that names `zip`; it never skips.
- [The older catalog and core are still real published versions under unchanged CUE source] → S9's first run moves all 13 CUE trees from the `oldest` to the `older` rows with a real `cue mod get` and `cue mod tidy` (`cascade.sh` phase C). A template or test tree that starts to use a definition the `older` catalog or core lacks breaks S9 the same way the library broke. This change does not remove that: the equivalent repair needs a local OCI registry in the test, which is not small. The `older.tsv` order check does not see it; only the network job does.
- [The task's `go mod tidy` in S2 has nothing to re-resolve, because the older library has the tree's requirements] → the warning path for a requirement that tidy moves cannot be reached by the suite. No scenario asserted it before.
- [S2 no longer downloads an older library from the real proxy] → the task's own `go get` of the tree version still goes through the real proxy path; the older download was setup, not an assertion.
