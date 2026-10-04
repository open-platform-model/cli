## ADDED Requirements

### Requirement: Instance delete keeps the ModuleInstance when a tracked resource could not be read

When `opm instance delete` deletes a CLI-owned instance and a tracked resource could not be read during discovery (the read failed with an error other than NotFound), that resource SHALL count as a failure for that resource, with the same outcome as a failed re-read: it SHALL be listed with its read error, the other tracked resources SHALL still be processed, the `ModuleInstance` record SHALL NOT be deleted, and the command SHALL exit 1. On a real run the output SHALL say that the `ModuleInstance` was kept and that re-running is safe. A dry run SHALL count the resource among those it could not check. A tracked `Namespace` in the core group or `CustomResourceDefinition` in `apiextensions.k8s.io` that could not be read SHALL instead be listed as `left behind`, as it would be if read, and SHALL NOT count as a failure. The operator-owned delete path SHALL NOT be affected.

#### Scenario: Discovery read error keeps the ModuleInstance

- **WHEN** running `opm instance delete` for a CLI-owned instance that tracks a Deployment and a ConfigMap
- **AND** reading the ConfigMap during discovery fails with Forbidden
- **THEN** the Deployment SHALL be deleted
- **AND** the ConfigMap SHALL be listed with the Forbidden error
- **AND** the `ModuleInstance` record SHALL NOT be deleted
- **AND** the output SHALL say that the ModuleInstance was kept and that re-running is safe
- **AND** the command SHALL exit 1

#### Scenario: Dry run counts an unreadable resource as not checked

- **WHEN** running `opm instance delete --dry-run` and a tracked resource cannot be read during discovery
- **THEN** nothing SHALL be deleted
- **AND** the command SHALL report that 1 resource could not be checked
- **AND** SHALL exit 1

#### Scenario: Unreadable Namespace is left behind, not a failure

- **WHEN** running `opm instance delete` and reading a tracked Namespace during discovery fails with Forbidden
- **AND** every other tracked resource is read and deleted
- **THEN** the Namespace SHALL be listed as `left behind`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

#### Scenario: Operator-owned delete ignores discovery read errors

- **WHEN** running `opm instance delete` for an operator-owned instance and a tracked resource cannot be read
- **THEN** the command SHALL delete the `ModuleInstance` and wait for the operator as before
