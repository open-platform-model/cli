## REMOVED Requirements

### Requirement: Bundle renderer renders a BundleRelease into resources
**Reason**: The bundle path is removed (enhancement 0002 D15, supersedes D7). The `pkg/render` `Bundle` struct, `NewBundle`, `Bundle.Execute`, and `BundleResult` are deleted with `pkg/render/bundle_renderer.go`; they had no production caller (the `NewBundle` reference lived only in `pkg/render/matchplan_test.go`).
**Migration**: None. Module rendering is unaffected — `pkg/render.Module` continues to render a single module instance. Multi-instance bundle rendering is reintroduced only if bundle support is built (a future enhancement), under the Instance vocabulary.

### Requirement: Bundle renderer uses Release map
**Reason**: Same removal — the `Bundle.Execute` iteration over `bundle.Release.Releases` is deleted along with the `Bundle` renderer and the `pkg/bundle` package.
**Migration**: None — no production code invoked this path.
