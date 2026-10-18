## Why

The org meta repo `.github` moves its README detail into `references/*.md`. Two normative lines of `release-cascade-wiring` cite the README for content that moves, so a reader following them would find nothing.

## What Changes

- The receiver `if:` is cited from `references/cascade-workflows.md` in `.github`.
- The cli's wiring-check values are cited from the table in `references/cascade-wiring-check.md` in `.github`.
- `Source:` lines that cite "README at <sha>" stay: git history resolves them.

## Capabilities

### Modified Capabilities

- `release-cascade-wiring`: two citations change. No behavior changes.

## Impact

Spec text only. No code, workflow or task changes.
