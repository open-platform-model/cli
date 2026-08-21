## 1. Fix the hand-off in the render funnel

- [x] 1.1 In `internal/workflow/render/render.go` `compileInstance`, assign `PlatformSpec: env.input` when constructing the `Result` (alongside the existing `Platform: env.resolution`)

## 2. Regression coverage

- [x] 2.1 Add a test in `internal/workflow/render/render_test.go` (following the package's existing hermetic test style) asserting the constructed `Result` carries the resolved platform input: non-empty `Type` and subscriptions equal to the input, guarding against the field being dropped again
- [x] 2.2 Verify the apply-side consumer path compiles unchanged and the seeded wire document from a populated input carries `type` and `registry` (covered by existing `internal/platform` tests; extend `cluster_test.go` only if the populated-input create path lacks an assertion)

## 3. Validation gates

- [x] 3.1 Run `task fmt` and `task lint`
- [x] 3.2 Run `go test ./internal/workflow/render/... ./internal/platform/...`, then `task test:unit`
