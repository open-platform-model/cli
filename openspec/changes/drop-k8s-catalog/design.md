## Context

`internal/config/templates.go` declares `DefaultCatalogPaths = []string{"opmodel.dev/catalogs/opm@v4", "opmodel.dev/catalogs/k8s@v1"}` and derives `DefaultCatalogPath = DefaultCatalogPaths[0]`; `internal/platform/catalog.go` re-exports the latter for `opm operator install`. Everything else that reads the slice is a test (`internal/config/platform_test.go`, which also checks it against `hack/platform`). The k8s path appears as a sample "second catalog" in seven test files, all over fake graphs or decoded data, none resolving it.

## Goals / Non-Goals

**Goals:**
- No first-party reference to `opmodel.dev/catalogs/k8s@v1` in code, fixtures, rule files or docs.

**Non-Goals:**
- Refusing or warning about a user platform that still subscribes to `k8s@v1`; it keeps resolving.
- Changing how a second catalog is handled. The samples keep exercising two catalogs; only the path changes.

## Decisions

### D1. One constant

```go
// DefaultCatalogPath is the major-suffixed CUE module path of the
// first-party catalog. `opm operator install` resolves a published
// version of it when it seeds a cluster Platform.
const DefaultCatalogPath = "opmodel.dev/catalogs/opm@v4"
```

`DefaultCatalogPaths` is deleted. `internal/config/platform_test.go` lines 59 and 143 loop over it to check `hack/platform`; they check the one path instead.

### D2. The drift test keeps its point with one catalog

`TestBuildPlatformModule_KeyImportDriftNamesTheEntry` proves that a `#registry` entry whose key disagrees with the embedded catalog's module path fails the 0019:D5 binding at a path naming the entry. With one catalog it re-keys the opm entry instead of swapping two bindings: the import stays `opm`, the key becomes another path the platform module also depends on. Section 1 measures which re-keying yields the binding refusal rather than an earlier resolution error; if none does without a second real dependency, the test is deleted and the scenario is left to core's own `#registry` tests, which section 1 names in this design.

**Measured (section 1).** The test re-keys the opm entry to `opmodel.dev/core@v2`, a path the platform module already depends on, leaving `#catalog: opm`. The build fails at `#registry."opmodel.dev/core@v2".#catalog.metadata.modulePath: conflicting values "opmodel.dev/catalogs/opm@v4" and "opmodel.dev/core@v2"`, the 0019:D5 binding refusal, with the hint naming the entry and `platform.cue`; no earlier resolution error intervenes. The test is kept, not deleted.

### D3. The neutral sample is `example.com/catalogs/extra@v1`

It sorts before `opmodel.dev/catalogs/opm@v4` exactly as `opmodel.dev/catalogs/k8s@v1` did, so `resolve_test.go` (the `Describe()` order), `spec_test.go` ("entries are sorted by path"), `replacements_test.go` (`got[2]`/`got[3]`) and the alias numbering they assert keep their order. `validation_test.go` used `k8s@v1#Expose` as an alternative implementer, which the k8s catalog never had (it shipped no traits); the sample fits it better. `replacements_test.go`'s `"../catalog_k8s"` becomes `"../catalog_extra"`.

## Research & Decisions

### Which references are real
**Context**: The k8s path appears in 16 cli files; most are samples.
**Explored**: Workspace triage of 2026-10-02 (every hit classified real or sample, with line numbers).
**Options considered**:
1. Delete every sample's second catalog - loses the two-catalog coverage the samples exist for.
2. Swap the samples to a neutral path that keeps their sort position - keeps coverage and every order-sensitive expectation.
**Decision**: Option 2 (D3); real references are removed (D1, D2, hack fixtures, rule files).
**Rationale**: The samples test generic multi-catalog behaviour, which outlives the k8s catalog.

## Risks / Trade-offs

- [`platform-pins.sh` fails on the missing key] → The workspace change lands first or with this PR (proposal, Impact).
- [The e2e kind platform loses a catalog an e2e test relied on] → Section 1 runs `task test:e2e` against the kind cluster with the edited `hack/kind-platform.yaml`; no e2e test renders a k8s member (none imports it).
