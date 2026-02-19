## MODIFIED Requirements

### Requirement: Namespace defaults to config

The `--namespace`/`-n` flag SHALL be optional for commands that discover resources (`delete`, `status`). When omitted, namespace SHALL be resolved using the precedence: `--namespace` flag → `OPM_NAMESPACE` environment variable → `kubernetes.namespace` in `~/.opm/config.cue` → `"default"`.

#### Scenario: Namespace omitted uses config default

- **WHEN** the user runs `opm mod delete --release-name my-app` without `-n`
- **AND** the config file sets `kubernetes: namespace: "staging"`
- **THEN** the command SHALL operate in the `staging` namespace

#### Scenario: Namespace omitted falls back to default

- **WHEN** the user runs `opm mod status --release-name my-app` without `-n`
- **AND** no config or env sets a namespace
- **THEN** the command SHALL operate in the `default` namespace

## REMOVED Requirements

### Requirement: Namespace always required

**Reason**: The namespace has a well-defined default resolution chain (flag → env → config → "default") that is already implemented via `ResolveKubernetes`. Requiring the flag explicitly is inconsistent with how other commands handle namespace and adds unnecessary friction for users who operate in a consistent default namespace.

**Migration**: No action required. Commands that previously required `-n` will now resolve namespace from config or default. Existing scripts that pass `-n` continue to work unchanged.

## REMOVED Requirements

### Requirement: Fail on no resources found (label-scan semantics)

**Reason**: The "no resources found" error was defined in terms of label-scan results. With inventory-only discovery, the error is now defined as "no inventory Secret found for this release" — which is semantically equivalent but mechanistically different. The new requirement is specified in `mod-status/spec.md` and `mod-delete` (TBD).

**Migration**: The error message changes from `"no resources found for release <name> in namespace <ns>"` to `"release '<name>' not found in namespace '<ns>'"`. Scripts using `--ignore-not-found` are unaffected.
