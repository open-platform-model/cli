## ADDED Requirements

### Requirement: Pull requests that move library or the operator warn about missing docs bundles

`pr.yml`'s `lint` job SHALL run the step "Docs bundles for moved pins" on every pull request. The step SHALL fetch the pull request's base commit and run `.github/scripts/docs-pins-check.sh --warn --moved-from <base sha>`. When the `github.com/open-platform-model/library` version in `go.mod` and `PinnedOperatorVersion` in `internal/operator/manifest.go` both equal the base's, it SHALL check nothing and say so. Otherwise it SHALL run `go run ./hack/docskit-dump pins` on the tree and look up each printed pin anonymously at `ghcr.io/open-platform-model/docs/<project>:<pin>`, and SHALL print a GitHub warning annotation, also written to the job summary, for each bundle that is missing (401, 403 or 404), each lookup that fails, a program that does not build, and a base commit that cannot be fetched. The step SHALL exit zero in every one of those cases, so it never fails the job: the blocking check stays G1 on the release pull request. The step runs with the job's read-only token. Source: security pass 2026-10-04, finding CAS-R2 (the check left the cascade task).

G1 (`release-pin-check.sh`) SHALL run the same script without `--warn` and report each line it prints as a G1 violation; a non-zero exit with no line SHALL itself be a violation.

#### Scenario: A cascade pull request moves library to a version without a bundle

- **WHEN** the cascade pull request moves library to a version whose docs bundle is not published
- **THEN** the `lint` job shows a warning naming `library`, the version and the bundle reference, and the job's result is decided by its other steps

#### Scenario: An ordinary pull request checks nothing

- **WHEN** a pull request changes neither the library version nor `PinnedOperatorVersion`
- **THEN** the step builds nothing, prints that both are unchanged, and exits zero

#### Scenario: G1 still fails a release pull request

- **WHEN** a release pull request's tree pins a library version whose docs bundle is not published
- **THEN** the release-pin step exits non-zero naming the bundle, as before
