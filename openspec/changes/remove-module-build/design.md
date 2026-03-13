## Context

The `opm module build` command maintains two preparation paths (Path A: pure module source without `release.cue`, Path B: module dir with sibling `release.cue`) that converge into the shared render tail. Both paths exist to support module authors previewing rendered output during development. However, this creates duplicated orchestration, inconsistent values-precedence rules between paths, and redundant processing (`SynthesizeModule` validates config and finalizes components, then `ProcessModuleRelease` does the same work again).

The `opm release build -r <release-file>` command (Path C) already handles the production use case. Removing `opm module build` simplifies the render pipeline to a single entrypoint.

## Goals / Non-Goals

**Goals:**

- Remove the `opm module build` command and all code exclusively supporting it
- Eliminate the `FromModule` workflow function and its two preparation paths
- Remove `SynthesizeModule` from `pkg/render/`
- Clean up orphaned types, helpers, and tests

**Non-Goals:**

- Do not change `opm module vet` or `opm module init` behavior
- Do not change the `FromReleaseFile` path or `ProcessModuleRelease`
- Do not change values resolution for the release-file path
- Do not reintroduce module-source rendering under a different design (that's future work)

## Decisions

**1. Delete, don't deprecate.**
`opm module build` is removed outright rather than deprecated with a warning. The project is pre-1.0 and under heavy development. A deprecation cycle adds maintenance burden for a command that has a direct replacement (`opm release build`). If module-source rendering returns, it will be redesigned from scratch using the `#ModuleRelease` schema properly.

**2. Keep `opm module vet` independent.**
`vet` validates config directly via `pkgrender.ValidateConfig` without going through the render workflow. It has no dependency on `FromModule`, `SynthesizeModule`, or any of the removed code. It stays as-is.

**3. Remove `SynthesizeModule` entirely.**
It has exactly one caller (`FromModule`). The function mixes skeleton construction with processing work that `ProcessModuleRelease` repeats. Rather than slim it down for hypothetical future use, delete it. If synthesis is needed later, it should be redesigned to produce output compatible with the `#ModuleRelease` CUE schema.

**4. Remove `ReleaseOpts` type but keep `ReleaseFileOpts`.**
`ReleaseOpts` is only used by `FromModule`. `ReleaseFileOpts` is used by `FromReleaseFile` and stays.

## Risks / Trade-offs

**[Risk] Users relying on `opm module build` lose their workflow.**
Mitigation: The project is pre-1.0 and the replacement (`opm release build -r`) is available. Document the migration in changelog.

**[Risk] Removing too aggressively could break shared code.**
Mitigation: Each function/type identified for removal has been traced to confirm it is only used by the module-build path. The removal map was built by grep, not guesswork.

**[Risk] `SynthesizeModule` tests cover edge cases that might be useful later.**
Mitigation: The test scenarios (success, gate failure, no components) are captured in specs and can be recreated when synthesis is redesigned.
