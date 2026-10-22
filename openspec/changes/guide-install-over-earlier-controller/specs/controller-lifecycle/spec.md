## ADDED Requirements

### Requirement: Install tells the user when the earlier controller is on the cluster

When the Deployment `opm-operator-controller-manager` exists in the namespace `opm-operator-system`, `opm controller install` (full form and `--crds-only`) SHALL print one line that says the cluster still runs the controller of the earlier install and names the docs page "Install over the earlier controller". The refusal, its text and the exit codes SHALL NOT change.

#### Scenario: A refusal on a cluster that holds the earlier controller

- **WHEN** `opm controller install` is refused by the apply guard on a cluster that holds the earlier controller Deployment
- **THEN** it exits 2 with the guard's refusal unchanged
- **AND** the output ends with one line that starts with `hint:` and names the Deployment, the namespace and the docs page

#### Scenario: An install that goes on

- **WHEN** the checks pass on a cluster that holds the earlier controller Deployment, for example because its CRDs carry the adopt annotation
- **THEN** one warning with the same words is printed before the first write
- **AND** the install runs as before and its exit code is unchanged

#### Scenario: The `--crds-only` form

- **WHEN** `opm controller install --crds-only` is refused, or goes on, on the same cluster
- **THEN** it prints the hint line, or the warning, and its exit code is unchanged

### Requirement: The line is a hint after a refusal and a warning when the install goes on

After a refusal of the apply guard the line SHALL start with `hint:` and follow the refusal. When the checks pass and the install goes on, the line SHALL be one warning printed before the first write.

#### Scenario: Placement of the line

- **WHEN** the guard refuses, or the checks pass
- **THEN** the line is the last line of the refusal, or one warning before the first write

### Requirement: Nothing is printed when the earlier Deployment is not found

The line SHALL NOT be printed when the earlier Deployment is absent, when the read of it fails for any reason, or when the install fails before or outside the apply guard. A failed read of the Deployment SHALL NOT change the exit code or the text of any outcome.

#### Scenario: A refusal with no earlier Deployment

- **WHEN** the apply guard refuses a foreign CustomResourceDefinition and the cluster has no Deployment `opm-operator-controller-manager`
- **THEN** the output holds the refusal and no `hint:` line

#### Scenario: A denied read of the Deployment

- **WHEN** the read of the earlier Deployment is forbidden
- **THEN** the output holds no hint and no warning, and the exit code is the one the install would have had

### Requirement: The line does not tell the user to delete an object

The hint line and the warning SHALL point to the docs page and SHALL NOT tell the user to delete or rename an object. The steps that delete the earlier controller SHALL be on the docs page only.

#### Scenario: The line holds no deletion advice

- **WHEN** the hint line or the warning is printed
- **THEN** it does not contain `delete`, `remove or rename` or a `kubectl` command

### Requirement: A docs page says how to move off the earlier controller

The docs site SHALL have a page `Install over the earlier controller` under `diagnostics`. It SHALL say that the earlier install is not migrated and that its controller keeps running. It SHALL give the steps to remove the earlier controller by hand before `opm controller install`, name the earlier objects, and say what happens to the four CustomResourceDefinitions and the ModuleInstances.

#### Scenario: The page names the objects

- **WHEN** a user reads the page
- **THEN** it names the namespace, the Deployment, the four CRDs and the cluster roles and bindings of the earlier install
- **AND** it keeps the CRDs by default and warns that deleting a CRD deletes every object of that kind

### Requirement: The page says that uninstall deletes nothing over an earlier install

The page SHALL say that `opm controller uninstall` deletes nothing on a cluster whose controller has no record, as with an earlier install, and SHALL give the `kubectl` steps instead.

#### Scenario: Uninstall over an earlier install

- **WHEN** a user reads the page's section on uninstall
- **THEN** it states that the command exits 2 and deletes nothing without a record
- **AND** it lists the `kubectl delete` steps
