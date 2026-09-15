# Design: platform-check-command

## Context

See `proposal.md` § Why. The enhancement is `enhancements/0015`; this command is the CLI consumer its `02-design.md` names for D1's inventory, D2's arity rule and D18's report-versus-gate split.

Current state, read 2026-09-15 against cli `v1.0.0-alpha.20` (library `v1.0.0-alpha.30`):

- **There is no `platform` command group.** `internal/cmd/` holds `catalog`, `config`, `instance`, `module`, `operator` and `registry`. "Platform" in the CLI today means the `--platform` flag, the `internal/platform/` resolution package, and the seeded `~/.opm/platform` module.
- Platform resolution is `internal/platform.Resolve`: the `--platform` flag, then a cluster Platform CR for commands that opt in by passing a getter, then the configured default beside the config file. `Resolution.Describe()` already produces the one-line provenance string this command should print.
- `config.BuildPlatformModule(ctx, dir, registry)` builds a platform through `config.NewKernel` and `AcquirePlatformFromDir`, and shapes the build failure into a CLI error with a hint. `opm config vet` is its only command caller.
- `Platform.Contracts()` exists in the pinned library and **nothing in the CLI calls it**. Over-subscription is surfaced only from `kernel.RenderDiagnostics` after a render refusal, in `internal/workflow/render/validation.go`.
- `internal/publish/check.go` is the repo's report precedent: a `CheckReport` with `Render()` and `Clean()`, driven by a thin cobra command that prints the report, then prints findings and returns a typed `ExitError` with `Printed: true`.
- There is no shared in-process command-test helper; tests construct the cobra command and call `Execute()`.

## Goals / Non-Goals

**Goals**

- Answer "is this platform usable" before anything is deployed against it, offline.
- Make the verdict machine-readable through the exit code, in a way that agrees with where the system actually refuses.
- Stop discarding the defining catalog the library now puts on every unresolved-demand row.

**Non-Goals**

- Enhancement 0015 D6, fetching a cluster's effective registry. It depends on the operator's registration acceptance and redefines what "which platform" means.
- D5's comparable-predicate duplicates, still deferred by OQ9.
- A `--output json` mode. No command in this repo has a shared one, and adding a schema before a consumer asks for it is surface without a reader; the report is built as a value so a later flag is a rendering choice, not a rewrite.

## Decisions

### The report is a value, rendered by the command

`internal/platform/check.go`:

```go
// Report is what `opm platform check` prints: the platform's contract
// inventory, plus where the platform came from.
type Report struct {
	Resolution     Resolution          // provenance, already carried by Resolve
	Defined        map[string]string   // contract FQN -> defining catalog registry key
	RequiredBy     map[string][]string // contract FQN -> implementation FQNs
	Unfulfilled    []string
	OverSubscribed []string

	// The inventory's two verdicts, unexported so NewReport stays the only
	// constructor and the lists cannot drift from the booleans.
	fulfilled bool
	routable  bool
}

func NewReport(res Resolution, inv *libplatform.ContractInventory) Report
func (r Report) Render() string
// Routable reports whether the platform may be generated from; it is the
// only value that decides the command's exit status (0015 D18).
func (r Report) Routable() bool
```

The two booleans are unexported deliberately: `Routable` cannot be both a field and the accessor the exit-code rule needs, and keeping them off the exported surface makes `NewReport` the only way to build a `Report`, so no caller can hand `Render()` a verdict its own lists contradict. `fulfilled` has no accessor because nothing outside `Render()` reads it; adding one would be surface without a reader.

The command builds the platform, reads `Contracts()`, constructs the report, prints `Render()`, and returns nil or a validation `ExitError`. Splitting the value from the command is what `internal/publish/check.go` already does here, and it is what lets the exit-code asymmetry be unit-tested without cobra.

### The exit code follows where the system refuses, not the severity of the word

Clean exits 0. Over-subscription exits `ExitValidationError`. An unfulfilled contract exits 0.

This is the only interesting decision in the change, and it is deliberate rather than an oversight: D18 splits the inventory into a report (`fulfilled`) and a gate (`routable`), and puts the gate at platform-package generation. A pre-flight that failed on `unfulfilled` would refuse platforms the operator will happily generate, which teaches users to ignore it. A pre-flight that passed on `overSubscribed` would bless a platform the generation step refuses. The exit code therefore tracks `Routable` alone, and the report explains the other list in words.

### Resolution reuses the existing precedence, plus an optional argument

The command takes an optional positional platform directory. Present, it wins; absent, `platform.Resolve` decides with the flag and the configured default. The cluster-CR arm is **not** wired in: this command reads a platform module, and a cluster's effective registry is D6's subject. `Resolution.Describe()` prints as the report's first line, so a report always says which platform it describes.

### The formatter change is additive

`cmdutil.FormatUnresolvedDemands` currently prints either the alternatives line or the nothing-implements line. The defining catalog is orthogonal to that choice — the capability requires every row carrying one to name it — so each of the two existing branches gains a defined-by variant rather than the formatter gaining a single third case:

```
component "api": unresolved trait demand "…/traits/backup@v1alpha1"
  defined by "opmodel.dev/catalogs/opm@v4", implemented by nothing on this platform

component "web": unresolved resource demand "…/resources/container@v1beta1"
  defined by "opmodel.dev/catalogs/opm@v4"
  implemented at: …/resources/container@v2
```

Folding both into one case would drop the alternatives line — the actionable half of the diagnostic — from exactly the rows that have one. Rows without a defining catalog keep today's wording exactly, so the only tests that move are the ones asserting a defined-but-unimplemented case, which is the case that could not previously be distinguished.

## Research & Decisions

### Why a new command rather than extending `opm config vet`

**Context**: `opm config vet` already builds the configured platform module and prints a check line for it, so "the platform builds" is answered today.
**Explored**: adding the inventory report to `config vet`; a new `platform check`.
**Decision**: a new command in a new group.
**Rationale**: `config vet` validates the CLI's own configuration, and its platform check exists because the seeded module is part of that configuration. It takes no platform argument and has no reason to grow one. A platform team checking a platform module in a repository is not validating their `~/.opm`; making them run `config vet` to do it would be the wrong noun, and the noun-first grouping rule in `cmd-structure` is explicit. The boundary is stated in the new capability so the overlap is deliberate rather than accidental.

### Reading the inventory costs the render path nothing

**Context**: the fold runs inside every platform build, so a naive worry is that reading it is expensive.
**Explored**: `Contracts()` in the pinned library decodes `#Platform.#contracts` from the already-built value on demand, per-field, and `defined` is deliberately not decoded.
**Decision**: read it in the command only; change no render path.
**Rationale**: the decode is lazy and local to this command. Nothing else in the CLI gains a call, so a render pays exactly what it paid before.

## Risks / Trade-offs

- [A user reads "exit 0" as "nothing to do" while the report names an unfulfilled contract] → the report states the consequence in words, and the capability spells out the asymmetry so it survives a later reader's instinct to "fix" it.
- [The report is vacuous against catalogs that list no contracts] → detected and reported as its own sentence rather than as a clean bill of health, which is the failure mode 0015's operational notes name.
- [A platform built against a pre-alpha.9 core] → `Contracts()` returns an error naming the missing field; the command surfaces it with the core release required rather than printing an empty report.
- [A later `--output json` would fix the report's shape as an interface] → accepted; the value type exists so that flag is additive, and nothing serialises it today.

## Migration Plan

Three sections in one PR, squash title `feat(platform): add opm platform check`. release-please cuts a minor. No existing command changes behaviour except the unresolved-demand line, which gains a clause only for rows that previously could not express it. Rollback is a revert.

## Open Questions

None.
