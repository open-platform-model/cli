## 1. internal/publish

- [x] 1.1 Add `gateKernelLoad(ctx, p, opts)` calling `loaderfile.LoadModulePackage(ctx, p.Dir, loaderfile.LoadOptions{Registry: opts.Registry})` for `KindModule`; map the error to a `Refusal` per design D4, branching on the loader sentinels with `errors.Is`.
- [x] 1.2 Wire it into `Run` after `resolveVersion` (so D3's open-identity skip can read the tristate) and before the derivation gates; accumulate, no short-circuit.
- [x] 1.3 Number the gate (msg 12) in its doc comment and the `Run` wiring comment, matching the existing per-gate numbering (there is no catalog listing in `refusal.go`).
- [x] 1.4 Ensure `VetChecks` (`vet.go`) runs the same gate at the same position.

## 2. Tests

- [x] 2.1 `gates_test.go`: a module tree with `identity.Version: #T | *"1.0.1"` and `metadata.version: id.Version` yields exactly one refusal (the kernel one) with the loader's message; the same tree with `Version: "1.0.1"` yields none from this gate.
- [x] 2.2 `gates_test.go`: a tree whose `kind` is `"Catalog"` published as a module yields the `ErrWrongKind` action.
- [x] 2.3 `vet_test.go`: `opm module vet` reports the kernel refusal.
- [x] 2.4 `tests/e2e/publish_test.go`: `--dry-run` on the defaulted tree exits 2 and prints the loader error.

## 3. Enhancement declaration

- [x] 3.1 Create `enhancement.yaml` in this change declaring `implements: [{enhancement: "0011", decisions: [...]}]` with the gate decisions cited in design.md.

## 4. Validation gates

- [x] 4.1 `task fmt`, `task lint`
- [x] 4.2 `task test` (library bumped to v1.0.0-alpha.20 so the gate quotes the loader's named-default diagnostic)
- [x] 4.3 `.github/scripts/publish-templates.sh --dry-run` and `hack/fixtures.sh check` still pass (templates and fixtures load through the kernel today).
