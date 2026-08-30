Two groups, about 10 minutes.

## 1. Remove the fixture (3 min)

- [x] 1.1 `git rm -r tests/fixtures/valid/secrets-module`
- [x] 1.2 `git grep -n 'secrets-module\|secrets_module\|res.#Secret\b\|\$secretName\|\$dataKey\|AutoSecrets\|opm-secrets' -- . ':!openspec/changes/archive' ':!openspec/specs' ':!CHANGELOG.md' ':!docs/rfc' ':!adr'` returns nothing (the excluded paths are design history, see proposal)

## 2. Verify (5 min)

- [x] 2.1 `go test ./internal/cmd/module/...` (vet tests still pass on `simple-module`, the only fixture they read)
- [x] 2.2 `task check`
