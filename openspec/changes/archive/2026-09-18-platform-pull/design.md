# Design: platform-pull

## Context

See `proposal.md` § Why. The decision is `enhancements/0015` D6 (reproduce a cluster's render by pulling the operator-generated package), reframed: the package is a pure function of inputs that are all on the Platform CR, and the CLI already runs that function.

Current state, read 2026-09-18 against cli main (library `v1.0.0-alpha.32`):

- `platform.ClusterSpecGetter` is `func(ctx) (spec map[string]any, name, unavailable string, err error)`; `ClusterSpecGetterFor(dyn)` reads the singleton and returns `spec` only, with NotFound and Forbidden as warn-fallback (D21). Three commands construct it (`module apply`, `instance apply`, `instance diff`) and hand it to `render.ModuleOpts`/`InstanceOpts.ClusterPlatform`.
- `Resolve` decodes the spec with `DecodeCRSpec` into `Spec{Name, Type, Entries, SkewPolicy}` and calls `GenerateClusterModule`, which feeds `platformmodule.Closure` and `platformmodule.Generate` under `ClusterPlatformModulePath = "opmodel.dev/platforms/cluster@v0"` and caches under `cache/platforms/<sha256 of files>/`. The operator's reconciler runs the same two library calls under the same path (`PlatformModulePath`), with entries from `platformEntries(spec, activeClaims)`: subscriptions first, then each active claim not already subscribed, enabled, sorted by path.
- The operator writes `status.registry` as `resolvedRegistry(entries)`: one row per generator entry, `{catalog, version, enabled, source: Subscription|Registration}`; `status.packageIdentity` (`gen-N[-digest8]`, over generation plus every active claim coordinate); `status.observedGeneration`; `status.operatorVersion`; `Ready` with reasons `Generated`, `BuildFailed`, `GenerateFailed`, `OverSubscribedContracts`, `ComparablePredicates`. Status is rewritten only where the held package is the one under that identity, so after a refusal it describes the last good package.
- `Resolution{Source, Location, Dir, SkewPolicy, Warning}` and `Describe()` carry provenance to the render output; `SpecFromPlatform` decodes the seed document from the built platform for write-if-absent, which runs only when no CR exists.
- Tests: `resolve_test.go` drives `Resolve` with `clusterGetterReturning(spec, name, unavailable, err)` and a fixture `ModFiles` graph (`fixtureGraph()`); `cluster_test.go` drives the getter with a fake dynamic client (`newFakeDynamic`, `clusterPlatformObj(spec)`).
- `inventory.GateOperatorVersionCeiling` already reads `status.operatorVersion` for the apply-time skew ceiling.

## Goals / Non-Goals

**Goals**

- Cluster-facing renders consume the package the operator consumes.
- The laptop can hold that package as an ordinary platform module directory.
- Every divergence between spec and effective package is visible in the provenance, never silent.

**Non-Goals**

- A CLI gate on `routable` or `discriminated` for the cluster arm (the recorded registry describes a gated package) or for a local directory (`opm platform check`).
- Recomputing or verifying the package identity or bytes (see Research & Decisions).
- Reading claims (`TransformerRegistration`) directly: `status.registry` is the operator's resolution of them, and reading claims would re-derive what the operator already decided (which subscription wins, which claim is active).
- `platform check --cluster`.

## Decisions

### The getter returns the document

```go
// ClusterPlatform is what the cluster getter returns: the singleton
// Platform's name, generation, spec and status, undecoded.
type ClusterPlatform struct {
	Name       string
	Generation int64
	Spec       map[string]any
	Status     map[string]any
}

type ClusterPlatformGetter func(ctx context.Context) (doc *ClusterPlatform, unavailable string, err error)
```

`ClusterSpecGetter` is renamed and returns the document; NotFound/Forbidden fallback semantics are unchanged. The three call sites and `render.ModuleOpts`/`InstanceOpts` follow in the same commit (a signature change cannot straddle a section).

### Decoding the effective registry

`spec.go` gains:

```go
// Effective is the registry the operator generated the running package from,
// decoded off Platform status; nil when the status records none.
type Effective struct {
	Entries          []Entry // catalog, version, enabled; sorted by path
	Sources          map[string]string // path -> "Subscription" | "Registration"
	PackageIdentity  string
	ObservedGeneration int64
	OperatorVersion  string
	Ready            *ReadyState // condition Ready: status and reason, nil when absent
}

func DecodeCR(doc *ClusterPlatform) (Spec, *Effective, error)
```

`Spec` keeps decoding the spec exactly as `DecodeCRSpec` does (legacy tolerance included); `Effective` decodes `status.registry` by the operator's JSON field names (`catalog`, `version`, `enabled`, `source`) through the same JSON round-trip idiom. A status row with an empty version is refused (the operator never writes one; it would mean a hand-edited status).

### Resolution prefers the effective registry

```go
s, eff, err := DecodeCR(doc)
entries, origin := s.Entries, "spec"
if eff != nil && len(eff.Entries) > 0 {
	entries, origin = eff.Entries, "effective"
	if eff.ObservedGeneration < doc.Generation { output.Warn(...) }
	if eff.Ready != nil && eff.Ready.Status == "False" { output.Warn(...) }
}
dir, err := GenerateClusterModule(ctx, Spec{Name: s.Name, Type: s.Type, Entries: entries}, opts)
```

`Resolution` gains `RegistryOrigin string` (`"effective"` or `"spec"`) and `PackageIdentity string`. `Describe()` for the CR source becomes:

```
platform: cluster Platform CR cluster (effective registry, package gen-7-3f9a1c2b, generated module ~/.opm/cache/platforms/<hash>)
platform: cluster Platform CR cluster (spec registry, no operator generation recorded, generated module ~/.opm/cache/platforms/<hash>)
```

Warnings, in the render's normal warning stream:

```
cluster Platform generation 5 is not yet generated by the operator (status describes generation 4); rendering against the effective package
cluster Platform is Ready=False (OverSubscribedContracts); rendering against the last good package the operator recorded
```

`SkewPolicy` keeps coming from the spec: it is policy, not registry, and the operator reads it from the spec too.

### `opm platform pull`

```
opm platform pull <dir> [--force] [--kubeconfig <path>] [--context <name>]
```

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--force` | bool | false | replace a non-empty `<dir>` |
| kubeconfig and context flags | as the apply commands | as the apply commands | which cluster to read |

Flow: build the cluster client (as `module apply` does) → `Resolve` with the cluster getter and no `--platform` (pull has no local arm: `Resolve` is called with a getter, and an `unavailable` result is turned into the not-found refusal instead of the local fallback, through a `ResolveOptions.NoLocalFallback` boolean) → copy the cached module directory's files into `<dir>` (create, or replace under `--force`; refuse otherwise) → print the report.

Example output:

```
platform: cluster Platform CR cluster (effective registry, package gen-7-3f9a1c2b, generated module ~/.opm/cache/platforms/8c1e…)
generation 7 (observed 7), operator v1.0.0-alpha.20

registry: 3 entries
  opmodel.dev/catalogs/opm@v4     4.4.0          enabled   Subscription
  opmodel.dev/catalogs/k8s@v1     1.0.0-alpha.3  disabled  Subscription
  opmodel.dev/catalogs/k8up@v1    1.2.0          enabled   Registration

wrote ./cluster-platform
next: opm module build --platform ./cluster-platform <module-dir>
```

Exit codes: 0 written; 1 non-empty target without `--force`; 2 the recorded registry does not generate (unpublished pin, legacy spec) with the generation diagnostic; 3 the cluster is unreachable; 5 no readable Platform CR (`cluster Platform "cluster" not found or not readable: nothing to pull`).

Error messages name the directory and the flag, the CR and the reason, or the failing dependency, in that order of cases.

### Files touched

| File | Change |
| --- | --- |
| `internal/platform/cluster.go`, `cluster_test.go` | getter returns the document |
| `internal/platform/spec.go`, `spec_test.go` | `Effective`, `DecodeCR` |
| `internal/platform/resolve.go`, `resolve_test.go` | effective-first generation, `RegistryOrigin`, `PackageIdentity`, warnings, `NoLocalFallback` |
| `internal/workflow/render/types.go`, `env.go` | getter type |
| `internal/cmd/module/apply.go`, `internal/cmd/instance/apply.go`, `internal/cmd/instance/diff.go` | constructor call |
| `internal/cmd/platform/pull.go`, `pull_test.go`, `platform.go` | the command, its tests, the group help |

## Research & Decisions

### Regenerate from the CR, not transfer bytes

**Context**: D6 says "retrieves the operator-generated platform package"; 0019 D6 says the package is never published or served.
**Explored**:
1. Serve the package from the operator (a ConfigMap, an OCI artifact, a subresource): new operator surface, a second copy of the bytes to keep consistent, and it contradicts the no-publish posture the reserved namespace exists for.
2. Regenerate from `status.registry` with the shared generator: no new surface; byte-identical by construction when CLI and operator embed the same library release, because the generator, the closure derivation and the module path are the same code and the core pin follows the library.
3. Regenerate from `spec.registry` plus a client-side read of active claims: re-derives the operator's arbitration (which subscription wins, which claim is active) on the client.
**Decision**: 2.
**Rationale**: D6 already conceded the shape ("the CR makes the effective set enumerable and therefore fetchable") and left the mechanics open; the operator writes `status.registry` for exactly this. Under 0026's draft, where the platform is generated per resolution, there is no single package to transfer at all, so 2 is the framing that survives. The enhancement entry gets a Revised line saying so.

### No identity or byte verification here

**Context**: `status.packageIdentity` hashes the generation and every active claim coordinate; `status.registry` cannot reproduce that list when a claim overlaps an authored subscription (the row is sourced Subscription at the subscription's version). The identity also does not cover the core pin, which comes from the library release each binary embeds.
**Explored**: recomputing the identity from claims read client-side; comparing bytes against a digest the operator does not publish; printing what the operator recorded and the operator version.
**Decision**: print, do not verify.
**Rationale**: the two verifications that would matter (bytes, core pin) need a content digest on Platform status, which is an operator change: a `status.packageDigest` over the generated files with the library's hash. Recorded as the follow-up; until then the report shows `operatorVersion` so a reader can see when the two binaries differ.

### Status wins over spec, with warnings instead of a choice

**Context**: after a spec edit the operator may not have generated yet, and after a refusal it has not.
**Explored**: generating from the spec when status lags; asking the user; generating from the effective registry and warning.
**Decision**: effective registry, warned.
**Rationale**: D6's property is reproducing what the cluster renders, and that is the effective package in both cases; the spec is a proposal until the operator accepts it. The warning names both generations or the refusal reason, so an operator editing the spec knows the laptop is behind on purpose.

### pull has no local fallback

**Context**: the render commands fall back to the local default platform when the CR is absent or unreadable (D21).
**Explored**: reusing the fallback; refusing.
**Decision**: refuse with the not-found code.
**Rationale**: a fallback would write the local default platform to `<dir>` and call it the cluster's; the command's one promise is that the directory is what the cluster renders against.

### Copy the cached module rather than generate twice

**Context**: `GenerateClusterModule` already writes the module under the cache.
**Explored**: generating straight into `<dir>`; copying the cached directory.
**Decision**: copy.
**Rationale**: one generator call, one cache entry; the cache keeps its idempotence and `<dir>` gets identical bytes. The copy writes files, not symlinks, so `<dir>` is a self-contained module a repository can commit.

## Risks / Trade-offs

- [A cluster whose operator predates `status.registry` keeps the spec path] → the provenance says "spec registry, no operator generation recorded", so the difference is visible, and the behaviour is exactly today's.
- [CLI and operator embed different library releases, so the core pin differs] → the report prints `operatorVersion`; exact reproduction needs the operator digest follow-up.
- [The effective registry names a catalog version the CLI's registry mapping cannot resolve] → the same failure as today for an unpublished pin, naming path and version.
- [A hand-edited status] → a row without a version is refused naming the row; anything else is the operator's to have written.

## Migration Plan

Two sections in one PR, squash title `feat(platform): render against the cluster's effective registry and add opm platform pull`. release-please cuts a minor. Rollback is a revert. No stored state changes; the cache directory names change only where the entries changed.

## Open Questions

None.
