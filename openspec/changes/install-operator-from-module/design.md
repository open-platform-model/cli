## Context

See proposal.md for motivation. Four owner decisions (2026-10-04) bound this design and are not reopened here: the operator's own instance (`opm-operator` in `opm-operator-system`) is CLI-owned for good and the operator never reconciles it; install pulls the module from a registry, with no embedded manifest and air-gapped clusters served by a mirror; the module renders through the catalog's abstractions; and the module has its own version train, so the CLI pins a module version and records the operator version it deploys. Moving the operator instance to another owner is out of scope.

What the code does today, and what this design changes:

- **Install** (`internal/operator/install.go:54-95`): resolve the embedded or fetched manifest (`resolveManifestFrom`, `install.go:139-160`), build a plan (`plan.go`), wait out terminating objects (`waitForTerminating`), `kubernetes.ApplyOne` each object, then `kubernetes.Wait` for CRDs `Established` and the Deployment rolled out. The command (`internal/cmd/operator/install.go:134-204`) resolves the catalog version first, then installs, then seeds the Platform (`platform.EnsureClusterPlatformForCatalog`).
- **Instance apply** (`internal/workflow/apply/apply.go:58-230`, `Execute`): cluster gates (CRD present, CRD field floor, operator ceiling; `RunClusterGates`, `apply.go:255-263`), ownership branch, status-RBAC preflight, existence check on a first apply (`RunPreApplyExistenceCheck`, `apply.go:454-464`), apply, prune with CRDs and Namespaces protected, record write (`WriteInstanceRecord`, `apply.go:323-378`, which records `spec.values`).
- **Render from a published module** (`internal/workflow/render/module.go:31-122`, `FromModule`): `AcquireModuleFromRegistry`, values from `-f` files or `debugValues` (`values.go:22-31`), `SynthesizeInstance`, platform by `platform.Resolve` (`internal/platform/resolve.go:222-272`), where a nil `Cluster` getter skips the cluster step and `Deps` falls back to the module's own pins.
- **Uninstall** (`internal/operator/uninstall.go:170-201`): finalizer guard, then deletes the embedded manifest's documents minus CRDs and Namespace.
- **The kernel** (library `v1.0.0-beta.3`, `opm/kernel/acquire.go:51`) loads modules through CUE's module cache under `CUE_CACHE_DIR`, keyed by `path@version` and not by registry (`cuelang.org/go/mod/modcache/cache.go:112`, `:147`); the kernel takes a registry mapping (`kernel.WithRegistry`) and no cache directory, although its `cueenv.Override(registry, cacheDir)` already supports one (`library/opm/internal/cueenv`). CUE modules carry no dependency checksums: a CUE module's `cue.mod/module.cue` names its dependencies by `path@version` only, and a registry accepted a re-push of one version with other bytes in the experiment (Evidence, render).
- **The Platform** carries `status.operatorVersion` written in the same status patch as its `Ready` or `Stalled` condition, with `status.observedGeneration` (`opm-operator/internal/controller/platform_controller.go:212-217`, `:302-306`, `:456-458`).

## Goals / Non-Goals

**Goals:**

- One install path: render the operator module, check, CRD step, instance apply, Platform seed, Platform-Ready wait.
- The default install applies exactly the bytes this CLI release pinned, for the module and every module its render resolves, without trusting the registry or the local CUE cache.
- No write before every refusing check has passed: a refused install changes nothing.
- Reuse the instance apply and instance delete machinery instead of a second apply or delete path.

**Non-Goals:**

- The migration of a manifest-installed operator. This change leaves a slot in the check phase and the write order for `migrate-manifest-installed-operator`; until it lands, install on such a cluster refuses at the apply guard.
- Signature verification of a non-default module version (no signing exists for modules yet).
- A `--dry-run` for install (not requested; YAGNI).
- Any change of the operator instance's owner. It stays `owner: cli`; the experiment's step 8 showed a self-owned operator instance cannot be reclaimed by re-running the CLI and destroys the operator when deleted.

## Decisions

### Command syntax and flags

```text
opm operator install [flags]
opm operator uninstall [--remove-finalizers] [k8s flags]
```

| Flag | Type | Default | Meaning |
| --- | --- | --- | --- |
| `--version` | string | `""` (the pinned default) | Module version selector, the `internal/modref` grammar: `0.3.1` pins, `v0` floats to the newest release of that major |
| `-f`, `--values` | []string | none | Values files layered, in order, over the values recorded on the operator's instance |
| `--reset-values` | bool | false | Ignore the recorded values; start from the module's defaults plus `-f` |
| `--allow-downgrade` | bool | false | Allow a target module (or, with no record, operator) version below the cluster's |
| `--crds-only` | bool | false | Unchanged meaning, now from the module render |
| `--rbac`, `--user`, `--group` | | | Unchanged (0006:D23) |
| `--catalog-prerelease`, `--skip-platform` | | | Unchanged (0006:D12, 0006:D22) |
| `--timeout` | duration | 5m | One budget: terminating wait, CRD `Established`, rollout, Platform Ready |

Flag validation (exit 1, before any registry or cluster call): `-f`, `--reset-values` and `--allow-downgrade` with `--crds-only` are refused as having no effect, as `--catalog-prerelease` already is (`installFlags.validate`, `internal/cmd/operator/install.go:127-132`). `--values` and `--reset-values` together are allowed (reset, then layer).

Exit codes: 0 success; 1 usage or general error, including the readiness timeout; 2 a refusal (digest mismatch, rejected values, a version rule, the apply guard, uninstall with no record); 3 registry unreachable; 4 permission denied.

Example output (messages are the contract's shape; exact wording is the implementation's):

```text
$ opm operator install
INFO operator module opmodel.dev/modules/opm_operator 0.1.0 (pinned; deploys opm-operator v1.0.0-beta.6)
INFO verified 4 module(s) against this CLI's pinned digests
INFO platform: module deps (cluster Platform not used for the operator)
INFO CustomResourceDefinition/moduleinstances.opmodel.dev   created
...
INFO opm-operator applied 19 resources successfully (19 created)
INFO Platform/cluster created (opmodel.dev/catalogs/opm v4.6.0)
[x] opm-operator v1.0.0-beta.6 installed from module 0.1.0; Platform/cluster Ready

$ opm operator install --version 0.4.0
ERROR refusing opm_operator 0.4.0: it deploys opm-operator v1.1.0, newer than this CLI (v1.0.0-beta.9) - upgrade the CLI first

$ opm operator install            # registry serves other bytes under the pinned tag
ERROR refusing the pinned operator module: opmodel.dev/catalogs/opm v4.6.0 has digest sha256:9f2... where this CLI pinned sha256:41c...; nothing was changed

$ opm operator install            # Platform not Ready within --timeout
ERROR opm-operator v1.0.0-beta.6 applied, but Platform/cluster is not Ready from it (Stalled: BuildFailed: ...); nothing was rolled back - fix the Platform and re-run 'opm operator install'
```

### Install flow

```text
flags --> catalog version (if seeding) --> module ref (pin | --version)
      --> [default] verified closed-world source  |  [other] registry as served + trust notice
      --> kube client --> read record opm-operator-system/opm-operator
      --> values = recorded (unless --reset-values) <- -f files
      --> render: SynthesizeInstance(opm-operator, opm-operator-system) + deps-only platform
      --> checks: values vs #config | operator MAJOR.MINOR <= CLI (V1)
                  downgrade (V2/V3) | CRD served versions (V4) | CRD floor on render (V5)
                  status-RBAC | terminating wait | apply guard on every rendered object | [migration proof slot]
      --> write 1: SSA rendered CRDs, wait Established
      --> [migration writes slot]
      --> write 2: workflowapply.Execute (CreateNS=false, ceiling skipped, prune on)
      --> wait: controller Deployment rollout
      --> Platform seed (write-if-absent, unless --skip-platform)
      --> wait: Platform Ready with status.operatorVersion == installed operator version
```

`--crds-only` stops after write 1 (checks V1, V4, V5 and the guard over the CRDs only), then applies `--rbac` objects as today.

Signatures (new or changed, `internal/operator`):

```go
// Plan is everything install decided before its first write.
type Plan struct {
    Module          ModuleRef          // path, version, Default bool
    OperatorVersion string             // tag of the rendered controller image
    Render          *render.Result     // instance opm-operator in opm-operator-system
    CRDs            []*unstructured.Unstructured
    PrevRecord      *inventory.Record  // nil on a fresh cluster
}

func PlanInstall(ctx context.Context, env InstallEnv, opts InstallOptions) (*Plan, error) // no writes
func Install(ctx context.Context, env InstallEnv, plan *Plan, opts InstallOptions) (*InstallResult, error)
func CheckTarget(ctx context.Context, client *kubernetes.Client, plan *Plan, cliVersion string, allowDowngrade bool) error // V1-V5
func OperatorVersionOf(objs []*unstructured.Unstructured) (string, error)
func Uninstall(ctx context.Context, client *kubernetes.Client, opts UninstallOptions) (*UninstallResult, error) // now record-driven
```

`internal/workflow/apply.Options` gains `SkipOperatorCeiling bool`, read by `RunClusterGates`. `internal/inventory` gains `CheckCRDFieldFloor(crd *unstructured.Unstructured) error`, which `GateCRDFieldFloor` calls after its read. `internal/workflow/render.ModuleOpts` gains `Values []kernel.Source` (used instead of `-f` files and `debugValues` when set) and `DepsOnly bool` (nil cluster getter, no `--platform`).

### Research & Decisions

#### Anchoring the default install to its pinned bytes

**Context**: the spec requires that the default install applies exactly the module and dependency content the CLI release pinned, and refuses before any write otherwise. The kernel fetches by `path@version` through the registry mapping and the CUE cache; neither checks content.
**Explored**: `cuelang.org/go/mod/modregistry` (`Client.GetModule`, `Module.ManifestDigest`, `GetZip`), CUE's cache layout (`mod/modcache/cache.go`), the kernel's options (`library/opm/kernel/kernel.go:87`, `:118`), experiment 01's local registry, which accepted a re-push of one version.
**Options considered**:
1. Check manifest digests through `modregistry`, then let the kernel load as usual. Simple, but the kernel may load from the CUE cache (keyed by `path@version`) or fetch again; the bytes it loads are never the bytes that were checked.
2. **A verified closed world.** For each pinned `path@version`, read the manifest from the configured registry, compare its digest with the pin, copy the manifest and its blobs byte for byte into an in-memory OCI registry (`cuelabs.dev/go/oci/ociregistry/ocimem`) served on `127.0.0.1:0` (`ociserver`), and run the kernel and the platform generation with a catch-all mapping to that address and a private, per-run CUE cache directory. Any module the render needs that was not pinned fails to fetch, and install refuses naming it.
3. A library API that loads with expected digests. Right long term; nothing like it exists, and it would hold this change on a library design.
**Decision**: option 2. The private cache directory needs one of: (a) a kernel option `WithCacheDir` in the library (its `cueenv` already takes the value), released and bumped before section 2; or (b) the CLI setting `CUE_CACHE_DIR` for the run under the package mutex `internal/cuemod` already uses for its one `os.Setenv`. Section 1 measures both; (a) is preferred because it keeps the CLI's single `os.Setenv` site, and choosing it adds a library release to the gates, which the spike reports to the supervisor before section 2 starts.
**Rationale**: only option 2 makes the loaded bytes the checked bytes with today's kernel, and the closed world turns "every module the render resolves" from a list someone keeps current into a property the render enforces.

#### What the pin records and how it is produced

**Context**: today `task operator:sync` downloads `install.yaml` and rewrites `PinnedOperatorVersion` (`Taskfile.yml:482-498`). Readers that must stay offline: `hack/docskit-dump` (`main.go:76`), `.github/scripts/release-evidence.sh:21`, `.tasks/cascade/cascade.sh:251`, `.tasks/cascade/pins.sh:48`.
**Options considered**:
1. Pin only the module version and read the operator version from the module at build time. Not offline: the docs pins and the release evidence check must read the operator version without a registry.
2. **A generated Go file** `internal/operator/pin.go` with `OperatorModulePath`, `PinnedModuleVersion`, `PinnedOperatorVersion` and a `pinnedClosure` slice of `{Path, Version, Digest}`, written only by `go run ./hack/operator-pin <version>` (`task operator:pin VERSION=<v>`). The program resolves the version, renders it in record mode (the closed world of the previous decision, filled through the real registry and recording every `path@version` the render fetched with its manifest digest), reads the operator version from the render, and writes the file.
3. A JSON file read with `go:embed`. Needs a parser in every shell reader.
**Decision**: option 2, keeping the name `PinnedOperatorVersion` for the recorded operator version, so the four offline readers above keep their `sed` and Go reads; its doc comment changes to "the operator version `PinnedModuleVersion` deploys".
**Rationale**: the pin is produced by the same render install performs, so the recorded closure is exactly what install will need; reusing the constant name avoids touching readers that the in-flight `join-release-cascade` change also edits.

#### Reading the operator version a module deploys

**Options considered**: a field the operator module exports (an interface opm-operator would have to add), or **the tag of the controller Deployment's image in the render**.
**Decision**: the rendered Deployment `opm-operator-controller-manager`, container `manager`, image `<repo>:<tag>@<digest>`; the tag must be a semver release tag. The operator module names the image by a published release's tag and digest, and its `#config` exposes the image repository but not the tag or digest (a recorded tag would survive an upgrade and keep the old binary running under a record that names the new version), so the tag is the module's own statement of what it deploys. Experiment 01's render showed the image as `<repo>:v1.0.0-beta.6` (no digest yet, before the module named one); the published module adds the digest. A render without that Deployment, or with an untagged or non-semver image, refuses. Section 1 confirms it against the published module.

#### Values on reinstall

**Context**: the record keeps `spec.values` (`inventory.Record.SpecValues`, written by `WriteInstanceRecord`). Kernel sources unify, so layering a changed field over a recorded one would conflict.
**Options considered**: typed flags per field (couples the CLI to the module's `#config`, which is the operator repo's to shape); a values file only; **values files layered over the recorded values**, Helm's `--reuse-values` behaviour by default, with `--reset-values` to start over.
**Decision**: the CLI deep-merges in Go: recorded values, then each `-f` file decoded in order; maps merge, scalars and lists from the later source replace. The merged map is rendered to CUE and handed to the kernel as one source with origin `values recorded on opm-operator-system/opm-operator plus <files>`. Before any write, the merged values are checked against the target module's `#config` (`validateValuesFiles`, `internal/workflow/render/module.go:69-74`), and every rejected path is named. `debugValues` are never used for the operator.
**Rationale**: the operator's settings live on its instance because `spec.values` is the one authoritative render input (0006:D19); a reinstall must not silently reset a value; without `--reset-values`, a recorded value could never be removed. Neither experiment exercised this: both re-ran install from an instance file that already held the values, so the table tests of task 3.2 are the first evidence. The image tag and digest are deliberately not values (see the previous decision).

#### Render platform

**Decision**: install calls the render with `DepsOnly`: no `--platform`, `Cluster: nil`, `Deps` from the module's `cue.mod/module.cue` (`moduleDepsOf`, `module.go:177-196`). The cluster Platform is never read for the render, so a missing, stalled or differently subscribed Platform neither changes nor refuses it. Experiment 02 step 4 is the case it removes: its reinstall rendered through the seeded Platform and produced the same render digest only because that Platform subscribed to the same catalog release (4.5.2) the module pins. Install is the recovery path, so it must not depend on the Platform it may be repairing. This narrows 0006:D11's precedence for this one instance; every other render keeps it.

#### The ceiling and the version rules

**Context**: every instance apply refuses when the operator the Platform reports has a `MAJOR.MINOR` above the CLI's (the ceiling of 0021:D9). For the operator's own install that is the wrong subject: install replaces the running operator, and a ceiling that refused it would leave a CLI below the running operator no way to repair the cluster. Install can instead check the operator it is about to install, because the module states the operator version it deploys, which the embedded manifest never allowed before applying.

**Decision**: `Execute` skips `GateOperatorVersionCeiling` when `SkipOperatorCeiling` is set, which only operator install sets; the CRD-presence and CRD-floor gates still run and cannot refuse after the CRD step, because the rendered CRDs already passed V5. `CheckTarget` applies, before any write, the five rules of the spec requirement "Install refuses a target version the CLI cannot drive or the cluster cannot take" (labelled V1 to V5 here only):

- V1: refuse when `MajorMinor(operatorVersion) > MajorMinor(CLI)`; a dev CLI skips with a warning, as the ceiling does (`internal/inventory/gates.go:74-79`).
- V2: with a record, refuse when the target module version is below `spec.module.version` unless `--allow-downgrade`.
- V3: with no record, compare the target's operator version with the Platform's `status.operatorVersion`, else the tag of the live Deployment at the fixed names; refuse below it unless `--allow-downgrade`; no running operator found means no check.
- V4: for each rendered CRD whose name exists in the cluster, every version the cluster serves must be served by the render; `--allow-downgrade` does not override.
- V5: `inventory.CheckCRDFieldFloor` on the rendered `moduleinstances.opmodel.dev`.

**Rationale**: install is the one recovery path, so it must not install an operator the CLI then cannot drive (V1), downgrade by accident when an older CLI re-runs it (V2, V3), or strand custom resources stored under a version a CRD stops serving (V4, which an explicit downgrade does not override; prior art is OLM v1's CRD upgrade-safety preflight). V5 reads the render rather than the cluster, so an explicit downgrade to a module whose CRD is below the floor refuses with nothing changed instead of failing after the older CRDs are applied.

#### Success means reconciling

**Decision**: after the instance apply, wait for the Deployment rollout (existing `kubernetes.Wait` with `DefaultPredicate`), seed the Platform as today, then poll the Platform until one read shows `Ready=True`, `status.observedGeneration == metadata.generation` and `status.operatorVersion` equal (after `v`-normalisation) to the installed operator version. The operator writes all three in one status patch, so a stale `Ready` from the previous release cannot pass. The rollout alone is not enough: in the experiment's upgrade the new pod started 28 s before its controllers did, so `opm instance apply --wait` reported healthy about 30 s before the operator reconciled anything, and the first workload reconcile in that window reported `PlatformNotReady` (Evidence, step 5). On timeout nothing is rolled back; the error names `Platform/cluster`, its `Ready`/`Stalled` reason and message, and that re-running install completes it.

**Provisional, needs a decision (the readiness wait against unchanged seeding):** with `--skip-platform` on a cluster that holds no Platform, nothing can report Ready. This design ends install after the rollout with a warning that reconciliation was not confirmed, because `task cluster:operator` and the lifecycle e2e test install with `--skip-platform` and apply their own Platform afterwards. The alternative is refusing `--skip-platform` for a full install on such a cluster. Reported to the supervisor; the spec scenario "Skipped seeding with no Platform" carries the provisional rule.

#### CRD step and the apply guard

**Decision**: the rendered CRDs already carry the instance's identity labels (the kernel stamps them), so they are applied with `kubernetes.ApplyOne` (force, field manager `opm-cli`, `internal/kubernetes/apply.go:244-272`) and then recorded by the instance apply as its own objects, not refused as foreign (Evidence, step 1). On a cluster with no record, install runs `inventory.PreApplyExistenceCheck` over every rendered object before the CRD step (the experiment's emulated CRD step relabelled all four CRDs before the existence check refused the Namespace, step 6), after `waitForTerminating` (kept from `install.go:99-112`) so install right after uninstall still waits instead of refusing. A CRDs-only install followed by a full install of the same version records the existing CRDs: they carry the same identity and an OPM managed-by label. `migrate-manifest-installed-operator` later extends this check with its proof list.

#### Uninstall over the record

**Decision**: read the record at the fixed coordinates; none means exit 2 with "no operator instance record in opm-operator-system; run 'opm operator install' to record the running operator, then uninstall". Otherwise the finalizer guard runs first, unchanged, then the record's inventory is deleted through `kubernetes.Delete` (inventory only, CRDs and Namespaces left behind, a live object without this instance's identity left behind) and the record last, the order `executeInstanceDelete` uses (`internal/cmd/instance/delete.go:207-249`). That function moves to `internal/workflow/apply` (or a sibling `delete` file there) so both commands share it.

**Rationale**: once the operator has an inventory, the inventory is the only honest answer to "what did install put here"; a list compiled into the CLI describes the CLI's release, not the cluster, and misses an object an older release installed. The finalizer guard of 0006:D34 stays first because it protects the instances the operator manages, which do not change when the operator's own record does: the experiment deleted the operator's instance with `opm instance delete`, which has no such guard, while an operator-managed fixture still carried the cleanup finalizer, and the fixture's later delete wedged until install was re-run (Evidence, step 7). The no-record refusal replaces a delete-by-name: deleting by the fixed names would reach objects the CLI cannot prove it installed.

#### Cascade, release gates and e2e

- `task operator:pin` replaces `operator:sync`; `.tasks/cascade/cascade.sh` moves the operator lane to the newest published module release in the pinned module's major whose deployed operator `MAJOR.MINOR` is not above the cli's, then runs `task -x operator:pin VERSION=<v>`. Whether the shared resolver can answer "newest module release" without a change to `.github` is a section 1 check; if it cannot, the lane stays at the current pin with a warning and the resolver change is reported.
- G1 (`release-pin-check.sh` step 5): refuse a `PinnedModuleVersion` the registry does not serve, a recorded digest that differs from the registry's manifest digest, or a `PinnedOperatorVersion` that differs from the tag the pinned module's render deploys (by running `go run ./hack/operator-pin --check`).
- G4 (`release-evidence.sh`): compare `PinnedModuleVersion` at the last tag with the PR's, so a module-only release also needs evidence.
- `.github/scripts/e2e-cluster-applies.sh` keeps triggering on `internal/operator/`, which now holds `pin.go`.
- `tests/e2e/operator_test.go`: `pinnedOperatorImage` reads the image from `opm instance build`-style render of the pinned module, or composes it from the pinned operator version; the lifecycle asserts the record, the inventory, Platform-Ready success, the uninstall-from-record behaviour and the no-record refusal.

## Evidence

Two hand-run experiments on 2026-10-04 back this design. Their working files are not in this repository; what they measured is summarised here.

**Experiment 01, the render.** A catalog-path operator module (the shape opm-operator publishes) was built with `opm module build` and `opm instance build` against core v2.0.0-beta.2, catalogs/opm v4.5.2 and operator v1.0.0-beta.5/beta.6.

- The render needs no cluster: with `KUBECONFIG=/nonexistent` it reported `platform: module deps (opmodel.dev/catalogs/opm@v4 v4.5.2 ...)`, and an instance build with no Platform CR produced identical output.
- 19 objects, the same kinds and namespaces as the release manifest; 16 of 19 names equal. The three role bindings take the catalog-derived names `opm-operator-manager-role`, `opm-operator-metrics-auth-role` and `opm-operator-leader-election-role` (the manifest has `...-rolebinding`). The Deployment selector differs from the manifest's and is immutable, so moving from the manifest needs a Deployment recreate (the migration change's job).
- The four CRDs rendered byte-equal to the controller's generated specs (CEL rules, list-map keys, status subresource, printer columns). The CRDs do not vary with the instance, so the module is a cluster singleton; with instance `opm-operator` in `opm-operator-system` every other name reproduces the manifest's, the Deployment being `opm-operator-controller-manager`.
- `#config` knobs rendered into the Deployment: image repository (the image carried a tag, no digest), `replicas: 2`, `--registry=...` and `--default-service-account=...` as controller arguments, extra arguments after them, resources.
- Cost of `opm module build`: cold cache 5.66 s wall and 507 MB peak RSS; warm 2.0 to 3.9 s and 494 to 510 MB.
- The local test registry accepted a re-push of a published version with other bytes, which is why the default install anchors to digests.

**Experiment 02, the install.** A shell script emulated this change's install (`opm instance build`, pick the 4 CRDs, `kubectl apply --server-side --field-manager=opm-cli` and wait `Established`, then `opm instance apply --wait`) on a fresh kind v0.32.0 cluster (node v1.36.1), with module versions 0.1.0 and 0.2.0 published to a local registry.

1. **Bootstrap.** The instance apply applied the Namespace first and then the rest: `applied 19 resources successfully (15 created, 4 unchanged)`, the CRDs unchanged because the CRD step had just applied them, and recorded all 19 objects, CRDs and Namespace included. With `--create-namespace` the CLI created the Namespace without OPM labels and the existence check then refused the module's own Namespace (`already exists and is not managed by OPM`).
2. **The operator left its own instance alone.** `spec.owner: cli`, no finalizer; the operator's only write was the status condition `Ready=Unknown`, reason `ManagedExternally`.
3. **End to end.** After the Platform seed (a plain create as `opm-cli`), Platform `cluster` reached Ready from operator v1.0.0-beta.5 and an operator-owned fixture reconciled.
4. **Reinstall.** All 19 objects kept uid and resourceVersion (`19 unchanged`), and the render digest was unchanged. But the second render went through the now-seeded cluster Platform, matching only because the Platform subscribed to the catalog the module pins.
5. **Upgrade 0.1.0 to 0.2.0** was a re-apply: `15 configured, 4 unchanged`, one new ReplicaSet, 13.9 s for apply plus `--wait`. The new pod started at 10:06:37, its controllers at 10:07:05; the Platform regenerated at 10:07:07.
6. **Over a manifest-installed operator** the first apply refused at the existence check on the Namespace, after the emulated CRD step had already relabelled the four CRDs. Forcing past it needed a Deployment delete (immutable selector) and left the three `*-rolebinding` objects uninventoried.
7. **Delete.** `opm instance delete opm-operator` deleted 14 objects and left the Namespace and the 4 CRDs, without checking the cleanup finalizer of an armed fixture; re-running the install (`14 created, 5 unchanged`) brought the operator back and it finished the fixture's cleanup 18 s later.
8. **Self-ownership.** Flipping the instance to `owner: operator` either failed closed but armed the finalizer (shipped RBAC) or, with cluster-admin, produced an instance the CLI could not take back and whose delete pruned the operator mid-cleanup; recovery needed a manual finalizer strip.

Install time, fresh cluster each, image through a mirror: module install 31.8 s, 28.6 s and 30.2 s (render plus CRDs 2.3 to 3.5 s), against 12.8 s (warm-up outlier), 28.7 s and 30.7 s for today's install. Readiness (image pull, probes) dominates both; the module path adds about 2.5 s of render.

The kind nodes on that host had no internet egress; the operator reached core and the catalog through two plain host-network pull-through mirrors set through the instance's registry value, which is also the evidence that the mirror path serves the operator's module fetches unchanged.

## Risks / Trade-offs

- [Install needs a registry; a fresh air-gapped cluster cannot install from the binary alone] → the mirror path (experiment 02 ran its operator through two plain pull-through mirrors); the error for an unreachable registry names the module, the registry mapping and `--registry`/config as the fix.
- [A cli release between this change and the migration refuses on every manifest-installed cluster] → release coupling in proposal.md; the refusal changes nothing and names the cause.
- [The closed world refuses when the kernel's floor pulls a core the module does not pin] → the pin is recorded from the same render, so the closure includes it; a cli whose library moves the floor needs `task operator:pin` again, which G1 enforces by re-rendering.
- [Platform-Ready wait fails where the operator cannot reach the catalog registry, as on this host's kind cluster] → by design; nothing is rolled back and the error names the Platform's condition.
- [`PinnedOperatorVersion` keeps its name with a narrower meaning] → doc comment and AGENTS.md state it; renaming would collide with `join-release-cascade`'s edits for no reader benefit.
- [The pre-write checks read the cluster and the write happens later (TOCTOU)] → the same window every instance apply has; the CRD served-version check is re-evaluated by the API server's own validation for removed stored versions.

## Migration Plan

- Users: the next beta's migration note says install needs a registry or mirror, `--version` takes a module version (the operator version it deploys is printed), settings move from Deployment patches to `-f` values, and uninstall needs an install-recorded operator. The first install on an existing cluster is `migrate-manifest-installed-operator`'s.
- Rollback: re-run install with the previous module version and `--allow-downgrade`; returning to a cli that embeds the manifest means deleting the operator Deployment first, because the module's Deployment selector differs from the manifest's and is immutable (Evidence, render), and then running the older cli's install; the three manifest bindings come back beside the module's.

## Open Questions

- Whether `hack/operator-pin --check` can share the render cache across G1's steps to stay under the `Lint` job's time budget; only affects CI time, not behaviour.
