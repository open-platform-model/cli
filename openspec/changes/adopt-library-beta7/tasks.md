## 1. Adopt library v1.0.0-beta.7 and pin its order

- [ ] 1.1 Run `task -x deps:cascade` in the work tree; verify `git diff` equals `git diff origin/main...origin/deps/cascade` of the main checkout (`go.mod` and `go.sum` only) and `go build ./...` passes
- [ ] 1.2 Rewrite the literals of `internal/kubernetes/order_parity_test.go` to the beta.7 table (constants with the three new ones, exact rows, cluster definitions in another version, kind rows, the `*Class` suffix rule, the default), extend the sort set with one object per new weight class and a PersistentVolumeClaim, record both expected sequences, rename the tests and data away from "retired", and rewrite the comments to say what the test is for; verify `go test ./internal/kubernetes/` passes and that changing one literal makes it fail
- [ ] 1.3 Check the remaining beta.7 changes against the cli as design.md lists them; verify `task test:unit` passes with no other edit
- [ ] 1.4 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `fix(deps)!: bump library to v1.0.0-beta.7`
