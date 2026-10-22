## ADDED Requirements

### Requirement: A refusal over the earlier controller carries one hint line

When the apply guard refuses an install and the Deployment `opm-operator-controller-manager` exists in the namespace `opm-operator-system`, `opm controller install` SHALL print one `hint:` line after the refusal that says the cluster still runs the controller of the earlier install and names the docs page "Install over the earlier controller". The refusal, its text and its exit code 2 SHALL NOT change.

#### Scenario: A refusal on a cluster that holds the earlier controller

- **WHEN** `opm controller install` is refused by the apply guard on a cluster that holds the earlier controller Deployment
- **THEN** it exits 2 with the guard's refusal unchanged
- **AND** the output ends with one line that starts with `hint:` and names the Deployment, the namespace and the docs page

#### Scenario: The `--crds-only` form

- **WHEN** `opm controller install --crds-only` is refused by the apply guard on the same cluster
- **THEN** it prints the same hint line and exits 2

### Requirement: The hint line is never printed on another cause

The hint line SHALL NOT be printed when the earlier Deployment is absent, when the read of it fails for any reason, or when the install fails for a cause other than a refusal of the apply guard. A failed read of the Deployment SHALL NOT change the exit code or the text of the refusal.

#### Scenario: A refusal with no earlier Deployment

- **WHEN** the apply guard refuses a foreign CustomResourceDefinition and the cluster has no Deployment `opm-operator-controller-manager`
- **THEN** the output holds the refusal and no `hint:` line

#### Scenario: A denied read of the Deployment

- **WHEN** the guard refuses and the read of the earlier Deployment is forbidden
- **THEN** the output holds the refusal and no `hint:` line, and the exit code is 2

### Requirement: The hint line does not tell the user to delete an object

The hint line SHALL point to the docs page and SHALL NOT tell the user to delete or rename an object. The steps that delete the earlier controller SHALL be on the docs page only.

#### Scenario: The hint holds no deletion advice

- **WHEN** the hint line is printed
- **THEN** it does not contain `delete`, `remove or rename` or a `kubectl` command

### Requirement: A docs page says how to move off the earlier controller

The docs site SHALL have a page `Install over the earlier controller` under `diagnostics`. It SHALL say that the earlier install is not migrated and that its controller keeps running. It SHALL give the steps to remove the earlier controller by hand before `opm controller install`, name the earlier objects, and say what happens to the four CustomResourceDefinitions and the ModuleInstances.

#### Scenario: The page names the objects

- **WHEN** a user reads the page
- **THEN** it names the namespace, the Deployment, the four CRDs and the cluster roles and bindings of the earlier install
- **AND** it warns that deleting a CRD deletes every object of that kind

### Requirement: The page says that uninstall deletes nothing over an earlier install

The page SHALL say that `opm controller uninstall` deletes nothing on a cluster whose controller has no record, as with an earlier install, and SHALL give the `kubectl` steps instead.

#### Scenario: Uninstall over an earlier install

- **WHEN** a user reads the page's section on uninstall
- **THEN** it states that the command exits 2 and deletes nothing without a record
- **AND** it lists the `kubectl delete` steps
