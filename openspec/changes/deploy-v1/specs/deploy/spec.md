# Delta Spec: CLI Deploy Commands

## Overview

This delta adds deployment lifecycle commands to the OPM CLI: `mod apply`, `mod delete`, `mod diff`, and `mod status`. These commands consume the Pipeline interface from render-pipeline-v1 (implemented by build-v1) and manage module resources on Kubernetes.

## Design Decisions

1. **Go API integration**: Commands call `build.NewPipeline().Render()` directly, not subprocess.
2. **Label-based discovery**: `mod delete` and `mod status` discover resources via labels, not re-rendering.
3. **Server-side apply**: Use SSA with force for idempotent operations.
4. **Weighted ordering**: Resources applied/deleted in weight order for dependency handling.

## Dependencies

- **render-pipeline-v1**: Consumes Pipeline interface, RenderResult, Resource types
- **build-v1**: Uses Pipeline implementation

---

## User Stories

### User Story 1 - Deploy Module to Kubernetes (Priority: P1)

A developer wants to deploy their rendered module to a Kubernetes cluster.

**Independent Test**: Given a valid module, `opm mod apply` deploys resources and `opm mod status` shows healthy.

**Acceptance Scenarios**:

1. **Given** a valid module, **When** running `opm mod apply`, **Then** resources are deployed.
2. **Given** a deployed module, **When** running `opm mod status`, **Then** health summary is displayed.
3. **Given** a deployed module, **When** running `opm mod delete`, **Then** all resources are removed.
4. **Given** a module with CRDs and CRs, **When** running `opm mod apply`, **Then** CRDs are created first.

### User Story 2 - Preview Changes Before Deployment (Priority: P2)

A developer needs to preview changes before applying them.

**Independent Test**: Deploy, modify locally, `opm mod diff` shows changes.

**Acceptance Scenarios**:

1. **Given** a deployed module with local modifications, **When** running `opm mod diff`, **Then** colorized diff is shown.
2. **Given** pending changes, **When** running `opm mod apply`, **Then** changes are applied.
3. **Given** dry-run request, **When** running `opm mod apply --dry-run`, **Then** no changes are made.

### User Story 3 - Delete Module Without Source (Priority: P2)

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
| FR-D-025 | `mod delete` MUST require `--name` and `-n` for identification. |
| FR-D-026 | `mod delete` MUST prompt for confirmation (unless --force). |

### mod diff

| ID | Requirement |
|----|-------------|
| FR-D-030 | `mod diff` MUST call Pipeline.Render() to get resources. |
| FR-D-031 | `mod diff` MUST fetch live state from cluster. |
| FR-D-032 | `mod diff` MUST show semantic diff using dyff. |
| FR-D-033 | `mod diff` MUST handle missing resources (show as additions). |
| FR-D-034 | `mod diff` MUST handle extra resources (show as deletions). |
| FR-D-035 | `mod diff` MAY process partial RenderResult (with warnings). |

### mod status

| ID | Requirement |
|----|-------------|
| FR-D-040 | `mod status` MUST discover resources via OPM labels. |
| FR-D-041 | `mod status` MUST evaluate health per resource category. |
| FR-D-042 | `mod status` MUST support `--watch` for continuous monitoring. |
| FR-D-043 | `mod status` MUST support `--output` (table, yaml, json). |
| FR-D-044 | `mod status` MUST show Ready/NotReady for workloads. |
| FR-D-045 | `mod status` MUST show Complete/Running for jobs. |

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
| FR-D-062 | All resources MUST have `module.opmodel.dev/namespace: <ns>`. |
| FR-D-063 | All resources MUST have `module.opmodel.dev/version: <version>`. |
| FR-D-064 | All resources MUST have `component.opmodel.dev/name: <component>`. |

---

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-D-001 | `mod apply` MUST be idempotent. |
| NFR-D-002 | `mod status` MUST report health within 60 seconds of change. |
| NFR-D-003 | No enforced limits on module complexity. |

---

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-D-001 | New user can init, build, apply in under 3 minutes. |
| SC-D-002 | `mod diff` accurately reflects delta 100% of time. |
| SC-D-003 | `mod apply` is fully idempotent. |
| SC-D-004 | `mod status` reports correct health within 60 seconds. |

---

## Edge Cases

| Case | Handling |
|------|----------|
| Render errors | `mod apply` fails before touching cluster |
| Partial render | `mod diff` can still compare successful resources |
| Cluster unreachable | Fail fast with connectivity error |
| RBAC denied | Pass through Kubernetes API error |
| Field conflict | Log warning, take ownership |
| Module source deleted | `mod delete` works via labels |
| Empty RenderResult | Success (nothing to apply) |

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
      --name string         Module name (required)
      --force               Skip confirmation prompt
      --dry-run             Preview without deleting
      --wait                Wait for resources to be deleted
      --kubeconfig string   Path to kubeconfig
      --context string      Kubernetes context
```

### mod diff

```text
opm mod diff [path] [flags]

Arguments:
  path    Path to module directory (default: .)

Flags:
  -f, --values strings      Additional values files
  -n, --namespace string    Target namespace
      --name string         Release name
      --kubeconfig string   Path to kubeconfig
      --context string      Kubernetes context
```

### mod status

```text
opm mod status [flags]

Flags:
  -n, --namespace string    Target namespace (required)
      --name string         Module name (required)
  -o, --output string       Output format: table, yaml, json (default: table)
      --watch               Continuous monitoring
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

---

## Resource Health Evaluation

| Category | Resources | Health Criteria |
|----------|-----------|-----------------|
| Workloads | Deployment, StatefulSet, DaemonSet | `Ready` condition True |
| Jobs | Job | `Complete` condition True |
| CronJobs | CronJob | Always healthy (passive) |
| Passive | ConfigMap, Secret, Service, PVC | Healthy on creation |
| Custom | CRD instances | `Ready` if present, else passive |
