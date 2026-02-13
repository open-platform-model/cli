## ADDED Requirements

### Requirement: K8s API warnings route through charmbracelet/log

The CLI SHALL route Kubernetes API deprecation warnings through the charmbracelet/log system instead of allowing raw klog output to stderr.

#### Scenario: Warning appears with consistent formatting
- **WHEN** the K8s API server returns a deprecation warning during discovery
- **THEN** the warning SHALL appear in charmbracelet/log format (e.g., `11:43:03 WARN k8s: v1 ComponentStatus is deprecated`)

#### Scenario: Warning includes source context
- **WHEN** a K8s API warning is logged
- **THEN** the log line SHALL include the standard caller information when verbose mode is enabled

### Requirement: Config option for warning behavior

The CLI SHALL provide a configuration option `log.kubernetes.apiWarnings` that controls how K8s API warnings are handled.

#### Scenario: Default behavior shows warnings
- **WHEN** the user has not set `log.kubernetes.apiWarnings` in config.cue
- **THEN** K8s API warnings SHALL appear as WARN level log entries

#### Scenario: Warn mode shows warnings at WARN level
- **WHEN** the user sets `log.kubernetes.apiWarnings: "warn"` in config.cue
- **THEN** K8s API warnings SHALL appear as WARN level log entries

#### Scenario: Debug mode shows warnings only with verbose flag
- **WHEN** the user sets `log.kubernetes.apiWarnings: "debug"` in config.cue
- **AND** the user runs a command WITHOUT `--verbose`
- **THEN** K8s API warnings SHALL NOT appear in output

#### Scenario: Debug mode shows warnings with verbose flag
- **WHEN** the user sets `log.kubernetes.apiWarnings: "debug"` in config.cue
- **AND** the user runs a command WITH `--verbose`
- **THEN** K8s API warnings SHALL appear as DEBUG level log entries

#### Scenario: Suppress mode hides all warnings
- **WHEN** the user sets `log.kubernetes.apiWarnings: "suppress"` in config.cue
- **THEN** K8s API warnings SHALL NOT appear in output regardless of verbose mode

### Requirement: Config schema validation

The CLI SHALL validate that `log.kubernetes.apiWarnings` contains only valid values.

#### Scenario: Invalid value rejected at config load
- **WHEN** the user sets `log.kubernetes.apiWarnings: "invalid"` in config.cue
- **THEN** the CLI SHALL fail with a CUE validation error before executing any command

#### Scenario: Valid values accepted
- **WHEN** the user sets `log.kubernetes.apiWarnings` to `"warn"`, `"debug"`, or `"suppress"`
- **THEN** the CLI SHALL load the config successfully
