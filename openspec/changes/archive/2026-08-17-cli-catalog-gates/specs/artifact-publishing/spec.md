# artifact-publishing — Delta

## ADDED Requirements

### Requirement: The compatibility gate

`opm catalog publish` SHALL, for every member of the tree at beta or GA level (transformers structurally excluded; alpha members exempt), compare the member against the last published build that shipped a member of that name at that apiVersion — found by scanning published versions strictly below the effective version, within the same major, prereleases included, newest first — under the additive-only rule via the library comparator, over provenance-stripped definition values. A member no published build has carried SHALL pass. Violations SHALL refuse as refusal 9: a header naming the member, its apiVersion, and the compared-against coordinate, followed by path-located violation lines. The gate makes a beta-or-GA name-and-apiVersion key a permanent claim on its own history: an incompatible re-introduction after a removal SHALL be refused identically.

#### Scenario: Field removal refused against the true predecessor

- **WHEN** a beta member removes a field relative to the newest published build that carried it
- **THEN** publish refuses with the violation path and the predecessor coordinate

#### Scenario: Prerelease predecessors count

- **WHEN** the newest build carrying the member is a prerelease newer than the latest stable
- **THEN** the comparison runs against that prerelease, not the stable

#### Scenario: Remove-then-readd refused

- **WHEN** a member absent from the immediate predecessor was shipped incompatibly-reshaped, and an older build carried it
- **THEN** the scan finds the older build and refuses

#### Scenario: apiVersion bump escapes

- **WHEN** a member ships at an apiVersion no build has ever carried
- **THEN** it passes the compatibility gate trivially

### Requirement: Member and posture gates

`opm catalog publish` SHALL unify every member of the tree — all four kinds, all levels — against core's member gate with concrete evaluation, surfacing CUE's error as the refusal (wrong-kind or wrong-depth filing, stale authored fqn, stale catalogVersion, missing declaredAPIVersion). Every trait's `optional` field SHALL be unified against core's posture gate, refusing an unstated or pinned posture. The alpha exemption applies to the compatibility gate only.

#### Scenario: Filing error caught structurally

- **WHEN** a contract member files under the wrong kind or depth
- **THEN** publish refuses with CUE's own conflict diagnostics

#### Scenario: Unstated posture caught only concretely

- **WHEN** a trait states no `optional` posture
- **THEN** publish refuses — the incomplete value is the finding

### Requirement: Connectivity across the member walk

A transport failure during predecessor enumeration or package loading SHALL abort as a connectivity error with no partial verdict — the artifact was never judged. A package absent at a given published version is the scan's negative signal, never an error.

#### Scenario: Mid-walk failure renders no verdict

- **WHEN** the registry becomes unreachable after some members have compared clean
- **THEN** the command exits 3 and no GO or REFUSED verdict is printed
