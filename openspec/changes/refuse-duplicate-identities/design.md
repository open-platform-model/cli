# Design: refuse-duplicate-identities

## Context

See `proposal.md` § Why. The decision is `enhancements/0015` D15, completed by the operator; this change is the CLI's matching refusal.

Current state, read 2026-09-19 against cli main (`809ea15`, library `v1.0.0-alpha.32`):

- `internal/workflow/render/render.go` `renderInstance(ctx, env, inst, k8sCfg, moduleRoot, sourceLocal)` is the one function both entry points reach: `FromInstanceFile` (instance build, apply, diff) and `FromModule` (module build, apply). It calls `env.kernel.Render`, prints and exits on error, words the replacement warnings, wraps each `Compiled` into `pkgcore.Resource`, computes the render digest, and builds the `Result`.
- `internal/workflow/render/validation.go` `printValidationError(err)` prints a `*kernel.RenderError` as `render failed: <message>` plus the diagnostics rows as details, a `*liberrors.SkewError` verbatim, and everything else through `cmdutil.PrintValidationError`. Tests capture the log stream and the details stream separately (`captureValidationOutput`).
- Library `alpha.33` `opm/helper/objectset`: `Duplicates(compiled []*kernel.Compiled) []Duplicate` and `*DuplicateIdentitiesError{Duplicates}` whose `Error()` is a header line (`N rendered objects share one identity, so the last apply would silently overwrite the first:`) followed by one indented line per identity naming every producer as `component "x" (transformer)`.
- The cli's render workflow tests are unit tests over pure pieces (values, provenance, formatting); no test drives a registry-backed refusal. The e2e build tests render published fixture modules pinned at a catalog build predating the registration contract.

## Goals / Non-Goals

**Goals**

- No CLI render passes two objects with one identity on to output, apply or diff.
- The refusal reads exactly as the operator's, split into the CLI's header and details streams.
- One check site, covering all five render-bearing commands.

**Non-Goals**

- Deduplication, arbitration, or a flag to allow duplicates.
- A registry-backed or e2e fixture for the collision (see Research & Decisions).
- Wording of the rows: the library owns it.

## Decisions

### The check sits in `renderInstance`, before any resource is built

```go
out, err := env.kernel.Render(ctx, newRenderInput(env, inst))
if err != nil { /* unchanged */ }
if err := refuseDuplicateIdentities(out); err != nil {
	printValidationError(err)
	return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
}
// replacement warnings, converted resources, digest, result: unchanged
```

```go
// refuseDuplicateIdentities returns the library's duplicate-identity error
// when two compiled objects share one apply identity (0015:D15), nil
// otherwise. Pure over the render output, so it is tested without a render.
func refuseDuplicateIdentities(out *kernel.RenderResult) error {
	if dups := objectset.Duplicates(out.Compiled); len(dups) > 0 {
		return &objectset.DuplicateIdentitiesError{Duplicates: dups}
	}
	return nil
}
```

Placed before the replacement warnings so a refused render prints no advisory beside a refusal, and before `converted`, `ComputeRenderDigest` and `newResult`, so nothing downstream (`ShowOutput`, apply, diff) can receive the set. Both entry points reach this line, so no per-command wiring exists to forget.

### Printing

`printValidationError` gains an arm before the generic funnel:

```go
var dupErr *objectset.DuplicateIdentitiesError
if errors.As(err, &dupErr) {
	header, rows, _ := strings.Cut(dupErr.Error(), "\n")
	output.Error(fmt.Sprintf("%s: %s", renderFailedMsg, header))
	if rows != "" {
		output.Details(rows)
	}
	return
}
```

Example:

```
ERROR render failed: 2 rendered objects share one identity, so the last apply would silently overwrite the first:

  opmodel.dev/v1alpha1 TransformerRegistration backup-system/backup-system.k8up rendered by component "registration" (opmodel.dev/catalogs/opm/transformers/transformer-registration-transformer@4.4.0) and component "registration-copy" (opmodel.dev/catalogs/opm/transformers/transformer-registration-transformer@4.4.0)
```

Exit code 2 (`ExitValidationError`), `Printed: true`, as every other render refusal.

### Files touched

| File | Change |
| --- | --- |
| `go.mod`, `go.sum` | library `v1.0.0-alpha.33` |
| `internal/workflow/render/render.go`, `render_test.go` | `refuseDuplicateIdentities` and its call; tests over hand-built `kernel.RenderResult` values |
| `internal/workflow/render/validation.go`, `validation_test.go` | the printing arm; a capture test for header and details |
| `tests/e2e/testdata/duplicate-identities/`, `tests/e2e/duplicate_identities_test.go` | the unpublished colliding module and the command-level refusal test |

## Research & Decisions

### Unit tests over the pure check, not a fixture

**Context**: the operator tested the same refusal over hand-built `kernel.RenderResult` values; the CLI's fixtures pin a catalog build without the registration contract, and its e2e tests run against published fixtures.
**Explored**:
1. A new e2e fixture module with two registration components, published through the fixture pipeline: moves the fixture catalog pin for one test and runs only where the registry is reachable.
2. `refuseDuplicateIdentities` as a pure function over `*kernel.RenderResult`, unit-tested with `cuecontext` values (two `TransformerRegistration` objects with one name from two components; distinct objects; a value without a name beside a duplicate), plus the print arm under `captureValidationOutput`.
**Decision**: 2, plus a repo-local e2e fixture that is neither published nor a registration module.
**Rationale**: the check reads four fields off values the kernel already returned; the two-registration case is expressible exactly, and the same shape proved the operator's refusal. The healthy path is what every existing render test exercises.

What option 1 costs is the fixture *pipeline*: a published module moves the fixture catalog pin and needs a reachable registry to be republished. A third option avoids that entirely — `tests/e2e/testdata/duplicate-identities`, a module on `test.example.com` that nothing publishes, rendered by `opm module build` the way `tests/e2e/testdata/vet-errors` is. It collides two `StatelessWorkload` components on one `metadata.name` rather than two registrations, because the collision the check refuses is a shared apply identity, not a registration: `TransformerRegistration` is the instance D15 was written from, not the class. That keeps the unit tests as the proof of the helper call and the wording, and makes `TestE2E_ModBuild_RefusesDuplicateIdentities` the proof that the refusal reaches a whole command — the exit code, the two streams, and an empty stdout, which no unit test over `renderInstance`'s callee can show.

### The library's wording, split, not rewritten

**Context**: `printValidationError` words other refusals itself (`FormatUnresolvedDemands`), and the operator puts the library's message whole into a condition.
**Explored**: a CLI formatter over `Duplicates` rows; the library message split at its first newline into the two streams.
**Decision**: split.
**Rationale**: the helper exists so the two runtimes say one thing; the only CLI-specific fact is which stream a line goes to, and `Cut` at the first newline is exactly that boundary in the library's format.

### Build refuses too

**Context**: `build` writes nothing to a cluster, so a duplicate is harmless there in isolation.
**Explored**: refusing only on apply and diff; refusing everywhere `renderInstance` runs.
**Decision**: everywhere.
**Rationale**: `build` is the pre-flight for `apply`; a build that succeeds on a set the apply refuses teaches users to ignore it (the same argument as `platform check`'s exit code), and one check site in `renderInstance` is simpler than two behaviours.

## Risks / Trade-offs

- [A module in use renders two objects with one name today] → it now fails every render naming both components; the cluster refuses the same module, and the CLI was silently dropping an object.
- [The library's message format changes] → `Cut` at the first newline still yields a header and rows; the capture test pins today's split, not the words.
- [Library bump] → alpha.32 to alpha.33 is the helper package alone; no kernel signature moved.

## Migration Plan

One PR, two sections, squash title `feat(render): refuse a render whose objects share one apply identity`. release-please cuts a minor. Rollback is a revert.

## Open Questions

None.
