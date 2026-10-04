## ADDED Requirements

### Requirement: Dependabot leaves cuelang.org/go to library releases
The Dependabot `gomod` update SHALL ignore the dependency `cuelang.org/go`, so the cli's CUE version moves only through a library release: the library bumps CUE in its own pull request, and the cli picks the new version up when the release cascade (or a hand-made `fix(deps)` pull request) moves the library pin and `go mod tidy` raises `cuelang.org/go` to the version the library requires. Other third-party Go modules SHALL keep their weekly updates. Source: owner decision j4 of the kernel beta.1 plan walkthrough (2026-10-03).

#### Scenario: New CUE release
- **WHEN** a new `cuelang.org/go` version is published and Dependabot's weekly `gomod` run executes
- **THEN** Dependabot opens no pull request for `cuelang.org/go`

#### Scenario: CUE arrives with a library release
- **WHEN** a library release that requires a newer `cuelang.org/go` is pinned by the cascade's library bump
- **THEN** that bump pull request moves `cuelang.org/go` in `go.mod` and `go.sum` to the version the library requires

#### Scenario: Other third-party modules still update
- **WHEN** a newer `k8s.io/client-go` is published
- **THEN** Dependabot still proposes the bump
