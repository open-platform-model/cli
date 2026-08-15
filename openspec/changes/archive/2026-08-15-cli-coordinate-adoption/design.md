# Design — cli-coordinate-adoption

## Overview

One crossing with a decoupling seam. The Go retarget, fixture crossing, coordinate collapse, D19 warning, and file-based platform surface land together and are green immediately. Two joints decouple the change from the operator track: CR-read tolerance (permanent) plus CR-write refusal (transitional), and skip-gated registry-pulling tests whose pins are pre-authored for the `examples-fleet-core-v2` republish (`podinfo v0.1.4`). Both joints tighten mechanically when operator `v1.0.0-alpha.9` ships; no second design pass is needed.

## Research & Decisions

### The spec.module write stays declared, verified downstream

**Context**: 0010 02-design:213 says the apply sites "write the coordinate actually fetched". Explored against the actual paths: `opm module apply` renders a local directory and fetches nothing (0006 D7; D9's own rationale cites this line); `opm instance apply` loads through `LoadInstancePackage`, which performs no identity verification; the only CLI registry fetch is handoff verification.
**Options considered**:
1. Plumb the CUE resolution (the instance `cue.mod` deps pin) up to the write — fragile (MVS may resolve above the pin; multiple module imports), and still unavailable on the module-apply path.
2. Write the declared coordinate, rely on the library's acquire-time verification (alpha.13) at every point the coordinate is later *consumed* (handoff verify, operator acquire), plus 0011's publish gates preventing lying artifacts at the source.
**Decision**: Option 2. The write reads `metadata.modulePath` verbatim and v-prefixes `metadata.version`. Recorded as a deviation against 0010 02-design when the slice lands.
**Rationale**: The silent-downgrade defect was declared-vs-published disagreement; it is killed where the coordinate meets a registry, which the CLI's write paths never do. The existing spec (`instance-inventory` "canonical declared path and version") already states the declared posture — the defect fix was never CLI-side plumbing.

### `CanonicalModuleRef` keeps its name, loses its arithmetic

**Context**: both call sites (`apply.go:273`, `thineditor.go:112`) want exactly `(path, v-prefixed version)`. Under v2 the path is `metadata.modulePath` verbatim; the version prefix is still needed (`#VersionType` is bare; the operator's `module.NewVersion` rejects bare — the prefix is load-bearing, per the method's existing doc).
**Decision**: body becomes `return m.ModulePath, ensureVPrefix(m.Version)`; delete `majorVersionTag`, `hasVPrefix` (fold into `ensureVPrefix` if still needed), the snake-leaf fallback, and the `NameSnakeCase`/`DefaultNamespace` fields (the former no longer exists in core v2 — D8; the latter never did and has zero readers). No v1 tolerance branch: a v1-authored module fails core-v2 schema validation before metadata is ever read, so a dual-line fallback (the library's loader keeps one) would be dead code here.
**Rationale**: keeping the method name keeps the two call sites and the SSA-completeness story (`SpecInput` docs) untouched; the five-case unit table is rewritten to pin the new contract (verbatim path; bare→prefixed version; empty version stays empty).

### CR tolerance split: read forever, write until the CRD knows `version`

**Context**: the vendored CRD (`internal/operator/dist/install.yaml`, `PinnedOperatorVersion v1.0.0-alpha.4`) has no `version` property — a written `version` is silently pruned by the API server (structural schema). On the read side, stored CRs keep `filter` in etcd until their next spec write, and `operator-library-retarget` may leave `Version` CRD-optional (ratcheting measurement), so filter-shaped/versionless CRs outlive the re-vendor.
**Decision**:
- `DecodeCRSpec`: decode `version` when present; ignore any `filter` key (no error — it is legacy stored data); a subscription with no version decodes with `Version: ""` and fails later at synthesis with the library's `ErrSubscriptionMissingVersion`, which the materialize call site wraps with a hint: "cluster Platform may predate the scalar-version shape; re-apply its spec with version set". Permanent behavior, not transitional.
- `EnsureClusterPlatform`: before writing, refuse when the vendored CRD manifest lacks the `version` property (checked against the embedded `install.yaml`, not the live cluster — the CLI installs that manifest), with an error naming `operator-library-retarget`. The refusal and its check are deleted in the blocked-on-release task group.
**Rationale**: read tolerance mirrors the operator's own posture (old CRs persist by design); the write refusal turns silent field-pruning — the worst failure mode, a platform that materializes nothing with no error — into an actionable message.

### Seed version: hardcoded pin, three-way mirror

**Context**: `DefaultPlatformTemplate` is a Go constant reachable by no automation (root `deps:update` skips it). `opm config init` is normatively forbidden registry access (`config-commands` spec, 0006 D39); a non-concrete placeholder fails `opm config vet` and three test consumers.
**Options considered**: (1) hardcode + hand-bump; (2) resolve at init (needs a spec delta and new network surface in a command defined by its offline-ness); (3) placeholder forcing user edit (breaks unattended consumers).
**Decision**: Option 1 — `version: "2.0.0-alpha.3"`. The mirror set is `internal/config/templates.go` + `hack/platform.cue` (same-commit contract, existing header) + the operator's `config/samples` Platform (recorded alignment, bumped on catalog releases as ordinary fixture updates). No new task (Principle VII): the drift mode is "old but resolvable" (immutable tags never dangle), caught when anyone bumps one peer.
**Rationale**: matches the library's load-bearing-pin precedent and the operator track's identical choice; the failure mode of a stale pin under v2 (an old catalog) is strictly milder than the v1 template's measured failure (a stale range selecting nothing).

### D19 trigger: the existing replacement predicate, on both render entries

**Context**: D19's text says file presence of `cue.mod/local-module.cue`; the repo's `loader.HasLocalModuleReplacement` requires a local-path `replaceWith` entry, and the `instance-inventory` spec's source-local annotation is already defined by that same predicate.
**Decision**: reuse `HasLocalModuleReplacement` (refinement recorded: a replacement-free `local-module.cue` carries none of D19's false-conclusion risk, and two predicates for one file invite drift). Instance-file entry: warn when the already-computed `SourceLocal` is true (`render.go:88-94` — the boolean is in hand). Module-directory entry: compute `HasLocalModuleReplacement(moduleRoot)` from the staged module root (distinct from that path's hardcoded `SourceLocal = true`, which means "local main module", not "replaced dependency"). Surface: direct `output.Warn` at render time (the repo's convention for pre-output warnings), one line: `module context carries cue.mod/local-module.cue replacements: demanded keys may not correspond to published bytes`.

### New-error surfacing: one branch, one message fix

**Context**: the CLI routes on zero library error types; everything funnels through `cmdutil.PrintValidationError`, whose CUE-position heuristic sends the new aggregates to a flat `error=` blob. The operator deliberately added no typed routing either.
**Decision**: minimal — `PrintValidationError` gains an `errors.As` branch for `*oerrors.UnresolvedDemandsError` (first `library/opm/errors` import in the CLI), rendering one line per demand: component, contract key, kind, and same-base alternatives ("implemented at: …" / "nothing on this platform implements this contract"). `handoff/verify.go` distinguishes `oerrors.IdentityError` from not-found ("artifact identity mismatch: declared X, fetched by Y" instead of "publish the module"). `MaterializeError`/`IdentityError` on the materialize path keep the generic wrap plus the legacy-CR hint above. Broader routing is a recorded follow-up, matching the operator's posture.

### Fixture identity reauthoring

Model: `tests/integration/module-apply/testdata` (already snake-cased, documented). Per fixture: `cue.mod` `module:` leaf, `metadata.name`, and `metadata.modulePath` leaf agree in snake_case with the `@vN` suffix on the path (`_leaf` assertion); CUE package renamed to match where it diverges (`simplemodule` → `simple_module`, etc.); `fqn` never authored. Catalog imports move to `…/{blueprints,resources,traits}/v1beta1` (package `v1beta1`, existing aliases absorb the rename); member bodies are unchanged (verified: `#StatelessWorkload`, `#Expose`, `#Container`, `#Image`, `#Secret` all present at v1beta1 with compatible fields). `vet_test.go` semantics survive (`debugValues: _` in v2); only comments and the skip message move.

### Gated tail keyed on operator `v1.0.0-alpha.9`

`render-parity` (env-gated) and the e2e handoff suite (kind-gated) pin `podinfo v0.1.4` now and stay gated; `examples/` pins the same and remains unverified-by-automation (unchanged posture). When the release ships: `task operator:sync`, `PinnedOperatorVersion` bump, delete the CR-write refusal, rewrite `hack/kind-platform.yaml` to the v2 shape, un-gate and run both suites. If the republish lands at a different version than `v0.1.4`, the pins move in that task — one greppable constant set.

## Technical Notes

### Signatures

```go
// pkg/module/module.go — after
func (m ModuleMetadata) CanonicalModuleRef() (path, version string) // returns (m.ModulePath, ensureVPrefix(m.Version))

// internal/platform/spec.go — after
type wireSubscription struct {
    Enable  *bool  `json:"enable,omitempty"`
    Version string `json:"version,omitempty"` // empty on legacy CRs; enforced at synthesis
}
```

### Data flow (coordinate)

```
module.cue metadata ──decode──▶ ModuleMetadata ──CanonicalModuleRef──▶ SpecInput ──ApplySpec──▶ spec.module.{path,version}
   (v2: full @vN path,            (verbatim path,                                      │
    bare semver)                   v-prefixed version)                                 ▼
                                                                    handoff verify / operator acquire
                                                                    (library verifies declared == fetched)
```

`sourceDigest(path, version)` consumes the same pair; golden pin test names the operator's `ModuleSourceDigest` as byte-identical peer.

### Error handling summary

| Failure | Surface |
| --- | --- |
| Unresolved demand (new) | typed branch: per-demand lines + alternatives |
| Versionless subscription from legacy CR | `ErrSubscriptionMissingVersion` + legacy-CR hint |
| Identity mismatch at handoff acquire | distinct message naming declared vs fetched |
| CR write against pre-v2 CRD | refusal naming `operator-library-retarget` |
| Named catalog build not published | library's error (lists published versions) via generic wrap |

### Exit codes

Unchanged: validation failures (including the new unresolved-demand class) exit through the existing validation-error path; the CR-write refusal is a validation error (2), not a crash.
