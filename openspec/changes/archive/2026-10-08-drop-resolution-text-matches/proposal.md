## Why

The cli still decides some answers by reading the text of an error. The library types every fetch
failure (`*FetchError`) and, since `v1.0.0-beta.7`, every author-defect resolution failure
(`*ResolutionError`), so the cli can stop (0021:D8:R12, owner ruling "Strict: type everything",
library ADR-013 row d1). The cli already builds on beta.7 (`go.mod:14`), so nothing waits on a
release.

## What Changes

Every place in non-test cli code that reads error text, found by a search of `cmd/`, `internal/`
and `pkg/` at `5aa8dc88` for string predicates on `err.Error()` and on messages taken from an
error:

| # | Site | What it reads today | Verdict |
| --- | --- | --- | --- |
| 1 | `internal/publish/compat.go:313-317` `unprovidedImport` | `cannot find module providing package` after "no `*FetchError`" | **Replaced.** `errors.As(liberrors.Classify(err), &re) && re.Kind == liberrors.ResolutionImportUnprovided`. |
| 2 | `internal/config/platform.go:111` `platformBuildHint` | `cannot find package`, `cannot expand module graph` | **Replaced.** "The chain holds a `*FetchError` or a `*ResolutionError`" after `liberrors.Classify`. |
| 3 | `internal/config/platform.go:113` `platformBuildHint` | `#registry` anywhere in the message | **Replaced.** A CUE error in the chain whose path starts with the `#registry` selector (`cueerrors.Errors(err)`, `Path()`). |
| 4 | `internal/config/loader.go:255-265` `removedFieldHint` | `providers`, `cacheDir`, `skewPolicy` anywhere in the message | **Replaced.** A CUE error whose path starts with that field. The error is the cli's own config validation, so no library type is involved. |
| 5 | `internal/cuemod/tidy.go:210` `classify` | `module is not tidy` prefix of `cmd/cue` output | **Stays.** Gap: CUE's `modload.ErrModuleNotTidy` is an internal type and `cmd/cue` prints it; the library has no tidy verb and `Classify` does not read this form. It is not a fetch or resolution failure, so 0021:D8:R12 does not ask for it. Pinned by `internal/cuemod/tidy_test.go`. |
| 6 | `internal/publish/identity.go:35` `conformIdentity` | `incomplete value` on each CUE error of the cli's own validation | **Stays.** Gap: the public CUE API gives an incomplete error no type or code. Not a fetch or resolution failure. Pinned by `internal/publish/identity_test.go`. |
| 7 | `pkg/errors/grouped_errors.go:55` | `errors in empty disjunction` summary lines | **Stays.** A display filter over CUE's summary line, which has no type. It picks no answer and no exit code. |
| 8 | `internal/workflow/render/validation.go:62` | `strings.Cut` of a `*object.DuplicateIdentitiesError` message at its first newline | **Stays.** The type decides (`errors.As`); the cut only lays out the library's own message. |

- Sites 1 to 4 lose their text match. No exit code moves: site 1 keeps "absent" against a
  `*ConnectivityError`, and sites 2 to 4 pick a hint, never an exit code.
- **Three platform build hint answers change**, because the text matched forms the types do not cover, or missed
  one they do. All three are in `platformBuildHint`, exit code 2 before and after:
  - A directly imported dependency whose module file does not parse
    (`ResolutionModuleFileInvalid`, direct path) gets the "Pin a published build" hint. Today it
    gets the default hint, while the same defect met during graph expansion already gets the pin
    hint.
  - A failed import that is neither a fetch nor a resolution failure (a package-name mismatch,
    `no files in package directory`, an import cycle) gets the default hint. Today it gets the
    pin hint through `cannot find package`.
  - A `#registry` entry the library's shape check refuses (it flattens the CUE cause into text
    and wraps `ErrMissingRequiredField`) gets the default hint. Today the `#registry` word in the
    text gives it the key-and-import hint, which does not describe that defect.

  These were an owner question (report T2.7, question 1), accepted at the proposal gate: each is
  pinned by a test.
- **One config hint answer changes** in `removedFieldHint`, exit code unchanged: a config that is
  invalid at another field while a value holds the word `providers`, `cacheDir` or `skewPolicy`
  gets the generic hint. Today the word in the message gives it that field's hint.
- Every other message and hint stays byte-identical, pinned by the tests named in `tasks.md`.
- A guard test refuses a new error-text predicate in non-test code, with sites 5 to 8 as its
  allowlist.

Not in this change:

- `TestPush_TokenEndpointRefusal_Pinned` (cli#339) keeps its pinned gap. The library fix
  (library#222, `c81e7fd`) is on library `main` and in no tag: the newest tag is `v1.0.0-beta.7`
  (`1d9fbbd`). The test moves with the cli's bump to the library release that carries it.
- The `module build` message for an unset required value (T9.20). It does not reproduce at
  `5aa8dc88`: `opm module build .` on a copy of `templates/minimal` with `note: string` in
  `#config` prints `incomplete value string`, `values.note` and `> module.cue:35:8`, the same
  three lines `module vet` prints. The typed resolution error has no part in it: the refusal is
  a CUE error the kernel wraps with `%w`, and the shared printer reads its path.
- Sites 5 and 6 would each need a new library or CUE surface (report T2.7, question 2).

SemVer: PATCH after GA (a refactor plus four corrected hints). Beta ships it as the next
`-beta.N`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `errors-domain`: adds the rule that the cli decides fetch and resolution answers by error type
  and never by message text, with the named exceptions, and the hint each platform module build
  failure gets.

## Impact

- `internal/publish/compat.go`, `internal/config/platform.go`, `internal/config/loader.go`, their
  tests, and one new guard test.
- Commands: `opm module publish` and `opm catalog publish` (the compat walk), every command that
  builds a platform module (`--platform <dir>`, `opm platform check`, `opm platform pull`), and
  config loading.
- No new dependency, no flag, no exit code change, no library change.
