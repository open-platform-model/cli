## ADDED Requirements

### Requirement: Dependabot leaves cuelang.org/go to library releases
The Dependabot `gomod` update SHALL ignore the dependencies `cuelang.org/go` and `cuelabs.dev/go/oci/ociregistry` (the CUE team's OCI module, whose version the library's `go.mod` sets, `cuelang.org/go` setting only a floor, so it reaches the cli through the library bump and moves with library releases), so the cli's CUE version moves only through a library release: the library bumps CUE in its own pull request, and the cli picks the new version up when the release cascade (or a hand-made `fix(deps)` pull request) moves the library pin, whose `go get` of the library raises `cuelang.org/go` by minimal version selection (kept by `go mod tidy`). Other third-party Go modules SHALL keep their weekly updates. Source: owner decision j4 of the kernel beta.1 plan walkthrough (2026-10-03), extended to `cuelabs.dev/go/oci/ociregistry` by the supervisor's round-1 triage; workspace RELEASING.md, section "Pin classes".

#### Scenario: New CUE release
- **WHEN** a new `cuelang.org/go` version is published and Dependabot's weekly `gomod` run executes
- **THEN** Dependabot opens no pull request for `cuelang.org/go` or `cuelabs.dev/go/oci/ociregistry`

#### Scenario: CUE arrives with a library release
- **WHEN** a library release that requires a newer `cuelang.org/go` is pinned by the cascade's library bump
- **THEN** that bump pull request moves `cuelang.org/go` in `go.mod` and `go.sum` to at least the version the library requires, and the cascade reports the move as a warning (`.tasks/cascade/cascade.sh`, Phase C)

#### Scenario: Other third-party modules still update
- **WHEN** a newer `k8s.io/client-go` is published
- **THEN** Dependabot still proposes the bump

## MODIFIED Requirements

### Requirement: Dependabot leaves OPM Go modules to the release cascade
The Dependabot `gomod` update SHALL ignore every dependency named `github.com/open-platform-model/*`, so library pins move only through the release cascade or by hand as `fix(deps)`. Other Go modules, except `cuelang.org/go` and `cuelabs.dev/go/oci/ociregistry` (see "Dependabot leaves cuelang.org/go to library releases"), and GitHub Actions SHALL keep their weekly updates. Source: workspace RELEASING.md, sections "The cascade" and "Rollout and changes" (Changes table).

#### Scenario: New library release
- **WHEN** a new `github.com/open-platform-model/library` version is tagged and Dependabot's weekly `gomod` run executes
- **THEN** Dependabot opens no pull request for library

#### Scenario: Third-party module update
- **WHEN** a newer `golang.org/x/term` is published
- **THEN** Dependabot still proposes the bump
