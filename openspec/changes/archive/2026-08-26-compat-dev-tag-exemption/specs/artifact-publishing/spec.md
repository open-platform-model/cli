# artifact-publishing — Delta

## MODIFIED Requirements

### Requirement: The compatibility gate

`opm catalog publish` SHALL, for every member of the tree at beta or GA level (transformers structurally excluded; alpha members exempt), compare the member against the last published build that shipped a member of that name at that apiVersion — found by scanning published versions strictly below the effective version, within the same major, release prereleases (alpha, beta, rc) included and dev builds excluded, newest first — under the additive-only rule via the library comparator, over provenance-stripped definition values. A member no published build has carried SHALL pass. Violations SHALL refuse as refusal 9: a header naming the member, its apiVersion, and the compared-against coordinate, followed by path-located violation lines. The gate makes a beta-or-GA name-and-apiVersion key a permanent claim on its own history: an incompatible re-introduction after a removal SHALL be refused identically.

A dev build, an effective tag whose prerelease segment carries a `dev` identifier (`v2.0.0-0.dev.1787747172.gcf5f131`), SHALL NOT be judged by the gate and SHALL NOT serve as any build's predecessor. The plan SHALL report the skip visibly on the `compat gate` row as `dev-exempt`; the row SHALL never read as a completed comparison for a dev build.

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

#### Scenario: Dev build is not judged

- **WHEN** the effective tag is a dev prerelease and a beta member is incompatible with the newest published release carrying it
- **THEN** publish does not refuse on compatibility
- **AND** the plan's `compat gate` row reads `dev-exempt`

#### Scenario: Dev build is never the baseline

- **WHEN** the published history holds a dev tag newer than the latest release tag, and only the dev tag carries an incompatible shape of a beta member
- **THEN** a release-tagged publish compares against the release tag and passes
- **AND** a member absent from every release tag but present in a dev tag is reported as new

#### Scenario: Release prereleases stay in the window

- **WHEN** the published history holds alpha, beta and dev tags below the effective tag
- **THEN** the predecessor window is the alpha and beta tags newest first, with no dev tag
