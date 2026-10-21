## MODIFIED Requirements

### Requirement: An undiscriminated platform fails the command

A pair of enabled transformers whose match predicates are comparable over a shared catalog-fulfilled contract SHALL be reported with both transformers and the shared contracts named, and the command SHALL exit with the validation error code (enhancement 0015). This mirrors where the refusal lives in the system: `Discriminated: false` is a condition platform-package generation refuses on, exactly as `Routable: false` is, so a pre-flight that exits 0 on it would bless a platform the controller rejects. No arbitration between the two transformers SHALL be applied or suggested. Transformers with incomparable predicates over a shared contract (a differing required label value, or a required trait the other lacks) SHALL be reported as implementations and SHALL NOT change the exit status.

#### Scenario: A comparable pair exits non-zero

- **WHEN** one enabled transformer requires a catalog-fulfilled resource alone and another enabled transformer requires the same resource plus a trait
- **THEN** the report names the first as broader, the second as narrower and the resource as shared, states the platform is not discriminated, and the command exits with the validation error code

#### Scenario: Discriminated plurality exits zero

- **WHEN** several enabled transformers require the same catalog-fulfilled contract and each adds a required label value or trait the others lack
- **THEN** the report shows them as implementations, states the platform is discriminated, and the command exits 0

#### Scenario: Both refusals are reported together

- **WHEN** a platform is over-subscribed on one contract and undiscriminated on another
- **THEN** both appear under their own headings, both verdict lines say no, the error names both counts, and the command exits with the validation error code once
