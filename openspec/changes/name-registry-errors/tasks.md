## 1. Publish names the registry failure

- [x] 1.1 Add tests that drive a real 401, 403, 429 and 503 answer through the already-published lookup and the push, and see them fail on the current code
- [x] 1.2 `internal/cuemod`: add `IsUnauthorized` beside `IsConnectivityError`, with tests
- [x] 1.3 `internal/publish`: add `RegistryError` and `RegistryFailure`, and use it in `gateAlreadyPublished` and `Push`
- [x] 1.4 `internal/cmdutil`: map `*RegistryError` to exit 3 in `publishError`, add the `opm registry login` hint, classify the core-schema fetch, with tests
- [x] 1.5 Update the exit-code line of the `module publish` and `catalog publish` help
- [x] 1.6 task fmt, task lint and task test:unit green, then commit fix(publish): name a refused credential instead of calling the registry unreachable

## 2. No registry configured

- [x] 2.1 `internal/config`: add `RegistryConfigured` and `NoRegistryError`, with tests
- [x] 2.2 `internal/cmdutil`: add `NoRegistryError`, use it in `RunPublish`, with tests
- [x] 2.3 `internal/cmd/module/vet.go`: use `NoRegistryError` for the failed schema fetch, with a test
- [x] 2.4 task fmt, task lint and task test:unit green, then commit fix(config): say when no registry is configured and point to opm config init

## 3. Docs

- [ ] 3.1 Update the publish pages under `docs/site` that list the exit codes and the registry failures
- [ ] 3.2 task docs:bundle:check and task openspec:check green, then commit docs(publish): list the registry failure classes
