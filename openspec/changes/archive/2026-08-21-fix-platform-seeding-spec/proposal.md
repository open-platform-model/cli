## Why

The D12 solo-cluster Platform seeding is broken: `render.Result.PlatformSpec` is declared as the carrier for the resolved platform spec but is never assigned, so `opm instance apply`'s write-if-absent seeding (`platform.EnsureClusterPlatform`) always receives a zero `synth.PlatformInput`. The resulting create document carries `spec.type: ""` and no registry, the CRD rejects it, and the seeding fails with a warning on every apply where it should fire. The `platform-resolution` spec's "Absent Platform is seeded" scenario is silently unmet.

## What Changes

- `compileInstance` in `internal/workflow/render/render.go` copies the resolved platform input (already held on `renderEnv.input`) onto `Result.PlatformSpec`, alongside the provenance it already copies (`Result.Platform`).
- Regression test asserting that a render's `Result.PlatformSpec` equals the resolved platform spec the render consumed (non-zero `type`, subscriptions intact), so the seeded document can never regress to the zero value again.
- No flag, command, or wire-shape changes. SemVer: PATCH (bug fix).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-resolution`: the "Solo-cluster Platform write-if-absent" requirement is tightened to state that the seeded document SHALL be the exact resolved spec the render consumed, carried through the render result with no re-read of the platform file at apply time (no TOCTOU), and a scenario is added asserting the seeded spec is non-empty and matches the render's platform input.

## Impact

- `internal/workflow/render/render.go` (`compileInstance`): one-line assignment.
- `internal/workflow/render/` tests: new regression coverage for `Result.PlatformSpec`.
- Consumers unchanged: `internal/workflow/apply/apply.go` already reads `Result.PlatformSpec`; `internal/platform/cluster.go` write contract untouched.
- `opm operator install` seeding path (`EnsureClusterPlatformForCatalog`) is unaffected.
