## Context

`hack/fixtures.sh` publishes the repo's module fixtures (`tests/fixtures/modules/*`) through
`opm module publish`, and `tests/fixtures/fixtures.go` reads their coordinates. Both are
byte-identical with opm-operator. opm-operator now needs a catalog fixture
(`testing.opmodel.dev/catalogs/operator/provider@v0`, its change `publish-test-catalog`, which holds
the full design and the verification table).

## Goals / Non-Goals

**Goals:** the shared flow publishes a catalog fixture with every catalog publish gate; behavior for
a repo without a catalog root is unchanged.

**Non-Goals:** a catalog fixture in the cli; a trigger on `tests/fixtures/catalogs/**` in this repo's
`publish-fixtures.yml` (add it with the first catalog fixture).

## Decisions

### D1. Kind follows the root

```bash
CATALOGS_DIR=${CATALOGS_DIR:-}   # default: $(dirname "$FIXTURES_DIR")/catalogs, only when it exists
fixture_dirs                     # catalogs first, then modules
kind_of <dir>                    # "catalog" under CATALOGS_DIR, else "module"
"$OPM_BIN" "$kind" publish [--dry-run] "$dir"
"$OPM_BIN" "$kind" version set "<ver>-<prerelease>" "$stagedcopy"
```

Output gains the kind: `==> podinfo (module): gates at v0.1.11`. The bump hint names
`opm <kind> version set`. Exit codes and the `already holds` idempotency rule are unchanged: both
commands share one publish pipeline and print the same refusal.

### D2. Go helper mirrors the root

`CatalogDir()` is `filepath.Join(filepath.Dir(Dir()), "catalogs")`; `LoadCatalog(name)` and `Load(name)`
share one unexported `loadIdentity`. `MustCatalog` is `LoadCatalog` for tests.

## Research & Decisions

### Where the catalog flow lives

**Context**: the operator needs to publish one catalog fixture with the gates and PR seeding its
modules already get.
**Explored**: `opm catalog publish` and `opm module publish` share `internal/publish` (same plan
output, same `already holds` refusal, same exit codes); `opm catalog version set` exists; the
namespace and kind-segment gates do not bind on `testing.opmodel.dev`.
**Options considered**:
1. Extend the shared script with a catalog root - one flow, one copy to keep identical; small diff.
2. An operator-local publish path - no cli change, but a second copy of the gates, seeding and
   changed-implies-bumped logic.
**Decision**: option 1.
**Rationale**: the workspace keeps one fixture flow on purpose (`.tasks/fixtures.yml`: diverging
copies get promoted, not forked).

### No regression here

**Context**: the cli must behave as before.
**Explored**: in this worktree with the `.opm-cli-version` release (`v1.0.0-beta.4`) and a throwaway
`registry:2` on `localhost:5055` (removed afterwards): `pins` prints the podinfo coordinate only;
`check` against GHCR passes (`already published and unchanged`); `seed` into the throwaway registry
pushes podinfo; `consumers examples tests/e2e/testdata/operator-owned` under the mixed mapping
prints `ok` twice. `go test ./tests/fixtures/` passes with `TestLoadCatalogs` skipped (no catalog
root).
**Decision**: no further change.
**Rationale**: the only visible difference is `(module)` in progress lines.
