# Tasks: operator-install-platform

Ordered so each group is green on its own: the offline constant first, then pure selection logic,
then the registry call, then the cluster write, then the command that wires them together.

## 1. Shared catalog path constant

- [x] 1.1 Add `DefaultCatalogPath = "opmodel.dev/catalogs/opm@v2"` to `internal/config` with a comment naming it as the single first-party catalog install seeds (0010 D47). It lives in `config`, not `platform`, because `platform` already imports `config` and the reverse would be an import cycle.
- [x] 1.2 Convert `config.DefaultPlatformTemplate` from a `const` to a `fmt.Sprintf` `var` over that constant, matching the shape `DefaultConfigTemplate` already uses directly above it.
- [x] 1.3 Confirm the rendered template is byte-identical: `go test ./internal/config/... ./internal/cmd/config/...` passes unchanged, including the `opmodel.dev/catalogs/opm@v2` assertion in `internal/cmd/config/init_test.go`.

## 2. Version selection (pure, no registry, no cluster)

- [x] 2.1 Implement the selector over a `[]string` of `v`-prefixed tags: release mode returns the highest empty-prerelease version; prerelease mode returns the highest version whose prerelease part starts with a non-numeric identifier; invalid SemVer entries are skipped. Return the bare version (no `v` prefix), the form a Platform subscription stores.
- [x] 2.2 Build the refusal for "nothing selectable" as a `publish.Refusal`: headline naming the module path, evidence rows for module path / registry / highest version seen, consequence, and an action naming `--catalog-prerelease`.
- [x] 2.3 Table-driven tests, one case per `platform-resolution` scenario: stable present wins over alphas; alphas only under release mode refuses (today's real registry); `[2.0.0-alpha.3, 2.0.0-0.dev.1754899200.g9ea5927]` under prerelease mode picks the alpha; all-dev-builds refuses in both modes; empty list refuses; unparseable tags ignored.

## 3. Registry lookup

- [x] 3.1 Implement `ResolveCatalogVersion(ctx, registry, modulePath string, prerelease bool) (string, error)` in `internal/platform`: `publish.NewRegistryClient` then `client.ModuleVersions`, feeding 2.1. Wrap transport failure in `*publish.ConnectivityError` naming the lookup and registry, mirroring `scaffold.ResolveTemplateVersion`.
- [x] 3.2 Test the error shapes: a failing client surfaces `*publish.ConnectivityError` and never a refusal; a successful listing returns the bare version.

## 4. Platform write for the install path

- [x] 4.1 Extract the create path of `EnsureClusterPlatform` into an unexported helper returning the outcome (created / already present / forbidden) instead of printing; leave `EnsureClusterPlatform`'s own output verbatim so `internal/workflow/apply/apply.go` is behaviorally unchanged.
- [x] 4.2 Add the install-path seeder that builds `synth.PlatformInput{Name: "cluster", Type: "kubernetes", Subscriptions: {DefaultCatalogPath: {Version: resolved}}}`, calls the helper, and reports provenance as the catalog coordinate and version.
- [x] 4.3 Tests against `dynamicfake`, reusing `newFakeDynamic` / `clusterPlatformObj` from `internal/platform/cluster_test.go`: absent Platform is created carrying the resolved version; a pre-seeded Platform keeps its stored version untouched; a `Forbidden` reactor warns and returns no error.
- [x] 4.4 Assert the existing apply-path behavior is unchanged by the extraction (its own tests stay green, provenance message unmodified).

## 5. Command wiring

- [x] 5.1 Add `--catalog-prerelease` and `--skip-platform` (both default `false`) to `opm operator install`, and extend `installFlags`.
- [x] 5.2 Add flag validation beside the existing `RBACOptions.Validate` call: `--catalog-prerelease` with `--crds-only` or `--skip-platform` is `ExitValidationError`, raised before any registry or cluster call.
- [x] 5.3 Reorder `runOperatorInstall` to validate, resolve the catalog version when seeding applies, then resolve kubernetes, install and wait, then seed. Map a refusal to exit 2 through `cmdutil.PrintRefusals` and a `ConnectivityError` to exit 3.
- [x] 5.4 Extend the success output with the Platform outcome and update the command's `Long` help text and examples to cover both new flags and the release-versus-prerelease default.
- [x] 5.5 Command-level tests: both invalid flag combinations exit 2 with no client construction; `--crds-only` and `--skip-platform` perform no registry lookup.

## 6. Validation gates

- [x] 6.1 `task fmt`, `task vet`, `task lint`.
- [x] 6.2 `task test:unit`.
- [x] 6.3 Kind cluster check against `kind-opm-dev`: bare `opm operator install` exits 2 with the refusal, before any cluster contact (verified with an unreadable kubeconfig, so nothing was applied); `--catalog-prerelease` against a cluster with no Platform creates `platform/cluster` pinned to the registry-resolved version; a second run against an existing Platform leaves it untouched (resourceVersion unchanged); `--crds-only --catalog-prerelease` exits 2 at flag validation.
- [x] 6.4 Confirm the seeded Platform is usable end to end: `kubectl wait --for=condition=Ready platform/cluster`, then an `opm instance apply` that reports the cluster CR as its platform source.
