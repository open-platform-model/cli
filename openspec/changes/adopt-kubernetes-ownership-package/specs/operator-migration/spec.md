## ADDED Requirements

### Requirement: The migration's deletes pass the shared delete verdict

Before install deletes the earlier Deployment or a superseded role binding, it SHALL take the decision from the delete verdict the CLI shares with the operator, in the check phase, on the object the proof read. The verdict SHALL admit a proven object although OPM does not manage it. When the verdict does not allow the delete of an object the migration would delete, which includes an object whose `opmodel.dev/adopt` annotation names an instance other than the operator's, install SHALL refuse before any object changes, with exit code 2, naming the object and the reason. Each delete of the migration SHALL carry a precondition on the UID of the object that was judged. Source: 0012:D8:R7, 0012:D8:R8.

#### Scenario: Proven Deployment is deleted with a UID precondition

- **WHEN** install migrates an operator whose earlier Deployment is proven and carries no adopt annotation
- **THEN** the delete request for that Deployment SHALL carry a precondition on the UID the check phase read

#### Scenario: Binding annotated for another instance refuses the install

- **WHEN** `ClusterRoleBinding opm-operator-manager-rolebinding` is proven and carries `opmodel.dev/adopt` with the UUID of another instance
- **THEN** install SHALL exit 2 before any object changes and SHALL name the binding
