# CLI Deploy Commands

## Purpose

The `opm mod apply` and `opm mod delete` commands manage the deployment lifecycle of OPM modules on Kubernetes clusters. `mod apply` renders the module via the Pipeline interface and applies resources using server-side apply. `mod delete` discovers and removes module resources via OPM labels without requiring the original source.

## Design Rationale

1. **Go API integration**: Commands call `build.NewPipeline().Render()` directly, not subprocess.
2. **Label-based discovery**: `mod delete` discovers resources via labels, not re-rendering.
3. **Server-side apply**: Use SSA with force for idempotent operations.
4. **Weighted ordering**: Resources applied/deleted in weight order for dependency handling.

## Dependencies

- **render-pipeline-v1**: Consumes Pipeline interface, RenderResult, Resource types
- **build-v1**: Uses Pipeline implementation

---

## User Stories

### User Story 1 - Deploy Module to Kubernetes (Priority: P1)

A developer wants to deploy their rendered module to a Kubernetes cluster.

**Independent Test**: Given a valid module, `opm mod apply` deploys resources successfully.

**Acceptance Scenarios**:

1. **Given** a valid module, **When** running `opm mod apply`, **Then** resources are deployed.
2. **Given** a deployed module, **When** running `opm mod delete`, **Then** all resources are removed.
3. **Given** a module with CRDs and CRs, **When** running `opm mod apply`, **Then** CRDs are created first.
4. **Given** pending changes, **When** running `opm mod apply`, **Then** changes are applied.
5. **Given** dry-run request, **When** running `opm mod apply --dry-run`, **Then** no changes are made.

### User Story 2 - Delete Module Without Source (Priority: P2)

A developer wants to delete a deployed module after deleting the source files.

**Independent Test**: Deploy module, delete source, `opm mod delete` still works.

**Acceptance Scenarios**:

1. **Given** a deployed module, **When** source is deleted and `opm mod delete -n <ns> --name <name>` runs, **Then** resources are removed.
2. **Given** delete request, **When** running `opm mod delete --dry-run`, **Then** resources to delete are listed but not removed.

---

## Functional Requirements

### mod apply

| ID | Requirement |
|----|-------------|
| FR-D-001 | `mod apply` MUST call Pipeline.Render() to get resources. |
| FR-D-002 | `mod apply` MUST use server-side apply with force for conflicts. |
| FR-D-003 | `mod apply` MUST apply resources in ascending weight order. |
| FR-D-004 | `mod apply` MUST add OPM labels to all resources. |
| FR-D-005 | `mod apply` MUST support `--dry-run` (server-side). |
| FR-D-006 | `mod apply` MUST support `--wait` with timeout. |
| FR-D-007 | `mod apply` MUST support `--values` / `-f` (repeatable). |
| FR-D-008 | `mod apply` MUST support `--namespace` / `-n`. |
| FR-D-009 | `mod apply` MUST log warning on field conflicts. |
| FR-D-010 | `mod apply` MUST fail fast if render has errors. |

### mod delete

| ID | Requirement |
|----|-------------|
| FR-D-020 | `mod delete` MUST discover resources via OPM labels. |
| FR-D-021 | `mod delete` MUST NOT require module source. |
| FR-D-022 | `mod delete` MUST delete in descending weight order. |
| FR-D-023 | `mod delete` MUST support `--force` to skip confirmation. |
| FR-D-024 | `mod delete` MUST support `--dry-run` to preview. |
| FR-D-025 | `mod delete` MUST require at least one of `--name` or `--release-id` for identification. The `--namespace` / `-n` flag remains required in all cases. |
| FR-D-026 | `mod delete` MUST prompt for confirmation (unless --force). |
| FR-D-027 | `mod delete` MUST support `--release-id` flag for discovery by release identity UUID. |
| FR-D-028 | `mod delete` MUST use dual-strategy discovery (both release-id and name+namespace selectors) when both are available, returning the union of results deduplicated by resource UID. |

### Kubernetes Integration

| ID | Requirement |
|----|-------------|
| FR-D-050 | MUST use client-go with default rate limiting. |
| FR-D-051 | MUST support kubeconfig from flag, env, or default path. |
| FR-D-052 | MUST support context selection via flag. |
| FR-D-053 | MUST fail fast with clear error on connectivity issues. |

### Resource Labeling

| ID | Requirement |
|----|-------------|
| FR-D-060 | All resources MUST have `app.kubernetes.io/managed-by: open-platform-model`. |
| FR-D-061 | All resources MUST have `module.opmodel.dev/name: <name>`. |
| FR-D-062 | ~~All resources MUST have `module.opmodel.dev/namespace: <ns>`.~~ Removed — namespace scoping is handled by K8s API calls, not labels. |
| FR-D-063 | All resources MUST have `module.opmodel.dev/version: <version>`. |
| FR-D-064 | All resources MUST have `component.opmodel.dev/name: <component>`. |
| FR-D-065 | All resources MUST have `module-release.opmodel.dev/uuid: <release-uuid>` when the release identity is available. |
| FR-D-066 | All resources MUST have `module.opmodel.dev/uuid: <module-uuid>` when the module identity is available. |

---

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-D-001 | `mod apply` MUST be idempotent. |
| NFR-D-003 | No enforced limits on module complexity. |

---

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-D-001 | New user can init, build, apply in under 3 minutes. |
| SC-D-003 | `mod apply` is fully idempotent. |

---

## Edge Cases

| Case | Handling |
|------|----------|
| Render errors | `mod apply` fails before touching cluster |
| Cluster unreachable | Fail fast with connectivity error |
| RBAC denied | Pass through Kubernetes API error |
| Field conflict | Log warning, take ownership |
| Module source deleted | `mod delete` works via labels |
| Empty RenderResult | Success (nothing to apply) |
| Delete with neither --name nor --release-id | Return validation error: "either --name or --release-id is required" |
| Catalog without identity support | Identity labels omitted; existing labeling unchanged |

---

## Command Syntax

### mod apply

```text
opm mod apply [path] [flags]

Arguments:
  path    Path to module directory (default: .)

Flags:
  -f, --values strings      Additional values files (can be repeated)
  -n, --namespace string    Target namespace
      --name string         Release name (default: module name)
      --provider string     Provider to use
      --dry-run             Server-side dry run
      --wait                Wait for resources to be ready
      --timeout duration    Wait timeout (default: 5m)
      --kubeconfig string   Path to kubeconfig
      --context string      Kubernetes context
```

### mod delete

```text
opm mod delete [flags]

Flags:
  -n, --namespace string    Target namespace (required)
      --name string         Module name (required if --release-id not provided)
      --release-id string   Release identity UUID (required if --name not provided)
      --force               Skip confirmation prompt
      --dry-run             Preview without deleting
      --wait                Wait for resources to be deleted
      --kubeconfig string   Path to kubeconfig
      --context string      Kubernetes context
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Usage error |
| 2 | Render error |
| 3 | Kubernetes error |
