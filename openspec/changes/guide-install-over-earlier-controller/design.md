## Context

The refusal comes from the apply guard (`internal/controller/plan_install.go`, `applyGuard`), wrapped as a `GuardError` and mapped to exit 2 by `installError` (`internal/cmd/controller/install.go`). It judges the objects the install applies, and only the four CRDs can collide with the earlier install: the new Namespace is `opm-system` and the new Deployment is `opm-controller-manager` (`internal/controller/names.go`), the earlier ones are different names.

What the guard says, by how the earlier install was made (read from `library` `opm/k8s/ownership/apply.go`, `CanApply`):

- Installed from the operator release manifest `install.yaml`: the CRDs carry no OPM labels, so the message is "exists and is not managed by OPM" with the adopt annotation to set.
- Installed by `opm operator install` (the operator module): the CRDs carry the managed-by label and the UUID of the earlier instance, so the message is "belongs to module instance <uuid>" with the same annotation.

Both exit 2. Neither message says that a controller is running.

The names of the earlier install, from the `opm-controller` repository at `1f0e247d^` (the commit before the rename), read only:

| Object | Name | Source |
| --- | --- | --- |
| Namespace | `opm-operator-system` | `dist/install.yaml`, first document; cli `internal/operator/names.go` at `105cadee^` (`OperatorNamespace`) |
| Deployment | `opm-operator-controller-manager`, in that namespace | `dist/install.yaml` (kind Deployment); same cli file (`ControllerDeploymentName`) |
| CRDs | the four names above | `dist/install.yaml` |
| ClusterRoles | `opm-operator-manager-role`, `opm-operator-metrics-auth-role`, `opm-operator-metrics-reader`, and the `opm-operator-*-admin-role`, `*-editor-role`, `*-viewer-role` of the CRDs | `dist/install.yaml` |
| ClusterRoleBindings | `opm-operator-manager-rolebinding`, `opm-operator-metrics-auth-rolebinding` | `dist/install.yaml` |

## Research & Decisions

### How the hint knows it is the earlier install

**Context**: The hint must not print where no earlier controller is: it keys on the Deployment, not on the cause of a refusal.
**Explored**: The CRD labels (manifest CRDs have none, module CRDs have the earlier instance UUID, which the CLI cannot tell from any other instance); the record `opm-operator` in `opm-operator-system` (absent for a manifest install, and the install does not read that coordinate); the Deployment.
**Options considered**:
1. Read the CRD labels. Fails for a manifest install, which has none.
2. Read the old record. Fails for a manifest install, and adds a second record read.
3. Get the Deployment `opm-operator-controller-manager` in `opm-operator-system`. Both install forms keep that name (the cli's own running-controller check relied on it for both), and the new install never writes it.
**Decision**: Option 3. When that Deployment exists, the install tells the user: a `hint:` line after an apply-guard refusal, a warning when the checks pass and the install goes on. When the read fails for any reason (not found, forbidden, other), print nothing.
**Rationale**: It is one read, observable from the cluster, and it states a fact: a controller of the earlier install is there. It cannot be fully reliable: a cluster that deleted the Deployment but kept the CRDs gets no hint, and any other Deployment with that name in that namespace gets one. Both are acceptable: the hint is advice, never a gate. A user who set the adopt annotation on the CRDs before removing the earlier controller is not refused, and now gets the warning, so two controllers do not run unannounced (owner ruling).

### Where the line goes

**Decision**: On a successful path the command prints the same text as one warning on stderr (the repo's warning output) before the first write; the exit code is unchanged. On a refusal the command wraps the refusal in an error whose text is the refusal followed by one `hint:` line. `errors.As` still finds the `GuardError`, and the exit code stays 2. The hint says where to look and does not tell the user to delete an object, which the requirement "Every refusal of the guard exits 2 and names the object, the instance and the annotation" forbids for a refusal. The deletion steps are on the page.
**Hint and warning text** (one line, same words; `opm-operator-system` and `opm-operator-controller-manager` are the old names):

```text
hint: the cluster still runs the controller of the earlier install (Deployment opm-operator-controller-manager in namespace opm-operator-system); this install does not migrate it, and the opm docs page "Install over the earlier controller" says how to move to this one
```

```go
// in internal/controller
func EarlierInstallHint(ctx context.Context, c *kubernetes.Client) string // "" when absent or unreadable
```

### What the page tells the user to do with the CRDs

**Context**: Deleting a CRD deletes every ModuleInstance of that kind. The earlier refusal text "remove or rename these objects" was retired for that reason.
**Options considered**:
1. Delete the four CRDs. Starts clean, and destroys every instance and its record.
2. Keep the CRDs, delete the earlier controller, annotate the four CRDs with the `opmodel.dev/adopt` value that the refusal prints, install. Keeps the instances.
**Decision**: The page teaches option 2 as the path, and shows option 1 behind a warning that it deletes every object of that kind, for a cluster with nothing to keep. Confirmed at the proposal gate. The owner decided "not migrated", which both options respect.

### The Deployment selector

The new Deployment has another name and namespace than the earlier one, so the install never applies over it. The page makes no claim about its selector; it says only to delete the earlier Deployment.

### Uninstall

`opm controller uninstall` reads the record `opm-controller` in `opm-system` and deletes nothing without one (requirement "Uninstall without an instance record deletes nothing"). An earlier install has no such record, so the page says to use `kubectl`. No code changes.

## Risks

- The hint can print for a Deployment of that name that is not the earlier controller. It states the name, so the user can see.
- The probe adds one read of one Deployment on every install. A denied read prints nothing.
- The page names old objects; if the earlier release manifests changed names between releases, a name could be missing. Task 2.1 checks the list against the release manifests that exist as tags.
