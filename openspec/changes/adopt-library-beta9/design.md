## Context

Library v1.0.0-beta.9 holds one change, library#228 (library ADR-016 and the closing note of ADR-015). A file-by-file comparison of the two releases in the Go module cache shows these production edits and no other: the `Admit` field and its helpers leave `opm/k8s/ownership`; `IdentityError` gets a pointer receiver and loses its `As` method; the alias `catalog.Source` goes; comments and one citation (`0012:D8:R6` to `0012:D8:R3`) change. The cli stopped setting `Admit` in cli#357 and never used the other names.

## Goals / Non-Goals

**Goals:**

- Pin library v1.0.0-beta.9 with `go.mod` and `go.sum` as the only changed files outside this change folder.
- Prove, for each removed name, that the cli does not use it.

**Non-Goals:**

- No other pin moves. The repo rules tie the catalog, core, template and fixture pins to catalog and core releases, not to a library release (the beta.8 adoption moved none).
- No rename and no cleanup.

## Decisions

The bump MUST change `go.mod` and `go.sum` only. If the build, `go vet`, the linter or a unit test needs any other edit, the work MUST stop and report what the earlier adoptions missed.

### Command syntax, flags, exit codes

None changes.

### The bump command

`go get github.com/open-platform-model/library@v1.0.0-beta.9`, then `go mod tidy`. The repo's own task, `task -x deps:cascade`, is not used: it also resolves the catalog, core and operator pins against the registry, which is outside this change. The beta.8 adoption made the same choice.

### The proof for each removed name

| Name | Proof |
| --- | --- |
| `Admit` (both inputs) | `go build ./...` and `go vet ./...` pass on the bumped tree; `grep -rnw Admit` finds the word only under `openspec/changes/archive/` |
| `IdentityError.As`, the value receiver | `grep -rn IdentityError` finds no Go file. `go vet` stays in the check: it, not the compiler, reports a value-typed `errors.As` target |
| `catalog.Source` | `grep -rn 'catalog\.Source'` finds no file |

### Ownership tests

Every test of the ownership guard (`internal/inventory`, `internal/kubernetes`, `internal/operator`, `internal/cmd`) MUST pass with no edit. An edited test is a behaviour change and stops the work.

## Risks / Trade-offs

- A value-typed `errors.As` target for `IdentityError` compiles and then panics. Mitigation: `task vet` and `task lint` run in the check, and the cli names the type nowhere.
- The archived changes under `openspec/changes/archive/` still name `Admit`. They are history and stay as written.
