## Context

See proposal.md, Why. The current published releases are core `v2.0.0-beta.1`, opm catalog `v4.4.4` and k8s catalog `v1.0.0-beta.1`. The repo's maintained consumers already pin them (`examples/cue.mod/module.cue`, `tests/e2e/testdata/operator-owned/cue.mod/module.cue:10,13`, `hack/platform/cue.mod/module.cue:12-16`).

The five stale trees and their consumers:

| Tree | Pins today | Consumed by | Runs in CI |
| --- | --- | --- | --- |
| `tests/fixtures/valid/simple-module` | core alpha.6 | `internal/cmd/module/vet_test.go:47`, `internal/workflow/render/module_test.go:316`; e2e `tests/e2e/vet_output_test.go:155` (`TestE2E_ModuleVet_OpenDebugValuesRefusedAtSynthesis`, copies the tree) | unit yes, `go test ./internal/...` (`pr.yml:76`); e2e yes (`pr.yml:246`) |
| `tests/fixtures/valid/module-with-debug-values` | core alpha.6 | `internal/cmd/module/eval_test.go:212`, `internal/workflow/render/module_test.go:258,288` | yes |
| `internal/instinit/testdata/initvalues` | core alpha.11, opm 4.4.1 (`default: true`) | `internal/instinit/values_test.go:20` | yes |
| `internal/workflow/render/testdata/skip-unprovided` | core alpha.10, opm 4.4.0 | `internal/workflow/render/skip_test.go:30-41`; e2e `tests/e2e/skip_unprovided_test.go:27` (`TestE2E_ModBuild_SkipUnprovided`); `tests/integration/skip-unprovided/main.go:55` | unit yes; e2e yes; integration no |
| `tests/e2e/testdata/duplicate-identities` | core alpha.6, opm 4.0.1 | `tests/e2e/duplicate_identities_test.go:25` | yes, e2e job (`pr.yml:246`) |
| `tests/integration/module-apply/testdata` | core alpha.6, opm 4.0.1 | `tests/integration/module-apply/main.go:59` | no: only local `task test:integration` (`Taskfile.yml:109`); CI integration job runs other programs (`pr.yml:202-209`) |

No tree pins `opmodel.dev/catalogs/k8s@v1` or any `testing.opmodel.dev` fixture, so `hack/fixtures.sh consumers` is unaffected and trap T3 does not apply. T3: re-pinning `examples` or `operator-owned` to an unpublished podinfo fixture breaks `TestResolveInstanceArg_RegistryBackedInstancePackage`, which resolves through `config.DefaultRegistry`. Neither of those consumers is touched here.

## Goals / Non-Goals

**Goals:**
- `.cascade-frozen` exists and names every core and catalog pin that is old on purpose, so cli `add-deps-cascade-task` and a reviewer can tell frozen pins from stale ones.
- The five trees are current, so the first cascade run starts from a clean baseline.

**Non-Goals:**
- Teaching any test to derive its pin from `schema.DefaultSchemaVersion()`. That is a cascade-maintenance improvement for later, if the trees keep going stale.
- Fixing `TestResolveInstanceArg_RegistryBackedInstancePackage`'s use of the shipped default registry (the GHCR trap). It concerns the podinfo consumers, which this change does not touch.
- Giving `tests/integration/module-apply/testdata` an `identity/` package. `opm module vet` on that tree fails with `cannot find package "./identity"` at both the old and the new pins. This was observed during planning and has nothing to do with the bump, and the apply program does not vet.

## Decisions

### D-a: Which literals go into `.cascade-frozen`

**Context**: The owner decision 2026-10-01 (RELEASING.md, "Pin classes") names two deliberate pins in `tests/e2e/instance_build_test.go`. A repo-wide grep for alpha-line and old catalog literals finds about 20 more in Go tests.

**Options considered**:
1. List only the two named pins. This misses `olderCatalogPin` in the same e2e file, and `internal/cmd/platform/check_test.go`, which holds the same two kinds of deliberate core pin: too-old platforms at `:727`, `:761` and `:785`, and `collisionCoreVersion` at `:351-357`. A reader of `.cascade-frozen` would then wrongly conclude those are safe to bump.
2. List every old literal. Most are parser inputs or fake registry listings that are never resolved (`internal/modref/*_test.go`, `internal/platform/catalog_test.go`, `internal/platform/resolve_test.go:252-253`, `internal/cmd/platform/pull_test.go:57`, `internal/cueedit/cueedit_test.go`). Listing them turns the file into noise and dilutes "frozen" into "anything old".
3. List every literal that a test resolves from a registry and that must stay old for the test to mean anything.

**Decision**: Option 3, plus the one golden literal below. The file holds three entries, each with a single-sentence reason:

```yaml
frozen:
  - path: tests/e2e/instance_build_test.go
    pins: ["opmodel.dev/core@v2", "opmodel.dev/catalogs/opm@v4"]
    reason: "The older-core platform pins core v2.0.0-alpha.11 so the build is refused as too old for the provider count, collisionCorePin v2.0.0-alpha.13 is the floor the colliding-platform seed re-pins up to, never down from, and olderCatalogPin v4.0.0 must stay older than the catalog examples requires so the platform shows catalog version skew."
  - path: internal/cmd/platform/check_test.go
    pins: ["opmodel.dev/core@v2"]
    reason: "Its too-old platforms pin core alpha.6, alpha.9 and alpha.11 to be refused naming the release that derives each missing field, and collisionCoreVersion v2.0.0-alpha.13 is the first core that reports contract collisions instead of failing the fold."
  - path: internal/instinit/render_test.go
    pins: ["opmodel.dev/core@v2"]
    reason: "TestRender_Golden feeds core v2.0.0-alpha.10 into Render and asserts the exact rendered module.cue text carrying it, so the literal is expected output that is never resolved and moving it would prove nothing new."
```

**Rationale**: The reasons paraphrase the tests' own doc comments (`instance_build_test.go:74-76,247-253,285-286,316-319`; `check_test.go:351-356,757-760,780-783`). `olderCatalogPin` is resolved from GHCR: `seedSkewPlatform` (`:123-135`) writes it into a platform `cue.mod`, and `:213` and `:244` assert the platform carries it. The golden literal `v2.0.0-alpha.10` in `internal/instinit/render_test.go:25,45` is not registry-resolved, so Option 3 alone would leave it out. It is listed anyway, by the cascade supervisor's ruling of 2026-10-02: it is an expected-output golden that asserts rendered text, not a pin, so it is frozen with that reason rather than bumped. Listing it tells the cascade and a reviewer that the old literal is intended, where silence would make it look stale.

### D-b: How to bump

**Decision**: In each tree run `cue mod get opmodel.dev/core@v2.0.0-beta.1`, plus `opmodel.dev/catalogs/opm@v4.4.4` where the tree pins the catalog, then `cue mod tidy`, with the canonical registry mapping `CUE_REGISTRY='testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'` (`.github/workflows/pr.yml:25-26`, `Taskfile.yml:18`, `internal/config/templates.go:11`). The mapping must include the `testing.opmodel.dev` domain: without it the module-apply program's version resolution (`tests/integration/module-apply/main.go:237-256`) is refused with "has no published versions", and `TestE2E_InstanceBuild_LayersValuesFile` fails on a cold cache. `pr.yml:60` is not the value to copy: it maps the testing domain to a job-local `localhost:5000` registry. `cue mod get` keeps the `default: true` marker in `initvalues` (verified during planning). Hand edits are banned because `cue mod tidy` is the check the cascade will run, and a hand edit can leave a tree that tidy would rewrite.

### D-c: Section cut

**Decision**: Four sections, each a single commit that leaves `main` green:
1. `.cascade-frozen` alone. It changes no test, so it is green trivially.
2. The four trees whose consumers are unit tests: `fixtures/valid/*`, `initvalues` and `skip-unprovided`. Their e2e consumers and the `skip-unprovided` cluster program run here too, so each tree is proven whole in one section.
3. The two trees consumed by the e2e suite and a cluster-only integration program: `duplicate-identities` and `module-apply`.
4. Archive the change on this branch, so the archive rides the implementing PR and nothing is pushed to `main` (owner decision 2026-10-01 (RELEASING.md, "Owner settings")).

**Rationale**: The cut follows consumer grouping. Section 2's trees are all read by unit tests in CI, section 3's trees are read only by the e2e suite and the cluster-only `module-apply` program. Every section's `task test` (`test:unit` + `test:integration` + `test:e2e`, `Taskfile.yml:75-80`) needs `kind-opm-dev` (`Taskfile.yml:98-100`), so no section can land without the cluster. Five one-tree sections would be valid but give five commits for six one-line diffs.

### D-d: Breakage rule

**Decision**: If a bumped tree's test fails, find out whether the cause is a schema change the fixture must follow. If it is, fix the fixture or the assertion in the same section. If the test asserts behaviour that only the old core has, revert that tree and add it to `.cascade-frozen` with the reason. Make that edit in this section's own commit: section 1 is already committed, and this change never rewrites a commit.

## Research & Decisions

### Are the old pins deliberate?

**Context**: The owner decision 2026-10-01 (RELEASING.md, "Pin classes") asks whether any of the five trees is old on purpose.
**Explored**: `git log` of each `cue.mod/module.cue`; grepping the trees for `alpha`, `pin`, `older`, `deliberate`; a reverted planning probe that bumped all six `cue.mod` files with `cue mod get` and `cue mod tidy` and ran the consumers; a review probe that repeated the bump under the canonical mapping.
**Findings**: No tree comments on its pin. The probes kept everything green:
- `go test ./internal/instinit/ ./internal/workflow/render/ ./internal/cmd/module/`: ok.
- `TestModVet_ValidModule` passes at both the old and the new pins under the GHCR mapping. Without any registry env it skips at both pins, because the kernel's own load of core `v2.0.0-beta.1` fails, not because of the fixture pin.
- `go test ./tests/e2e/ -run 'DuplicateIdentities|OlderCorePlatform'`: both pass.
- `TestE2E_ModBuild_SkipUnprovided` and `TestE2E_ModuleVet_OpenDebugValuesRefusedAtSynthesis` pass at the bumped pins.
- `opm module build tests/integration/module-apply/testdata` renders the same three objects (Service and two Deployments) at both pins.

The probes never ran the two cluster programs, `module-apply` and `skip-unprovided`.
**Decision**: Bump all five. Freeze only the Go-literal pins of D-a.

### Why no spike section

The one unverified assumption is that the two cluster programs still pass at the bumped pins. The repo's rule makes section 1 a spike when design.md carries an unverified assumption; here it does not, for two reasons. First, the evidence is strong: the programs' inputs render the same objects at both pins, and every other consumer of the same trees passes. Second, the outcome cannot change the plan: tasks 2.4 and 3.3 run each program against `kind-opm-dev` before their section commits, and D-d already decides what happens on failure (fix the fixture, or revert and freeze in that section's own commit). A spike would run the same commands one section earlier and decide nothing new.

### Silent skip without a registry

**Context**: Without `CUE_REGISTRY`, `TestModVet_ValidModule` skips: the default registry cannot find `opmodel.dev/core@v2.0.0-beta.1`, and the test treats that as a connectivity error (`vet_test.go:69-74`). A local `go test` can therefore look green without exercising the fixture.
**Decision**: Every task runs the tests with the canonical registry mapping of D-b and checks that the named fixture-consuming tests report PASS, not SKIP. A whole-run no-SKIP check cannot hold, because `TestModVet_CUEValidationError` skips unconditionally (`internal/cmd/module/vet_test.go:122`). Changing either skip is out of scope.

## Risks / Trade-offs

- [The cluster programs are not in CI] A bumped `module-apply` or `skip-unprovided` tree could break `task test:integration` without CI noticing. → Sections 2 and 3 run those programs against the local `kind-opm-dev` cluster before committing.
- [`.cascade-frozen` format drift] The workspace RELEASING.md that defines the format is still in review. → The file uses the fixed format verbatim. If the doc changes it, cli `add-deps-cascade-task` rewrites this file, and the content (paths, pins, reasons) carries over.
- [The trees go stale again] Nothing enforces currency until the cascade task lands. → Requirement on cli `add-deps-cascade-task` (proposal.md, "Depends on / gates"): its test-class set SHALL include the six `cue.mod` files bumped here, and workspace RELEASING.md SHALL list them. Until then, the new test-fixture-lineage requirement is the written rule.

## Migration Plan

No migration. Rolling back means reverting the section commits.
