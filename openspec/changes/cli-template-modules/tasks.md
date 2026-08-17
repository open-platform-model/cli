# Tasks — cli-template-modules

> Prerequisites: `cli-authoring-commands` merged (writers); the enhancements batch merged (templates-segment DN + gate pins — task 3.1 cites it). Groups land independently green. Bare-`@` ban on every commit.

## 1. Template module trees

- [x] 1.1 `cli/templates/standard/` — real module at `opmodel.dev/templates/standard@v1`: identity subpackage (defaulted `1.0.0`-line version + marker), D12 wiring, D49 catalog imports, `source: {kind: "self"}`, snake package == leaf.
- [x] 1.2 `cli/templates/minimal/` — same shape, smallest useful module (replaces `simple`).
- [x] 1.3 `cli/templates/advanced/` — the showcase tree: multiple components, trait attachments, values plumbing.
- [x] 1.4 All three pass `opm module vet` and `publish --dry-run` locally against GHCR; root `deps:update:templates` retargeted to the real `cue.mod` files and run once.

## 2. Writers (cueedit additions)

- [x] 2.1 Self-import rewriter: every import of path P → path Q across the tree; splice-style; tests incl. the `import id` line and a multi-file case.
- [x] 2.2 Package-clause renamer: old leaf → new leaf, all `.cue` files; tests preserve following comments/whitespace.
- [x] 2.3 Identity `ModulePath` rewrite + `Version` reset to the defaulted `*"0.1.0"` form (extends the existing writer set); tests.

## 3. Publish gates learn the segment

- [x] 3.1 Namespace/kind tables gain `templates/<name>` as module-kind under owned domains (cites the 0011 DN); refusals for nested paths under the segment; passing + failing gate tests mirroring the DN's target.cue pins.

## 4. init rewrite

- [x] 4.1 Shortcut expansion + `--from` + `-t` alias: the grammar table as a pure function with exhaustive tests (bare word, majors, exact semver, literal paths never expanded).
- [x] 4.2 Scaffold path: resolve (stable-float via `compat.HighestStable`), acquire, copy staged tree, re-identify via the writer set, post-rewrite parse+derivation assertion, file-tree output + vet pointer.
- [x] 4.3 Interactive form: template-only argument prompts for the module path; non-TTY without a path refuses; offline refusal names the expansion and registry.
- [x] 4.4 Repair mode relocated from the authoring draft: detector, second confirmation (file tree + aligned pairs), `--yes`, never-invent, stranding refusal; stdin-driven tests.
- [x] 4.5 Delete `internal/templates/` + embed machinery + old init rendering/tests.

## 5. template list

- [x] 5.1 Baked table (drives expansion + listing); `opm module template list` under a new `template` subgroup; tests assert table and expansion share one source.

## 6. CI publish job

- [x] 6.1 Release workflow: build `opm`, per template resolve declared version, caller-side tag-existence filter, publish unpublished ones; a gate refusal fails the release. Dry-run path exercised in PR CI (no push).

## 7. E2E, specs, gates

- [ ] 7.1 Hermetic e2e: publish `standard` into the in-process registry, init from it via shortcut, scaffold passes vet + dry-run GO; template identity never leaks into the scaffold. Revive `TestE2E_ModInit_ThenVet` in this form.
- [ ] 7.2 Spec deltas: new `template-modules`; `core`/`core/project-structure` variant-enumeration removal.
- [ ] 7.3 `CLAUDE.md` command map + layout; `task fmt vet lint test` green.

## 8. Record

- [ ] 8.1 `enhancements/0011/`: history event — the `cli-authoring-commands` slice lands as two changes (`openspec_ref` cites both); `HighestStable` disposition resolved (float selector; first caller is template resolution — update the cli-catalog-gates task note); templates DN implemented by tasks 1/3/6.
