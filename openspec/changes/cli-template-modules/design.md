# Design — cli-template-modules

## Overview

```
 cli repo                          registry                      user machine
──────────────────────────────────────────────────────────────────────────────
 templates/standard/   release CI  opmodel.dev/templates/       opm mod init
 templates/minimal/    ──publish──▶  standard@v1 vX.Y.Z          my.corp/m/app@v0 standard
  real modules: vetted,  (own opm,    minimal@v1  vX.Y.Z              │
  gated, deps updated    caller-side                                  ▼
  by normal tooling)     filter)                          1 expand shortcut
                                                          2 resolve version (stable-float)
                                                          3 AcquireModuleFromRegistry
                                                          4 copy staged tree
                                                          5 RE-IDENTIFY:
                                                              cue.mod module: line
                                                              identity ModulePath
                                                              identity Version → *"0.1.0"
                                                              self-imports → new path
                                                              package clauses → new leaf
                                                          6 metadata: derives (D12) — untouched
```

## Research & Decisions

### Fetch and copy: the kernel's acquire, staged source

**Decision**: `Kernel.AcquireModuleFromRegistry(templatePath, version)` — it already returns the module with its staged source tree (`module.Source{Root, Overlay}`), runs the identity verification (a tampered template artifact refuses at fetch), and shares the CUE disk cache. The copy takes the staged tree minus registry-manager artifacts; the template's `cue.mod` deps come along verbatim (they are the template's tested pins).
**Rationale**: one fetch path for the whole CLI; the acquire-time identity check means the shortcut namespace's trust story is enforced, not assumed.

### Version selection: stable-float via `compat.HighestStable`

**Decision**: shortcut and default-template resolution pick the newest *release* within the major — `compat.HighestStable` over `ModuleVersions` — falling back to the highest prerelease on a prerelease-only history. An exact semver in the ref pins the tag.
**Rationale**: a scaffold wants the newest released starting point; stale-pin maintenance is the disease this change removes. This is the float selector's one correct home — the predecessor-selection misuse is being struck from D9 in the same enhancements batch, and this caller is what settles the function's disposition (re-document, keep).

### Shortcut expansion: syntactic, airtight, never over paths

**Decision**: template refs are disambiguated by shape — bare word (`^[a-z0-9_]+$` before an optional `@…`) expands to `opmodel.dev/templates/<word>`; any ref containing `/` or `.` is a literal module path, never expanded. `@vN` floats within the major; `@X.Y.Z` is exact. A bare word can never be a valid module path (paths require a dotted first segment), so the expansion is collision-free by grammar. Typos 404 inside the reserved segment with a refusal naming the expansion — never a silent fallback.
**The trust precondition is the DN**: expansion into an *unreserved* namespace would be a typosquat surface; the enhancements-batch decision reserving the segment (module-kind, cli-CI-published-only, `index` name reserved) is a hard prerequisite, and the gate tables here land citing it.

### Re-identification: the writers, applied wholesale

**Decision**: the rewrite set is exactly D16's three statements plus the package clause: `SetCueModModule` (from `cli-authoring-commands`), the identity rewrite (`ModulePath` set to the user's path; `Version` reset to the defaulted `#VersionType | *"0.1.0"` form, keeping the release-automation marker line shape), a new self-import rewriter (every import of the template's own path → the user's path — in a fresh clone this is typically just the `import id ".../identity"` line), and a new package-clause renamer (template leaf → user leaf, all files). Post-rewrite assertion: the tree parses and `metadata` still derives — then init prints the file tree and a pointer to `opm module vet`.
**Rationale**: repair mode *refuses* to rewrite self-imports because the user owns the tree; init-from-template is the one context where the wholesale rewrite is correct because they don't yet. Same writers, opposite consent situation — which is why both live in one command.

### Repair mode relocates here unchanged

The prior `cli-authoring-commands` draft's repair design (detector via the pipeline's checks, second confirmation with file-tree + aligned current→replacement pairs, `--yes`, non-TTY refusal, never-invent, stranding refusal) moves into this change verbatim so `init` is rewritten once, coherently. `cli-authoring-commands` slims to the writers and `version set` — recorded in both changes.

### `template list`: baked, release-coupled

**Decision**: one constant table (name, description, default major) drives both shortcut expansion and `opm module template list`. No published index artifact, no registry listing — the binary that knows the table belongs to the release train that published the templates, so drift cannot occur between a binary and *its* template set. The `template` subgroup leaves room for future verbs; registry-resolved current versions are recorded as deferred.

### CI publish job: the caller-side filter

**Decision**: the release workflow builds `opm`, then per template: read the declared version (the same grep-free route — `cue eval` the identity package, or the CLI's own plan output), check tag existence via the registry, and invoke `opm module publish ./templates/<name>` only for unpublished versions. Publish itself never skips (D15: already-published is always a refusal); idempotency is the caller's filter, exactly as D15 assigns it. A gate refusal fails the release — that is the feature.

### Offline stance

No embedded fallback. Init without a reachable registry refuses with the expansion named and the registry it tried; after any successful fetch, CUE's module cache serves repeats offline. First-run-without-config uses the built-in default registry mapping. Stated in the spec so it is a decision, not an accident.

## Technical Notes

- Gate tables: `internal/publish`'s namespace/kind gates gain `templates/` as module-kind under owned domains; the 0011 `target.cue` pins move with the DN in the enhancements batch; a passing pin for `opmodel.dev/templates/standard@v1` and a failing one for a nested `templates/x/y` land there.
- Grammar precedence in the two-positional form: first positional = new module path (has `/` or `.`), second = template ref; a single bare-word positional = template ref → prompt for path.
- Hermetic e2e: publish `standard` into the in-process registry, `mod init` from it (shortcut expansion pointed at the test registry via the mapping), vet + dry-run the result; assert the template's own identity never leaks into the scaffold.
- Deletions: `internal/templates/` (embed.go, all `.tmpl`), init's old template rendering, `embed_test.go`; the root `deps:update:templates` task retargets to `cli/templates/*/cue.mod/module.cue` (real files now — simpler than `.tmpl` surgery).
- `advanced` variant: carried as the third template module — the showcase tree (multiple components, trait attachments, values plumbing), which doubles as the strongest re-identification test subject (most files, most package clauses, most self-import exposure).
