## ADDED Requirements

### Requirement: Match components against transformer definitions
The system SHALL evaluate each component in a `*core.ModuleRelease` against every transformer definition in a `*provider.LoadedProvider`, producing a `MatchPlan` that records which transformers matched each component and why unmatched components failed to match.

#### Scenario: Component satisfies all required labels
- **WHEN** a component's labels include all labels declared as required by a transformer
- **THEN** the transformer is included in the match plan for that component

#### Scenario: Component satisfies all required resources and traits
- **WHEN** a component's resources and traits include all entries declared as required by a transformer
- **THEN** the transformer is included in the match plan for that component

#### Scenario: Component missing a required label
- **WHEN** a component does not have a label declared as required by a transformer
- **THEN** the transformer is excluded from the match plan for that component with a reason stating which label was missing

#### Scenario: Component missing a required resource
- **WHEN** a component does not have a resource type declared as required by a transformer
- **THEN** the transformer is excluded from the match plan for that component with a reason stating which resource was missing

#### Scenario: Component matches no transformer
- **WHEN** a component does not satisfy the requirements of any transformer in the provider
- **THEN** the component's name is returned in the unmatched slice alongside the `*core.TransformerMatchPlan`

#### Scenario: Component matches multiple transformers
- **WHEN** a component satisfies the requirements of more than one transformer
- **THEN** all matching transformers are recorded in the `*core.TransformerMatchPlan` for that component

### Requirement: Match details recorded for rejection reporting
The system SHALL include a detail entry for every (component, transformer) pair evaluated, regardless of whether the pair matched, so that callers can report which transformers were considered for an unmatched component and why each was rejected.

#### Scenario: Unmatched component with rejection reasons
- **WHEN** a component matches no transformers
- **THEN** a detail entry per transformer evaluated SHALL be accessible, each with a human-readable reason for rejection

### Requirement: Optional labels, resources, and traits do not affect matching
The system SHALL treat optional labels, resources, and traits declared by a transformer as informational only — their presence or absence on a component SHALL NOT affect whether the transformer matches.

#### Scenario: Component lacking an optional trait still matches
- **WHEN** a component satisfies all required fields of a transformer but lacks an optional trait
- **THEN** the transformer still matches the component
- **THEN** the missing optional trait is recorded as unhandled in the match detail

### Requirement: Unhandled traits are tracked per match
The system SHALL record which optional traits a matched transformer does not handle, so that the pipeline can warn when no transformer handles a given trait on a component.

#### Scenario: Trait handled by at least one transformer
- **WHEN** a component has an optional trait and at least one matched transformer declares it as optional or required
- **THEN** the trait is NOT reported as unhandled

#### Scenario: Trait unhandled by all matched transformers
- **WHEN** a component has an optional trait and no matched transformer declares it
- **THEN** the trait is reported as unhandled for that component
