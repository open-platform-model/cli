## Context

See proposal.md for the defects. This is change D of the set in `orchestration.md`. Its inputs are change A's core release (expected `v2.0.0-alpha.13`, `fold-colliding-contract-keys`) and change B's library release (`refuse-colliding-contracts`), whose surface is:

- `platform.ContractInventory.Collisions []string`: every contract key more than one enabled registry entry's catalog lists, ascending. Such a key is in none of `DefinedBy`, `RequiredBy`, `Unfulfilled` or `Comparable`, so `Fulfilled` and `Discriminated` can read true while `Collisions` is non-empty.
- `platform.ContractInventory.CollidingEntries map[string][]string`: each colliding key to the sorted registry keys (path with major) of the enabled entries listing it.
- `ContractInventory.Routable`: true exactly when `OverSubscribed` and `Collisions` are both empty. `ProvidedBy` and `OverSubscribed` are unaffected and can co-occur with a collision.
- `errors.ContractCollision{Key, Catalogs}` and `*errors.ContractCollisionsError{Contracts}`, raised first in the render gate's joined causes, platform-wide, under `SkipUnprovided` too. `*errors.NotRoutableError{}`, raised only when `routable` is false and no over-subscription or collision row explains it.
- `kernel.RenderDiagnostics.Collisions []errors.ContractCollision` (sorted by key) and `RenderDiagnostics.Routable bool`.
- `errors.UnresolvedDemand.Colliding []string`: diagnostic only, the registry entries defining a demanded key that collides.
- `schema.DefaultSchemaModule` at A's tag. `schema.ProvidedBySince` (the floor) is unchanged. `Contracts()` decodes an absent `collisions`/`collidingEntries` as empty: every core before A's tag fails to evaluate a colliding platform at acquire.

What goes wrong in the CLI on B's release, verified against `origin/main` 93d0bc3 (library `v1.0.0-alpha.35`):

- `Report.Render` returns the vacuous report when `len(r.Defined) == 0 && len(r.ProvidedBy) == 0` (`internal/platform/check.go:103`). Experiment 06 case A (two majors of one catalog, same keys, no provider) has both empty, so the report says `routable:  yes (vacuously ...)` while `runPlatformCheck` exits with the validation code (`internal/cmd/platform/check.go:130`).
- Past the vacuous branch, when `Defined` is empty the report prints the note "The enabled catalogs define no contracts" (`check.go:115-118`), false under a collision: the catalogs define the keys, twice.
- The routable verdict line counts only `OverSubscribed` (`check.go:164`); a collision-only platform reads `routable:  no` followed by `0 contracts are over-subscribed`.
- The exit message names over-subscribed and comparable counts only (`internal/cmd/platform/check.go:133-134`).
- `formatRenderDiagnostics` (`internal/workflow/render/validation.go:150-175`) prints no collision row; `cmdutil.FormatUnresolvedDemands` (`internal/cmdutil/output.go:75-92`) falls to `nothing on this platform implements this contract` for a demand whose `DefinedBy` is empty, which is every demand on a colliding key. `refusalHint` (`validation.go:96-124`) adds the deps-source provider hint to any refusal carrying unresolved rows, including one whose rows exist only because of the collision.

The CLI carries no count of its own: the only inventory reader is `opm platform check` (`p.Contracts()`, `internal/cmd/platform/check.go:119`), and the render path prints `RenderDiagnostics` rows. The fixture platforms of `internal/cmd/platform/check_test.go` pin `schema.DefaultSchemaVersion()` (`fixtureModule`, line 29), which is alpha.12 on library alpha.35; a colliding platform pinned there fails to build, so the colliding fixtures carry their own module file pinning A's core.

## Goals / Non-Goals

**Goals:**

- `opm platform check` names every colliding contract and the registry entries defining it, from the inventory; never words a colliding platform as vacuous; counts collisions in the routable verdict and the exit message.
- A render refused for a collision prints one row per colliding contract, first, naming the entries.
- Red-first proof on the check: the colliding command tests run red on library alpha.35 and again on B's head before the report changes.
- The library pinned at B's release.

**Non-Goals:**

- Any count, fallback or presence check in the CLI. The report prints `Collisions` and `CollidingEntries` as the inventory carries them; `Contracts()` already decodes an absent field as empty.
- Side-by-side majors (enhancement 0026 D9) and any hint that suggests them. No special hint for a module importing two majors on the platform-less path (0026 OQ18): it gets the generic collision row, because B moves `DefaultSchemaModule` to the fold core.
- Rewording the `fulfilled` and `discriminated` verdicts. They can read yes under a collision (core's stated limitation); the colliding section's note and the exit status carry the refusal.
- `hack/platform`, `examples/`, `templates/`: the workspace `task deps:update` owns them and runs only after B's release.

## Decisions

### 1. The report carries `Collisions` and `CollidingEntries`, copied from the inventory

`Report` gains two fields, filled in `NewReport` (the only constructor) from `inv.Collisions` and `inv.CollidingEntries`:

```go
// Collisions lists the contract keys more than one enabled registry entry's
// catalog lists. A colliding key is in none of Defined, RequiredBy,
// Unfulfilled or Comparable, so the fulfilled and discriminated verdicts can
// read yes while Collisions is non-empty; routable reads no.
Collisions []string

// CollidingEntries maps each colliding key to the registry keys (catalog
// path plus major) of the enabled entries listing it.
CollidingEntries map[string][]string
```

The `Routable` method doc becomes "reports whether no provider-fulfilled contract is over-subscribed and no contract key collides". Nothing is derived: no count of `CollidingEntries` values, no scan of `Defined`.

### 2. The vacuous branch also requires an empty `Collisions`

The early return at `check.go:103` fires only when `len(r.Defined) == 0 && len(r.ProvidedBy) == 0 && len(r.Collisions) == 0`; the vacuous text stays byte-identical. The existing `defined contracts: 0` note ("The enabled catalogs define no contracts. The provider counts below come from the enabled transformers' own requirements.") is printed only when `Collisions` is also empty; with a collision the header `defined contracts: 0` stands alone and the colliding section's note explains the gap. (This refines the approved design, which did not address the note; without it a collision-only report would claim the catalogs define nothing.)

### 3. The colliding section, before over-subscribed

Printed after the unfulfilled section and before the over-subscribed section, when `Collisions` is non-empty:

```text
colliding contracts: 2
  testing.opmodel.dev/catalogs/base/resources/container@v1beta1
    defined by  testing.opmodel.dev/catalogs/base@v1, testing.opmodel.dev/catalogs/base@v2
  testing.opmodel.dev/catalogs/base/traits/backup@v1alpha1
    defined by  testing.opmodel.dev/catalogs/base@v1, testing.opmodel.dev/catalogs/base@v2
  (a colliding contract is left out of the defined, required, unfulfilled and comparable sections; keep one of its defining entries enabled)
```

Keys iterate `sortedCopy(r.Collisions)`; entries print `sortedCopy(r.CollidingEntries[key])` joined by `, `, like every other list in the report, never trusting the inventory's order. The header line is `colliding contracts: N` with `N = len(r.Collisions)`.

### 4. The routable verdict names both counts

`verdictLine` stays as is for the other two verdicts and for routable when `Collisions` is empty, so every existing report line is byte-identical. When `Collisions` is non-empty the routable line is `routable:  no`, verdictLine's existing separator, then `<n> contract(s) collide, <m> over-subscribed`, worded `1 contract collides` / `2 contracts collide`, for example `2 contracts collide, 0 over-subscribed` and `1 contract collides, 1 over-subscribed`. The invariant the tests pin: under a collision the line never reads `0 contracts are over-subscribed` alone, and never `yes`.

### 5. The exit message names the collision count

`internal/cmd/platform/check.go`: the exit condition is unchanged (`!Routable() || !Discriminated()`; a collision makes routable false). The message becomes:

```text
platform <dir> cannot generate a platform package: <c> colliding contract(s), <o> over-subscribed contract(s), <p> comparable transformer pair(s)
```

The existing tests asserting the substring `1 over-subscribed contract(s), 0 comparable transformer pair(s)` still hold. The `Long` help's inventory sentence adds "the contract keys more than one enabled registry entry defines, with the entries defining each", and the exit-code table gains:

```text
  colliding         exits with the validation error code, because a platform
                    package cannot be generated until all but one of the
                    registry entries defining a contract key is disabled
```

### 6. The render prints the collision rows first

`formatRenderDiagnostics` prints, before the unresolved rows:

```text
contract "<key>": defined by more than one enabled registry entry: <entry>, <entry>
```

one line per `d.Collisions` row, in the kernel's order (sorted by key), entries joined with `, ` as the kernel carries them (the over-subscribed row does the same). `printValidationError` needs no new case: the collision cause arrives inside a `*kernel.RenderError`, whose `Err` (the joined causes, collision message first) prints under `render failed`. A `NotRoutableError` carries no row, so the details block is empty and the kernel's message is the whole output; it still exits with the validation code (`render.go:215-219`, unchanged).

Two refinements beyond the approved design, found while tracing the rows B emits (listed under deviations in the worker report so the supervisor can strike them):

- **`cmdutil.FormatUnresolvedDemands`** gains a case before its default: when `d.Colliding` is non-empty it prints `  defined by more than one enabled registry entry: <entries>` instead of `  nothing on this platform implements this contract`. B added `Colliding` because that default is false for a colliding key; the CLI has its own formatter and would otherwise print the very wording B retired, directly under a collision row naming the same key.
- **`refusalHint`** returns `""` when `errors.As(err, &*liberrors.ContractCollisionsError)`, checked first. Under a collision the unresolved rows are distorted (the key left `definedBy`), so the deps-source provider hint ("the module's own catalogs do not implement this contract") would misdirect; the collision row already names the fix.

### 7. Pins and sequencing

Development runs against B's pushed head as a Go pseudo-version (`go get github.com/open-platform-model/library@<B head sha>`); the last section replaces it with B's release (the supervisor supplies the version). `go.mod` never merges on a pseudo-version: the last section stops and reports if B's release is not out. If B's release is already out when section 1 starts, section 1 pins the release and section 3 runs only its gates, with no commit.

### 8. Section plan

1. **Red first, then platform check names collisions** (`fix(platform)`): the colliding fixtures and command tests, red on library alpha.35 and red on B's head; the report, help and exit message; unit and command tests green.
2. **The render words a collision** (`fix(render)`): the collision row first, the unresolved `Colliding` case, the hint guard; printer and formatter unit tests and an e2e `opm instance build --platform` on a copy of `hack/platform` re-pinned to A's core with a second entry (`testing.opmodel.dev/catalogs/opm-shadow@v1`) listing a hand-authored member carrying the opm container resource's FQN (never the opm catalog's own stamped `#resources`/`#traits` values: `#Catalog` stamps `metadata.modulePath` and `catalogVersion`, so re-listing them conflicts, measured in review), with and without `--skip-unprovided`, all red first.
3. **Pin B's release** (`fix(deps)`): the release replaces the pseudo-version; full gates.

## Command surface

No command, flag or exit code is added or removed. `opm platform check [dir] [--platform <dir>] [--kubeconfig <path>] [--context <name>]`: a colliding platform exits with the validation error code (2), as it did before (it failed to build). Render-bearing commands: unchanged flags; a colliding platform exits with the validation code (2).

Example, `opm platform check ./p` where `./p` enables `testing.opmodel.dev/catalogs/base@v1` and `testing.opmodel.dev/catalogs/base@v2`, both listing container and backup, and no provider:

```text
platform: ./p (argument)

defined contracts: 0

colliding contracts: 2
  testing.opmodel.dev/catalogs/base/resources/container@v1beta1
    defined by  testing.opmodel.dev/catalogs/base@v1, testing.opmodel.dev/catalogs/base@v2
  testing.opmodel.dev/catalogs/base/traits/backup@v1alpha1
    defined by  testing.opmodel.dev/catalogs/base@v1, testing.opmodel.dev/catalogs/base@v2
  (a colliding contract is left out of the defined, required, unfulfilled and comparable sections; keep one of its defining entries enabled)

fulfilled: yes
routable:  no <separator> 2 contracts collide, 0 over-subscribed
discriminated: yes
```

(`<separator>` is verdictLine's existing one.) Exit: validation code, `Printed: true`, message `platform ./p cannot generate a platform package: 2 colliding contract(s), 0 over-subscribed contract(s), 0 comparable transformer pair(s)`.

Render refusal, the same platform under `opm instance build ./inst --platform ./p`:

```text
render failed: <kernel message: 2 colliding contract(s): contract "..." is defined by 2 enabled registry entries (...); ...>
contract "testing.opmodel.dev/catalogs/base/resources/container@v1beta1": defined by more than one enabled registry entry: testing.opmodel.dev/catalogs/base@v1, testing.opmodel.dev/catalogs/base@v2
contract "testing.opmodel.dev/catalogs/base/traits/backup@v1alpha1": defined by more than one enabled registry entry: testing.opmodel.dev/catalogs/base@v1, testing.opmodel.dev/catalogs/base@v2
```

## Risks / Trade-offs

- [`fulfilled: yes` and `discriminated: yes` print under a collision] → core's stated limitation: a colliding key leaves `defined`, so it leaves `requiredBy`, `unfulfilled` and `comparable`. The CLI does not recompute them (no count in the CLI); the colliding section's note says which sections omit the key, and the exit status follows `routable`.
- [B renames a surface symbol] → code against what B reports under `surface`; list the difference under deviations.
- [A's tag differs from `v2.0.0-alpha.13`] → the colliding fixtures' module file uses the actual tag the supervisor reports.
- [The colliding fixture pins A's core explicitly while the other fixtures pin `schema.DefaultSchemaVersion()`] → required for the red run on library alpha.35 (whose default is alpha.12, where the platform does not build). After B's pin the two agree; a unit assertion is not added, because the fixture must keep evaluating on the pinned core whatever the default moves to.
- [An old `opm` (alpha.25 and earlier) given a colliding platform pinned to A's core] → it renders it, duplicates included when a bridge transformer is present (measured in B's planning). No CLI change can fix an old binary; the set's mitigation is ordering (B right after A, `deps:update` after B, C and D right after B).

## Migration Plan

None for users: no platform that evaluated before changes its result. Rollback is reverting the change; the library pin is the only coupling.

## Research & Decisions

### Where the CLI reads the collision

**Context**: The set's rule is one computation, in core, read by everyone else.
**Explored**: `grep -rn "OverSubscribed\|Routable\|DefinedBy" --include=*.go internal/` on 93d0bc3: the inventory reader is `internal/cmd/platform/check.go:119`; the render path prints `RenderDiagnostics` rows (`validation.go:150-175`) and unresolved rows through `cmdutil.FormatUnresolvedDemands`.
**Options considered**:
1. Print `Collisions` and `CollidingEntries` straight from the inventory, and `RenderDiagnostics.Collisions` from the render. No count; the check and the render name the same entries.
2. Infer collisions from `CollidingEntries` lengths or from registry iteration. A second computation, the defect class this change set keeps removing.
**Decision**: Option 1.
**Rationale**: Core computes it once; the library decodes it; the CLI prints it.

### What the vacuous wording guards

**Context**: The vacuous report exists so an empty inventory is not read as a verified clean platform.
**Explored**: experiment 06 case A reads empty `definedBy` and `providedBy` with two collisions.
**Options considered**:
1. Keep the condition and patch the verdict lines. A report saying "nothing was verified" above a refusal.
2. Add `len(Collisions) == 0` to the condition.
**Decision**: Option 2.
**Rationale**: A colliding platform was verified and failed.

### Where the colliding section goes

**Context**: A collision blinds the defined, required, unfulfilled and comparable sections.
**Options considered**:
1. First, before defined contracts. Most prominent, but moves every existing report's layout for a rare case.
2. Before over-subscribed, after unfulfilled (the approved design). Existing reports keep their layout; the refusing sections sit together.
**Decision**: Option 2, with the note naming the sections that omit the key.
**Rationale**: Keeps every non-colliding report byte-identical and groups the refusals.

### Red first

**Context**: The bug must be shown to reach `opm platform check` before the report changes.
**Explored**: A's core evaluates a colliding platform; library alpha.35's `Contracts()` ignores the new fields and decodes `routable: false` with empty `definedBy`.
**Decision**: Section 1 writes the colliding fixtures (own module file pinning A's core) and command tests first and runs them twice: on library alpha.35 (expected: the collision-only report is vacuous while the exit is non-zero; the collision-plus-over-subscription report reads `1 contract is over-subscribed` with no colliding section) and on B's head (expected: the same wording, since `Report` does not carry `Collisions` yet). Nothing red is committed.
**Rationale**: The first run shows the defect reaches the command on today's kernel; the second shows the part only this change fixes.
