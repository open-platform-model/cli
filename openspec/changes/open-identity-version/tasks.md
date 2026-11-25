# Tasks

## 1. Identity check without text reads

- [ ] 1.1 Add core's `VersionMajor` derivation (`VersionMajor: "v" + strings.SplitN(Version, ".", 2)[0]` and `VersionMajor: Major`, with `Major` derived from `ModulePath`) to the test stub `#IdentityPackage` in `internal/publish/fixtures_test.go`, and rewrite its comment; verify `TestVetChecks_OpenVersionIsNotPoliced` now fails against the unchanged code
- [ ] 1.2 Add offline `VetChecks` and `Run` tests for an open, absent, skewed and malformed `Version`, and for `--version` filling an open `Version` with a matching and with a different major; verify they fail before 1.3 and pass after it
- [ ] 1.3 Change `conformIdentity` in `internal/publish/identity.go` to plain validation plus a refusal for each absent field of `identityFieldNames`, and drop the `incomplete value` text filter; verify the tests of 1.2 pass
- [ ] 1.4 In `Run` (`internal/publish/gates.go`), pass the `--version` fill into the conformance check so it validates the identity as filled; verify the fill tests of 1.2 pass and `--dry-run` refuses a filled version with another major
- [ ] 1.5 Add one registry-backed test in `internal/publish` that runs the open and fill cases against the real core `#IdentityPackage` (as `realtree_test.go` loads it) and skips only when the schema cannot be fetched; verify it passes with the default registry
- [ ] 1.6 Remove the `conformIdentity` entry from the allowlist in `pkg/errors/textmatch_refusal_test.go`, and change "four" to "three" in `AGENTS.md`; verify the refusal test passes
- [ ] 1.7 Rewrite the `VetChecks` godoc in `internal/publish/vet.go` and the comment of `TestVetChecks_OpenMetadataVersionKeepsName` to the new rule, with no issue reference; verify `task cite-lint` passes
- [ ] 1.8 Run `task lint` and `task test`; verify both pass

## 2. Vet skips the render on an open Version

- [ ] 2.1 Make `TestModVet_ValidModule` set `cfg.Registry = config.DefaultRegistry`, skip only on the connectivity exit code, and fail on `NoRegistryError`; delete the `providers` block from `setupTestConfig`; verify the test runs and passes with a reachable registry
- [ ] 2.2 Add a fixture under `tests/fixtures/valid/` with an open identity `Version` wired into `metadata.version`, following `references/fixtures.md`, and a vet command test that asserts the render-skip line, exit 0 and no `Module valid` line; verify it fails before 2.3
- [ ] 2.3 In `runVetModuleOnly` (`internal/cmd/module/vet.go`), after the values check, log the render-skip line and return nil when the plan's `Version` state is open; verify the test of 2.2 passes
- [ ] 2.4 Run `task lint`, `task test` and `task test:fixtures`; verify all pass
