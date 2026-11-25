# Design

## Context

See `proposal.md` for the motivation. The facts below were checked in the explore (evidence in
`evidence/01-open-version-conformance/`).

- `conformIdentity` (`internal/publish/identity.go`) unifies the identity with `#IdentityPackage`,
  validates with `cue.Concrete(true)`, and drops each error whose text contains `incomplete value`.
  Core's `VersionMajor: "v" + strings.SplitN(Version, ".", 2)[0]` gives `error in call to
  strings.SplitN: non-concrete value string` on an open `Version`, which the filter keeps.
- Plain `Validate()` (not concrete) passes an open and an absent `Version`, and still refuses a
  major skew and a malformed value. Only concrete mode reports a missing required field.
- `identityStates` already classifies each field of `identityFieldNames` as absent, open or
  concrete. `resolveVersion` returns early on an absent `Version` and relies on the conformance
  refusal to report it.
- `resolveVersion` records a `--version` fill on the plan (`FillVersion`); Push writes it into the
  working tree. Today the conformance check runs before `resolveVersion` and sees the open value.
- The kernel-load gate skips when a field is not concrete. The library loader requires a concrete
  `metadata.version` (`opm/internal/loader/shape.go`), so vet's render fails on a wired open
  `Version`.
- The publish test stub (`internal/publish/fixtures_test.go`) leaves out the `VersionMajor`
  derivation on purpose, so no unit test sees the real failure.
- `TestModVet_ValidModule` builds `GlobalConfig{ConfigPath}` without `config.Load`, so
  `cfg.Registry` is empty and vet raises `NoRegistryError`, which the test skips on.
  `setupTestConfig` writes a `providers` block that the config no longer reads.

## Goals / Non-Goals

**Goals:**

- One rule for an open `Version` in vet and publish: it passes the identity check.
- No error text read in the identity check.
- Tests that fail offline when the stub and the code disagree with core's derivation.

**Non-Goals:**

- The kernel-load gate on a filled `Version` (it stays skipped while a field is open in the tree).
- Vet output glyphs (the Unicode check mark), tracked apart.
- A catalog vet command.

## Decisions

### Research & Decisions

#### Is an open identity `Version` valid

**Context**: The issue asked whether the schema or the godoc claim is wrong.
**Explored**: The publishing spec, the 0011 design archive, core beta.4, and a CUE prototype.
**Options considered**:
1. Keep it valid, fix the cli: matches the spec and enhancement 0011; core needs no change.
2. Retire it: one fewer state, but removes a documented publish flow.
**Decision**: Option 1 (owner).
**Rationale**: The spec scenario "Fill then publish" and enhancement 0011's design both state it.
Core already treats the open value as incomplete; only the cli's text filter is wrong.

#### How the identity check tells open from wrong

**Context**: The check must refuse a conflict and an absent field, and pass an open one.
**Explored**: The prototype in `evidence/01-open-version-conformance/`.
**Options considered**:
1. Plain validation, plus a presence check from the field states, plus the publish fill before
   the check. Removes a text match.
2. Widen the text filter to `non-concrete value`: adds a text match against the repo rule.
3. Fill before the check only: vet keeps refusing an open `Version`.
**Decision**: Option 1 (owner).
**Rationale**: The repo rule "read no error text" in `AGENTS.md`; the prototype shows that plain
validation separates the cases.

Sketch:

```go
// conformIdentity checks the identity against #IdentityPackage. fill, when
// non-empty, is the --version value for an open Version.
func conformIdentity(a *artifact, schema cue.Value, fill string) *Refusal {
	id := a.identity
	if fill != "" {
		id = id.FillPath(cue.ParsePath("Version"), fill)
	}
	unified := schema.Unify(id)
	if err := unified.Validate(); err != nil { // conflicts only; open passes
		return &Refusal{Headline: "the identity package does not conform to core's #IdentityPackage", Err: err}
	}
	// An absent schema-fixed field is a malformed identity.
	if missing := absentIdentityFields(id); len(missing) > 0 {
		return &Refusal{...} // names each missing field and its file
	}
	return nil
}
```

The gate order in `Run` changes so that the fill value is known before the check: the version
resolution reads the field states, which do not depend on the conformance result. The concrete
pass after the fill is the same `Validate()` call: once every field is concrete, plain validation
reports every derivation conflict. The absent-field refusal MUST name the field and
`identity/identity.cue`, as CUE's "field is required but not present" did.

#### What vet does with an open `Version`

**Context**: After the fix, the identity and `#config` checks pass on an open `Version`, and the
render fails in the kernel loader.
**Options considered**:
1. Skip the render, say why, exit 0.
2. Refuse with an action, exit 2: two rules for one state.
3. Leave the loader error: it names `metadata.version`, not the identity file.
**Decision**: Option 1 (owner).
**Rationale**: One rule for the open state, and output that says what was not checked
(`CONSTITUTION.md` principle IV).

`runVetModuleOnly` reads the `Version` field state from the plan after the values check. On
`StateOpen` it logs one line and returns nil:

```text
Render skipped: identity Version is open. Fill it with opm module version set <version>, or let opm module publish --version fill it.
```

### Error handling

| Case | Command | Exit | Output |
| --- | --- | --- | --- |
| Open `Version`, no flag | vet | 0 | checks pass, render-skip line |
| Open `Version`, no flag | publish | 2 | the existing open-field refusal |
| Open `Version`, `--version` with matching major | publish `--dry-run` | 0 | plan, GO |
| Open `Version`, `--version` with another major | publish | 2 | major refusal |
| Absent `Version` | vet, publish | 2 | conformance refusal naming the field |
| Malformed or skewed concrete `Version` | vet, publish | 2 | CUE's diagnostic |

## Slices

### S1: Identity check without text reads

- **Delivers**: the publish requirement "The identity check treats an open field as fillable and
  checks the identity as filled"; the vet scenarios for an absent, malformed or skewed `Version`;
  the errors-domain list of three text reads.
- **Seam**: `internal/publish`, `VetChecks` and `Run` with a stub `#IdentityPackage` that carries
  core's `VersionMajor` derivation. Offline cases: open, absent, skew, malformed, and
  fill-then-check. One registry-backed test runs the open and fill cases against the real core
  schema and skips only when the registry is unreachable.
- **Depends on**: none.
- **Done when**: the offline cases pass with the derivation in the stub; the text-match allowlist
  in `pkg/errors` no longer lists `conformIdentity`; the `VetChecks` godoc and the stub comment
  state the new rule.

### S2: Vet skips the render on an open Version

- **Delivers**: the vet requirement "An open identity `Version` passes vet's identity checks and
  skips the render"; the fix for `TestModVet_ValidModule`.
- **Seam**: `internal/cmd/module`, `NewModuleVetCmd` run with `cfg.Registry =
  config.DefaultRegistry`. Skip only on the connectivity exit code; `NoRegistryError` fails. A new
  fixture with an open, wired `Version` asserts the render-skip line, exit 0 and no `Module valid`
  line.
- **Depends on**: S1.
- **Done when**: `TestModVet_ValidModule` runs (not skips) with a reachable registry; the open
  fixture test passes; `setupTestConfig` writes no `providers` block.

## Risks / Trade-offs

- [The stub drifts from core again] → the registry-backed test runs the same cases against the
  real schema wherever a registry is reachable.
- [Core adds a required identity field] → plain validation does not report its absence. Mitigation:
  `identityFieldNames` is the one list to extend; the real-schema test fails on a fixture that
  lacks a new field only if the fixture is updated. Accepted.
- [A filled publish skips the kernel-load gate] → unchanged behaviour, outside this change.

## Assumptions

- The vet render-skip line text above is a proposal; the build may reword it within the spec.
- Release class PATCH; no enhancement task line is delivered.
- The new fixture lives under `tests/fixtures/valid/`, beside `simple-module`
  (`references/fixtures.md` holds the fixture rules).
