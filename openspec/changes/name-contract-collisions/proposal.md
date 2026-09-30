## Why

A platform that enables two majors of one catalog whose contract maps share keys makes core's `#Platform.#contracts` fold fail to evaluate today (enhancement 0026 OQ17, experiments 01 and 06). This is change D of a five-change set (`orchestration.md`): core (A) folds only single-definer keys and reports the rest as `collisions` and `collidingEntries` with `routable` false; the library (B) decodes them as `ContractInventory.Collisions` and `CollidingEntries` and refuses every render of such a platform with a typed `ContractCollisionsError`. This change makes `opm` name the collision instead of misreporting it.

Measured against `origin/main` 93d0bc3 (library `v1.0.0-alpha.35`), once a colliding platform evaluates the CLI words it wrong:

- `opm platform check` short-circuits to the vacuous report when `Defined` and `ProvidedBy` are both empty (`internal/platform/check.go:103`). A platform whose only contracts collide has both empty (experiment 06 case A), so the report prints `routable:  yes (vacuously ...)` while the command exits with the validation code.
- With a provider present, the routable verdict counts only `OverSubscribed` (`check.go:164`), so it reads `no` followed by `0 contracts are over-subscribed`, and the exit message (`internal/cmd/platform/check.go:133`) names 0 over-subscribed contracts and 0 comparable pairs: a refusal with no stated reason.
- A render refused with B's collision cause prints no row for it (`internal/workflow/render/validation.go:150-175`), and the CLI's own unresolved-demand formatter (`internal/cmdutil/output.go:75-92`) words a demand on a colliding key as `nothing on this platform implements this contract`.

## What Changes

- **`opm platform check` reports colliding contracts.** A `colliding contracts: N` section, printed before the over-subscribed section, lists each colliding key with the registry entries defining it, read from the inventory (`Collisions`, `CollidingEntries`), and closes with a note that a colliding contract is left out of the defined, required, unfulfilled and comparable sections and that one of its defining entries must stay enabled. The vacuous report is reserved for a platform where nothing is defined, provided or colliding. The routable verdict names the collision count beside the over-subscribed count, and the exit message names colliding, over-subscribed and comparable counts.
- **A colliding render is worded.** The render refusal prints one row per colliding contract, first, naming the registry entries defining it; an unresolved demand on a colliding key says so instead of "nothing implements"; the unresolved-demand hints are withheld under a collision, whose fix comes first. A `NotRoutableError` (routable false with no row explaining it) prints the kernel's message under `render failed`.
- **`opm platform check --help`** lists colliding contracts in the inventory sentence and the exit-code table.
- **Library pinned to B's release** (`refuse-colliding-contracts`), which moves `schema.DefaultSchemaModule` to A's core tag. No platform that renders today is refused: every core before A's tag fails to evaluate a colliding platform, and B raises no floor.
- `hack/platform`, `examples/` and `templates/` are not touched; the workspace `task deps:update` owns their pins and runs only after B's release.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-check`: the report names colliding contracts and their defining registry entries, a colliding platform is never worded as vacuous, and a collision fails the command with its count named.
- `kernel-render`: a render refused because the platform's enabled entries share contract keys prints a row per colliding contract, first, naming the entries to disable.

## Impact

- **Release class: `fix(platform)`, PATCH, pre-GA.** Sections: `fix(platform)`, `fix(render)`, `fix(deps)`. No flag, command or exit code is added or removed. A colliding platform failed before (it did not evaluate) and still fails, now naming why. Nothing in the CLI derives a count: every list is printed as the inventory or the render diagnostics carry it.
- Commands: `opm platform check`; every render-bearing command (`module build|vet|apply`, `instance build|vet|diff|apply`) for the refusal wording.
- Packages: `internal/platform` (`check.go`), `internal/cmd/platform` (`check.go` help and exit message), `internal/workflow/render` (`validation.go`), `internal/cmdutil` (`output.go`, unresolved-demand formatter), `go.mod`/`go.sum`.
- Tests: `internal/platform/check_test.go`, `internal/cmd/platform/check_test.go` (colliding fixture platforms with their own module file pinning A's core), `internal/workflow/render/validation_test.go`, `internal/cmdutil/output_test.go`, `tests/e2e/instance_build_test.go` (a colliding `--platform` render, with and without `--skip-unprovided`).
- Merge gate: B's release is out before the last section; `go.mod` never merges on a pseudo-version.
- Downstream: the `opmodel.dev` generated CLI reference picks up the changed `opm platform check --help` on its next regeneration (supervisor follow-up).
