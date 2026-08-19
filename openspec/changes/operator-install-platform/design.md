# Design: operator-install-platform

## Context

See `proposal.md` (Why) for motivation. The design-relevant state:

- `opm operator install` is a thin command over `internal/operator`, which knows only about the
  embedded manifest, applying, and waiting. It has no registry dependency today.
- The Platform write already exists and is already correct: `platform.EnsureClusterPlatform`
  (`internal/platform/cluster.go:50-91`) builds the unstructured `Platform`, creates it with field
  manager `opm-cli`, treats `AlreadyExists` as a noop and `Forbidden` as a warning. Its one
  install-hostile detail is a hardcoded success line naming the local default platform file.
- Registry version listing also already exists as a pattern: `publish.NewRegistryClient`
  (`internal/publish/registry.go:21`) plus `modregistry.Client.ModuleVersions`, used by
  `gateAlreadyPublished`, by `publish.Check`, and by `scaffold.ResolveTemplateVersion`
  (`internal/scaffold/scaffold.go:68-110`). A major-suffixed path scopes the listing to that major
  and the tags come back `v`-prefixed and SemVer-sorted.
- The catalog module path `opmodel.dev/catalogs/opm@v2` is currently a bare literal inside
  `DefaultPlatformTemplate` (`internal/config/templates.go:80-88`).
- Measured against the live registry: the v2 catalog line publishes `v2.0.0-alpha.1` through
  `v2.0.0-alpha.3` and no stable release. `catalog_opm`'s `branch-publish` workflow pushes
  `v2.0.0-0.dev.<count>.g<sha>` tags to the same GHCR repository, so a version listing mixes releases
  and branch builds.

## Goals / Non-Goals

**Goals:**

- A cluster that finished `opm operator install` has a Platform the operator can reconcile.
- The subscribed catalog version is resolved at install time, not compiled into the binary.
- A registry problem never leaves a partially installed cluster.
- The Platform write keeps exactly one contract across both of its callers.

**Non-Goals:**

- No update path for an existing Platform. Bumping a cluster's catalog pin stays a deliberate act
  through the CR or a platform file, not a side effect of re-running install.
- No `--catalog-version <x>` pin flag. `--platform <file>` already covers a fully authored platform,
  and CONSTITUTION VII says an unrequested flag is a cost with no buyer.
- No change to the hand-pinned literal in `DefaultPlatformTemplate` or its mirror peers.
  `opm config init` stays normatively offline.
- No configurable catalog. The first-party catalog is the one install seeds; a platform needing more
  or different catalogs is an authored platform.
- No operator `--registry` argument patching. What the operator can pull is the operator's concern.

## Decisions

### D1: A purpose-built selector, not `compat.HighestStable`

`compat.HighestStable` (`library/opm/compat/predecessor.go:25-36`) looks like the right helper and is
not. It walks backwards for the first empty-prerelease version and, failing that, returns the last
element of the list. On today's catalog history that fallback silently returns `2.0.0-alpha.3` under
what the user asked to be a release-only default, and on a list containing branch builds it can
return a dev build. Its semantics belong to template floating, where a best-effort answer is right.

Decision: implement selection here, over the same `ModuleVersions` listing, with the two explicit
modes the specs define. `HighestStable` stays untouched and keeps its single existing caller.

*Alternative considered:* extend `HighestStable` with a strictness parameter. Rejected: it lives in
the library, its current caller wants the loose behavior, and a boolean parameter that inverts a
function's failure mode is a trap for the next reader.

### D2: Development builds are excluded by the shape of their prerelease, not by a name pattern

The listing mixes release-please releases (`-alpha.N`, and later `-beta.N` / `-rc.N`) with
branch-publish builds (`-0.dev.<count>.g<sha>`). The discriminator is deliberate upstream:
`catalog_opm/.tasks/branch-tag.sh` gives branch builds a leading numeric identifier precisely so
SemVer 2.0.0 section 11.4.3 ranks them below every alphanumeric prerelease.

Decision: prerelease mode accepts a version only when the first identifier of its prerelease part is
non-numeric. This reads the property the tag scheme was built to have, instead of matching on the
literal string `dev`, which would silently accept a future `nightly.` or `ci.` prefix.

*Alternative considered:* deny-list the `dev` identifier. Rejected: it is a blocklist over an open
set, and it inverts the safe default. An unknown tag shape should be rejected from a cluster
subscription, not accepted.

### D3: The resolver lives in `internal/platform`

It produces a `synth.PlatformInput`, which is what `internal/platform` exists to produce and what
`EnsureClusterPlatform` consumes. Putting it there keeps `internal/operator` free of registry
concerns, so the installer stays a manifest-and-cluster component. The import direction
`platform -> publish` is new but acyclic: `publish` does not import `platform`.

*Alternative considered:* `internal/operator`. Rejected: it would give the installer a registry
dependency it otherwise has no reason to hold, and the resulting value would still have to travel to
`internal/platform` to be written.

### D4: Resolve before the kubeconfig, not between install and wait

Ordering is validate flags, then resolve the catalog version, then resolve kubernetes, then install
and wait, then seed. Resolution is a cheap read with a decisive outcome, so running it first turns a
registry failure into a refusal with an untouched cluster. This is CONSTITUTION I (validate early)
applied across a process boundary.

*Alternative considered:* install first, seed last, and let a resolution failure surface after the
operator is up. Rejected: it trades a clean refusal for a half-provisioned cluster plus a non-zero
exit, and the user has to reason about which half landed.

### D5: Extract the create, keep the messages honest

`EnsureClusterPlatform` is reused by extracting its create path into an unexported helper that
returns what happened (created, already present, or forbidden) instead of printing. The existing
exported function keeps its current output verbatim, so `internal/workflow/apply/apply.go:189` is
behaviorally untouched. The install path gets its own thin caller that reports the catalog coordinate
and version it resolved.

This is what the `platform-resolution` delta means by provenance: one write contract, two truthful
narrations. Reusing the function as-is would have install announce that it seeded from
`~/.opm/platform.cue`, which it did not read.

### D6: Flag validation follows the `--rbac` precedent

`--catalog-prerelease` with `--crds-only` or `--skip-platform` is a validation error rather than a
silently inert flag, matching `--user`/`--group` requiring `--rbac` (`internal/operator/rbac.go`). A
flag that appears to select a prerelease and in fact selects nothing is worse than an error,
especially on a command that writes to a cluster.

### D7: One constant for the catalog path

`config.DefaultCatalogPath` becomes the single spelling of `opmodel.dev/catalogs/opm@v2`, and
`DefaultPlatformTemplate` converts from a `const` to a `fmt.Sprintf` `var` over it. The constant
lives in `internal/config` rather than `internal/platform`: `platform` already imports `config`
(`resolve.go`, `spec.go`), so putting it the other way round would be an import cycle. The template
directly above it (`DefaultConfigTemplate`) already uses exactly this shape, and the rendered bytes
are unchanged, so `internal/cmd/config/init_test.go:146` continues to pass without edit.

The version literal in that template stays hand-pinned. It is a different concern: an offline
scaffold cannot resolve latest, and its documented drift mode ("old but resolvable") is acceptable
precisely because published tags are immutable.

### D8: Exit-code mapping reuses the established vocabulary

A refusal (no selectable version) is exit 2 through the refusal funnel `cmdutil.PrintRefusals`, and
an unreachable registry is exit 3 through `publish.ConnectivityError`. Both mappings already exist
and are already exercised by `opm catalog registry check`
(`internal/cmd/catalog/registry.go:114-127`). Nothing new is invented for this command.

## Risks / Trade-offs

- **The default refuses on today's registry** → Accepted deliberately, and it is the user's stated
  choice over a silent prerelease fallback. It self-resolves when `catalog_opm` cuts `v2.0.0`; until
  then the refusal names `--catalog-prerelease`, and `--skip-platform` restores the previous
  behavior exactly. The proposal states this in Impact so it is not discovered at runtime.
- **Install now needs network for its default path** → `--skip-platform` is the offline install, and
  `--crds-only` was already the minimal path. Both are documented in the command's help text.
- **Seeding changes which platform later commands resolve** → After install, the cluster CR exists,
  so `opm instance apply` uses precedence source 2 (cluster CR) instead of falling back to
  `~/.opm/platform.cue`. That is the intended end state and is exactly what the first apply used to
  cause, only earlier and with a resolved version rather than a compiled-in one. Mitigations already
  in the surface: install reports the coordinate it pinned, `--platform <file>` still overrides
  everything, and the apply path already reports which source it resolved.
- **A resolved version the operator cannot pull** → The Platform will report NotReady if the
  operator's own registry configuration cannot reach the catalog. Out of scope here, and visible:
  the operator surfaces it on the Platform's conditions, and the install output names the exact
  coordinate to check.
- **Listing cost** → One extra registry round trip on the default install path, against a public
  anonymous GHCR read. Negligible next to pulling and applying the manifest.

## Migration Plan

No migration. Clusters that already carry a Platform are untouched by construction, so an existing
install re-run is a noop for the Platform. Rollback is `--skip-platform`, which reproduces the
pre-change command exactly.

## Open Questions

None that would change the specs, the approach, or the task breakdown. One deferred observation
worth revisiting only if it is actually asked for: if operators later need install to bump an
existing cluster's catalog pin, that is a separate change with its own refusal semantics, not a flag
grafted onto this one.
