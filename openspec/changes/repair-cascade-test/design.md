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

The name MUST sort below the tree's version in SemVer, in the stub's `semver-cmp` and in Go. `v<X.Y.Z>-0.cascade.<suffix>` does: a numeric first prerelease identifier sorts below every alphanumeric one (`alpha`, `beta`, `rc`) and below the release. `<suffix>` is the tree's prerelease (`beta.6`), or `release` when it has none, so one name always holds one content in the module cache. The test asserts the order with the stub before it uses the name.

### Environment of the setup

`go get` and `go mod tidy` in `setup_older` run with:

- `GOPROXY=file://$TMP/goproxy,<the user's GOPROXY>`: the file proxy answers only the made-up version; a missing file falls through.
- `GONOSUMDB=$LIB`: the checksum database cannot know the made-up version. Scoped to the setup; the task run still verifies the real library.
- `GOPRIVATE=` and `GONOPROXY=` empty, so a developer's private settings cannot route around the file proxy; `GOFLAGS=-mod=mod`; `GOWORK=off` as before.

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
- [S2 no longer downloads an older library from the real proxy] → the task's own `go get` of the tree version still goes through the real proxy path; the older download was setup, not an assertion.
