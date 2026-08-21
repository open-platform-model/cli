## Context

See proposal.md - Why. The render pipeline already resolves the platform input and holds it on the internal `renderEnv.input` (`internal/workflow/render/kernel.go`); the apply workflow already consumes `Result.PlatformSpec` (`internal/workflow/apply/apply.go`). The only gap is the missing hand-off in `compileInstance` (`internal/workflow/render/render.go`), which builds the `Result` without copying `env.input`.

## Goals / Non-Goals

**Goals:**

- `Result.PlatformSpec` carries the exact resolved (pre-materialize) platform input every render consumed, for both render entrypoints (`FromInstanceFile`, `FromModule`) since both funnel through `compileInstance`.
- A regression test that fails if the field is ever dropped again.

**Non-Goals:**

- No change to the seeding write contract (`EnsureClusterPlatform` stays plain-create, AlreadyExists-noop, Forbidden-warn).
- No change to the seeding trigger condition in the apply workflow.
- No change to `opm operator install`'s catalog-derived seeding path.

## Decisions

- **Fix at the single funnel, not per entrypoint.** `compileInstance` is the one place every render builds its `Result`, so the assignment `PlatformSpec: env.input` there covers instance-file and module-dir renders alike. Alternative (assigning in each `From*` function) duplicates the fix and invites the same drift that caused the bug.
- **Test at the render layer, not via a fake cluster.** A unit test around the render result asserting `Result.PlatformSpec` equals the resolved input (non-empty `Type`, subscriptions preserved) pins the contract cheaply. An end-to-end seeding test against a fake dynamic client would also exercise `wireFromInput`, but that mapping is already covered by `internal/platform` tests; the regression to guard is the hand-off. If the render-layer test cannot run hermetically (kernel materialize needs registry access), fall back to a focused test that drives `compileInstance`'s `Result` construction with a stubbed `renderEnv`.

## Risks / Trade-offs

- [Render tests may require registry access to materialize a platform] → keep the regression test at the narrowest hermetic seam (struct hand-off), mirroring how existing render tests in the repo are structured; do not add network-dependent unit tests.
