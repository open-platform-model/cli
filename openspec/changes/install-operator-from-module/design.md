## Context

See proposal.md for motivation. Four owner decisions (2026-10-04) bound this design and are not reopened here: the operator's own instance (`opm-operator` in `opm-operator-system`) is CLI-owned for good and the operator never reconciles it; install pulls the module from a registry, with no embedded manifest and air-gapped clusters served by a mirror; the module renders through the catalog's abstractions; and the module has its own version train, so the CLI pins a module version and records the operator version it deploys. Moving the operator instance to another owner is out of scope. The amendments that record these decisions are in enhancements PR #94: 0021:D11 (the operator's install artifact is a registry module on its own train; this change implements R1, R2, R6, R7, R8, R10 and R11, while R3 to R5 and R12 are the module release's and R9 the operator's), 0021:D9:R3 and 0021:D9:R4 (the ceiling does not refuse install; install checks the target), and 0012:D8:R6 and 0012:D8:R7 (install admits, as if adopted, exactly the objects it proves came from an earlier operator manifest, and may delete only the proven earlier Deployment and superseded role bindings; implemented by `migrate-manifest-installed-operator`, not here).

What the code does today, and what this design changes:

- **Install** (`internal/operator/install.go:54-95`): resolve the embedded or fetched manifest (`resolveManifestFrom`, `install.go:139-160`), build a plan (`plan.go`), wait out terminating objects (`waitForTerminating`), `kubernetes.ApplyOne` each object, then `kubernetes.Wait` for CRDs `Established` and the Deployment rolled out. The command (`internal/cmd/operator/install.go:134-204`) resolves the catalog version first, then installs, then seeds the Platform (`platform.EnsureClusterPlatformForCatalog`).
- **Instance apply** (`internal/workflow/apply/apply.go:58-230`, `Execute`): cluster gates (CRD present, CRD field floor, operator ceiling; `RunClusterGates`, `apply.go:255-263`), ownership branch, status-RBAC preflight, existence check on a first apply (`RunPreApplyExistenceCheck`, `apply.go:454-464`), apply, prune with CRDs and Namespaces protected, record write (`WriteInstanceRecord`, `apply.go:323-378`, which records `spec.values`).
- **Render from a published module** (`internal/workflow/render/module.go:31-122`, `FromModule`): `AcquireModuleFromRegistry`, values from `-f` files or `debugValues` (`values.go:22-31`), `SynthesizeInstance`, platform by `platform.Resolve` (`internal/platform/resolve.go:222-272`), where a nil `Cluster` getter skips the cluster step and `Deps` falls back to the module's own pins.
- **Uninstall** (`internal/operator/uninstall.go:170-201`): finalizer guard, then deletes the embedded manifest's documents minus CRDs and Namespace.
- **The operator module** (opm-operator's `add-operator-module`) carries, beside `identity/`, an `operator` package with `Version` (the operator release, without `v`, for example `"1.0.0-beta.5"`) and `Image{repository, tag, digest}`, where `tag` is `"v\(Version)"`. It is readable from the module source without a render. `#config` is closed and refuses `image.tag` and `image.digest`, so the rendered image is always `<repository>:<Image.tag>@<Image.digest>`. The module states its names as constants and refuses any instance other than `opm-operator` in `opm-operator-system`. Its release (opm-operator's `release-operator-module`) publishes it and attaches an `install.yaml` rendered from it to a module release tagged `opm_operator-vX.Y.Z`; signing the module artifact is a follow-up there, not part of that change.

## Goals / Non-Goals

**Goals:**

- One install path: render the operator module, check, CRD step, instance apply, rollout wait, Platform seed.
- No write before every refusing check has passed: a refused install changes nothing.
- Reuse the instance apply and instance delete machinery instead of a second apply or delete path.
- The CLI's pin says, offline, which module version it installs by default and which operator version that deploys.

**Non-Goals:**

- The migration of a manifest-installed operator. This change leaves `migrate-manifest-installed-operator` a slot in the check phase, immediately before the apply guard, and one in the write order, between the CRD step and the instance apply; until it lands, install on such a cluster refuses at the apply guard. The two ship in the same cli release (proposal.md, "Release coupling").
- **Content-digest anchoring of the default install.** Recording the digest of the module and of every module its render resolves, and refusing other bytes, would need an in-memory OCI registry, a private CUE cache directory (a second `os.Setenv` site or a new library option and release) and a record-mode render in the pin tool and in G1. No owner decision or amendment asks for it (0021:D11 asks for a pinned module version and its operator version), so the module is trusted as the registry serves it, like every other module the CLI installs. Experiment 01 showed a registry accepting a re-push of a version; that risk is the same for every module and is a candidate follow-up change with its own owner approval.
- **Verifying a signature on the module.** The module release does not sign the artifact yet (`release-operator-module` defers it); checking a signature at install is out of scope here, as it is for every other module.
- **Downgrade and CRD served-version refusals** (an `--allow-downgrade` override, a target below the recorded module version or the running operator, a CRD that stops serving a version the cluster serves). Today's install has none of these; only the `MAJOR.MINOR` target check is mandated (0021:D9:R4). Candidate follow-up change.
- **Waiting for the cluster Platform to report Ready from the installed operator.** 0021:D11 keeps install's wait at "CRDs served and the operator rolled out". Experiment 02 measured a 28 s gap between the new pod starting and its controllers starting (Evidence, step 5), so a workload apply right after install can meet `PlatformNotReady` for that window, as it can today. Candidate follow-up change.
- A `--dry-run` for install (not requested; YAGNI).
- Any change of the operator instance's owner. It stays `owner: cli`; the experiment's step 8 showed a self-owned operator instance cannot be reclaimed by re-running the CLI and destroys the operator when deleted.
- A CLI check that the cluster holds one operator instance. The module itself refuses any instance coordinates other than `opm-operator` in `opm-operator-system`, so a second operator instance cannot render.

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
| `--crds-only` | bool | false | Unchanged meaning, now from the module render |
| `--rbac`, `--user`, `--group` | | | Unchanged (0006:D23) |
| `--catalog-prerelease`, `--skip-platform` | | | Unchanged (0006:D12, 0006:D22) |
| `--timeout` | duration | 5m | One budget: terminating wait, CRD `Established`, rollout |

Flag validation (exit 1, before any registry or cluster call): `-f` and `--reset-values` with `--crds-only` are refused as having no effect, as `--catalog-prerelease` already is (`installFlags.validate`, `internal/cmd/operator/install.go:127-132`). `--values` and `--reset-values` together are allowed (reset, then layer).

Exit codes (`internal/exit/exit.go`): 0 success; 1 usage or general error, including a rollout timeout; 2 a refusal (rejected values, the operator-version rule, an image that disagrees with the module's stated operator version, the CRD floor, the apply guard, uninstall with no record); 3 registry unreachable; 4 permission denied.

Example output (messages are the contract's shape; exact wording is the implementation's):

```text
$ opm operator install
INFO operator module opmodel.dev/modules/opm_operator 0.1.0 (pinned; deploys opm-operator v1.0.0-beta.6)
INFO platform: module deps (cluster Platform not used for the operator)
INFO CustomResourceDefinition/moduleinstances.opmodel.dev   created
...
INFO opm-operator applied 19 resources successfully (19 created)
INFO Platform/cluster created (opmodel.dev/catalogs/opm v4.6.0)
[x] opm-operator v1.0.0-beta.6 installed from module 0.1.0

$ opm operator install --version 0.4.0
ERROR refusing opm_operator 0.4.0: it deploys opm-operator v1.1.0, newer than this CLI (v1.0.0-beta.9) - upgrade the CLI first

$ opm operator install --version v1.0.0-beta.5
ERROR no operator module version matches "v1.0.0-beta.5": --version now takes an operator module version (for example 0.1.0), not an opm-operator release tag
```

### Install flow

```text
flags --> catalog version (if seeding) --> module ref (pin | --version)
      --> read operator.Version from the module's operator package
      --> kube client --> read record opm-operator-system/opm-operator
      --> values = recorded (unless --reset-values) <- -f files
      --> render: SynthesizeInstance(opm-operator, opm-operator-system) + deps-only platform
      --> checks: values vs #config | operator MAJOR.MINOR <= CLI (V1)
                  rendered image tag == v<operator.Version> | CRD floor on render (V5)
                  status-RBAC | terminating wait | [migration proof slot]
                  | apply guard on every rendered object (last check; admits what the proof admits)
      --> write 1: SSA rendered CRDs, wait Established
      --> [migration writes slot]
      --> write 2: workflowapply.Execute (CreateNS=false, ceiling skipped, prune on)
      --> wait: controller Deployment rollout
      --> Platform seed (write-if-absent, unless --skip-platform)
```

`--crds-only` stops after write 1 (checks V1, the image agreement, V5 and the guard over the CRDs only), then applies `--rbac` objects as today.

Signatures (new or changed, `internal/operator`):

```go
// Plan is everything install decided before its first write.
type Plan struct {
    Module          ModuleRef          // path, version, Default bool
    OperatorVersion string             // operator.Version of the module, "v"-prefixed
    Render          *render.Result     // instance opm-operator in opm-operator-system
    CRDs            []*unstructured.Unstructured
    PrevRecord      *inventory.Record  // nil on a fresh cluster
}

func PlanInstall(ctx context.Context, env InstallEnv, opts InstallOptions) (*Plan, error) // no writes
func Install(ctx context.Context, env InstallEnv, plan *Plan, opts InstallOptions) (*InstallResult, error)
func CheckTarget(plan *Plan, cliVersion string) error               // V1, image agreement, V5
func ReadOperatorVersion(ctx context.Context, src ModuleSource) (string, error) // operator.Version, no render
func RenderedOperatorImage(objs []*unstructured.Unstructured) (tag string, err error)
func Uninstall(ctx context.Context, client *kubernetes.Client, opts UninstallOptions) (*UninstallResult, error) // now record-driven
```

`internal/workflow/apply.Options` gains `SkipOperatorCeiling bool`, read by `RunClusterGates`. `internal/inventory` gains `CheckCRDFieldFloor(crd *unstructured.Unstructured) error`, which `GateCRDFieldFloor` calls after its read. `internal/workflow/render.ModuleOpts` gains `Values []kernel.Source` (used instead of `-f` files and `debugValues` when set) and `DepsOnly bool` (nil cluster getter, no `--platform`).

### Research & Decisions

#### What the pin records and how it is produced

**Context**: today `task operator:sync` downloads `install.yaml` and rewrites `PinnedOperatorVersion` (`Taskfile.yml:482-498`). Readers that must stay offline: `hack/docskit-dump` (`main.go:76`), `.github/scripts/release-evidence.sh:21`, `.tasks/cascade/cascade.sh:251`, `.tasks/cascade/pins.sh:48`. 0021:D11:R6 asks that each CLI release name one default module version and the operator version it deploys, and that a release where the two disagree is refused.
**Options considered**:
1. Pin only the module version and read the operator version from the registry when needed. Not offline: the docs pins and the release evidence check must read the operator version without a registry.
2. **A generated Go file** `internal/operator/pin.go` with `PinnedModuleVersion` and `PinnedOperatorVersion`, written only by `go run ./hack/operator-pin <version>` (`task operator:pin VERSION=<v>`). The program resolves the version through the registry, reads `operator.Version` from the module's `operator` package (the next decision), and writes the file. `OperatorModulePath` is not repeated: `pin.go` uses the constant `locate-operator-and-guard-its-instance` adds in `names.go`, in the same package.
3. A JSON file read with `go:embed`. Needs a parser in every shell reader.
**Decision**: option 2, keeping the name `PinnedOperatorVersion` for the recorded operator version, `v`-prefixed as today (`"v" + operator.Version`), so the four offline readers above keep their `sed` and Go reads; its doc comment changes to "the operator version `PinnedModuleVersion` deploys". While `manifest.go` still declares `PinnedOperatorVersion` (sections 2 to 4), the generated field is named `pinnedModuleOperatorVersion`, and a unit test asserts the two are equal; section 5 deletes `manifest.go` and the generator then writes `PinnedOperatorVersion` itself.
**Rationale**: the pin is two strings a human can review; it is produced from the module's own statement of what it deploys, so the CLI cannot record an operator version the module does not deploy, and G1 re-reads that statement to enforce 0021:D11:R6.

#### Reading the operator version a module deploys

**Context**: install needs the operator version before any write (the `MAJOR.MINOR` rule, 0021:D9:R4), the pin tool needs it, and the cascade lane needs it for every candidate module version it considers.
**Options considered**:
1. **The module's `operator` package**, `operator.Version`, which opm-operator's `add-operator-module` adds precisely so a consumer can read it without a render or a cluster (`cue eval ./operator -e Version`).
2. The tag of the controller Deployment's image in the render. Needs a full render (5.66 s cold, about 500 MB peak RSS in experiment 01) per read, which is acceptable once per install but not for each cascade candidate.
**Decision**: option 1 as the source, option 2 as a consistency assertion. Install, the pin tool and the cascade lane read `operator.Version` from the module source the registry serves (fetched with the CLI's registry mapping; section 1 picks between the kernel's acquired source and a `cuelang.org/go/mod/modregistry` zip fetch, whichever needs no second cache setting). Install then requires the rendered Deployment `opm-operator-controller-manager`, container `manager`, to run an image whose tag equals `v<operator.Version>`, and refuses before any write when it does not, when the Deployment is missing, or when `operator.Version` is not a semver release.
**Rationale**: the operator package is the interface the sibling change provides for this reader; it costs no render where none is needed. The image assertion keeps the two statements honest in the CLI too, although the module derives `tag` from `Version` and its release gate already checks the image.

#### Values on reinstall

**Context**: the record keeps `spec.values` (`inventory.Record.SpecValues`, written by `WriteInstanceRecord`). Kernel sources unify, so layering a changed field over a recorded one would conflict.
**Options considered**: typed flags per field (couples the CLI to the module's `#config`, which is the operator repo's to shape); a values file only; **values files layered over the recorded values**, Helm's `--reuse-values` behaviour by default, with `--reset-values` to start over.
**Decision**: the CLI deep-merges in Go: recorded values, then each `-f` file decoded in order; maps merge, scalars and lists from the later source replace. The merged map is rendered to CUE and handed to the kernel as one source with origin `values recorded on opm-operator-system/opm-operator plus <files>`. Before any write, the merged values are checked against the target module's `#config` (`validateValuesFiles`, `internal/workflow/render/module.go:69-74`), and every rejected path is named. `debugValues` are never used for the operator.
**Rationale**: the operator's settings live on its instance because `spec.values` is the one authoritative render input (0006:D19); a reinstall must not silently reset a value; without `--reset-values`, a recorded value could never be removed. Neither experiment exercised this: both re-ran install from an instance file that already held the values, so the table tests of task 3.2 are the first evidence. The image tag and digest are deliberately not values: the module's closed `#config` refuses them, because a recorded tag would survive an upgrade and keep the old binary running under a record that names the new version.

#### Render platform

**Decision**: install calls the render with `DepsOnly`: no `--platform`, `Cluster: nil`, `Deps` from the module's `cue.mod/module.cue` (`moduleDepsOf`, `module.go:177-196`). The cluster Platform is never read for the render, so a missing, stalled or differently subscribed Platform neither changes nor refuses it. Experiment 02 step 4 is the case it removes: its reinstall rendered through the seeded Platform and produced the same render digest only because that Platform subscribed to the same catalog release (4.5.2) the module pins. Install is the recovery path, so it must not depend on the Platform it may be repairing. This narrows 0006:D11's precedence for this one instance; every other render keeps it.

#### The ceiling and the target check

**Context**: every instance apply refuses when the operator the Platform reports has a `MAJOR.MINOR` above the CLI's (the ceiling, 0021:D9:R1). For the operator's own install that is the wrong subject: install replaces the running operator, and a ceiling that refused it would leave a CLI below the running operator no way to repair the cluster (0021:D9:R3). Install can instead check the operator it is about to install, because the module states the operator version it deploys (0021:D9:R4).

**Decision**: `Execute` skips `GateOperatorVersionCeiling` when `SkipOperatorCeiling` is set, which only operator install sets; the CRD-presence and CRD-floor gates still run and cannot refuse after the CRD step, because the rendered CRDs already passed V5. The main-spec ceiling and gate-ordering requirements are modified to say so, so the archived specs do not contradict each other. `CheckTarget` applies, before any write:

- V1: refuse when `MajorMinor(operator.Version) > MajorMinor(CLI)`; a dev CLI skips with a warning, as the ceiling does (`internal/inventory/gates.go:74-79`).
- Image agreement: the rendered controller image tag equals `v<operator.Version>` (previous decision).
- V5: `inventory.CheckCRDFieldFloor` on the rendered `moduleinstances.opmodel.dev`.

**Rationale**: install must not install an operator the CLI then cannot drive (V1). V5 reads the render rather than the cluster, so a module whose CRD is below the floor refuses with nothing changed instead of failing after its CRDs are applied. A downgrade and a CRD that stops serving a version are not refused (Non-Goals); today's install refuses neither.

#### Success means the operator rolled out

**Decision**: after the instance apply, wait for the Deployment rollout (existing `kubernetes.Wait` with `DefaultPredicate`), within the one `--timeout` budget that also covers the terminating wait and CRD `Established`, then seed the Platform as today. This is the wait 0021:D11 keeps from the embedded install. On a rollout timeout nothing is rolled back; the error names the Deployment and that re-running install completes it.

#### CRD step and the apply guard

**Decision**: the rendered CRDs already carry the instance's identity labels (the kernel stamps them), so they are applied with `kubernetes.ApplyOne` (force, field manager `opm-cli`, `internal/kubernetes/apply.go:244-272`) and then recorded by the instance apply as its own objects, not refused as foreign (Evidence, step 1). On a cluster with no record, install runs `inventory.PreApplyExistenceCheck` over every rendered object before the CRD step (the experiment's emulated CRD step relabelled all four CRDs before the existence check refused the Namespace, step 6), after `waitForTerminating` (kept from `install.go:99-112`) so install right after uninstall still waits instead of refusing. A CRDs-only install followed by a full install of the same version records the existing CRDs: they carry the same identity and an OPM managed-by label. The guard is the last check of `PlanInstall`, and the migration proof slot sits immediately before it: `migrate-manifest-installed-operator` fills that slot with its proof and hands the guard the set of proven objects to admit (0012:D8:R6). A proof placed after the guard would never run on a manifest-installed cluster, because the guard refuses at `Namespace/opm-operator-system` first (Evidence, step 6). This change only orders `PlanInstall` so that nothing but the guard follows the terminating wait; the migration adds its proof there and the admission set to both guard calls.

#### Uninstall over the record

**Decision**: read the record at the fixed coordinates; none means exit 2 with "no operator instance record in opm-operator-system; run 'opm operator install' to record the running operator, then uninstall". Otherwise the finalizer guard runs first, unchanged, then the record's inventory is deleted through `kubernetes.Delete` (inventory only, CRDs and Namespaces left behind, a live object without this instance's identity left behind) and the record last, the order `executeInstanceDelete` uses (`internal/cmd/instance/delete.go:207-249`). That function moves to `internal/workflow/apply` (or a sibling `delete` file there) so both commands share it.

**Rationale**: once the operator has an inventory, the inventory is the only honest answer to "what did install put here"; a list compiled into the CLI describes the CLI's release, not the cluster, and misses an object an older release installed. The finalizer guard of 0006:D34 stays first because it protects the instances the operator manages, which do not change when the operator's own record does: the experiment deleted the operator's instance with `opm instance delete`, which has no such guard, while an operator-managed fixture still carried the cleanup finalizer, and the fixture's later delete wedged until install was re-run (Evidence, step 7). The no-record refusal replaces a delete-by-name: deleting by the fixed names would reach objects the CLI cannot prove it installed.

#### Cascade, release gates and e2e

- `task operator:pin` replaces `operator:sync`; `.tasks/cascade/cascade.sh` moves the operator lane to the newest published module release in the pinned module's major whose `operator.Version` `MAJOR.MINOR` is not above the cli's, read from each candidate's `operator` package without a render, then runs `task -x operator:pin VERSION=<v>`. Whether the shared resolver can answer "newest module release" without a change to `.github` is a section 1 check; if it cannot, the lane stays at the current pin with a warning and the resolver change is reported.
- `.tasks/cascade/pins.sh` reports a fifth pin, `opmodel.dev/modules/opm_operator@v0` from `PinnedModuleVersion`, so a module-only release (same operator, 0021:D11:R3) shows in the cascade title and body; the opm-operator pin stays read from `PinnedOperatorVersion`.
- G1 (`release-pin-check.sh` step 5, via `go run ./hack/operator-pin --check`): refuse a `PinnedModuleVersion` the registry does not serve, or a `PinnedOperatorVersion` other than `v` plus that module's `operator.Version` (0021:D11:R6).
- G4 (`release-evidence.sh`): compare `PinnedModuleVersion` at the last tag with the PR's, so a module-only release also needs evidence.
- Spec names: the release-gates requirements and scenarios that name the embedded operator are REMOVED and ADDED under new names, since a MODIFIED requirement cannot drop a main-spec scenario. The CI check names stay: the `main` ruleset of `open-platform-model/cli` requires the status checks `E2E (kind, embedded operator)` and `G4 operator-embed evidence` (read 2026-10-04), so renaming the job or the check would block every PR until the owner edits the ruleset. The specs keep those names and a later change may rename them together with the ruleset.
- `.github/scripts/e2e-cluster-applies.sh` keeps triggering on `internal/operator/`, which now holds `pin.go`.
- `tests/e2e/operator_test.go`: `pinnedOperatorImage` composes the image from `PinnedOperatorVersion`; the lifecycle asserts the record, the inventory, the rollout, the uninstall-from-record behaviour and the no-record refusal.

## Evidence

Two hand-run experiments on 2026-10-04 back this design. Their working files are not in this repository; what they measured is summarised here.

**Experiment 01, the render.** A catalog-path operator module (the shape opm-operator publishes) was built with `opm module build` and `opm instance build` against core v2.0.0-beta.2, catalogs/opm v4.5.2 and operator v1.0.0-beta.5/beta.6.

- The render needs no cluster: with `KUBECONFIG=/nonexistent` it reported `platform: module deps (opmodel.dev/catalogs/opm@v4 v4.5.2 ...)`, and an instance build with no Platform CR produced identical output.
- 19 objects, the same kinds and namespaces as the release manifest; 16 of 19 names equal. The three role bindings take the catalog-derived names `opm-operator-manager-role`, `opm-operator-metrics-auth-role` and `opm-operator-leader-election-role` (the manifest has `...-rolebinding`). The Deployment selector differs from the manifest's and is immutable, so moving from the manifest needs a Deployment recreate (the migration change's job).
- The four CRDs rendered byte-equal to the controller's generated specs (CEL rules, list-map keys, status subresource, printer columns). The CRDs do not vary with the instance, so the module is a cluster singleton; with instance `opm-operator` in `opm-operator-system` every other name reproduces the manifest's, the Deployment being `opm-operator-controller-manager`.
- `#config` knobs rendered into the Deployment: image repository (the image carried a tag, no digest), `replicas: 2`, `--registry=...` and `--default-service-account=...` as controller arguments, extra arguments after them, resources.
- Cost of `opm module build`: cold cache 5.66 s wall and 507 MB peak RSS; warm 2.0 to 3.9 s and 494 to 510 MB.
- The local test registry accepted a re-push of a published version with other bytes. This change does not guard against that (Non-Goals); it is the same exposure every registry module has.

**Experiment 02, the install.** A shell script emulated this change's install (`opm instance build`, pick the 4 CRDs, `kubectl apply --server-side --field-manager=opm-cli` and wait `Established`, then `opm instance apply --wait`) on a fresh kind v0.32.0 cluster (node v1.36.1), with module versions 0.1.0 and 0.2.0 published to a local registry.

1. **Bootstrap.** The instance apply applied the Namespace first and then the rest: `applied 19 resources successfully (15 created, 4 unchanged)`, the CRDs unchanged because the CRD step had just applied them, and recorded all 19 objects, CRDs and Namespace included. With `--create-namespace` the CLI created the Namespace without OPM labels and the existence check then refused the module's own Namespace (`already exists and is not managed by OPM`).
2. **The operator left its own instance alone.** `spec.owner: cli`, no finalizer; the operator's only write was the status condition `Ready=Unknown`, reason `ManagedExternally`.
3. **End to end.** After the Platform seed (a plain create as `opm-cli`), Platform `cluster` reached Ready from operator v1.0.0-beta.5 and an operator-owned fixture reconciled.
4. **Reinstall.** All 19 objects kept uid and resourceVersion (`19 unchanged`), and the render digest was unchanged. But the second render went through the now-seeded cluster Platform, matching only because the Platform subscribed to the catalog the module pins.
5. **Upgrade 0.1.0 to 0.2.0** was a re-apply: `15 configured, 4 unchanged`, one new ReplicaSet, 13.9 s for apply plus `--wait`. The new pod started at 10:06:37, its controllers at 10:07:05 (28 s later); the Platform regenerated at 10:07:07. `opm instance apply --wait` therefore reported healthy about 30 s before the new operator reconciled anything, and a workload reconcile in that window reported `PlatformNotReady`.
6. **Over a manifest-installed operator** the first apply refused at the existence check on the Namespace, after the emulated CRD step had already relabelled the four CRDs. Forcing past it needed a Deployment delete (immutable selector) and left the three `*-rolebinding` objects uninventoried.
7. **Delete.** `opm instance delete opm-operator` deleted 14 objects and left the Namespace and the 4 CRDs, without checking the cleanup finalizer of an armed fixture; re-running the install (`14 created, 5 unchanged`) brought the operator back and it finished the fixture's cleanup 18 s later.
8. **Self-ownership.** Flipping the instance to `owner: operator` either failed closed but armed the finalizer (shipped RBAC) or, with cluster-admin, produced an instance the CLI could not take back and whose delete pruned the operator mid-cleanup; recovery needed a manual finalizer strip.

Install time, fresh cluster each, image through a mirror: module install 31.8 s, 28.6 s and 30.2 s (render plus CRDs 2.3 to 3.5 s), against 12.8 s (warm-up outlier), 28.7 s and 30.7 s for today's install. Readiness (image pull, probes) dominates both; the module path adds about 2.5 s of render.

The kind nodes on that host had no internet egress; the operator reached core and the catalog through two plain host-network pull-through mirrors set through the instance's registry value, which is also the evidence that the mirror path serves the operator's module fetches unchanged.

## Risks / Trade-offs

- [Install needs a registry; a fresh air-gapped cluster cannot install from the binary alone] → the mirror path (experiment 02 ran its operator through two plain pull-through mirrors); the error for an unreachable registry names the module, the registry mapping and `--registry`/config as the fix.
- [A cli release between this change and the migration refuses on every manifest-installed cluster] → decided release coupling in proposal.md: both changes ship in the same cli release, and the cli release PR is held until the migration merges; the refusal changes nothing and names the cause.
- [The registry may serve other bytes under the pinned module version] → accepted, as for every module the CLI installs; content-digest anchoring is a deferred follow-up (Non-Goals).
- [An older CLI re-running install can downgrade the operator, and a module can drop a served CRD version] → as today; refusals for both are a deferred follow-up (Non-Goals). V1 still stops installing an operator newer than the CLI.
- [Install returns before the new operator reconciles (28 s gap in experiment 02)] → as today; the Platform-Ready wait is a deferred follow-up (Non-Goals).
- [`PinnedOperatorVersion` keeps its name with a narrower meaning] → doc comment and AGENTS.md state it; renaming would collide with `join-release-cascade`'s edits for no reader benefit.
- [The pre-write checks read the cluster and the write happens later (TOCTOU)] → the same window every instance apply has.

## Migration Plan

- Users: the next beta's migration note says install needs a registry or mirror, `--version` takes a module version (the operator version it deploys is printed), settings move from Deployment patches to `-f` values, and uninstall needs an install-recorded operator. The first install on an existing cluster is `migrate-manifest-installed-operator`'s.
- Rollback: re-run install with the previous module version (`--version`); returning to a cli that embeds the manifest means deleting the operator Deployment first, because the module's Deployment selector differs from the manifest's and is immutable (Evidence, render), and then running the older cli's install; the three manifest bindings come back beside the module's.

## Open Questions

None blocking. Section 1 settles how the module's `operator` package is read (kernel-acquired source or a `modregistry` zip fetch) and whether the shared cascade resolver can list module releases.
