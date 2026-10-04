## ADDED Requirements

### Requirement: Operator install is not refused by the running-operator ceiling

The instance apply that `opm operator install` performs for the operator's own instance SHALL skip the operator-version ceiling, because install replaces the operator the Platform reports rather than driving it. Install applies the same `MAJOR.MINOR` rule to the operator version of the module it installs instead (see the `operator-lifecycle` capability). Every other apply keeps the ceiling unchanged, and the CRD-presence and CRD field-floor gates still run for the operator's instance. This narrows the ceiling of 0021:D9 for the operator's own install.

#### Scenario: Older CLI repairs a newer operator

- **WHEN** `Platform/cluster` reports `status.operatorVersion` `1.1.0`, the CLI is `1.0.0`, and `opm operator install --allow-downgrade` targets a module deploying `1.0.2`
- **THEN** no ceiling refusal occurs and install proceeds

#### Scenario: Ordinary apply still refused

- **WHEN** the same CLI runs `opm instance apply` for any other instance against that cluster
- **THEN** apply refuses with the upgrade-the-CLI error
