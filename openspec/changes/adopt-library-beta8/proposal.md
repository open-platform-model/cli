## Why

Library v1.0.0-beta.8 is released. The release cascade proposed the pin bump (cli#355), but the bump fails the cli's unit tests: two tests pin, on purpose, a gap that beta.8 closes. A registry that hands out bearer tokens and whose token endpoint refuses the caller was read as "no response"; the library now reads it as a refused credential (library#222). The cli pinned the wrong answer so that the bump would fail and the right expectation would be a reviewed edit. This change is that edit, with every other behaviour change of beta.8 checked against the cli.

## What Changes

- Bump `github.com/open-platform-model/library` from v1.0.0-beta.7 to v1.0.0-beta.8 (`go.mod` and `go.sum` only, the same two files the cascade branch carries).
- Token endpoint answers (library#222). The library reads every answer of a token endpoint by its status: a 401 or 403 is a refused credential, and a 429 or 5xx is a failed registry operation, no longer "no response". What a user sees changes in three places:

  | Command | Token endpoint answers | Before | After |
  | --- | --- | --- | --- |
  | `opm module publish`, `opm catalog publish` (the push) | 403 | exit 3, `registry unreachable: pushing ...: 403 Forbidden` | exit 4, `registry refused the credentials (authentication or permission): pushing ...: 403 Forbidden`, then `opm registry login <host>` |
  | `opm platform check` (the platform build) | 401 | exit 2, hint to pin a published build | exit 4, `Log in to the registry, then retry:  opm registry login <host>` |
  | `opm instance init` (resolving the staged package's dependencies) | 401, 429 or 5xx | exit 3, `registry unreachable` | exit 1, the registry's own answer |

  The pinned gap tests are replaced by tests of the right answer, and the notes that named the gap in the code, the publish page and two specs are removed.
- On the push, a 429 or 5xx from the token endpoint keeps exit 3; by the library's classification its text moves from `registry unreachable` to `registry operation failed` (not driven in a cli test: the local fake fails the lookup first).
- One limit stays and is stated where the old note stood: a 403 from the token endpoint on a fetch (not a push) reaches the cli as "module not found", exactly as a 403 answer to a tag lookup does. The library keeps that reading on purpose (library ADR-014).
- Public surface (library#223): no edit. The cli uses none of the removed or deprecated names.
- Unset required values (library#227): no code edit. A render that leaves required `#config` values unset now prints one finding for each unset value at `values.<field>`, read by a component or not, and no finding at the place inside a component that reads it. A new test pins that output.
- No change to commands, flags or output formats.

SemVer class: PATCH after GA (a failing run fails with a truer message and exit code; no successful run changes). During beta it ships as the next `-beta.N`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artifact-publishing`: the requirement "Exit codes" said a refusing token endpoint is reported as "registry unreachable" on the push. That sentence is replaced by the right answer and a scenario.
- `errors-domain`: the requirement "Platform module build failure hints" listed a token endpoint refusal among the refusals the cli receives as not found. A 401 from the token endpoint is now a refused credential; the 403 stays.
- `instance-building`: the requirement "The resulting instance must be fully concrete" gains the shape of the report for unset required values.

## Impact

- `go.mod`, `go.sum`: library pin.
- Tests: `internal/publish/registryanswer_test.go`, `internal/config/platform_hint_forms_test.go`, `internal/cmd/platform/check_refused_test.go`, `internal/cmd/module/vet_test.go`, `internal/cmdutil/publish_test.go`, `internal/cmd/instance/init_pin_test.go`, `internal/cmd/catalog/registry_pin_test.go`, `internal/cuemod/connectivity_pin_test.go`, `internal/cuemod/connectivity_test.go`, `internal/workflow/render/unset_values_test.go` (new).
- Comments: `internal/publish/publish.go`.
- Docs: `docs/site/authoring/publish-a-module.md`.
- Users: the three rows above, and the text of a render refusal for unset required values.
