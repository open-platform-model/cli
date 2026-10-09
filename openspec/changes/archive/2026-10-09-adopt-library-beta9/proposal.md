## Why

Library v1.0.0-beta.9 is released (library#228). It is a breaking release for Go callers: it removes five names that beta.8 kept as deprecated or unused. The cli must pin a library release without those names before v1.0.0, and the pin bump must be a reviewed change that proves the cli uses none of them.

## What Changes

- Bump `github.com/open-platform-model/library` from v1.0.0-beta.8 to v1.0.0-beta.9. Only `go.mod` (one line) and `go.sum` (two lines) change. The library's own `go.mod` hash is the same in both releases, so no transitive dependency moves.
- No production code, test or documentation edit. The cli left every removed name before this change:

  | Removed in beta.9 | Last cli use | Removed from the cli by |
  | --- | --- | --- |
  | `ownership.ApplyInput.Admit` | `opm operator install` set it for objects of a manifest install | cli#357 (3c3dbe03) |
  | `ownership.DeleteInput.Admit` | the same install, for the superseded role bindings and Deployment | cli#357 (3c3dbe03) |
  | `errors.IdentityError.As` (the value target) | never used; the one `*IdentityError` pointer target went with `opm instance handoff` | cli#196 (7ae153f7) |
  | the value receiver of `errors.IdentityError.Error` | never used | not applicable |
  | `catalog.Source` | never used; no Go file in the history of the cli holds the name | not applicable |

- No verdict of `ownership.CanApply` or `ownership.CanDelete` changes for a caller that sets no `Admit`. The cli sets none, so no apply, prune or delete decision changes.
- No change to commands, flags, exit codes or output.

SemVer class: PATCH after GA (no behaviour a user sees changes). During beta it ships as the next `-beta.N`. The commit type is `fix(deps)` with no `!`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. No spec-level behaviour changes: the requirement "Ownership guard judges every apply and dry run" of `apply-pruning` already says that the guard takes no admission set. `.openspec.yaml` sets `skip_specs: true`.

## Impact

- `go.mod`, `go.sum`: the library pin.
- Users: none.
- This change replaces the bump pull request that the release cascade opens for the same release.
