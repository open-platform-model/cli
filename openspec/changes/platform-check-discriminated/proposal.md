## Why

`opm platform check` is the offline pre-flight for "is this platform usable", and its exit code was designed to agree with the tool that refuses a platform later: the operator's platform-package generation. Since opm-operator `platform-inventory-gate` (enhancement 0015 D5, merged 2026-09-18) that step refuses a platform whose enabled transformers carry comparable match predicates over a shared catalog-fulfilled contract, because every component the narrower one matches is also matched by the broader one and both would render. The check knows nothing of this: it reads six inventory fields off library `alpha.31`, prints no comparable pair and exits 0 on a platform the operator now rejects, which is the exact inversion the command exists to prevent. OQ9's resolution asked for the check to "land identically in the operator and `opm platform check`"; library `v1.0.0-alpha.32` (`Comparable`, `Discriminated`) makes that one more field pair on the same read.

## What Changes

- **Pins.** Library to `v1.0.0-alpha.32` and, through the workspace `task deps:update`, core to `v2.0.0-alpha.10` in `hack/platform`, `DefaultCorePin` and the templates (the catalog pins move to their latest builds with it). Required, not incidental: on alpha.32, `Contracts()` refuses a platform built against core `alpha.9` for lacking the report, so every test platform pinning `DefaultCorePin` would stop decoding.
- **The report prints comparable pairs.** `internal/platform.Report` carries the inventory's `Comparable` rows and a `discriminated` verdict. `Render()` gains a section listing each pair's broader transformer, narrower transformer and shared contracts, worded as the operator's refusal is (broader, narrower, over), and a third verdict line, `discriminated`. The empty-inventory case states the vacuous verdict for it as it does for the other two.
- **The exit code follows the operator.** A platform that is not discriminated exits with the validation error code, exactly as an over-subscribed one does, because both are conditions the generation step refuses on. An unfulfilled contract keeps exiting 0 (D18). The error names the counts of both refusals; the command's help text lists the new case.
- **The plurality fixture is split.** The test platform proving "catalog-fulfilled plurality is not over-subscription" pairs a transformer requiring the container resource alone with one requiring it plus a trait, which is a comparable pair on alpha.10. It gains a required label so it stays discriminated, and the label-less shape becomes the new undiscriminated fixture. The existing assertion keeps meaning what it says.
- **Not in this change.** A resolution-time gate for the render commands (`module build`, `module apply`, `instance apply`, `instance diff`): a local platform module has no generation step and the cluster arm generates one in `GenerateClusterModule`, so that gate belongs with the change that makes cluster resolution read the effective registry (0015 D6). The kernel's render-time over-subscription gate is untouched. No `--output json`.

## Impact

- **Affected commands and packages:** `opm platform check`; `internal/platform/check.go` (report value), `internal/cmd/platform/check.go` (exit rule and help text), their tests; `go.mod`, `hack/platform/cue.mod`, `internal/config/templates.go`, `templates/*/cue.mod` (pins).
- **Behaviour change:** a platform with comparable predicates exited 0 and now exits 2. Intended: the operator refuses that platform, so the pre-flight was wrong, not lenient. No other command changes.
- **SemVer:** MINOR (new report content and a new refusal case in a command that already refuses; library and core pin bumps are `fix(deps)`).
- **Complexity (Principle VII):** two fields, one accessor, one section and one verdict line on an existing value; no new type.
- **Flags:** none added.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-check`: the report requirement gains the comparable-pair listing and the third verdict; a new requirement makes an undiscriminated platform fail the command beside over-subscription; the build-failure requirement gains the core-with-inventory-but-without-the-report case.
