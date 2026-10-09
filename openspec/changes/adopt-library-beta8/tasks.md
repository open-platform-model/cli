## 1. Adopt library v1.0.0-beta.8 and state the token endpoint answer

- [x] 1.1 Bump the library pin to v1.0.0-beta.8 with `go get` and `go mod tidy`; verify `git diff origin/deps/cascade -- go.mod go.sum` is empty and `go build ./...` and `go vet ./...` pass
- [x] 1.2 Run `task test:unit` on the bumped tree with no other edit and record every failing test with its cause in design.md
- [x] 1.3 Replace `TestPush_TokenEndpointRefusal_Pinned` with `TestPush_TokenEndpointRefusal` (a `*RegistryError` marked `Unauthorized` that names the host) and remove the "Known gap" comment of `publish.RegistryFailure`; verify the test fails with the old `go.mod`
- [x] 1.4 Move the token 401 row of `TestPlatformBuildHint_RefusalNotTypedAsOne_Pinned` to `TestPlatformBuildHint_RefusedCredential` and add the same case to `TestPlatformCheck_BuildFailureExitCodes` (exit 4, the login hint with the host)
- [x] 1.5 Add token endpoint rows (401 and 403) to the schema-fetch tests of vet and publish, to `TestAcquireModule_Pinned`, `TestInitWrite_Pinned`, `TestRegistryCheck_ExitCodes_Pinned` and `TestIsConnectivityError_Pinned`, and add `TestIsUnauthorized_TokenEndpoint`
- [x] 1.6 Check library#223 against the cli: search every removed or deprecated name; verify `task lint` reports 0 issues with no new exclusion
- [x] 1.7 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `fix(deps): bump library to v1.0.0-beta.8`

## 2. Pin the report for unset required values

- [x] 2.1 Add `TestFromModule_UnsetRequiredValuesAreNamed` in `internal/workflow/render`: a module whose `debugValues` leave two required values unset, one read by a component; verify it passes on beta.8 and fails with the old `go.mod`
- [x] 2.2 Search `tests/e2e`, `tests/integration` and `docs/` for a quoted kernel refusal that the new text makes stale; correct each one
- [x] 2.3 `task lint` and `task test:unit` green, then commit `test(render): pin the report for unset required values`

## 3. Correct the known-limit notes

- [x] 3.1 Rewrite the registry failure sentence of `docs/site/authoring/publish-a-module.md`: a refusing token endpoint is a refused credential on the push; a 403 on a fetch still reads as not found
- [x] 3.2 `task docs:bundle:check` and `task openspec:check` green, then commit `docs(publish): name a token endpoint refusal as a refused credential`
