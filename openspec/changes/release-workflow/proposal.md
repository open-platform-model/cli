## Why

The CLI has no CI/CD pipeline. There is no automated testing on commits or PRs, no binary release process, and no changelog generation. This blocks reliable delivery and makes contribution riskier.

## What Changes

- Add GitHub Actions workflow (`ci.yml`) that runs lint and unit tests on every push
- Add GitHub Actions workflow (`pr.yml`) that runs the full test suite (lint, unit, registry, integration, e2e) on pull requests
- Add GitHub Actions workflow (`release.yml`) that runs the full test suite then builds and publishes a GitHub Release via goreleaser on tag pushes
- Add `.goreleaser.yml` to define build targets, binary artifacts, checksums, and changelog generation
- All workflows target a self-hosted runner and are initially disabled (triggers commented out, `workflow_dispatch` only)

## Capabilities

### New Capabilities

- `ci-workflow`: Automated lint and unit test checks on every push to any branch
- `pr-workflow`: Full test suite (lint, unit, registry tests, integration tests with kind cluster, e2e) gating pull requests
- `release-workflow`: Goreleaser-based release pipeline triggered on `v*` tags — produces raw binaries for 5 platforms, a checksums file, and an auto-generated changelog from conventional commits

### Modified Capabilities

## Impact

- New files: `.github/workflows/ci.yml`, `.github/workflows/pr.yml`, `.github/workflows/release.yml`, `.goreleaser.yml`
- No changes to application source code
- Requires a self-hosted GitHub Actions runner (to be configured separately)
- Requires goreleaser installed on the runner, or via `goreleaser/goreleaser-action`
- Requires `kind` and `kubectl` on the runner for integration tests
- SemVer impact: MINOR (new tooling, no breaking changes)
