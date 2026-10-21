## MODIFIED Requirements

### Requirement: Controller install renders against the controller module's own pins

A full `opm controller install` SHALL render the controller module against a platform generated from the module's own committed dependency pins, and SHALL NOT read the cluster `Platform` or accept a `--platform` directory for that render. Every other command keeps its precedence under "Platform source precedence by command". The same module version and values SHALL render the same objects whether or not the cluster holds a Platform and whatever it subscribes to, and a Platform that is missing, not Ready or unreadable SHALL NOT refuse the render. The provenance line SHALL name the module's deps as the source. The CRDs-only form renders nothing at install time: it applies the CRDs the pin task rendered from the pinned version in the same way. This narrows the platform-source precedence of enhancement 0006 for the controller's own instance only, so repairing the controller never depends on the Platform it serves. Source: enhancement 0021.

#### Scenario: Seeded Platform on another catalog release

- **WHEN** the cluster Platform subscribes to a catalog release other than the one the controller module pins, and install re-runs with the same module version and values
- **THEN** the render is the one the module's own pins produce, and no live object changes

#### Scenario: Stalled Platform does not block repair

- **WHEN** the cluster Platform is `Stalled` and install runs
- **THEN** the render succeeds from the module's pins and install proceeds to its checks
