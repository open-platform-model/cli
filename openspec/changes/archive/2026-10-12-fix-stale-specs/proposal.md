# Proposal

## Why

Three statements in the `template-modules` spec no longer describe the code. Seeding the
enhancement store from the cli specs found them. A reader who trusts the spec expects behaviour
the cli does not have. The code is right and the spec follows it. No code changes.

## What Changes

- Repair: the scenario `Scaffold identity is a literal` says that `opm mod init` writes
  `Version: "0.1.0"` when it repairs a tree. Only a fresh scaffold writes `0.1.0`. When repair
  creates a missing identity package, it takes the version from the tree's own
  `metadata.version`, and refuses when the tree states none.
- Default major: the requirement `Shortcut expansion is syntactic and safe` says that no suffix
  selects the cli's default major. Only a bare word that names an official template gets one.
  An unknown bare word and a literal module path with no suffix float with no major constraint.
- Reserved name `index`: the requirement `Templates are published modules in a reserved segment`
  says that the name `index` is reserved in the `templates` segment. No code enforces it.
  The spec now states that no gate reserves any name in the segment.

Release class: none. The change edits specs only, so every commit is `docs:` or `docs(openspec):`.
After GA it would be PATCH; beta ships nothing for it.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `template-modules`: three MODIFIED requirements. Every live scenario name stays; four scenarios
  are new.

## Enhancement

None.

## Impact

- Spec: `openspec/specs/template-modules/spec.md` (by the archive sync).
- Code: none. Each new statement is traced to code in the design.
- Not changed here: `docs/site/reference/registry-namespaces.md` still says that `index` is
  reserved. It is published site text outside the scope of this change; the report names it.
