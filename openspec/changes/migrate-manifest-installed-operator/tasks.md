# Tasks: migrate-manifest-installed-operator

One PR, titled with the highest class among the section commits. The release coupling is decided (proposal.md): this change ships in the same cli release as `install-operator-from-module`, whose merge holds the cli release PR; this change's PR body says the held release PR may proceed once it merges. Workers report that to the supervisor and never touch the release PR, never tag and never publish anything.

## 1. Gate and spike: the install flow to plug into, and client-side field ownership

- [x] 1.1 Gate: merge `origin/feat/install-operator-from-module` (cli PR #307, change `install-operator-from-module` archived on that branch) into this branch with a merge commit, since the two ship in one cli release and this change's PR stacks on #307. Confirm enhancements PR #94 is merged with 0012:D8:R3/R6/R7, 0012:D1:R4 and 0012:D4:R1 reading as this change cites them. If either fails, stop and report. Add `enhancement.yaml` to this change declaring enhancement `0012` with no whole decision (it delivers 0012:D8:R6 and 0012:D8:R7, and the schema allows a decision number only for a decision delivered whole), and `Source:` citations: `0012:D8:R6` in "Install adopts the proven objects the module renders and no others" and in the apply-pruning admission-set requirement; `0012:D8:R7` in "Install recreates the earlier Deployment once", "Install deletes the superseded role bindings" and "The migration deletes only proven objects, and refuses otherwise". Verify with `openspec validate migrate-manifest-installed-operator --strict`
- [x] 1.2 Read the merged install flow. In design.md, "Flow", write the concrete names of its check phase (`PlanInstall`, the proof slot before its guard, and its `inventory.PreApplyExistenceCheck` call), its CRD step and its instance-apply call, and how that call reaches `apply.RunPreApplyExistenceCheck`. Confirm that the instance apply applies the CRDs again as `opm-cli` with force after the CRD step; the ownership move after the CRD step relies on it. If the integration points differ from the sketch, adjust sections 3 and 4
- [x] 1.3 Spike on a throwaway kind cluster; no operator image is needed:
  - client-side `kubectl apply` the v1.0.0-beta.5 `install.yaml`;
  - server-side-apply as `opm-cli` (force) the module-shaped CRD, standing in for the CRD step;
  - send `csaupgrade.UpgradeManagedFieldsPatch` for `kubectl-client-side-apply` -> `opm-cli` on the Namespace, a ClusterRole, the Service and the `moduleinstances.opmodel.dev` CRD;
  - server-side-apply as `opm-cli` (force) module-shaped objects without the kustomize labels, the CRD included, standing in for the instance apply;
  - record whether `app.kubernetes.io/managed-by=kustomize`, `control-plane` and `kubectl.kubernetes.io/last-applied-configuration` are gone;
  - repeat over an `opm-cli` server-side install of the same manifest.

  Write the result into design.md, "Field ownership of client-side installs". If the labels or the annotation survive, switch the design there to the named-field JSON-patch option before section 4
- [x] 1.4 `task fmt`, `task lint`, `task test` and `task openspec:check` green, then commit `docs(openspec): record the operator migration integration points and ownership spike`

## 2. The proof list of earlier operator manifests

- [x] 2.1 Add `hack/operator-legacy/main.go` (`go run`, not linked into `opm`). It downloads `install.yaml` from every opm-operator operator release (tag `v<semver>`) that has that asset, and from no other release: an operator module release (`opm_operator-vX.Y.Z`) is skipped even though it attaches an `install.yaml`. Put the tag filter in its own function with an offline table test (`v1.0.0-beta.5` and `v0.5.0` read; `opm_operator-v0.1.0`, `opm_operator-v1.0.0` and a malformed tag skipped). If an operator release after v1.0.0-beta.5 attaches a manifest by then, its objects join the union and the counts in design.md and the specs are updated to match. The program writes `internal/operator/testdata/legacy-manifests.json` with the group, kind, namespace, name, labels and (for the Deployment) selector per release, and no specs. Run it once and commit the output
- [x] 2.2 Add `internal/operator/legacy.go` with `LegacyObject`, `LegacyObjects` (the 32-entry union from `FirstLegacyRelease` v0.5.0 to v1.0.0-beta.8, each entry with the release range that shipped it) and `SupersededBindings` (the three `*-rolebinding` entries). No replacement name is hardcoded: the plan finds the replacement in the render by `roleRef` (3.2). Build the operator Namespace and Deployment entries from `OperatorNamespace` and `ControllerDeploymentName` in `internal/operator/names.go`
- [x] 2.3 Add `internal/operator/legacy_test.go`, offline. It checks that `LegacyObjects` equals the union in `testdata/legacy-manifests.json`, that every listed object keeps one label set and the Deployment one selector across releases, and that the only bindings on the list are the three superseded ones
- [x] 2.4 `task fmt`, `task lint`, `task test` and `task openspec:check` green, then commit `feat(operator): carry the proof list of earlier operator manifests`

## 3. Proof and plan, refusing in the check phase

- [x] 3.1 Add `internal/operator/migration_proof.go` (package `operator`, see design.md "Flow"): `ProveLegacy(live, want, instanceUUID)` returns `Absent`, `Proven`, `Ours` or `Unproven` with a reason. Add table tests for:
  - a proven object;
  - an added label (still proven);
  - a CRD with no listed labels;
  - a missing or changed listed label;
  - this instance's identity (`Ours`), including an object an operator module release's `install.yaml` created (the spec scenario "A module release's manifest is not a source of the list");
  - another instance's identity;
  - a partial identity
- [x] 3.2 Add `internal/operator/migration_plan.go`. `PlanMigration(ctx, client, rendered, instanceUUID, crdsOnly)` GETs every list entry and every rendered object and returns either a plan or a `RefusalError` naming every object that blocks it. The plan holds:
  - `Adopt`: proven and rendered;
  - `Ours`: rendered and carrying this instance's identity;
  - `MoveOwnership`: objects in `Adopt` or `Ours` with a `kubectl-client-side-apply` manager;
  - `RecreateDeployment`: proven, with the listed selector;
  - `DeleteBindings`: the proven superseded bindings, each with the rendered binding of the same kind and namespace whose `roleRef` equals the live one;
  - `LeftInPlace`: proven, not rendered, and not a binding.

  The objects that block the plan, all named in one `MigrationRefusalError` (exit 2): an `Unproven` list entry that is rendered, is the Deployment, or is a superseded binding; and a proven superseded binding with no replacement in the render. An `Unproven` list entry outside those sets is neither refused nor reported. A GET error other than NotFound refuses with the exit code `cmdutil.ExitCodeFromK8sError` gives (4 on Forbidden or Unauthorized). Fake dynamic client tests:
  - a fresh cluster gives an empty plan;
  - an opm-cli origin;
  - a client-side origin;
  - a module-rendered kubectl install gives `Ours` everywhere and an empty plan;
  - a pre-v1 cluster with leftovers;
  - an unproven binding;
  - a proven binding with no replacement in the render;
  - an unproven leftover ClusterRole that does not refuse;
  - an unreadable entry, with Forbidden and Unauthorized both exit 4;
  - a resumed run with the Deployment and one binding gone
- [x] 3.3 Call `PlanMigration` in the proof slot of the install check phase found in 1.2: after the other refusing checks and the terminating wait, immediately before the check-phase `inventory.PreApplyExistenceCheck` call, so the proof runs before the guard. If the merged flow does not place the guard last, with only the proof slot between it and the checks before it, in the full flow and in the `--crds-only` flow alike, stop and report to the supervisor. Return the plan's refusal there. The plan is not executed and nothing is admitted yet, so a manifest-installed cluster with every object proven is still refused by the guard as before; one with an unproven object is now refused by the proof, naming every blocking object. Test with a fake client that fails the test on any write verb (create, update, patch, delete, apply) that a refusal sends no write
- [x] 3.4 `task fmt`, `task lint`, `task test` and `task openspec:check` green, then commit `feat(operator): prove the objects of an earlier operator manifest before install writes`

## 4. Execute the migration and admit the proven objects

- [ ] 4.1 Add the comparable key `K8sIdentity{Group, Kind, Namespace, Name}` beside `K8sIdentityEqual` in `pkg/inventory/entry.go`, and `AdmitSet map[K8sIdentity]struct{}`. In `internal/inventory/stale.go`, `PreApplyExistenceCheck` takes an `AdmitSet`. An admitted entry passes the managed-by test only; a terminating admitted entry is still refused. Every existing caller passes `nil`. `apply.Request` carries the set to `RunPreApplyExistenceCheck`. Tests: an admitted kustomize-labelled object passes, an admitted terminating object is refused, an unadmitted one is refused, and `nil` keeps today's behaviour
- [ ] 4.2 Add `internal/operator/migration_execute.go` with two functions:
  - `MoveOwnership(ctx, client, plan)` sends the ownership patches;
  - `Delete(ctx, client, plan, budget)` deletes the Deployment with foreground propagation and waits with `kubernetes.WaitAbsent`, then deletes `DeleteBindings`.

  Every step treats NotFound as done. An error after the first write wraps the re-run hint and exits 1. Fake client tests for the order (Deployment before bindings), for idempotence on a second call, for no write to any object outside the plan, and for an interrupted run: a reactor that fails the second binding delete, then a recomputed plan and a second call that completes it
- [ ] 4.3 Wire the migration into the install write phase: `MoveOwnership`, then `Delete`, both after the CRD step and before the instance apply. Pass `plan.Admit()` (`Adopt` plus `Ours`) to both guard calls: the check-phase `PreApplyExistenceCheck` in `PlanInstall` and the instance apply's `RunPreApplyExistenceCheck`. Tests: `PlanInstall` admits proven kustomize-labelled objects on a fake client holding the beta.5 manifest's objects, and the write order is CRD step, ownership moves, Deployment delete, binding deletes, instance apply
- [ ] 4.4 `--crds-only` runs the proof over the CRD entries only, admits the proven CRDs through its guard and refuses an unproven one, calls neither write function and prints no migration lines. Tests: over the beta.5 manifest's objects it applies the CRDs and leaves the earlier Deployment and bindings without any write; an unproven CRD refuses; a full install after it migrates the rest and records the CRDs
- [ ] 4.5 Report: print the migration lines from design.md, "Report", before the ordinary apply lines, and only when the plan adopts, recreates or deletes something; `left` lines print only on such a run. Add a golden-output test for a full migration, for a pre-v1 cluster with leftovers, for a re-run after a completed migration with leftovers present (no migration lines), and for a refusal naming two objects
- [ ] 4.6 In `README.md`, operator section, add the migration note from design.md, "Migration note": the first install over a manifest-installed operator recreates the Deployment and loses its patches (pass them as `-f` values), the controller pauses briefly, and an older CLI cannot reinstall over a migrated cluster
- [ ] 4.7 `task fmt`, `task lint`, `task test` and `task openspec:check` green, then commit `feat(operator): migrate an operator installed from an earlier release manifest`

## 5. Measure the migration on a cluster

- [ ] 5.1 Add `tests/integration/operator-migration/main.go` (`//go:build ignore`, context `kind-opm-dev`) and add it to `task test:integration`. It drives `migration.Plan`, the two write functions and the instance apply against objects applied from a copy of the v1.0.0-beta.5 manifest in its testdata, in three runs:
  - an `opm-cli` server-side origin;
  - a client-side `kubectl apply` origin;
  - an interrupted run: the Deployment delete and one binding delete executed from a partial plan, then a full re-run.

  It asserts that:
  - every rendered object is recorded;
  - every uid is kept except the Deployment's;
  - the three bindings are gone;
  - the earlier labels and `last-applied-configuration` are gone;
  - the CRDs' spec, and a seeded ModuleInstance and its workload, are unchanged (`uid`, `resourceVersion`)
- [ ] 5.2 Run 5.1 on `kind-opm-dev`. Then measure with the real binary: `opm operator install` of the pinned operator module over a v1.0.0-beta.5 operator installed by the last manifest-installing CLI release, again over a client-side `kubectl apply` install, and an interrupted run completed by re-running install. Afterwards every CRD, custom resource and managed workload must be unchanged and the operator must be reconciling. The cluster needs the operator image; when the cluster has no egress, use a single-platform image with `kind load docker-image` or a host-network pull-through mirror. Record the timings, the controller gap and the object outcomes in design.md under a new "Measured" heading
- [ ] 5.3 `task fmt`, `task lint`, `task test` and `task openspec:check` green, then commit `test(operator): measure the operator migration over both install origins and an interrupted run`
