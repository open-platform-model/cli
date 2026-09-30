## Context

See proposal.md for the two bugs and why the render build's count becomes the only one. This is change D of the set in `orchestration.md`; its input is change B's library release (`read-provider-count-from-core`), whose surface is:

- `platform.ContractInventory.ProvidedBy map[string][]string`: every provider-fulfilled contract FQN some enabled transformer requires, defined by an enabled catalog or not, to the sorted registry keys (`path@major`) of the enabled entries whose transformers require it. `OverSubscribed` is exactly its keys with two or more entries.
- `errors.PlatformCoreTooOldError{Platform, Field, Since}`, pointer receiver, returned by `Platform.Contracts()` for every since-guard (the missing `#contracts` case included, `Since: "2.0.0-alpha.9"`) and, wrapped as `render refused before staging: %w`, by `Kernel.Render` before staging when the platform package lacks `#contracts.providedBy`. Not a `*kernel.RenderError`.
- `schema.ContractsProvidedBy`, `schema.ProvidedBySince = "2.0.0-alpha.12"`, and `schema.DefaultSchemaModule` at `opmodel.dev/core@v2.0.0-alpha.12`.
- The render's over-subscription rows (`OverSubscribedContract{Key, Catalogs}`) and refusal text are unchanged, so `formatRenderDiagnostics` (`internal/workflow/render/validation.go:151-153`) already prints the right catalogs and needs no edit.

The CLI keeps no provider count of its own: the only inventory reader is `opm platform check` (`internal/cmd/platform/check.go:114`, `p.Contracts()`), and the report (`internal/platform/check.go`) prints what the inventory carries. What goes wrong on B's release is wording, verified against `origin/main` e9a9e4e:

- `Report.Render` returns early when `len(r.Defined) == 0` (`internal/platform/check.go:94-103`) with all three verdicts worded `yes (vacuously ...)`, whatever the inventory's booleans say. On a definer-disabled or definer-absent over-subscription B's inventory reads `Routable: false`, so `runPlatformCheck` exits with the validation code (`internal/cmd/platform/check.go:128-135`) under a report that says `routable: yes`.
- The over-subscribed rows print `required by <RequiredBy[fqn]>` (`internal/platform/check.go:128-131`). `RequiredBy` is keyed by defined contracts only, so a definer-less row prints "nothing on this platform"; and it names transformers, while the fix the heading prose asks for ("until one of the competing catalogs is disabled") is a registry entry.
- A `PlatformCoreTooOldError` reaches the user unworded: from `Contracts()` as `reading the contract inventory of the platform at <dir>: <err>` (`internal/cmd/platform/check.go:114-120`), and from `Render` through `printValidationError`'s generic funnel (`internal/workflow/render/validation.go:60`) with no hint (`refusalHint`, `validation.go:86-107`).

Pins in this repo, verified: `go.mod:13` pins library `v1.0.0-alpha.34`. The only in-tree platform module is `hack/platform/` (core `v2.0.0-alpha.10`, `hack/platform/cue.mod/module.cue:16`); `examples/cue.mod/module.cue:13` pins the same. Both are moved by the workspace `task deps:update` commit the supervisor lands on `main` (the merge gate below), and B's floor would refuse every `--platform hack/platform` render in `tests/e2e/instance_build_test.go` (`seedPlatform`, line 95), `internal/config/platform_test.go` (`hackPlatformDir`, line 22) and `tests/integration/{platform-build,render-parity}` until it lands. Generated platforms move by themselves: the cluster-CR and deps sources pin `schema.DefaultSchemaVersion()` (asserted at `internal/platform/generate_test.go:85` and `moduledeps_test.go:186`), and the platform cache is keyed by a hash over the generated files (`internal/platform/generate.go:111-133`), so a platform cached at alpha.10 is never reused. The platform check test fixtures pin `schema.DefaultSchemaVersion()` too (`internal/cmd/platform/check_test.go:27-30`). The modules pinning an older core (`internal/workflow/render/testdata/skip-unprovided`, `internal/instinit/testdata/initvalues`, `templates/*`, `tests/fixtures/*`, `tests/e2e/testdata/*`) are modules, not platforms, and B's floor reads only the platform.

## Goals / Non-Goals

**Goals:**

- `opm platform check` prints the providing registry keys of every over-subscribed contract, from `ProvidedBy`, and never words a false verdict as a vacuous yes.
- Red-first proof: command tests for the two bug shapes run red on the current pin (exit 0, the bug) and red again on B's head (the report wording), before the report changes.
- `PlatformCoreTooOldError` is worded as a re-pin of the directory the user named, on `opm platform check` and on every render-bearing command.
- The library pinned at B's release.

**Non-Goals:**

- Any count, fallback or presence check in the CLI. If the report and the render disagree, the fix is in core or the library.
- The render refusal text for over-subscription (unchanged in B; `formatRenderDiagnostics` already prints the rows' catalogs).
- `hack/platform`, `examples/`, `templates/*` pins: the workspace `task deps:update` owns them (supervisor).
- The operator (change C) and hand-written platforms outside this repo (follow-ups in `orchestration.md`).

## Decisions

### 1. The over-subscribed row prints `provided by`, from `ProvidedBy`

`Report` gains `ProvidedBy map[string][]string`, filled from `inv.ProvidedBy` in `NewReport` (the only constructor). Each over-subscribed row prints:

```text
  <fqn> (defined by <registry key>)          or  (defined by no enabled catalog)
    provided by  <sorted registry keys, comma-separated>
```

The `required by <transformer FQNs>` line is dropped from this section. The transformers stay visible under "defined contracts" (`implemented by`), where `RequiredBy` is complete. The heading prose becomes:

```text
over-subscribed contracts: N
  Provider-fulfilled contracts required by transformers of more than one
  enabled registry entry. Two majors of one catalog are two entries. A
  platform package cannot be generated from this platform until all but one
  of the providing entries is disabled.
```

`definedByClause("")` returns ` (defined by no enabled catalog)` instead of `""`. It is only called for over-subscribed and unfulfilled rows, and an unfulfilled row is always defined (core scopes `unfulfilled` to defined contracts), so only definer-less over-subscribed rows change. The keys are printed through `sortedCopy`, never trusting the inventory's order, like every other list in the report.

**Alternatives.** Keeping `required by` beside `provided by`: two lines naming the same competition in different units; on the definer-less shape the first would read "nothing on this platform", contradicting the second. Deriving the keys from transformer FQNs: that is a second count, the defect this change set removes.

### 2. The vacuous branch requires an empty `ProvidedBy`

The early return at `internal/platform/check.go:94` fires only when `len(r.Defined) == 0 && len(r.ProvidedBy) == 0`. On that platform nothing is defined and nothing provides, so all three verdicts are true and the vacuous wording stays byte-identical. Otherwise the full report runs; when `Defined` is empty it prints, in place of the "defined contracts" block:

```text
defined contracts: 0
  The enabled catalogs define no contracts. The provider counts below come
  from the enabled transformers' own requirements.
```

and then the unfulfilled (always empty here), over-subscribed and comparable sections and the three ordinary verdict lines, each read from the inventory's boolean. The invariant the tests pin: a verdict line never says `yes` when the inventory's boolean is `false`.

Example, definer absent (two provider entries, no base catalog):

```text
platform: /tmp/p (argument)

defined contracts: 0
  The enabled catalogs define no contracts. The provider counts below come
  from the enabled transformers' own requirements.

over-subscribed contracts: 1
  Provider-fulfilled contracts required by transformers of more than one
  ...
  testing.opmodel.dev/catalogs/base/traits/backup@v1alpha1 (defined by no enabled catalog)
    provided by  testing.opmodel.dev/catalogs/k8up@v2, testing.opmodel.dev/catalogs/velero@v1

fulfilled: yes
routable:  no (the existing verdictLine wording: 1 contract is over-subscribed)
discriminated: yes
```

The command's exit is unchanged in form: validation code, `Printed: true`, message `platform <dir> cannot generate a platform package: 1 over-subscribed contract(s), 0 comparable transformer pair(s)`.

### 3. One re-pin hint, owned by `internal/platform`

```go
// CoreRepinHint is the remediation for a platform module pinning a core
// older than a field the kernel reads: the exact command re-pinning core
// in dir to the release the kernel was verified against.
func CoreRepinHint(dir string) string
// "the platform module at <dir> pins a core release older than this opm reads:
//  run 'cue mod get opmodel.dev/core@<schema.DefaultSchemaVersion()>' in <dir>"
```

The version is `schema.DefaultSchemaVersion()`, not the error's `Since`: `Since` names the floor that fired first (for an alpha.6 platform it is `2.0.0-alpha.9`, `#contracts`), and re-pinning to it would only reach the next floor. The kernel's verified release satisfies every floor. The error's own message still names `Field` and `Since`.

- **`opm platform check`** (`internal/cmd/platform/check.go:114-120`): when `errors.As(err, &*liberrors.PlatformCoreTooOldError)`, the `ExitError`'s `Err` becomes a `*oerrors.DetailError{Type: "platform module error", Message: "reading the contract inventory of the platform at <dir>: <err>", Location: dir, Hint: platform.CoreRepinHint(dir), Cause: err}`. `DetailError` unwraps (`pkg/errors/errors.go:74`), so `errors.As` to the library type still holds, and the existing message substrings (`#contracts`, `2.0.0-alpha.9`, `comparable`, `2.0.0-alpha.10`, the directory) are kept. Any other `Contracts()` error keeps today's plain wrap. The check only ever resolves a directory the user named (`[dir]`, `--platform`) or the cluster Platform; a cluster-generated platform pins the verified core, so the hint always names a user directory in practice. It is still emitted only for `SourceArgumentDir` and `SourceFlagDir`.
- **Render-bearing commands** (`internal/workflow/render/validation.go`): `printValidationError` gains a case beside the skew one, printing `render failed: <err>` verbatim (the library message already names platform, field and release). `refusalHint` gains, before its unresolved checks, `PlatformCoreTooOldError` with `res.Source == platform.SourceFlagDir` returning `platform.CoreRepinHint(res.Dir)`; any other source returns `""`. The exit stays `ExitValidationError` with `Printed: true` (`render.go:215-219`, unchanged).

**Alternatives.** A hint for generated sources: they pin `DefaultSchemaVersion()` by construction, so the branch would be dead code (Principle VII). Rewriting the library's message: the operator (C) prints the same error; one message in two frontends is the library's job, the CLI adds only its location-specific fix.

### 4. Pins and sequencing

Development runs against B's pushed head as a Go pseudo-version (`go get github.com/open-platform-model/library@<B head sha>`); the last section replaces it with B's release (expected `v1.0.0-alpha.35`; the supervisor supplies the actual version). `go.mod` never merges on a pseudo-version: the last section is the pin, and it stops and reports if B's release is not out.

### 5. Section plan

1. **Red first, then the report names the providers** (`fix(platform)`): the three inline fixture platforms and their command tests run red on library alpha.34 (exit 0 on both bug shapes) and red on B's head (vacuous wording, no `provided by`); the pseudo-version pin; the report change; unit and command tests green.
2. **The deps platform keeps its generated pins** (`fix(platform)`): § 6, added during section 1.
3. **An older core is a re-pin** (`fix(render)`): `CoreRepinHint`, the check's `DetailError`, the render printer and hint, the typed-error assertions on the existing alpha.6 and alpha.9 check tests, a new alpha.11 check test and an e2e older-core `--platform` render.
4. **Pin B's release** (`fix(deps)!` with a `BREAKING CHANGE:` footer): the release replaces the pseudo-version; full gates on `kind-opm-dev`. As delivered, B's release was out before section 1 pinned the library, so section 1 pinned it and carries the `BREAKING CHANGE:` footer as `fix(platform)!`; section 4 needs no commit and runs only its gates.

### 6. The deps platform's local module file carries every generated pin

Found in section 1's e2e gate: `TestE2E_InstanceBuild_InstanceDepsHonorThePackageReplacement` failed with `platform "module-deps" carries no "providedBy"`. `carryReplacements` (`internal/platform/moduledeps.go`) wrote the generated module's `cue.mod/local-module.cue` with only the replaced paths. In main-module mode `cue/load` (cue v0.17.1, `modfile.ParseLocal`) takes the dependency list from the local file alone, restoring version and default mark from `module.cue` only for a path the local file names, so core resolved from the replaced catalog checkout's own pin (`v2.0.0-alpha.10`) rather than the generated floor (`v2.0.0-alpha.12`), and B's floor refused the render. The local file now lists every dependency of the generated module file, with `replaceWith` only on the replaced paths (`FormatLocal` omits the versions and default marks `module.cue` already states). The e2e fixture's catalog keeps its older core pin: that is the real-world case (a catalog checkout pinning an older core than the kernel floors to). Its own section, before the re-pin section, with a red unit test first.

## Command surface

No command, flag or exit code is added or removed.

- `opm platform check [dir] [--platform <dir>] [--kubeconfig <path>] [--context <name>]`: the `Long` help's inventory sentence says "the provider-fulfilled contracts required by transformers of more than one enabled registry entry, with the registry keys providing each"; the exit-code table is unchanged (`over-subscribed` and `comparable` exit 2, the validation code; `unfulfilled` exits 0).
- Render-bearing commands: unchanged flags; an older-core `--platform` directory exits with the validation code (2) and the hint.

Error output, older-core render (`opm instance build ./inst --platform ./p`, `./p` pinning core `v2.0.0-alpha.11`):

```text
render failed: render refused before staging: <library PlatformCoreTooOldError message naming the platform, providedBy and 2.0.0-alpha.12>
Hint: the platform module at ./p pins a core release older than this opm reads: run 'cue mod get opmodel.dev/core@v2.0.0-alpha.12' in ./p
```

## Risks / Trade-offs

- [The workspace `task deps:update` commit re-pinning `hack/platform` and `examples/` is not on `main` when D starts] → every `--platform hack/platform` test is refused on B's head. Task 1.1 checks `origin/main` and stops with a report if it is missing; this change never edits those files.
- [B's `Contracts()` decode order makes an alpha.9 platform name `providedBy` instead of `comparable`] → B appends the `providedBy` row, so `comparable` should still fire first (`internal/cmd/platform/check_test.go:488-505` stays valid). Plausible, not measured: section 1 runs that test on B's head and records the result; if it changed, the test asserts what the typed error reports and the change is listed under deviations.
- [The e2e older-core render builds a core-only platform module pinned to `v2.0.0-alpha.11`] → it carries no catalog, so without B's floor the render would fail unmatched rather than pass: the test asserts the typed error and the hint, never only a non-zero exit.
- [The report reads `ProvidedBy` but still shows transformer FQNs under "defined contracts"] → intentional: the transformers are what implement a contract, the registry keys are what a user disables.
- [A hand-written platform outside this repo pins core older than `2.0.0-alpha.12`] → it is refused by every `opm` built on this change, with the re-pin hint. Named in the `BREAKING CHANGE:` footer; the personal `opm-kind-demo` and `opm-suite-installer` platforms are follow-ups in `orchestration.md`.

## Migration Plan

Pre-GA. The `BREAKING CHANGE:` footer on the commit that pins B's release (section 1, as delivered) says: platform modules passed with `--platform` or to `opm platform check` must pin `opmodel.dev/core` at `v2.0.0-alpha.12` or later (`cue mod get opmodel.dev/core@v2.0.0-alpha.12` in the module); platforms enabling two majors of one provider catalog, or two providers of a contract whose defining catalog is disabled or absent, now fail `opm platform check` (every render on them already failed). Rollback is reverting the change; the library pin is the only coupling.

## Research & Decisions

### Where the CLI reads the count

**Context**: The change set's rule is one count, computed by core and read by everyone else.
**Explored**: `grep -rn "OverSubscribed\|RequiredBy\|Routable" --include=*.go` on `origin/main` e9a9e4e: the only inventory reader is `internal/cmd/platform/check.go:114`; the render path prints `RenderDiagnostics.OverSubscribed` rows (`validation.go:151`). No CLI code counts providers.
**Options considered**:
1. Print `ProvidedBy` straight from the inventory - no count, matches the render rows' `catalogs` exactly.
2. Rebuild provider keys from `RequiredBy` transformer FQNs - a second count, and impossible for a definer-less contract.
**Decision**: Option 1.
**Rationale**: It is the value the render's rows are built from, so the check and the render name the same entries.

### What the vacuous wording guards

**Context**: The early return was added so an empty inventory is not read as a verified clean platform.
**Explored**: `internal/platform/check.go:94-103` and the `platform-check` spec scenario "A platform whose catalogs list nothing reports an empty inventory".
**Options considered**:
1. Keep the early return on `len(Defined) == 0` and patch the verdict lines - a report that says "nothing was verified" above a refusal.
2. Narrow the condition to "nothing defined and nothing provided" - the vacuous case stays exactly what the scenario describes.
**Decision**: Option 2.
**Rationale**: With B's inventory, a definer-less platform can be over-subscribed; it has been verified, and it failed.

### Which release the re-pin hint names

**Context**: `PlatformCoreTooOldError.Since` is the floor that fired, which may be older than another floor the platform also misses.
**Options considered**:
1. `Since` - precise for the field, but an alpha.6 platform re-pinned to alpha.9 is refused again on `comparable`.
2. `schema.DefaultSchemaVersion()` - the release the kernel was verified against, which satisfies every floor.
**Decision**: Option 2.
**Rationale**: A hint that needs a second round trip is not the fix.

### Red first

**Context**: The bug shapes must be proven to reach `opm platform check` before the report changes.
**Explored**: Change A measured on core that the three bad shapes read `routable=true` on today's core; the check's fixtures pin `schema.DefaultSchemaVersion()`, so they follow the library pin.
**Decision**: Section 1 writes the fixtures and tests first and runs them twice: on library alpha.34 (expected: the two-majors and both definer-less shapes exit 0) and on B's head (expected: validation exit, but the definer-less reports still word `routable` as vacuous and no report prints `provided by`). Nothing red is committed.
**Rationale**: The first run shows the bug the change set fixes reaches this command; the second shows the part only this change fixes.
