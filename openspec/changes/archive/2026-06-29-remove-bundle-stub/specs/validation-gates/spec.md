## REMOVED Requirements

### Requirement: Bundle Gate validates consumer values against #bundle.#config
**Reason**: The bundle path is removed (enhancement 0002 D15, supersedes D7). There is no bundle-loading or bundle-processing path left to gate: `LoadBundleReleaseFromValue` and `ProcessBundleRelease` are deleted, so the Bundle Gate (validating bundle-level values against `#bundle.#config` before per-release Module Gates) has no call site.
**Migration**: None. The Module Gate is unaffected — consumer values are still validated against `#module.#config` before processing. A bundle-level gate is reintroduced only if bundle support is built (a future enhancement).
