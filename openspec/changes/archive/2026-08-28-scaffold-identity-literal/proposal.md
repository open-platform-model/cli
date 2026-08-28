## Why

The identity package the CLI writes and ships declares `Version: #VersionType | *"x.y.z"`, a defaulted disjunction. The kernel's loader shape gate (`library` `opm/helper/loader/internal/shape`) tests `IsConcrete()`, which is false for that form, so a module carrying it is refused by `opm module build` and by the operator's registry load with `required field "metadata.version" is not concrete`. The official templates and the podinfo fixture only load because their `module.cue` interpolates the reference (`version: "\(id.Version)"`) to force the default; the repair path (`identityFileContent`) writes the bare disjunction into trees that may not. The `modules` fleet copied the scaffold's form and was unloadable for its whole v2 life until 2026-08-28. Publish never sees the problem because it reads `Version` through `String()`, which resolves defaults. The local `#VersionType` copy in each identity file is redundant: publish unifies the package against core's `#IdentityPackage`, whose `Version!: #VersionType` already constrains it, and nothing references `id.#VersionType`.

## What Changes

- The identity file the scaffold repair writes declares `Version: "x.y.z"` (plain literal) and no local `#VersionType`.
- The three official templates (`templates/minimal|standard|advanced`) declare `Version: "1.0.0"` with no `#VersionType` and no `// x-release-please-version` marker (release-please has no `extra-files` entry for templates; `publish-templates.sh` reads the version with `cue eval`, which is unaffected). Their `module.cue` returns to `version: id.Version` and drops the interpolation comment.
- `Reidentify` resets the scaffold's `Version` with the plain writer (`SetIdentityVersion`) instead of `ResetIdentityVersion`; `ResetIdentityVersion` and `spliceVersionReset`, which exist only to rebuild the disjunction form, are removed.
- `version set` keeps tolerating all three authored forms (literal, `&` chain, defaulted disjunction); only what the CLI *emits* changes.
- Templates get a version bump so the release pipeline republishes them (a template is a published artifact; `publish-templates.sh` filters to unpublished versions).

Not in scope: the podinfo fixture (sibling change `fixtures-identity-literal`), a publish-side loader gate (sibling change `publish-runs-kernel-loader`), the kernel's message (library change `loader-gate-defaulted-disjunction-diagnostic`).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `template-modules`: "init scaffolds by fetch and re-identification" (`Version` reset to a plain literal, not "the defaulted initial form"); "Templates are published modules in a reserved segment" gains the requirement that a template's identity is a literal and loads through the kernel unmodified.
- `authoring-commands`: "version set writes in place, idempotently, offline" gains the scenario that the CLI-emitted form is a literal; the existing tolerance scenarios stand.

## Impact

- Commands: `opm module init` (scaffold and repair). Packages: `internal/scaffold` (`repair.go` template, `scaffold.go` `Reidentify`), `internal/cueedit` (delete `ResetIdentityVersion`), `templates/*`.
- Tests: `tests/e2e/mod_init_test.go:164` (asserts `*"0.1.0"`), `internal/scaffold/repair_test.go`, `internal/cmd/module/init_test.go`, `internal/cueedit` reset tests.
- SemVer: PATCH (`fix(scaffold)`): the scaffold emitted a form the kernel refuses. Complexity: net negative (one writer and its tests removed).
- Enhancement: none; 0011 D3/D8/D12 (identity is written by `version set`, located by schema path) are unchanged.
