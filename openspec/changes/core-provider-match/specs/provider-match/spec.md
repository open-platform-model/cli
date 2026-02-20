## ADDED Requirements

### Requirement: Provider exposes a Match method
`core.Provider` SHALL expose a `Match` method that accepts a map of components and returns a `*TransformerMatchPlan` describing which transformers were matched to which components and which components were unmatched.

The method signature SHALL be:
```
Match(components map[string]*Component) *TransformerMatchPlan
```

#### Scenario: All components matched
- **WHEN** `provider.Match(components)` is called and every component satisfies at least one transformer's requirements
- **THEN** the returned `TransformerMatchPlan.Matches` SHALL contain one entry per matched (component, transformer) pair
- **AND** `TransformerMatchPlan.Unmatched` SHALL be empty

#### Scenario: Some components unmatched
- **WHEN** `provider.Match(components)` is called and one or more components satisfy no transformer's requirements
- **THEN** `TransformerMatchPlan.Unmatched` SHALL list the names of all unmatched components
- **AND** `TransformerMatchPlan.Matches` SHALL only contain entries for successfully matched pairs

#### Scenario: Component matches multiple transformers
- **WHEN** a component satisfies the requirements of more than one transformer
- **THEN** `TransformerMatchPlan.Matches` SHALL contain one entry for each (component, transformer) pair
- **AND** the component SHALL NOT appear in `TransformerMatchPlan.Unmatched`

### Requirement: Provider stores CUE context at construction time
`core.Provider` SHALL store a `*cue.Context` received from the loader at construction time. This context SHALL be passed into the `TransformerMatchPlan` returned by `Match()` so that the plan can use it during execution.

#### Scenario: CUE context propagated to match plan
- **WHEN** `transform.LoadProvider()` constructs a `*core.Provider` with a given `*cue.Context`
- **AND** `provider.Match()` is called
- **THEN** the returned `TransformerMatchPlan` SHALL hold the same `*cue.Context`

### Requirement: Matching algorithm evaluates required labels, resources, and traits
The `Match()` method SHALL implement the same O(components × transformers) matching algorithm currently in `transform/matcher.go`. A component matches a transformer when all of the transformer's required labels, required resources, and required traits are satisfied by the component.

#### Scenario: Required label not present causes no match
- **WHEN** a transformer requires label `"app.opmodel.dev/type": "web"` and a component does not have that label
- **THEN** that (component, transformer) pair SHALL NOT appear in `TransformerMatchPlan.Matches`

#### Scenario: Required resource not present causes no match
- **WHEN** a transformer requires resource FQN `"networking.example.dev#Ingress"` and a component does not declare that resource
- **THEN** that (component, transformer) pair SHALL NOT appear in `TransformerMatchPlan.Matches`

#### Scenario: All requirements satisfied yields a match
- **WHEN** a component satisfies all of a transformer's required labels, resources, and traits
- **THEN** that (component, transformer) pair SHALL appear in `TransformerMatchPlan.Matches` with a non-empty `Reason` field

### Requirement: TransformerMatchPlan carries match details for diagnostics
`core.TransformerMatchPlan` SHALL carry sufficient detail for verbose output and error reporting, including per-transformer match decisions for each component.

#### Scenario: Match plan includes unhandled trait diagnostics
- **WHEN** a component has a trait that no matched transformer handles
- **THEN** the match plan SHALL carry enough information for the pipeline to emit a warning about the unhandled trait
