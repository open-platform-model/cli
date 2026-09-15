## Why

Core `v2.0.0-alpha.9` derives `#Platform.#contracts`, library `v1.0.0-alpha.30` exposes it as `Platform.Contracts()`, and `catalog_opm` 4.1.0 populated the contract maps that make it non-vacuous. Enhancement 0015 names the CLI consumer of all three: a pre-flight that answers "is this platform usable" with no module in hand, reporting the contracts its catalogs define, the provider-fulfilled ones nothing implements, and the ones two catalogs compete for.

Today that answer only exists as a render refusal. A platform is diagnosed by trying to deploy something against it: `internal/workflow/render/validation.go` prints over-subscription from `kernel.RenderDiagnostics` after a render has already failed, and nothing in the CLI reads a platform's inventory outside one. A platform team cannot check a platform before handing it to anyone, and nothing consumes `Contracts()` at all.

The same libraries also added the defining catalog to the unresolved-demand row, which the CLI's formatter still drops.

## What Changes

- `opm platform check [dir]` (new command, new `platform` group): resolves a platform the way every other command does (the `--platform` flag, else the configured default), builds it through the kernel, reads `Contracts()`, and prints a report: the defined contracts and the catalog that defines each, the required demands per contract, the unfulfilled set, the over-subscribed set, and the two verdict lines. Read-only; it applies nothing and touches no cluster.
- **Exit code carries the verdict**: clean exits 0; over-subscription exits with the validation code, because `Routable: false` is the condition the operator's generation step refuses on. An unfulfilled contract exits 0 with the report naming it, because enhancement 0015 D18 makes it a report and never a gate. That asymmetry is the whole point of the command and is spelled out in the spec.
- `internal/platform/check.go` (new): the report type and its rendering, mirroring `internal/publish/check.go`'s `CheckReport` / `Render()` / `Clean()` split so a report is testable without a cobra command.
- `cmdutil.FormatUnresolvedDemands` gains the defining catalog: a demand whose contract an enabled catalog lists now reads "defined by `<catalog>`, nothing on this platform implements it" instead of the bare nothing-implements line. The row has carried `DefinedBy` since library alpha.30; the formatter has been discarding it.
- `internal/cmd/root.go` registers the new group.

**Not in this change**: enhancement 0015 D6, reproducing a cluster's render by fetching the effective registry. That needs the operator's registration acceptance to exist first, and it changes what "which platform" means; this command reads a platform module, not a cluster. D5's comparable-predicate duplicates are also out, still deferred by OQ9.

## Impact

- **Affected packages**: new `internal/cmd/platform/`, new `internal/platform/check.go`, modified `internal/cmdutil/output.go` and `internal/cmd/root.go`.
- **Downstream**: none. The command is additive and the formatter change only adds a clause to an existing message. A consumer asserting the old nothing-implements sentence verbatim would move, which is the CLI's own tests and nothing outside the repo.
- **Cluster access**: none. `platform check` is offline; it resolves a platform directory and builds it.
- **SemVer**: MINOR. New command, new exported report type in an internal package, no signature changes.
- **Complexity (Principle VII)**: one command, one report type, one added clause. The report type earns its place by being what makes the verdict testable without driving cobra; `internal/publish/check.go` is the precedent the repo already accepted for exactly this shape.

## Capabilities

### New Capabilities

- `platform-check`: what `opm platform check` reports about a platform's contracts, how it resolves which platform, and the exit-code contract that distinguishes a report from a refusal.

### Modified Capabilities

- `cmd-structure`: the root command gains a `platform` group, and `internal/cmd/platform/` joins the list of command packages.
- `cmdutil`: the unresolved-demand formatter names the defining catalog when the row carries one.
