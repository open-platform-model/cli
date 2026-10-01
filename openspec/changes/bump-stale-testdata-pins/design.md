## Context

See proposal.md, Why. The current published releases are core `v2.0.0-beta.1`, opm catalog `v4.4.4` and k8s catalog `v1.0.0-beta.1`. The repo's maintained consumers already pin them (`examples/cue.mod/module.cue`, `tests/e2e/testdata/operator-owned/cue.mod/module.cue:10,13`, `hack/platform/cue.mod/module.cue:15-16`).

The five stale trees and their consumers:

| Tree | Pins today | Consumed by | Runs in CI |
| --- | --- | --- | --- |
| `tests/fixtures/valid/simple-module` | core alpha.6 | `internal/cmd/module/vet_test.go:47`, `internal/workflow/render/module_test.go:316` | yes, `go test ./internal/...` (`pr.yml:76`) |
| `tests/fixtures/valid/module-with-debug-values` | core alpha.6 | `internal/cmd/module/eval_test.go:212`, `internal/workflow/render/module_test.go:258,288` | yes |
| `internal/instinit/testdata/initvalues` | core alpha.11, opm 4.4.1 (`default: true`) | `internal/instinit/values_test.go:20` | yes |
| `internal/workflow/render/testdata/skip-unprovided` | core alpha.10, opm 4.4.0 | `internal/workflow/render/skip_test.go:30-41`; `tests/integration/skip-unprovided/main.go:55` | unit yes; integration no |
| `tests/e2e/testdata/duplicate-identities` | core alpha.6, opm 4.0.1 | `tests/e2e/duplicate_identities_test.go:25` | yes, e2e job (`pr.yml:246`) |
| `tests/integration/module-apply/testdata` | core alpha.6, opm 4.0.1 | `tests/integration/module-apply/main.go:59` | no: only local `task test:integration` (`Taskfile.yml:109`); CI integration job runs other programs (`pr.yml:202-209`) |

No tree pins `opmodel.dev/catalogs/k8s@v1` or any `testing.opmodel.dev` fixture, so `hack/fixtures.sh consumers` is unaffected and trap T3 does not apply. T3: re-pinning `examples` or `operator-owned` to an unpublished podinfo fixture breaks `TestResolveInstanceArg_RegistryBackedInstancePackage`, which resolves through `config.DefaultRegistry`. Neither of those consumers is touched here.

## Goals / Non-Goals

**Goals:**
- `.cascade-frozen` exists and names every core pin that is old on purpose, so the future cascade task (B4) and a reviewer can tell frozen pins from stale ones.
- The five trees are current, so the first cascade run starts from a clean baseline.

**Non-Goals:**
- Teaching any test to derive its pin from `schema.DefaultSchemaVersion()`. That is a cascade-maintenance improvement for later, if the trees keep going stale.
- Fixing `TestResolveInstanceArg_RegistryBackedInstancePackage`'s use of the shipped default registry (the GHCR trap). It concerns the podinfo consumers, which this change does not touch.
- Giving `tests/integration/module-apply/testdata` an `identity/` package. `opm module vet` on that tree fails with `cannot find package "./identity"` at both the old and the new pins. This was observed during planning and has nothing to do with the bump, and the apply program does not vet.

## Decisions

### D-a: Which literals go into `.cascade-frozen`

**Context**: D7 names two deliberate pins in `tests/e2e/instance_build_test.go`. A repo-wide grep for alpha-line literals finds about 20 more in Go tests.

**Options considered**:
1. List only the two named pins. This misses `internal/cmd/platform/check_test.go`, which holds the same two kinds of deliberate pin: too-old platforms at `:727`, `:761` and `:785`, and `collisionCoreVersion` at `:351-357`. A reader of `.cascade-frozen` would then wrongly conclude that file is safe to bump.
2. List every alpha literal. Most are parser inputs or fake registry listings that are never resolved (`internal/modref/*_test.go`, `internal/platform/catalog_test.go`, `internal/platform/resolve_test.go:252-253`, `internal/cmd/platform/pull_test.go:57`, `internal/cueedit/cueedit_test.go`). Listing them turns the file into noise and dilutes "frozen" into "anything old".
3. List every literal that a test resolves from a registry and that must stay old for the test to mean anything.

**Decision**: Option 3. The file holds two entries, each with a single-sentence reason:

```yaml
frozen:
  - path: tests/e2e/instance_build_test.go
    pins: ["opmodel.dev/core@v2"]
    reason: "The older-core platform pins core v2.0.0-alpha.11 so the build is refused as too old for the provider count, and collisionCorePin v2.0.0-alpha.13 is the floor the colliding-platform seed re-pins up to, never down from."
  - path: internal/cmd/platform/check_test.go
    pins: ["opmodel.dev/core@v2"]
    reason: "Its too-old platforms pin core alpha.6, alpha.9 and alpha.11 to be refused naming the release that derives each missing field, and collisionCoreVersion v2.0.0-alpha.13 is the first core that reports contract collisions instead of failing the fold."
```

**Rationale**: The reasons paraphrase the tests' own doc comments (`instance_build_test.go:247-253,285-286,316-319`; `check_test.go:351-356,757-760,780-783`). The golden literal `v2.0.0-alpha.10` in `internal/instinit/render_test.go:25,45` is out. The plan's pin inventory lists it, but the test never resolves it: it is a string in a golden file, and no cascade that edits `cue.mod` files would touch it. This exclusion is a deliberate departure from the plan, flagged for the owner.

### D-b: How to bump

**Decision**: In each tree run `cue mod get opmodel.dev/core@v2.0.0-beta.1`, plus `opmodel.dev/catalogs/opm@v4.4.4` where the tree pins the catalog, then `cue mod tidy`, with `CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'`. `cue mod get` keeps the `default: true` marker in `initvalues` (verified during planning). Hand edits are banned because `cue mod tidy` is the check the cascade will run, and a hand edit can leave a tree that tidy would rewrite.

### D-c: Section cut

**Decision**: Three sections, each a single commit that leaves `main` green:
1. `.cascade-frozen` alone. It changes no test, so it is green trivially.
2. The four trees whose consumers are unit tests in CI: `fixtures/valid/*`, `initvalues` and `skip-unprovided`. `skip-unprovided` also has a cluster program, which runs here so the tree is proven whole in one section.
3. The two trees consumed by the e2e suite and a cluster-only integration program: `duplicate-identities` and `module-apply`.

**Rationale**: Section 3 needs the kind cluster and section 2 mostly does not. Splitting along that line lets section 2 land on a machine without a cluster. Five one-tree sections would be valid but give five commits for six one-line diffs.

### D-d: Breakage rule

**Decision**: If a bumped tree's test fails, find out whether the cause is a schema change the fixture must follow. If it is, fix the fixture or the assertion in the same section. If the test asserts behaviour that only the old core has, revert that tree and add it to `.cascade-frozen` with the reason. Make that edit in this section's own commit: section 1 is already committed, and this change never rewrites a commit.

## Research & Decisions

### Are the old pins deliberate?

**Context**: D7 asks whether any of the five trees is old on purpose.
**Explored**: `git log` of each `cue.mod/module.cue`; grepping the trees for `alpha`, `pin`, `older`, `deliberate`; a reverted planning probe that bumped all six `cue.mod` files with `cue mod get` and `cue mod tidy` and ran the consumers.
**Findings**: No tree comments on its pin. The probe kept everything green:
- `go test ./internal/instinit/ ./internal/workflow/render/ ./internal/cmd/module/`: ok.
- `TestModVet_ValidModule` now runs instead of skipping.
- `go test ./tests/e2e/ -run 'DuplicateIdentities|OlderCorePlatform'`: both pass.
- `opm module build tests/integration/module-apply/testdata` renders the same three objects (Service and two Deployments) at both pins.

The probe never ran the two cluster programs, `module-apply` and `skip-unprovided`.
**Decision**: Bump all five. Freeze only the Go-literal pins of D-a.

### Silent skip without a registry

**Context**: Without `CUE_REGISTRY`, `TestModVet_ValidModule` skips: the default registry cannot find `opmodel.dev/core@v2.0.0-beta.1`, and the test treats that as a connectivity error (`vet_test.go:69-74`). A local `go test` can therefore look green without exercising the fixture.
**Decision**: Every task runs the tests with the CI registry mapping (`pr.yml:60`) and checks that the run reports no `SKIP`. Changing the skip itself is out of scope.

## Risks / Trade-offs

- [The cluster programs are not in CI] A bumped `module-apply` or `skip-unprovided` tree could break `task test:integration` without CI noticing. → Sections 2 and 3 run those programs against the local `kind-opm-dev` cluster before committing.
- [`.cascade-frozen` format drift] The workspace RELEASING.md that defines the format is still in review. → The file uses the fixed format verbatim. If the doc changes it, B4's change rewrites this file, and the content (paths, pins, reasons) carries over.
- [The trees go stale again] Nothing enforces currency until the cascade task (B4) lands. → B4 runs `cue mod get` and `cue mod tidy` over these `cue.mod` files on every upstream release. Until then, the new test-fixture-lineage requirement is the written rule.

## Migration Plan

No migration. Rolling back means reverting the section commits.
