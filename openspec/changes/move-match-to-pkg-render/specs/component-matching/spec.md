## MODIFIED Requirements

### Requirement: Match components against transformer definitions
The system SHALL evaluate each component in a module release against every transformer definition in a `*provider.Provider`, producing a `*render.MatchPlan` that records which transformers matched each component and why unmatched components failed to match. The implementation SHALL reside in `pkg/render` (previously `internal/match`).

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
- **THEN** the component's name is returned in the unmatched slice of the `*render.MatchPlan`

#### Scenario: Component matches multiple transformers
- **WHEN** a component satisfies the requirements of more than one transformer
- **THEN** all matching transformers are recorded in the `*render.MatchPlan` for that component
