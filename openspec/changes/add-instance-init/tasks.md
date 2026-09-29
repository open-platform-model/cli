## 1. Gates and spike

- [x] 1.1 Gate: verify `add-dependency-tidy` section 1 is on the base branch (`rg -n "^func Tidy" internal/cuemod` finds `cuemod.Tidy`). If it is not, stop and report the gate as open; do not reimplement tidy
- [x] 1.2 Gate: verify `split-module-and-instance-inputs` section 1 is on the base branch (`rg -n "^func Resolve" internal/modref` finds `modref.Resolve`). If it is not, stop and report the gate as open
- [x] 1.3 Spike the three unverified assumptions from design.md in `tests/e2e/instance_init_test.go`: hand-write the three files for the podinfo fixture (coordinate from `tests/fixtures`, never a literal; core pin read from the fixture's module file), run `cuemod.Tidy` on them, then verify that the module and core pins and the language version are unchanged, that the closure was added, that `Kernel.AcquireInstanceFromDir` loads the package, that a `Check` tidy passes, and that the package still loads with the registry mapped to an unreachable host (warm cache). Then run `cuemod.Tidy` on the same files with an empty module cache and the registry mapped to an unreachable host, and record which error shape identifies the failure (`errors.As` to `net.Error`, or cmd/cue's error text). Record any contradiction in design.md (Risks) before continuing
- [x] 1.4 `task lint` and `task test` green, then commit `test(cmd): prove a generated instance package tidies and loads`

## 2. Package renderer and values ladder (`internal/instinit`)

- [x] 2.1 Add `Input`, `Files` and `Render`; verify golden tests pin the bytes of `cue.mod/module.cue` (modfile canonical format, module and core pins), `instance.cue` (`package instance`, both imports, `core.#ModuleInstance`, metadata, `#module`) and `values.cue` (source comment plus `values:`), cover the `--module-path` override, and show that two renders of the same input are identical
- [x] 2.2 Add `PickValues` with the `debugValues` and empty rungs; verify tests on compiled CUE values cover concrete struct `debugValues`, a defaulted field counting as concrete, non-concrete `debugValues` (`_`) falling to `values: {}`, a partly concrete `debugValues` (`{image: "nginx:1.27", replicas: int}`) falling to `values: {}`, `debugValues: {}` keeping its source name while flagging the empty file, absent `debugValues`, and a concrete non-struct value rendered verbatim
- [x] 2.3 Add the tidy connectivity classifier beside `cuemod.Tidy` (design.md, "A registry failure during tidy exits 3"), using the error shape section 1 recorded, and `Write` (sibling staging directory, `cuemod.Tidy` behind an injectable function, rename into place); verify tests show a successful write leaves only the target, a failing tidy or rename leaves neither the target nor any staging directory, and a tidy connectivity failure comes back as a `*publish.ConnectivityError` while any other tidy failure does not
- [x] 2.4 `task lint` and `task test` green, then commit `feat(cmd): add the instance package renderer`

## 3. `opm instance init`

- [x] 3.1 Move the terminal and prompt helpers (`stdinReader`, the prompt and refusal funnels) from `internal/cmd/module/init.go` to `internal/cmdutil`; verify `go test ./internal/cmd/module/...` passes unchanged
- [x] 3.2 Add `internal/cmd/instance/init.go`: positional classification and `--from` merging, prompts (name, then module path, then namespace; each prompted value checked like its flag), name and namespace checks against the core name rule, `--module-path` check, the existing-directory and enclosing-CUE-module refusals, then resolve, acquire, read the core pin, pick values, render, write and report, with exit codes per design.md; verify command tests cover every refusal scenario of the `instance-init` spec without registry access
- [x] 3.3 Register the command in the instance group with long help and examples (cert_manager and web_app only); verify a command test asserts every flag and its default, and that `opm instance --help` lists `init`
- [x] 3.4 Extend `tests/e2e/instance_init_test.go`: `opm instance init` on the podinfo fixture's major-free path with `-n` exits 0, prints the resolution line, the values source and the vet hint; the three files exist; `opm instance build <dir>/instance.cue` renders against the e2e platform and `opm instance vet <dir>/instance.cue` exits 0; `cuemod.Tidy` in `Check` mode passes on the result; a rerun into the same directory exits 2; an unpublished `--version` exits 2 and leaves nothing. Verify `task test:e2e` passes against GHCR
- [x] 3.5 Update `README.md` (instance init in the command list and quickstart) and the `AGENTS.md` package map (`internal/instinit`); verify `task openspec:check` passes
- [x] 3.6 `task lint` and `task test` green, then commit `feat(cmd): add opm instance init`

## 4. `initValues` rung (gated on core)

- [x] 4.1 Gate: verify a core release on the CLI's core major accepts `initValues` on `#Module` (the core change for 0016 D3/D4 is released; a module setting `initValues` vets against it). If none is released, stop and report the gate as open
- [x] 4.2 Add a test module under `internal/instinit/testdata` carrying `initValues` (a defaulted field, an undefaulted disjunction, an optional field) and different `debugValues`, its `cue.mod` written by `opm module tidy`, never by hand; verify it vets
- [x] 4.3 Put the `initValues` rung first in `PickValues` and name it in the report; verify tests show `initValues` wins over `debugValues`, no `debugValues` content reaches the file, and the rendering keeps the default, keeps the disjunction and omits the optional field
- [x] 4.4 `task lint` and `task test` green, then commit `feat(cmd): scaffold instance values from initValues`
