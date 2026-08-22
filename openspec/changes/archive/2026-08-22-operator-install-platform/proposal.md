# Proposal: operator-install-platform

## Why

`opm operator install` server-side-applies the embedded opm-operator manifest, waits for the CRDs to
reach `Established` and the Deployment to finish its rollout, and stops there
(`internal/cmd/operator/install.go:89-131`, `internal/operator/install.go:49-80`). The singleton
`Platform` CR, the object the operator's `PlatformReconciler` needs before it can materialize
anything, is never created. A cluster that just ran `opm operator install` has a running operator
with nothing to reconcile.

The Platform appears today only as a side effect of the first cluster-facing apply
(`internal/workflow/apply/apply.go:181-191`), seeded write-if-absent from `~/.opm/platform.cue`. That
file's catalog pin is a hand-bumped literal compiled into the binary
(`internal/config/templates.go:80-88`, `"opmodel.dev/catalogs/opm@v2": {version: "2.0.0-alpha.3"}`),
and its own comment concedes the limitation: `opm config init` is normatively offline, so it cannot
resolve "latest". The result is that a cluster's catalog subscription is decided by which CLI build
the operator happened to be installed from, months after that literal was last touched.

Install is the moment the CLI already has a cluster, a network, and the user's attention. It is where
the subscription should be resolved and written.

## What Changes

- **`opm operator install` creates the singleton `cluster` Platform** after the readiness wait,
  subscribing it to the newest **released** build of `opmodel.dev/catalogs/opm@v2` resolved from the
  registry. The write is the existing create-only, write-if-absent contract (0006 D22): plain create
  with field manager `opm-cli`, `AlreadyExists` is a success-noop, `Forbidden` degrades to a warning.
  An existing Platform is never rewritten.
- **New flag `--catalog-prerelease`** (default `false`): subscribe to the newest catalog prerelease
  instead of the newest release. Branch dev builds are never selectable under either mode.
- **New flag `--skip-platform`** (default `false`): install the operator and create no Platform.
- **Resolution happens before any cluster contact.** The registry lookup runs after flag validation
  and before the kubeconfig is even resolved, so a lookup that cannot produce a version fails with
  nothing applied. A half-installed cluster is not an acceptable outcome of a registry problem.
- **Strict release selection.** With no `--catalog-prerelease`, a catalog major with no stable
  published release is a refusal naming the flag, never a silent fall back to a prerelease. The
  version a cluster subscribes to is not something the CLI should guess at.
- **`--crds-only` creates no Platform**, keeping its current contract that nothing beyond the
  manifest's CRDs is created. `--catalog-prerelease` combined with `--crds-only` or `--skip-platform`
  is a flag-validation error, following the existing `--user`/`--group` requires `--rbac` precedent
  (`internal/operator/rbac.go`).
- **The catalog module path becomes one constant** shared by the new resolver and
  `DefaultPlatformTemplate`, which currently spells it out as a literal.

## Capabilities

### New Capabilities

None. The behavior extends two capabilities that already exist.

### Modified Capabilities

- `operator-lifecycle`: `opm operator install` gains the Platform-seeding step, the two flags, and the
  requirement that catalog resolution precede all cluster contact. `--crds-only` gains an explicit
  no-Platform guarantee.
- `platform-resolution`: adds registry-resolved catalog subscription (the selection rules for
  release, prerelease, and the dev-build exclusion) and extends the write-if-absent contract to a
  second caller with its own provenance reporting.

## Impact

- **SemVer: MINOR.** Two new flags, both defaulted to today's behavior for the surfaces that had one;
  `opm operator install` with no flags gains a step it did not have.
- **Packages**: `internal/platform` (new catalog-version resolver, extraction of the create helper in
  `cluster.go`), `internal/cmd/operator` (flags, validation, ordering, output), `internal/config`
  (`DefaultPlatformTemplate` reads the shared constant). `internal/publish.NewRegistryClient` and
  `golang.org/x/mod/semver` are consumed, not modified. `internal/operator` is untouched, keeping the
  installer free of registry concerns.
- **A default install now needs registry access.** `--skip-platform` is the offline path.
- **Immediate consequence, stated plainly: the strict default refuses today.** The
  `opmodel.dev/catalogs/opm@v2` line has no stable release; it carries `v2.0.0-alpha.1` through
  `v2.0.0-alpha.3`, plus `v2.0.0-0.dev.<count>.g<sha>` branch builds published to the same GHCR path.
  Until `catalog_opm` cuts `v2.0.0`, a bare `opm operator install` exits non-zero with nothing
  applied, and the working invocations are `--catalog-prerelease` or `--skip-platform`. This is the
  accepted cost of never guessing a version: the refusal is clean, names the flag that clears it, and
  disappears on its own the day a stable catalog ships.
- **Mirror peers unaffected**: `hack/platform.cue`, `hack/kind-platform.yaml`, and the operator's
  sample Platform keep their hand-pinned versions; this change adds a resolved path beside them, it
  does not replace the authored ones.
