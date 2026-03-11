## Context

CUE validation errors during `opm rel vet` and `opm mod vet` were losing their structured grouped format and getting flattened into noisy logs because the formatting layer was disconnected from the validation layer's raw output. While the underlying fix is to use the `cmdutil.PrintValidationError` helper, we lacked test coverage to catch this regression.

## Goals / Non-Goals

**Goals:**
- Add E2E tests covering the CLI output of the validation commands to prevent future regressions.
- Ensure tests are isolated from changing examples.

**Non-Goals:**
- Modifying the underlying validation logic or schema structure.

## Decisions

- **Isolated Fixtures**: Use a dedicated minimal CUE fixture at `tests/e2e/testdata/vet-errors/` rather than modifying or relying on the full `jellyfin` example. This avoids test breakage when examples are updated.
- **Substring Matching**: Assert on stable string fragments in `stderr` rather than doing exact golden-file matching to avoid fragility around timestamps or minor formatting tweaks.
- **Test Both Entrypoints**: Add tests for both `opm rel vet` and `opm mod vet` since they exercise slightly different paths to the shared validation printing logic.

## Risks / Trade-offs

- [Risk] Substring matching in tests might be fragile to minor wording changes. → Mitigation: Assert on very stable fragments like `field not allowed` and `conflicting values` rather than full lines.
