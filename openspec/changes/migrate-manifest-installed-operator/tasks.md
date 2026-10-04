## 1. Gate and spike: the install flow to plug into, and client-side field ownership

- [ ] 1.1 Gate: confirm that change `install-operator-from-module` is merged on `origin/main`: its archived change directory exists under `openspec/changes/archive/` on `origin/main` and its implementation is in the tree. If it is not, stop and report, and do not start 1.2. If it is, rebase this branch onto `origin/main`
- [ ] 1.2 Read the merged install flow. In design.md, "Flow", write the concrete names of its check phase, its CRD step and its instance-apply call, and how it reaches `apply.RunPreApplyExistenceCheck`. Confirm where the ownership move fits before the CRD step. If the integration points differ from the sketch, adjust sections 3 and 4
- [ ] 1.3 Spike on a throwaway kind cluster; no operator image is needed:
  - client-side `kubectl apply` the v1.0.0-beta.5 `install.yaml`;
  - send `csaupgrade.UpgradeManagedFieldsPatch` for `kubectl-client-side-apply` -> `opm-cli` on the Namespace, a ClusterRole, the Service and the `moduleinstances.opmodel.dev` CRD;
  - server-side-apply as `opm-cli` (force) module-shaped objects without the kustomize labels;
  - record whether `app.kubernetes.io/managed-by=kustomize`, `control-plane` and `kubectl.kubernetes.io/last-applied-configuration` are gone;
  - repeat over an `opm-cli` server-side install of the same manifest.

  Write the result into design.md, "Field ownership of client-side installs". If the labels or the annotation survive, switch the design there to the named-field JSON-patch option before section 4
- [ ] 1.4 `task lint` and `task test` green, then commit `docs(openspec): record the operator migration integration points and ownership spike`

## 2. The proof list of earlier operator manifests

- [ ] 2.1 Add `hack/operator-legacy/main.go` (`go run`, not linked into `opm`). It downloads `install.yaml` from every opm-operator release that has that asset and writes `internal/operator/testdata/legacy-manifests.json` with the group, kind, namespace, name, labels and (for the Deployment) selector per release, and no specs. Run it once and commit the output
- [ ] 2.2 Add `internal/operator/legacy.go` with `LegacyObject`, `LegacyObjects` (the 29-entry union, each entry with the release range that shipped it) and `SupersededBindings` (the three `*-rolebinding` entries, each with the module binding that replaces it)
- [ ] 2.3 Add `internal/operator/legacy_test.go`, offline. It checks that `LegacyObjects` equals the union in `testdata/legacy-manifests.json`, that every listed object keeps one label set and the Deployment one selector across releases, and that the only bindings on the list are the three superseded ones
- [ ] 2.4 `task lint` and `task test` green, then commit `feat(operator): carry the proof list of earlier operator manifests`

## 3. Proof and plan, refusing in the check phase

- [ ] 3.1 Add `internal/operator/migration/proof.go`: `Prove(live, want, instanceUUID)` returns `Absent`, `Proven`, `Ours` or `Unproven` with a reason. Add table tests for:
  - a proven object;
  - an added label (still proven);
  - a CRD with no listed labels;
  - a missing or changed listed label;
  - this instance's identity (`Ours`);
  - another instance's identity;
  - a partial identity
- [ ] 3.2 Add `internal/operator/migration/plan.go`. `Plan(ctx, client, rendered, instance)` GETs every list entry and every rendered object and returns either a plan or a `RefusalError` naming every unproven listed object. The plan holds:
  - `Adopt`: proven and rendered;
  - `MoveOwnership`: adopted objects with a `kubectl-client-side-apply` manager;
  - `RecreateDeployment`: proven, with the listed selector;
  - `DeleteBindings`: the proven superseded bindings;
  - `LeftInPlace`: proven, not rendered, and not a binding.

  A GET error other than NotFound refuses with exit 4 on Forbidden, 3 on a connectivity error, else 1. Fake dynamic client tests:
  - a fresh cluster gives an empty plan;
  - an opm-cli origin;
  - a client-side origin;
  - a module-rendered kubectl install gives `Ours` everywhere and an empty plan;
  - a pre-v1 cluster with leftovers;
  - an unproven binding;
  - an unreadable entry;
  - a resumed run with the Deployment and one binding gone
- [ ] 3.3 Call `Plan` in the install check phase found in 1.2, after the other refusing checks and before the first write, and return its refusal there. The plan is not executed yet, so the guard still refuses a manifest-installed cluster as before, now naming every unproven object when there are any. Test that a refusal leaves every object's `resourceVersion` unchanged
- [ ] 3.4 `task lint` and `task test` green, then commit `feat(operator): prove the objects of an earlier operator manifest before install writes`

## 4. Execute the migration and admit the proven objects

- [ ] 4.1 In `internal/inventory/stale.go`, `PreApplyExistenceCheck` takes an `AdmitSet`. An admitted entry passes the managed-by test, and every existing caller passes `nil`. `apply.Request` carries the set to `RunPreApplyExistenceCheck`. Tests: an admitted kustomize-labelled object passes, an unadmitted one is refused, and `nil` keeps today's behaviour
- [ ] 4.2 Add `internal/operator/migration/execute.go` with two functions:
  - `MoveOwnership(ctx, client, plan)` sends the ownership patches;
  - `Delete(ctx, client, plan, budget)` deletes `DeleteBindings`, then deletes the Deployment with foreground propagation and waits with `kubernetes.WaitAbsent`.

  Every step treats NotFound as done. An error after the first write wraps the re-run hint and exits 1. Fake client tests for the order, for idempotence on a second call, and for no write to any object outside the plan
- [ ] 4.3 Wire the migration into the install write phase: `MoveOwnership` before the CRD step, and `Delete` after the CRD step and before the instance apply. Pass `plan.Admit()` (`Adopt` plus `Ours`) to the instance apply's guard
- [ ] 4.4 `--crds-only` computes no migration and calls neither function. Test that it leaves the earlier Deployment and bindings untouched
- [ ] 4.5 Report: print the migration lines from design.md, "Report", before the ordinary apply lines, and nothing when the plan is empty. Add a golden-output test for a full migration, for a pre-v1 cluster with leftovers, and for a refusal naming two objects
- [ ] 4.6 `task lint` and `task test` green, then commit `feat(operator): migrate an operator installed from an earlier release manifest`

## 5. Measure the migration on a cluster

- [ ] 5.1 Add `tests/integration/operator-migration/main.go` (`//go:build ignore`, context `kind-opm-dev`) and add it to `task test:integration`. It drives `migration.Plan`, the two write functions and the instance apply against objects applied from a copy of the v1.0.0-beta.5 manifest in its testdata, in three runs:
  - an `opm-cli` server-side origin;
  - a client-side `kubectl apply` origin;
  - an interrupted run, cancelled after the first binding delete and then re-run.

  It asserts that:
  - every rendered object is recorded;
  - every uid is kept except the Deployment's;
  - the three bindings are gone;
  - the earlier labels and `last-applied-configuration` are gone;
  - the CRDs' spec, and a seeded ModuleInstance and its workload, are unchanged (`uid`, `resourceVersion`)
- [ ] 5.2 Run 5.1 on `kind-opm-dev`. Then measure with the real binary: `opm operator install` of the pinned operator module over a v1.0.0-beta.5 operator installed by the last manifest-installing CLI release, again over a client-side `kubectl apply` install, and an interrupted run completed by re-running install. Afterwards every CRD, custom resource and managed workload must be unchanged and the operator must be reconciling. The cluster needs the operator image; when the cluster has no egress, use a single-platform image with `kind load docker-image` or a host-network pull-through mirror. Record the timings, the controller gap and the object outcomes in design.md under a new "Measured" heading
- [ ] 5.3 `task lint` and `task test` green, then commit `test(operator): measure the operator migration over both install origins and an interrupted run`
