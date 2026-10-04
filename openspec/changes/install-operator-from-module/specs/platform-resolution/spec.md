## ADDED Requirements

### Requirement: Operator install renders against the operator module's own pins

`opm operator install`, the CRDs-only form included, SHALL render the operator module against a platform generated from the module's own committed dependency pins, and SHALL NOT read the cluster `Platform` or accept a `--platform` directory for that render. Every other command keeps its precedence under "Platform source precedence by command". The same module version and values SHALL render the same objects whether or not the cluster holds a Platform and whatever it subscribes to, and a Platform that is missing, not Ready or unreadable SHALL NOT refuse the render. The provenance line SHALL name the module's deps as the source. This narrows 0006:D11's precedence for the operator's own instance only, so repairing the operator never depends on the Platform it serves. Source: 0021:D11:R10.

#### Scenario: Seeded Platform on another catalog release

- **WHEN** the cluster Platform subscribes to a catalog release other than the one the operator module pins, and install re-runs with the same module version and values
- **THEN** the render is the one the module's own pins produce, and no live object changes

#### Scenario: Stalled Platform does not block repair

- **WHEN** the cluster Platform is `Stalled` and install runs
- **THEN** the render succeeds from the module's pins and install proceeds to its checks
