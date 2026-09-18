# Design: platform-check-discriminated

## Context

See `proposal.md` § Why. The decision is `enhancements/0015` D5, completed in opm-operator by `platform-inventory-gate`; this change lands the same verdict in the CLI pre-flight, per OQ9's resolution.

Current state, read 2026-09-18 against cli main (library `v1.0.0-alpha.31`, `DefaultCorePin` `v2.0.0-alpha.9`):

- `internal/platform.Report` carries `Defined`, `RequiredBy`, `Unfulfilled`, `OverSubscribed` and the unexported `fulfilled`, `routable`; `NewReport` is the only constructor; `Routable()` is the one exit-status input; `Render()` prints the provenance line, the defined contracts, the two lists under headings with explanatory prose, and two `verdictLine`s formatted `%-10s` with a hard-coded "contract(s)" noun.
- `internal/cmd/platform/check.go` resolves, builds through `config.BuildPlatformModule`, reads `Contracts()`, prints the report and returns a validation `ExitError` (`Printed: true`) when `!report.Routable()`. Its help text lists the exit-code contract for over-subscribed and unfulfilled.
- Library `v1.0.0-alpha.32` adds `Comparable []ComparablePredicates{Broader, Narrower, Contracts}` and `Discriminated` to `ContractInventory`, and `Contracts()` errors on a platform whose `#contracts` lacks them: `platform #contracts carries no "comparable" field (core derives it from release 2.0.0-alpha.10 on)`.
- The command tests author catalogs inline in a temp platform module pinning `config.DefaultCorePin`; nothing pins a published catalog. `catalogFulfilledPluralityPlatform` pairs `velero/mirror` (container alone) with `k8up/schedule` (container plus the backup trait): on core alpha.10 that is one comparable row, mirror broader.
- The operator's refusal wording (`internal/controller/platform_inventory.go`): `<broader> (broader) and <narrower> (narrower) over <contracts>`, lists sorted.
- The workspace `task deps:update` is the only sanctioned way to move `cue.mod` pins and `DefaultCorePin`; it rewrites every sibling repo's pins in one pass, and `catalog_opm` still pins core alpha.9, so a workspace bump is due regardless of this change.

**Spike (task 1.2), run 2026-09-18 on library `v1.0.0-alpha.32` with the pins at core
`v2.0.0-alpha.10` / `opmodel.dev/catalogs/opm` `v4.4.0` / `opmodel.dev/catalogs/k8s`
`v1.0.0-alpha.3`:**

- Every command fixture decodes `comparable` and `discriminated` — none is refused by
  `Contracts()`, because they all pin `config.DefaultCorePin`, which now reads alpha.10.
- Exactly one fixture carries a row, as predicted:

  | Fixture | `discriminated` | `comparable` |
  | --- | --- | --- |
  | `routablePlatform` | `true` | 0 |
  | `baseOnlyPlatform` | `true` | 0 |
  | `catalogFulfilledPluralityPlatform` | `false` | 1 — broader `…/velero/transformers/mirror@1.4.0`, narrower `…/k8up/transformers/schedule@2.0.0`, over `…/base/resources/container@v1beta1` |
  | `overSubscribedPlatform` | `true` | 0 |

  `overSubscribedPlatform`'s two transformers are incomparable (velero requires the trait
  alone, k8up the resource plus the trait — neither predicate contains the other), which is
  why over-subscription and undiscrimination are separate verdicts on separate fixtures.
- `go test ./internal/cmd/platform/... ./internal/platform/... ./internal/config/...` is green
  on the new pins with no source change: nothing reads the row yet, and the sole row lands on a
  fixture whose assertions are about over-subscription. § The plurality fixture is split is
  therefore about keeping that fixture's *claim* honest, not about repairing a failure.
- Deviation from task 1.1 as written: the workspace `task deps:update` deliberately skips
  `.claude/worktrees`, so it cannot rewrite this checkout. The same tooling was run scoped to
  this worktree instead — `cue mod get` per `cue.mod` directory (`hack/platform`, the three
  templates), and `platform-pins.sh`'s own `awk` rewrite applied to `internal/config/templates.go`
  — so no pin was hand-picked and no sibling repo's tree was disturbed. `examples/cue.mod` is
  `task deps:pins:fixtures`' business and was left alone.

**Gate result (task 3.4), 2026-09-18:** `task fmt`, `task vet`, `task lint`, `task openspec:check`
(58 passed, 0 failed), `task test:unit` and `task test:integration` are green. Two e2e tests fail,
both pre-existing and both already tracked as cli issue 214:
`TestE2E_ThinEditor_ValuesRoundTrip` and `TestE2E_Delete_OperatorOwnedDelegates`, at `operator did
not reconcile generation 2 … within 3m`. The cause is not this change and not a missing operator —
with a verified-reconciling opm-operator `v1.0.0-alpha.14` the cluster Platform still stalls:

```
Platform cluster: Stalled=True MaterializeFailed:
  subscription version is not a concrete string:
  platform.#registry."opmodel.dev/catalogs/opm@v4".version: required field missing: version
ModuleInstance default/e2e-operator-owned: Ready=False PlatformNotReady
```

That release still materializes the Platform against the retired scalar-subscription shape; the
render switch that replaces it is on opm-operator `main` (PR 119) and in no release yet. Failing
since at least 2026-09-12, independent of any CLI change. The other 29 e2e tests pass. Note also
that `TestE2E_Operator_InstallUninstallLifecycle` is destructive by design and restores the dev
operator in a `t.Cleanup`; a suite run that hits go's default 10-minute test timeout is killed
before that cleanup, leaving the cluster without an operator for the next run.

No other test in the repo asserted the two-verdict report shape — `Routable()`, `NewReport` and the
verdict lines appear only in the two packages this change edits.

## Goals / Non-Goals

**Goals**

- The pre-flight's verdict agrees with the operator's gate on both refusals.
- A comparable pair is printed the way the operator names it, so one vocabulary covers both surfaces.
- Every existing assertion keeps meaning what it says; fixtures move, expectations do not weaken.

**Non-Goals**

- A gate at platform resolution for the render commands (0015 D6's change).
- Any change to the kernel's render-time diagnostics or `internal/workflow/render/validation.go`.
- `--output json`.

## Decisions

### The report gains the rows and a third verdict

```go
type Report struct {
	Resolution     Resolution
	Defined        map[string]string
	RequiredBy     map[string][]string
	Unfulfilled    []string
	OverSubscribed []string
	Comparable     []libplatform.ComparablePredicates // the library's row type, not a copy

	fulfilled, routable, discriminated bool
}

func NewReport(res Resolution, inv *libplatform.ContractInventory) Report
func (r Report) Render() string
func (r Report) Routable() bool      // unchanged
func (r Report) Discriminated() bool // new: true exactly when no pair is comparable
```

The library's row type is used directly (Principle VII: a CLI copy would exist only to be converted into). `discriminated` stays unexported like the other two booleans so `NewReport` remains the only constructor and the rows cannot drift from the verdict. `Routable()` keeps its meaning and its D18 test; the command combines the two accessors rather than the report growing a third "may generate" accessor, because the two refusals are worded and fixed differently.

### Rendering

After the over-subscribed section:

```
comparable transformer pairs: 1
  Enabled transformers whose match predicates are comparable over a shared
  catalog-fulfilled contract: every component the narrower one matches is
  also matched by the broader one, so both would render (enhancement 0015
  D5). A platform package cannot be generated from this platform until one
  catalog is disabled or the transformers are discriminated by a required
  label value or a required trait.
  testing.opmodel.dev/catalogs/velero/transformers/mirror@1.4.0 (broader)
    and  testing.opmodel.dev/catalogs/k8up/transformers/schedule@2.0.0 (narrower)
    over  testing.opmodel.dev/catalogs/base/resources/container@v1beta1

fulfilled: yes
routable:  yes
discriminated: no — 1 pair is comparable
```

Rows sort by broader then narrower, shared contracts sort within a row. The two existing verdict lines keep their `%-10s` layout byte for byte (tests assert `routable:  yes` with two spaces); the third label is longer than the pad and prints as `discriminated: <verdict>`. `verdictLine` gains the noun as a parameter (`contract`/`contracts` for two lines, `pair`/`pairs` for the third) instead of a second function. The empty-inventory branch gains `discriminated: yes (vacuously — no contract is defined)`.

### Exit code

```
opm platform check [dir] [--platform <dir>]
```

| Outcome | Exit |
| --- | --- |
| routable and discriminated, whatever `fulfilled` says | 0 |
| over-subscribed, undiscriminated, or both | 2 (`ExitValidationError`) |
| platform does not build, or its inventory cannot be read | 2 |
| resolved directory is not a platform module | 5 (`ExitNotFound`) |

The refusal error, returned with `Printed: true` after the report:

```
platform ./platforms/staging cannot generate a platform package: 0 over-subscribed contract(s), 1 comparable transformer pair(s)
```

The help text's exit-code list gains `comparable` beside `over-subscribed`, with the D5 one-liner.

### The plurality fixture is split

`catalogFulfilledPluralityPlatform`'s mirror transformer gains `requiredLabels: "testing.opmodel.dev/mirror": "true"`, so its predicate (container plus a label) and schedule's (container plus a trait) are incomparable and the "many suppliers, still routable" assertion holds on alpha.10 with `discriminated: yes` added. The label-less mirror becomes `undiscriminatedPlatform`, built on `routablePlatform`, and drives the new refusal test: mirror broader, schedule narrower, container shared. Both fixtures stay inline and core-only, as every fixture in the file is.

### Files touched

| File | Change |
| --- | --- |
| `go.mod`, `go.sum` | library `v1.0.0-alpha.32` |
| `hack/platform/cue.mod/module.cue`, `internal/config/templates.go`, `templates/*/cue.mod/module.cue` | pins moved by the workspace `task deps:update` |
| `internal/platform/check.go`, `check_test.go` | `Comparable`, `discriminated`, `Discriminated()`, the section, the third verdict, the noun parameter |
| `internal/cmd/platform/check.go`, `check_test.go` | the combined exit rule, the help text, the fixture split, the new tests |

## Research & Decisions

### Where the pins move

**Context**: on alpha.32, `Contracts()` refuses an alpha.9-built platform, and every command test pins `DefaultCorePin`; `DefaultCorePin` and the `cue.mod` pins may only move through the workspace `task deps:update`, which rewrites every sibling repo too.
**Explored**:
1. Hand-edit `DefaultCorePin` and `hack/platform`'s pin: minimal diff, but the workspace rule forbids hand-edited `cue.mod` pins for a reason (the Go pins and the module pins drift).
2. Run the workspace `task deps:update`, commit only the cli's rewritten files here, leave the sibling repos' rewritten trees for their own `fix(deps)` PRs: the sanctioned path, and the bump is due workspace-wide (catalog_opm still pins alpha.9).
3. Bump the library first in its own PR and wait for Dependabot: leaves main with a library that refuses the seeded platform's core until the pin follows.
**Decision**: 2, in one section with the library bump, as one `fix(deps)` commit.
**Rationale**: the two pins are one fact (the library's verified core release) and must land together; the precedent commit on this repo bundled core, catalogs and library the same way. The catalog pins moving with it is accepted, since the seeded platform wants the newest build anyway.

### The library's row type is used, not mirrored

**Context**: `Report`'s other list fields are `[]string`; a comparable row has three parts.
**Explored**:
1. A CLI row type with the same three fields: isolates the report from the library's naming, at the cost of a copy loop and a second definition of the same fact.
2. `[]libplatform.ComparablePredicates` directly: no conversion, one definition.
**Decision**: 2.
**Rationale**: the report already imports the library type for construction; the row is data the command only prints, and a rename in the library would be a compile error here either way.

### Exit rule combined in the command, not in a new accessor

**Context**: `Routable()` is documented as the only exit input, and its test pins D18's asymmetry.
**Explored**:
1. A `Generable()` accessor returning `routable && discriminated`, the command reading only that.
2. Two accessors, the command refusing when either is false.
**Decision**: 2.
**Rationale**: the two refusals have different fixes and the error names both counts; a combined accessor would hide which one fired. `Routable()`'s doc comment is corrected to "one of the two values that decide the exit status", and a sibling test pins that `Discriminated()` is the other while `fulfilled` still decides nothing.

### No resolution-time gate here

**Context**: the render commands build a platform (a local module, or one generated from the cluster CR) and nothing refuses an undiscriminated one; the render then doubles.
**Explored**: reading `Contracts()` in `resolvePlatformEnv` for every render command in this change; deferring.
**Decision**: defer to the 0015 D6 change.
**Rationale**: for the cluster arm the CLI generates the platform module itself (`GenerateClusterModule`), which is the step the operator gates, and that arm is about to change shape to read the effective registry; adding the gate twice, before and after that change, is the wrong order. The pre-flight is the surface this change owns.

## Risks / Trade-offs

- [A platform that passed `platform check` yesterday exits 2 today] → intended and named in the proposal; the report says which two transformers and how to discriminate them, and the operator refuses the same platform.
- [The workspace bump moves catalog pins the seeded platform and templates carry] → the same `fix(deps)` commit; `task test:unit` covers the seeded platform's build; e2e and fixture pins are the root `task deps:pins:fixtures`' business, not this change's.
- [Sibling repos' trees are left rewritten by `task deps:update`] → stated in the tasks; each sibling ships its own `fix(deps)` PR or resets its tree; nothing here depends on them.
- [Comparable rows arrive in comprehension order] → sorted before printing, so the report is deterministic; the determinism test covers the new section.

## Migration Plan

Three sections in one PR, squash title `feat(platform): fail opm platform check on comparable transformer predicates`. release-please cuts a minor. Rollback is a revert; the pin bump alone is safe to keep.

## Open Questions

None.
