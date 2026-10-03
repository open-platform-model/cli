## Why

The CLI can delete objects whose loss cascades far past the instance that owns them.

- **Prune deletes CRDs.** `PruneStaleResources` (`internal/inventory/stale.go:102-149`) skips a core `Namespace` and nothing else. When a module stops rendering a `CustomResourceDefinition`, the next `opm instance apply` or `opm module apply` deletes it, and the API server deletes every custom resource of that kind in the cluster with it, including resources other instances and other tools own.
- **The prune preview hides what it skips.** `previewPrune` (`internal/workflow/apply/apply.go:208-216`) drops a stale `Namespace` from the list without saying so, so a dry run does not show that the object will be left on the cluster.
- **Instance delete deletes everything it tracks.** `kubernetes.Delete` (`internal/kubernetes/delete.go`) issues a foreground delete for every live inventory object, `Namespace` and `CustomResourceDefinition` included. Deleting a Namespace deletes everything in it, including other instances' resources.
- **Instance delete trusts a stale read.** The objects it deletes were read by `query.ResolveInventory` before the loop. It does not check that an object is still managed by OPM and still carries this instance's UUID before deleting it. The operator does both checks before every prune and instance-delete (`opm-operator/internal/apply/prune.go`), so the two frontends delete differently today.
- **The first-install refusal promises a flag that does not work.** `PreApplyExistenceCheck` refuses an existing object that OPM does not manage with "use --force to proceed", but nothing reads `--force` on that path, so the hint sends the user to a flag that changes nothing.

## What Changes

- **One protected-kind predicate.** `kubernetes.IsProtectedKind(group, kind)` is true for `Namespace` in the core group and `CustomResourceDefinition` in `apiextensions.k8s.io`. Prune, the prune preview and instance delete all use it. There is no override flag.
- **Prune leaves protected kinds behind and says so.** A stale CRD or Namespace is not deleted. Apply lists each one as `left behind`, and the dry-run preview lists it the same way next to its `would prune` lines. Like a stale Namespace today, a left-behind object drops out of the recorded inventory, so later applies no longer track it.
- **Instance delete leaves protected kinds behind and re-checks ownership.** Delete never deletes a CRD or a Namespace. Before each delete it GETs the live object again and skips it unless its `app.kubernetes.io/managed-by` label is an OPM value and its `module-instance.opmodel.dev/uuid` label matches the instance's recorded UUID, with the same tolerances as the operator. Each skipped object is listed as `left behind` with its reason, in the dry run and in the real run. A re-read that fails with anything but NotFound fails that object and keeps the `ModuleInstance` for a re-run; a read that already failed during discovery still drops the entry silently, as today (cli issue #283). A left-behind object is not an error: the `ModuleInstance` is still deleted last and the command exits 0. The closing line counts what was left behind.
- **The first-install refusal drops the `--force` text.** The message says what to do instead (remove or rename the existing object, or render a different name).
- `opm instance delete --help` states that CRDs and Namespaces are never deleted and that objects no longer owned by the instance are left behind. The generated command reference is regenerated.

**Behaviour change for users.** `opm instance delete` no longer deletes a Namespace or a CRD that the instance rendered, and `opm instance apply` / `opm module apply` no longer prune a CRD that a module stopped rendering. Both are listed as left behind; remove them with `kubectl delete` when nothing else needs them. Instance delete also skips an object whose labels show it no longer belongs to the instance. The section 1 and section 2 commit bodies carry this wording, but under the repo's squash settings (title from the PR title, blank body) only the PR title reaches `main` and CHANGELOG.md. The PR title is therefore the changelog carrier: `fix: never delete CRDs or Namespaces on prune or instance delete; list them as left behind`, and this paragraph goes into the draft GitHub release notes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `apply-pruning`: prune and its dry-run preview never delete a CRD or a Namespace, and list each as left behind; the first-install refusal no longer names `--force`.
- `deploy`: `opm instance delete` never deletes a CRD or a Namespace, re-reads each object and deletes it only while it is still OPM-managed and carries the instance's UUID, and lists every skipped object as left behind.

## Impact

- **Release class: `fix`, PATCH (after GA as well), shipped as the next beta.N.** No flag, command or exit code is added or removed. The change is a safety fix that makes the CLI match the operator, which already never deletes these kinds. It is deliberately not a `feat!`: no input that worked stops working, but the delete outcome changes, so the PR title and the draft release notes carry the changelog wording above (the commit bodies repeat it for `git log` readers).
- Commands: `opm instance apply`, `opm module apply` (prune and dry-run preview), `opm instance delete` (CLI-owned path; the operator-owned path is unchanged), and every first-install apply for the refusal wording.
- Packages: `internal/kubernetes` (`IsProtectedKind`, `delete.go`), `internal/inventory` (`stale.go`), `internal/workflow/apply` (`apply.go`), `internal/cmd/instance` (`delete.go`), `internal/output` (a `left behind` status), `docs/site/reference/cli/` (regenerated).
- Tests: `internal/kubernetes/delete_test.go` (the existing tracked-object fixture gains OPM labels), `internal/inventory/stale_test.go`, `internal/workflow/apply/dryrunprune_test.go`, `internal/cmd/instance/delete_test.go`; the integration programs under `tests/integration/` that call `kubernetes.Delete` and `inventory.PruneStaleResources`.
- Out of scope: a live ownership re-check in prune and the shared `CanApply`/`CanDelete` verdicts with the adopt annotation (the later ownership change for both frontends), and any change to the operator.
