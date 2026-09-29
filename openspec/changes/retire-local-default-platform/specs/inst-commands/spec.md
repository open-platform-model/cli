## MODIFIED Requirements

### Requirement: instance render commands accept --platform flag

Render commands (`vet`, `build`, `apply`, `diff`) SHALL accept a `--platform <dir>` flag naming a platform module directory, the highest-precedence platform source (see the `platform-resolution` capability), through the shared `cmdutil.InstanceFileFlags` set alongside `-f`/`--values`. The flag help SHALL say it overrides the cluster Platform and the instance's own deps. The platform SHALL NOT be specified in the instance file, and there SHALL be no `--provider` flag: it was retired together with the provider concept.

#### Scenario: platform flag overrides the resolved platform source

- **WHEN** `opm instance build instance.cue --platform ./my-platform` is run
- **THEN** the render SHALL use the platform module in `./my-platform` regardless of the cluster Platform CR or the instance's deps

### Requirement: instance cluster-connectivity commands accept K8s flags

The commands that connect to the cluster (`apply`, `diff`, `delete`, `status`, `tree`, `events`, `list`) SHALL accept `--kubeconfig` and `--context` flags. `build` and `vet`, which read the cluster only to find its `Platform`, SHALL accept the same two flags plus `--offline`, a boolean defaulting to false that forbids any cluster contact.

#### Scenario: kubeconfig flag

- **WHEN** `opm instance apply instance.cue --kubeconfig ~/.kube/prod-config` is run
- **THEN** the CLI SHALL use the specified kubeconfig file for cluster connectivity

#### Scenario: build reads the Platform from the named context

- **WHEN** `opm instance build instance.cue --context staging` is run
- **THEN** the CLI SHALL look up the cluster `Platform` through the `staging` context

#### Scenario: offline build and vet

- **WHEN** `opm instance vet instance.cue --offline` is run
- **THEN** the CLI SHALL NOT load a kubeconfig or contact a cluster
