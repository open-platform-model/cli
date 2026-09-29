## Why

The single-provider rule for a provider-fulfilled contract was counted in two places that disagreed: core's `#Platform.#contracts` and the library render build's own guard. On two measured platform shapes (two majors of one provider catalog enabled; two providers enabled while the defining catalog is disabled or absent) the inventory read `routable: true` while every render was refused. This is change D of a four-change set (`orchestration.md`): core (A) now computes the count once as `#contracts.providedBy`, the library (B) reads it in the render build, exposes it as `ContractInventory.ProvidedBy`, and refuses a platform whose core predates it with `errors.PlatformCoreTooOldError`. This change makes `opm` consume B's release.

Two defects remain in the CLI once B is embedded:

- `opm platform check` short-circuits on an empty `definedBy` (`internal/platform/check.go:94-103`) and prints the vacuous `routable: yes` line. On a platform whose providers' defining catalog is disabled or absent, B's inventory reads `Routable: false`, so the command would exit with the validation code under a report that says the platform is routable.
- The over-subscribed rows print `required by <transformer FQNs>` from `RequiredBy`, which is keyed by defined contracts only: with the definer absent it prints "nothing on this platform", and with two majors of one catalog it names two transformers without saying which registry entries to disable.

And B's new refusal, a platform module pinning core older than `2.0.0-alpha.12`, reaches the CLI as a bare error on every render-bearing command and on `opm platform check`, with no word on where to re-pin.

## What Changes

- **`opm platform check` names the providing registry entries.** Each over-subscribed contract prints `provided by <registry keys>` from `ContractInventory.ProvidedBy` (path plus major, sorted), replacing the `required by <transformer FQNs>` line. A contract no enabled catalog defines prints `(defined by no enabled catalog)`. The heading prose says the count is per enabled registry entry, so two majors of one catalog are two providers.
- **The vacuous report is reserved for a platform with nothing to report.** The empty-inventory wording is printed only when no enabled catalog defines a contract AND no enabled transformer provides one; otherwise the full report runs, so a definer-disabled or definer-absent over-subscription is reported under its heading with `routable: no`.
- **An older core is worded as a re-pin.** When a render or `opm platform check` is refused with `PlatformCoreTooOldError`, the CLI prints the library's message and, for a platform directory the user named (`--platform <dir>` or the check's `[dir]`), a hint naming the directory and the exact `cue mod get opmodel.dev/core@<version>` to run there, `<version>` being the core release the kernel was verified against. Generated platforms (cluster CR, render deps) pin the kernel's verified core and never reach it.
- **Library pinned to B's release** (`read-provider-count-from-core`), which moves `schema.DefaultSchemaModule` to core `2.0.0-alpha.12`. **BREAKING** for hand-written platform modules: `opm` refuses to render against, or check, a platform pinning core older than `2.0.0-alpha.12`. Platforms of the two bug shapes now fail `opm platform check`, where they passed before; every render on them was already refused.
- `hack/platform` and `examples/` are re-pinned by the workspace `task deps:update` commit the supervisor lands on `main` first; this change does not touch them.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-check`: the report names the providing registry entries of every over-subscribed contract, counts per registry entry, stops reporting a definer-less over-subscription as vacuously routable, and words a platform whose core predates the provider count as a re-pin.
- `kernel-render`: a render refused because the platform pins a core older than the kernel reads is worded as a re-pin of the named platform directory.

## Impact

- **Release class: `fix(platform)!`, pre-GA, with a `BREAKING CHANGE:` footer on the library pin commit.** No flag or command is added or removed; the report text changes; `opm` refuses platforms pinning core older than `2.0.0-alpha.12` (through the library), and the two bug shapes now fail `opm platform check`. Pre-GA (`1.0.0-alpha.*`), so no major bump.
- Commands: `opm platform check`; every render-bearing command (`module build|vet|apply`, `instance build|vet|diff|apply`) for the re-pin hint only.
- Packages: `internal/platform` (`check.go` report, one shared re-pin hint), `internal/cmd/platform` (`check.go` help text and inventory error), `internal/workflow/render` (`validation.go` printer and `refusalHint`), `go.mod`/`go.sum`.
- Tests: `internal/platform/check_test.go`, `internal/cmd/platform/check_test.go` (three new inline fixture platforms), `internal/workflow/render/validation_test.go`, `tests/e2e/instance_build_test.go` (an older-core `--platform` render).
- Merge gate: the workspace `task deps:update` commit re-pinning `hack/platform` and `examples/` to core `2.0.0-alpha.12` is on `main` before this change merges; B's release is out before the last section.
- Downstream: the `opmodel.dev` generated CLI reference picks up the changed `opm platform check --help` on its next regeneration; hand-written platforms outside this repo (the personal `opm-kind-demo`, `opm-suite-installer/platform`) must re-pin core before they use an `opm` built on this change.
