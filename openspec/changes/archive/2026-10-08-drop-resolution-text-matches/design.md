## Context

See `proposal.md` for the reason and the full list of sites. The facts the approach rests on:

- The cli builds on library `v1.0.0-beta.7` (`go.mod:14`), which carries `*ResolutionError` and
  its three kinds (`opm/errors/resolution.go`). `liberrors.Classify` returns an error unchanged
  when the chain already holds a `*FetchError` or a `*ResolutionError`, so the cli MAY call it on
  an error the kernel already classified.
- `Classify` matches the fetch forms before the resolution forms, so a registry failure anywhere
  in the text never reads as an author defect. That is the guard `unprovidedImport` writes by
  hand today (`!errors.As(..., &fe) && strings.Contains(...)`).
- A `*ResolutionError` keeps the cause's message and unwraps to it. No printed message moves.
- The library change that added the type already proved sites 1 and 2 in a scratch clone of the
  cli: all 18 rows of `TestLoadPublishedPackage_Pinned` and all five rows of
  `TestPlatformBuildHint_Pinned` kept their answers (library
  `openspec/changes/archive/2026-10-05-type-author-resolution-errors/design.md`, "cli
  replacement proof (4.3)"). It also named the first two hint rows that move.

Command syntax, flags and exit codes do not change. The exit codes in play stay 2 (validation:
a platform module or a config file that does not build) and 3 (connectivity: the compat walk met
a registry failure).

## Goals / Non-Goals

**Goals:**

- No non-test cli code decides a fetch or resolution answer from message text.
- Every exit code and every message outside the three named hint rows is pinned by a test that
  passes before and after.
- A guard keeps the next text match out.

**Non-Goals:**

- Typing the forms the library leaves unclassified on purpose (a package-name mismatch,
  `no files in package directory`, an import cycle). That is a library change.
- A typed "module is not tidy" or "incomplete value". Each needs a surface CUE or the library
  does not have (sites 5 and 6 of the proposal).
- The token-endpoint gap test of cli#339. It moves with the library bump.
- New hints. An unauthorized registry keeps the pin hint it has today.

## Decisions

### D1. `unprovidedImport` reads the kind

```go
// unprovidedImport reports a load failure the library types as an import no
// module of the loaded build provides.
func unprovidedImport(err error) bool {
	var re *liberrors.ResolutionError
	return errors.As(liberrors.Classify(err), &re) && re.Kind == liberrors.ResolutionImportUnprovided
}
```

`loadPublishedPackage` keeps its switch: `cuemod.IsFetchNotFound` first, then
`unprovidedImport`, then the `*ConnectivityError`. `strings` stays imported only if another use
in the file needs it.

### D2. The pin hint reads "a fetch or a resolution failure"

```go
func platformBuildHint(dir string, err error) string {
	classified := liberrors.Classify(err)
	var fe *liberrors.FetchError
	var re *liberrors.ResolutionError
	switch {
	case errors.Is(err, liberrors.ErrWrongKind), errors.Is(err, liberrors.ErrInvalidPackage):
		// shape hint, unchanged
	case errors.As(classified, &fe), errors.As(classified, &re):
		// "Pin a published build in <modFile>, then try again"
	case cueErrorUnder(err, "#registry"):
		// key-and-import hint, unchanged text
	default:
		// default hint, unchanged text
	}
}
```

The order of the cases stays as it is today.

### D3. `#registry` and the config fields are read from the CUE error path

One helper in `internal/config`:

```go
// cueErrorUnder reports whether the chain of err holds a CUE error whose
// path starts with the selectors of prefix.
func cueErrorUnder(err error, prefix ...string) bool
```

`removedFieldHint` takes the error, not its text, and asks for a CUE error under
`#CLIConfig.config.providers`, then `.cacheDir`, then `.skewPolicy`, in today's order.

### D4. The guard test

`TestNoErrorTextMatch` (`pkg/errors/textmatch_refusal_test.go`) parses every non-test `.go` file
under `cmd/`, `internal/` and `pkg/` with `go/parser`. It reports a `strings` predicate
(`Contains`, `HasPrefix`, `HasSuffix`, `Index`, `EqualFold`, `Cut`, `Split`, `Fields` and their
variants) or a `regexp` match method over an error's text, a comparison of an `Error()` call, and
a switch over one. An error's text is an `Error()` call, a CUE error's `Msg()`, or a variable
assigned from either or from an expression over such a variable, inside one function. The four
sites that stay are an allowlist of `file: function` pairs, each with its reason, and an entry
that is no longer in the code fails the test too. The pattern follows `TestNoLocalHealthEvaluator`
(`internal/kubernetes/health_refusal_test.go`). The text may be wrapped in other calls
(`strings.ToLower(err.Error())`). Two limits remain, because the guard reads syntax and no types:
a text made by formatting the error (`fmt.Sprint(err)`), and a text handed to another function as
a string. `TestErrorTextMatches_Detects` holds both as rows.

Justification under Principle VII: without it the rule is prose, and the search for this change
found four text matches where the swarm's notes named two.

## Research & Decisions

### The registry key mismatch carries a typed CUE path

**Context**: The owner's ruling asks the `#registry` match to read the CUE error path. That only
works if the kernel hands the CUE error on with `%w`.
**Explored**: A temporary test in the worktree (removed, never committed) built the repo's
platform module with one entry keyed at `opmodel.dev/core@v2` and printed the chain:
`cueerrors.Errors(err)` returned one `*cue.valueError` with path
`["#registry" "\"opmodel.dev/core@v2\"" "#catalog" "metadata" "modulePath"]`;
`errors.Is(err, liberrors.ErrMissingRequiredField)` was false.
**Options considered**:
1. Read the first path selector. Typed, and narrower than the word match.
2. Ask the library for a typed "registry key mismatch". A library change for one hint.
**Decision**: Option 1.
**Rationale**: The path is data CUE gives; no library work is needed.

### The shape check flattens its CUE cause

**Context**: The library's shape check refuses an incomplete `#registry` entry with
`fmt.Errorf("required field %q entry %q ...: %v: %w", ..., err, ErrMissingRequiredField)`
(library `opm/internal/loader/shape.go:159-170` at beta.7). The CUE cause goes through `%v`, so
no CUE path is left in the chain; `#registry` is only in the text.
**Options considered**:
1. Let that form take the default hint.
2. Keep a text match for it, recorded as an exception.
3. Add a typed field path to the library's missing-field error first.
**Decision**: Option 1, as the recommended answer to report question 1.
**Rationale**: The key-and-import hint does not describe an incomplete entry, so the word match
gave it a wrong hint. Option 3 stays open as a library follow-up; it is additive.
**Measured (section 1)**: a platform file reaches this branch. A second entry that embeds no
catalog (`"example.com/x@v1": {}`) is refused with `required field "#registry" entry
"example.com/x@v1" is incomplete at "version" ...`, `errors.Is(err, ErrMissingRequiredField)` is
true, and the chain holds no CUE path. `TestBuildPlatformModule_EntryWithoutCatalogIsRefusedByShape`
pins it. An entry with a wrong `enable` type fails in the schema's own evaluation instead, at a
path under `#registry`, and keeps the key-and-import hint.

### The two hint rows the library change named

**Context**: `cannot find package` is cue/load's prefix for every failed import, so it also
matched forms that are not resolution failures; and the direct-path form of a module file that
does not parse carries neither matched text.
**Options considered**:
1. Accept both moves and pin them.
2. Keep `cannot find package` as an exception so the package-name row keeps the pin hint. The
   direct-path row still moves, because one kind covers both paths.
3. Wait for a library change that types the content-defect forms.
**Decision**: Option 1 (report question 1).
**Rationale**: A package-name mismatch is not cured by another pin, and a module file that does
not parse is, whichever path met it. Both moves make the hint truer.

### Config field hints

**Context**: `removedFieldHint` matches a field name anywhere in the message.
**Decision**: Read the path, as D3.
**Measured (section 1)**: CUE reports all three at the path `#CLIConfig.config.<field>`
(`TestConfigHint_FailingFieldIsOnTheCUEPath`). The path carries the schema definition and the
`config` struct first, so `cueErrorUnder` takes a path prefix, not one selector:
`cueErrorUnder(err, "#CLIConfig", "config", "providers")` and `cueErrorUnder(err, "#registry")`.
No row keeps a text match.

### The unset required value at `module build` (T9.20)

**Explored**: A cli built from `5aa8dc88`, run on a scratch copy of `templates/minimal`:
- `note: string` in `#config`, `opm module build .`: exit 2,
  `render failed: 1 issue`, `incomplete value string`, `values.note`, `> module.cue:35:8`;
- the same with `-f <values file>`: the same three lines, exit 2;
- `note!: string`: `field is required but not present`, `values.note`, `> module.cue:35:2`,
  exit 2;
- `opm module vet .` on the first form: the same three lines under
  `values do not satisfy #config: 1 issue`, exit 2.
**Decision**: Nothing to fix and nothing added to this change.
**Rationale**: The path and the position are printed. The typed resolution error is not on this
path.

## Error handling

| Failure | Typed signal | Answer | Exit |
| --- | --- | --- | --- |
| Predecessor load: registry does not hold it | `*FetchError`, `FetchNotFound` | probe, then absent or connectivity | 0 or 3 |
| Predecessor load: import no module provides | `*ResolutionError`, `ResolutionImportUnprovided` | absent | walk goes on |
| Predecessor load: anything else | none of the above | `*ConnectivityError` | 3 |
| Platform build: wrong shape | `ErrWrongKind`, `ErrInvalidPackage` | shape hint | 2 |
| Platform build: fetch or resolution | `*FetchError` or `*ResolutionError` | pin hint | 2 |
| Platform build: CUE error under `#registry` | CUE path | key-and-import hint | 2 |
| Platform build: other | none | default hint | 2 |
| Config: CUE error under a known field | CUE path | that field's hint | unchanged |

Example, unchanged by this change:

```text
Error: platform module error
  ... #registry."opmodel.dev/core@v2".#catalog.metadata.modulePath: conflicting values ...
  Hint: Each #registry entry's key must equal the module path of the catalog it imports (#catalog); fix the entry named above in <dir>/platform.cue
```

## Risks / Trade-offs

- [A CUE bump rewords a form] → The library pins every form `Classify` reads against the
  embedded CUE (`classify_cue_test.go`); the cli's own rows (`TestLoadPublishedPackage_Pinned`,
  `TestPlatformBuildHint_Pinned`) fail on a changed answer.
- [A failure the text caught and the types miss gets the default hint] → Section 1 pins every
  form found before the code moves; only the three named rows may flip.
- [The guard test reports a false positive] → It is narrow on purpose (an `.Error()` value into
  a string predicate), and the allowlist takes a reason per entry.

## Open Questions

None that change the tasks. Report T2.7 carries the two owner questions; the tasks follow the
recommended answers.
