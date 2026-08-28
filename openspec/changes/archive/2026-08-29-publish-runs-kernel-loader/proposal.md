## Why

`opm module publish` loads the tree with raw `cue/load` and judges identity by unifying `identity/` against core's `#IdentityPackage` and reading `Version` through `String()`, which resolves defaults. The kernel's module loader judges the same tree with its shape gate, which asks `IsConcrete()`. The two disagree on a defaulted identity `Version`: publish says GO, the loader refuses. Measured 2026-08-28: all 20 `modules` v2 fleet modules passed every publish gate and were unloadable by `opm module build` and by the operator. Enhancement 0011 states that publish "does not resolve dependencies differently from a consumer"; today it *validates* differently from a consumer, and the only frontend that can catch "publishable but unloadable" before a tag exists is publish.

## What Changes

- `opm module publish` (and `opm module vet`, which shares the pipeline) runs the kernel's module loader (`library` `loader/file.LoadModulePackage`) on the tree as a gate. A loader refusal becomes a publish refusal carrying the loader's own error, branched on the loader's sentinels (`ErrWrongKind`, `ErrMissingRequiredField`, `ErrInvalidPackage`) for the action line.
- Gate ordering: after the tree loads and before the identity gates; accumulates like the other gates (one cause, one refusal: the identity tristate does not also refuse a defaulted field, so no double report).
- `--dry-run` runs it; the plan prints the outcome.

Not in scope: catalog publish (catalogs go through `materialize`, which reads via `String()` and has no equivalent gate; if one is added it is a separate change). Instances are never published, so no instance-side gate.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artifact-publishing`: "Gates and refusal catalog" gains the kernel-loader gate; "One pipeline, two entry points" gains the scenario that a module the kernel would refuse is refused by publish with the loader's error.

## Impact

- Commands: `opm module publish`, `opm module vet`. Packages: `internal/publish` (`gates.go` `Run`, new gate function, refusal catalog entry), `internal/publish/vet.go` if `VetChecks` does not already pass through `Run`.
- Dependency: the loader is already embedded (`internal/workflow/render` uses it); no new module.
- SemVer: MINOR (`feat(publish)`): a new gate can refuse trees that previously passed. That is the intent; every such tree was already unloadable.
- Complexity: one gate function; the loader call already exists elsewhere in the CLI. Justified by closing the class of defect rather than the instance.
- Enhancement: 0011 (D16-D18 gate family, and the "publish never judges differently from a consumer" principle). Create `enhancement.yaml` declaring `implements: 0011` with the gate decisions the design cites.
