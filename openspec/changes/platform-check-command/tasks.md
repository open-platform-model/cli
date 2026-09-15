# Tasks: platform-check-command

Three sections. design.md carries no unverified assumption: the library surface was read at the pinned version and the report shape follows an existing precedent in this repo, so section 1 is not a spike.

## 1. The report value

- [x] 1.1 Add `internal/platform/check.go` per design.md § The report is a value: a `Report` carrying the resolution, the defined-to-catalog map, the per-contract implementations, the two lists and the two booleans, built from a `*libplatform.ContractInventory` by a constructor. Verify: `go build ./internal/...` passes and nothing outside the package constructs a `Report` field-by-field.
- [x] 1.2 Add `Render()` producing the report text: the provenance line first, then the defined contracts with their defining catalogs, the implementations per contract, and the two lists under separate headings, with the empty-inventory case worded as "these catalogs define no contracts" rather than as a clean result. Verify: table tests cover clean, unfulfilled-only, over-subscribed-only, both, and empty.
- [x] 1.3 Add the verdict accessor that reports routability alone, and a test asserting the asymmetry directly: an unfulfilled-only report is routable, an over-subscribed one is not. Verify: the test names enhancement 0015 D18 in a comment so the asymmetry is not "fixed" later.
- [x] 1.4 `task fmt lint test:unit` green, then commit `feat(platform): add the contract-inventory check report`.

## 2. The command

- [x] 2.1 Add `internal/cmd/platform/platform.go` (group constructor taking `*config.GlobalConfig`) and `internal/cmd/platform/check.go` (the `check` subcommand, optional positional platform directory), following the `config vet` template: flags wired in the constructor, `RunE` delegating to a package-level runner, `c.Context()` for the context. Verify: `opm platform --help` lists `check`.
- [x] 2.2 Register the group in `internal/cmd/root.go` beside the existing five. Verify: a structural test in the new package asserts the group's `Use`, its `check` subcommand and its flag set, following the `catalog_test.go` pattern.
- [x] 2.3 Wire the runner: resolve the platform (positional argument first, else the existing precedence, cluster-CR arm deliberately not passed), build it through the kernel, read `Contracts()`, print the report, and return nil or a validation `ExitError` with `Printed: true` when it is not routable. Verify: a test drives the command in-process against a temp platform module and asserts both the printed report and the exit code for a routable and a non-routable platform.
- [x] 2.4 Error paths: a resolved directory that is not a platform module fails before any build; a platform that does not build reports the grouped CUE diagnostic and the validation code; a platform whose core predates the inventory fails naming the missing field and the required core release. Verify: one test per path, each asserting the message and not only the error.
- [x] 2.5 `task fmt lint test:unit` green, then commit `feat(cmd): add opm platform check`.

## 3. The defining catalog on unresolved demands

- [ ] 3.1 Extend `cmdutil.FormatUnresolvedDemands` with the defined-by clause per design.md § The formatter change is additive; rows without a defining catalog keep today's wording byte-for-byte. Verify: the existing formatter tests pass unchanged, and a new case asserts the defined-by line.
- [ ] 3.2 Run the cross-cutting checks: `task test` in full, and confirm no other test asserted the nothing-implements sentence for a row that now carries a catalog. Verify: `task check` green, including `openspec:check`.
- [ ] 3.3 `task check` green, then commit `feat(cmdutil): name the defining catalog on unresolved demands`.
