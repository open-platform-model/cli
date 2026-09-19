# Tasks: refuse-duplicate-identities

Two sections per design.md. design.md carries no unverified assumption: the render site and the print funnel were read at the current commit and the helper at its released version, so section 1 is not a spike.

## 1. Library pin to v1.0.0-alpha.33 (go.mod)

- [x] 1.1 `go get github.com/open-platform-model/library@v1.0.0-alpha.33 && go mod tidy`, then `go build ./...`. Verify: `go.mod` names alpha.33 and `go doc github.com/open-platform-model/library/opm/helper/objectset` lists `Duplicates` and `DuplicateIdentitiesError`.
- [x] 1.2 `task fmt lint test:unit` green, then commit `fix(deps): bump library to v1.0.0-alpha.33`.

## 2. The refusal (internal/workflow/render)

- [x] 2.1 Add `refuseDuplicateIdentities(out *kernel.RenderResult) error` to `render.go` per design.md § The check sits in `renderInstance`, and call it right after the render error check, before the replacement warnings, exiting with the validation code and `Printed: true`; update `renderInstance`'s doc comment to list the refusal. Verify: `go build ./... && go vet ./...` pass.
- [x] 2.2 Tests in `render_test.go` over hand-built `kernel.RenderResult` values (`cuecontext.New().CompileString`): two `TransformerRegistration` objects with one name from components `registration` and `registration-copy` return a `*objectset.DuplicateIdentitiesError` naming the identity and both producers; distinct objects return nil; a value without `metadata.name` beside a duplicate is skipped. Verify: `go test ./internal/workflow/render/...` passes.
- [x] 2.3 Add the `*objectset.DuplicateIdentitiesError` arm to `printValidationError` per design.md § Printing, and a `captureValidationOutput` test asserting the log stream carries `render failed: ` plus the library's header line and the details stream carries the identity line with both components. Verify: the existing `validation_test.go` cases pass unchanged.
- [x] 2.4 Cross-cutting: `task check` in full (including `openspec:check`). Verify: green.
- [x] 2.5 `task check` green, then commit `feat(render): refuse a render whose objects share one apply identity`.
