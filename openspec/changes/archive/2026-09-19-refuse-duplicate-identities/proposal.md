## Why

The operator now refuses a render whose compiled objects share one Kubernetes apply identity (opm-operator `refuse-duplicate-identities`, enhancement 0015 D15: a module shipping two `transformer-registration` components renders two `TransformerRegistration` objects under one instance-derived name, and the last apply silently won). The CLI renders the same modules through the same kernel and still passes both objects on: `opm module build` prints them, `opm module apply` and `opm instance apply` write them in turn so the last wins, and `opm instance diff` compares against a set the cluster will refuse. A laptop that says a module is fine while the cluster refuses it is the inversion the CLI's pre-flights exist to prevent. Library `v1.0.0-alpha.33` ships the detector and its wording (`opm/helper/objectset`); this change calls it where every CLI render passes through.

## What Changes

- **Library pin to `v1.0.0-alpha.33`**, the release carrying `opm/helper/objectset`. Nothing else moved between alpha.32 and alpha.33.
- **Every render refuses duplicate identities.** `renderInstance`, the one function both entry points (`FromModule`, `FromInstanceFile`) reach after `Kernel.Render`, calls `objectset.Duplicates` on the compiled objects before building resources or a digest, and exits as a validation failure with the library's error. That covers `module build`, `instance build`, `instance vet`, `module apply`, `instance apply` and `instance diff` in one place: the build refuses what the apply would have written.
- **The refusal prints in the CLI's two-stream shape.** The library's message is a header line plus one line per shared identity naming every producing component and transformer; `printValidationError` prints the header under `render failed` and the identity lines as details, wording unchanged, so the CLI and the operator say the same thing.
- **Not in this change.** Deduplication or arbitration (D15: forbid, one per module). A CLI fixture with two registrations: the fixtures pin a catalog build that predates the registration contract, and the refusal is a pure function of the kernel's output, tested without a registry (design.md). Any change to the kernel, the render glue or `opm platform check`.

## Impact

- **Affected commands and packages:** `opm module build`, `opm instance build`, `opm instance vet`, `opm module apply`, `opm instance apply`, `opm instance diff` (a new validation refusal, no flag changes); `internal/workflow/render/render.go` and `validation.go`; `go.mod`. `opm instance vet` renders through `FromInstanceFile` like the rest, so it refuses too: it is the pre-flight the same argument as `build` applies to. `opm module vet` does not render and is unaffected.
- **Behaviour change:** a module whose components render two objects with one identity exited 0 from `build` and lost an object on `apply`; it now exits 2 from every render with both components named. Intended: the cluster refuses the same module.
- **SemVer:** MINOR. A new refusal in commands that already refuse; the library bump is `fix(deps)`.
- **Complexity (Principle VII):** one helper call and one formatting arm. Justified by D15 and by the class it closes: any two components rendering one object name, not only registrations.
- **Flags:** none added.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-render`: the "All renders go through the library kernel" requirement gains the post-render duplicate-identity refusal, as a validation failure carrying the library's wording, with a scenario for two registrations in one module and one for the build refusing what the apply would write.
