
## Requirements

### Requirement: Status discovers resources via OPM labels

The `opm mod status` command SHALL discover deployed resources by querying the cluster using the OPM label selector (`module.opmodel.dev/name`) within the target namespace. It MUST NOT require module source or re-rendering.

#### Scenario: Status shows deployed resources

- **WHEN** the user runs `opm mod status -n my-namespace --name my-module`
- **THEN** the command SHALL list all resources with matching OPM labels

#### Scenario: No resources found

- **WHEN** no resources match the given name and namespace labels
- **THEN** the command SHALL print "No resources found for module <name> in namespace <namespace>" and exit with code 0

### Requirement: Status evaluates health per resource category

The command SHALL evaluate resource health using category-specific criteria:
- **Workloads** (Deployment, StatefulSet, DaemonSet): healthy when `Ready` condition is True
- **Jobs** (Job): healthy when `Complete` condition is True
- **CronJobs**: always reported as healthy (scheduled)
- **Passive** (ConfigMap, Secret, Service, PVC): healthy on creation
- **Custom** (CRD instances): healthy if `Ready` condition is present and True, otherwise treated as passive

#### Scenario: Deployment not yet ready

- **WHEN** a Deployment has `Ready` condition set to False
- **THEN** the status output SHALL show the Deployment as "NotReady"

#### Scenario: Job completed

- **WHEN** a Job has `Complete` condition set to True
- **THEN** the status output SHALL show the Job as "Complete"

#### Scenario: ConfigMap always healthy

- **WHEN** a ConfigMap exists in the cluster
- **THEN** the status output SHALL show the ConfigMap as "Ready"

#### Scenario: Custom resource with Ready condition

- **WHEN** a CRD instance has a `Ready` condition set to True
- **THEN** the status output SHALL show the resource as "Ready"

#### Scenario: Custom resource without Ready condition

- **WHEN** a CRD instance has no `Ready` condition in its status
- **THEN** the status output SHALL treat the resource as passive and show it as "Ready"

### Requirement: Status supports table output format

The default output format SHALL be a table showing: resource kind, name, namespace, health status, and age. Columns SHALL be aligned and human-readable.

#### Scenario: Default table output

- **WHEN** the user runs `opm mod status -n ns --name mod` without `--output`
- **THEN** the output SHALL be a formatted table with KIND, NAME, NAMESPACE, STATUS, and AGE columns

### Requirement: Status supports structured output formats

The command SHALL support `--output`/`-o` with values `table` (default), `yaml`, and `json` for machine-readable output.

#### Scenario: JSON output

- **WHEN** the user runs `opm mod status -n ns --name mod -o json`
- **THEN** the output SHALL be a valid JSON array of resource status objects

#### Scenario: YAML output

- **WHEN** the user runs `opm mod status -n ns --name mod -o yaml`
- **THEN** the output SHALL be valid YAML containing resource status entries

### Requirement: Status supports watch mode

The command SHALL support `--watch` for continuous monitoring. In watch mode, the status table SHALL refresh at a regular interval (2 seconds), clearing the previous output and displaying the updated table.

#### Scenario: Watch mode updates on change

- **WHEN** the user runs `opm mod status -n ns --name mod --watch`
- **THEN** the status table SHALL refresh every 2 seconds until the user interrupts (Ctrl+C)

#### Scenario: Watch mode exits cleanly on interrupt

- **WHEN** the user presses Ctrl+C during watch mode
- **THEN** the command SHALL exit with code 0 and restore the terminal state

### Requirement: Status requires name and namespace flags

The `--name` and `--namespace`/`-n` flags SHALL be required. The command SHALL reject execution and display a usage error if either is missing.

#### Scenario: Missing namespace flag

- **WHEN** the user runs `opm mod status --name my-module` without `-n`
- **THEN** the command SHALL exit with code 1 and display a usage error indicating `-n` is required

#### Scenario: Missing name flag

- **WHEN** the user runs `opm mod status -n my-namespace` without `--name`
- **THEN** the command SHALL exit with code 1 and display a usage error indicating `--name` is required

### Requirement: Status accepts kubernetes connection flags

The command SHALL accept `--kubeconfig` and `--context` flags for cluster connection. Kubeconfig resolution SHALL follow: explicit flag > `KUBECONFIG` env var > default path (`~/.kube/config`).

#### Scenario: Custom kubeconfig

- **WHEN** the user runs `opm mod status --kubeconfig /path/to/config -n ns --name mod`
- **THEN** the command SHALL use the specified kubeconfig file

### Requirement: Status fails fast on connectivity errors

The command SHALL fail immediately with a clear error message if the Kubernetes cluster is unreachable.

#### Scenario: Cluster unreachable

- **WHEN** the cluster specified by kubeconfig/context is not reachable
- **THEN** the command SHALL exit with code 3 and display a connectivity error message
