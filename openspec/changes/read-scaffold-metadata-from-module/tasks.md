## 1. Read scaffold identity from mod.Metadata

- [ ] 1.1 In `internal/scaffold/scaffold.go`, rewrite `assertDerives` to compare `mod.Metadata.ModulePath` and `mod.Metadata.Version` against `newPath` and `InitialVersion` in a fixed order (modulePath, then version), and replace the "does not evaluate" branch with an internal error when `mod.Metadata` is nil. Keep the `RefusalError` text and evidence unchanged
- [ ] 1.2 In `internal/scaffold/repair.go`, make `statedVersion` read `mod.Metadata.Version` (nil `Metadata` or an empty version refuses as "not stated"), keeping the `cueedit.CheckVersion` gate
- [ ] 1.3 Drop the `cuelang.org/go/cue` import from both files if nothing else in them uses it, and confirm `grep -n 'Package.LookupPath' internal/scaffold` finds nothing
- [ ] 1.4 Run `go test ./internal/scaffold/...`, then the mod-init e2e tests under the kind-opm-dev cluster lock (the e2e package is a cluster suite; take the lock file the supervisor names): `flock <lock-file> go test ./tests/e2e/... -run 'TestE2E_ModInit' -v -timeout 25m`; verify `TestE2E_ModInit_NonDerivingDonorRefuses` still reports "does not derive metadata"
- [ ] 1.5 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` green, then commit `refactor(scaffold): read module identity from mod.Metadata`
