## ADDED Requirements

### Requirement: Pull requests that move library or the operator warn about missing docs bundles

`pr.yml`'s `lint` job SHALL run the step "Docs bundles for moved pins" on every pull request. The step SHALL fetch the pull request's base commit and run `.github/scripts/docs-pins-check.sh --warn --moved-from <base sha>`. When the `github.com/open-platform-model/library` version in `go.mod` and `PinnedOperatorVersion` in `internal/operator/manifest.go` both equal the base's, it SHALL check nothing and say so. Otherwise it SHALL run `go run ./hack/docskit-dump pins` on the tree and look up each printed pin anonymously at `ghcr.io/open-platform-model/docs/<project>:<pin>`, and SHALL print a GitHub warning annotation, also written to the job summary, for each bundle that is missing (401, 403 or 404), each lookup that fails, a program that does not build, a base commit that cannot be fetched, and a library version or `PinnedOperatorVersion` it cannot read on either side (an empty read never counts as unchanged). Each GHCR request SHALL be bounded (`--connect-timeout 10 --max-time 30`) and the step SHALL have `timeout-minutes: 10` and `continue-on-error: true`, because the `lint` job is a required check. The step SHALL exit zero in every one of those cases, so it never fails the job: the blocking check stays G1 on the release pull request. The step runs with the job's read-only token. Source: security pass 2026-10-04, finding CAS-R2 (the check left the cascade task).

G1 (`release-pin-check.sh`) SHALL run the same script without `--warn` and report each line it prints as a G1 violation; a non-zero exit with no line SHALL itself be a violation.

#### Scenario: A cascade pull request moves library to a version without a bundle

- **WHEN** the cascade pull request moves library to a version whose docs bundle is not published
- **THEN** the `lint` job shows a warning naming `library`, the version and the bundle reference, and the job's result is decided by its other steps

#### Scenario: An ordinary pull request checks nothing

- **WHEN** a pull request changes neither the library version nor `PinnedOperatorVersion`
- **THEN** the step builds nothing, prints that both are unchanged, and exits zero

#### Scenario: An unreadable pin is a warning, not a pass

- **WHEN** `go.mod` or `internal/operator/manifest.go` changes shape so the step reads no library version or no `PinnedOperatorVersion`
- **THEN** the step prints a warning that the bundles were not checked, instead of saying both are unchanged

#### Scenario: G1 still fails a release pull request

- **WHEN** a release pull request's tree pins a library version whose docs bundle is not published
- **THEN** the release-pin step exits non-zero naming the bundle, as before
