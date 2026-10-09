## 1. Adopt library v1.0.0-beta.9

- [ ] 1.1 Bump the library pin to v1.0.0-beta.9 with `go get` and `go mod tidy`; verify `git status --porcelain` lists only `go.mod` and `go.sum` and that `go build ./...` and `go vet ./...` pass
- [ ] 1.2 Search every file outside `openspec/changes/archive/` for `Admit`, `IdentityError` and `catalog.Source`; verify no match
- [ ] 1.3 Run `task test:unit` with no test edited; verify every ownership test passes
- [ ] 1.4 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `fix(deps): bump library to v1.0.0-beta.9`
