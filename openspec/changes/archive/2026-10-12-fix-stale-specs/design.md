## Context

The code is the source of truth for this change. Each statement below was read from Go and
checked against a test or a run. Paths are relative to the cli repo at the base of this branch.

## Decisions

- **Repair version.** `planIdentity` creates `identity/identity.cue` only when the file is
  absent (`internal/scaffold/repair.go:193-228`). It gets the version from `statedVersion`
  (`repair.go:237-264`), which loads the tree and reads `metadata.version`. It refuses when the
  tree does not load, states no version, or states a non-SemVer value. `InitialVersion = "0.1.0"`
  is used by `Reidentify` on a fresh scaffold only (`internal/scaffold/scaffold.go:25,202`).
  Tests: `repair_test.go` "missing identity adopts the version the acquired module states" and
  "missing identity refuses when the module states no version". When the identity file exists,
  repair leaves its `Version` alone (`repair.go:196-208` plans a `ModulePath` edit only). Init repairs when the directory holds a module and no template is named (`internal/cmd/module/init.go:118-125`).
  The scenario `Scaffold identity is a literal` keeps its name and its literal-form promise.
  Its WHEN now covers a scaffold and a repair that creates the identity file; a repair that
  finds the file leaves its `Version` form alone, so the scenario does not claim it. Two new
  scenarios, `Repair adopts the version the tree states` and `Repair refuses a tree that
  states no version`, name the repair cases.
- **Default major.** `ParseTemplateRef` sets `Major` from the baked table only for a bare word
  that names an official template and has no suffix (`internal/scaffold/ref.go:116-121`). A
  literal path takes only its own suffix (`ref.go:122-123`). With no major, `LookupPath` is the
  bare base and resolution takes the newest stable version of that path (`scaffold.go:62-107`).
  Tests: `ref_test.go` cases "bare unknown word expands unconstrained", "literal path without a
  suffix floats unconstrained". A new scenario `Only official templates get a default major`
  names this. The `template list` requirement is unchanged: it already lists a default major per
  official template.
- **Reserved name `index`.** The namespace gate admits `opmodel.dev/templates/<name>` for any
  `<name>` that matches `[a-z0-9._-]+` (`internal/publish/gates.go:23,260-282`). The kind gate
  admits the `templates` segment as module kind (`gates.go:289-310`). No other code names
  `index` (grep over `internal/`, `cmd/`, `pkg/` finds none). A throwaway test with
  `RegistryRepo: "opmodel.dev/templates/index"` produced zero refusals; it was deleted. The
  spec drops the reservation and says that no gate reserves a name. A new scenario
  `No name in the segment is reserved` records it. The official templates stay `minimal`,
  `standard`, `advanced`.
- **MODIFIED, not REMOVED plus ADDED.** All three requirements keep their names and all live
  scenario names, so OpenSpec accepts the deltas.
- **Wording.** The one citation in the modified text (`Source: enhancement 0011 (...)`) is
  kept as it is, with its reason in words.

## Risks

- A reader may want `index` reserved. The spec then records a gap rather than hiding it. A
  later change can add the gate and bring the reservation back.
- The published site page `docs/site/reference/registry-namespaces.md` still says that `index`
  is reserved. It is outside this change; the report asks the owner.
