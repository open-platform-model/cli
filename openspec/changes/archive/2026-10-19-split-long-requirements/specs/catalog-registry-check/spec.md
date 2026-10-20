## REMOVED Requirements

### Requirement: Out-of-band verification of a published catalog
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm catalog registry check` pulls the named published build and verifies out of band what a platform verifies when it imports the catalog", "`opm catalog registry check` reports the member inventory per kind and apiVersion", "`opm catalog registry check --compat` runs the predecessor comparison for the fetched build exactly as publish would have", "The help text of `opm catalog registry check` states that the check is an aid and not a guarantee".

### Requirement: An unpublished coordinate exits 5 and every other registry answer exits 3
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry", "`opm catalog registry check` takes the not-held decision from the library's typed fetch classification and never from the message text".

## ADDED Requirements

### Requirement: `opm catalog registry check` pulls the named published build and verifies out of band what a platform verifies when it imports the catalog

`opm catalog registry check <path@version>` SHALL pull the named published build and verify, out of band, what a platform verifies when it imports the catalog (enhancement 0019, "A platform imports its catalog and embeds it whole into the registry entry, which derives everything else"): the declared identity is concrete and its `modulePath` and `version` agree with the coordinate the build was fetched by.
Exit codes: 0 clean, 2 findings, 3 registry unreachable, 5 not published.

#### Scenario: Identity mismatch found out of band

- **WHEN** a published build's declared version disagrees with the tag it was fetched by
- **THEN** the check reports both values and exits 2

### Requirement: `opm catalog registry check` reports the member inventory per kind and apiVersion

`opm catalog registry check <path@version>` SHALL report the catalog's member inventory per kind and apiVersion.

#### Scenario: The member inventory is reported per kind and apiVersion

- **WHEN** `opm catalog registry check <path@version>` verifies a published build
- **THEN** it SHALL report the catalog's member inventory per kind and apiVersion

### Requirement: `opm catalog registry check --compat` runs the predecessor comparison for the fetched build exactly as publish would have

With `--compat`, `opm catalog registry check <path@version>` SHALL additionally run the predecessor comparison for the fetched build exactly as publish would have.

#### Scenario: Compat aid over a published build

- **WHEN** `--compat` runs against a build that broke a beta contract relative to its predecessor
- **THEN** the same violations publish would have refused are reported

### Requirement: The help text of `opm catalog registry check` states that the check is an aid and not a guarantee

The `opm catalog registry check <path@version>` command's help text SHALL state that the check is an aid and not a guarantee: enforcement exists only at publish.

#### Scenario: Aid, not guarantee

- **WHEN** the command's help is displayed
- **THEN** it states the check is an aid and that only publish enforces

### Requirement: `opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry

`opm catalog registry check <path@version>` SHALL exit 5 when the registry answers that it does not hold the named version, which includes a registry that answers the tag lookup with 403, because CUE reports that answer as not found.
A registry answer other than not-held, or no response, while the named build is fetched SHALL exit 3, the build never judged.
A failure that asks no registry (a registry mapping that does not parse) exits 1.

**Source**: enhancement 0011 ("A published catalog can be verified out of band").

#### Scenario: Version not published

- **WHEN** the check names a version the registry does not hold
- **THEN** the check exits 5 and prints no report

#### Scenario: Forbidden tag lookup reads as not published

- **WHEN** the registry answers the tag lookup for the named version with status 403
- **THEN** the check exits 5

### Requirement: `opm catalog registry check` takes the not-held decision from the library's typed fetch classification and never from the message text

`opm catalog registry check <path@version>` SHALL exit 5 when the registry answers that it does not hold the named version, which includes a registry that answers the tag lookup with 403, because CUE reports that answer as not found.
The CLI SHALL take the not-held decision from the library's typed fetch classification and SHALL NOT match the registry error's message text itself.

**Source**: enhancement 0011 ("A published catalog can be verified out of band").

**Source**: enhancement 0021 ("A beta line reaches GA only when its exit criteria hold").

#### Scenario: The not-held decision comes from the typed fetch classification

- **WHEN** the check decides whether the registry holds the named version
- **THEN** the CLI SHALL take the not-held decision from the library's typed fetch classification and SHALL NOT match the registry error's message text itself
