## 1. Goreleaser Configuration

- [x] 1.1 Create `.goreleaser.yml` with project name, build targets (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64), and ldflags matching `internal/version/version.go`
- [x] 1.2 Configure `archives.format: binary` for raw binary output (no tarballs)
- [x] 1.3 Configure `checksum` section to produce `checksums.txt`
- [x] 1.4 Configure `changelog` section with conventional commit grouping (feat, fix, perf, refactor) and exclusions (docs, test, ci, chore)
- [x] 1.5 Verify goreleaser config is valid by running `goreleaser check`

## 2. CI Workflow

- [x] 2.1 Create `.github/workflows/ci.yml` with `workflow_dispatch` as the only active trigger; add push trigger commented out
- [x] 2.2 Add `lint` job: `runs-on: self-hosted`, steps for checkout, setup-go 1.25.0, and `golangci-lint run ./...`
- [x] 2.3 Add `unit` job: `runs-on: self-hosted`, steps for checkout, setup-go 1.25.0, and `go test ./internal/...`
- [x] 2.4 Confirm both jobs have no `needs:` dependency (parallel execution)

## 3. PR Workflow

- [x] 3.1 Create `.github/workflows/pr.yml` with `workflow_dispatch` as the only active trigger; add pull_request trigger (branches: [main]) commented out
- [x] 3.2 Add `lint` job matching ci.yml lint job
- [x] 3.3 Add `unit` job matching ci.yml unit job
- [x] 3.4 Add `registry` job: setup-go, set `OPM_REGISTRY` env var, run `go test ./internal/builder/... -v`
- [x] 3.5 Add `integration` job: setup-go, install kind, `kind create cluster --name opm-dev --config hack/kind-config.yaml`, run all 5 integration test scripts (`go run tests/integration/*/main.go`), `kind delete cluster` in a cleanup step that runs even on failure
- [x] 3.6 Add `e2e` job: setup-go, run `go test ./tests/e2e/... -v`
- [x] 3.7 Confirm all five jobs have no `needs:` dependency (parallel execution)

## 4. Release Workflow

- [x] 4.1 Create `.github/workflows/release.yml` with `workflow_dispatch` as the only active trigger; add tag push trigger (`tags: ['v*']`) commented out
- [x] 4.2 Add `test` job: `runs-on: self-hosted`, runs lint + unit + registry + integration (with kind cluster) + e2e
- [x] 4.3 Add `release` job: `needs: [test]`, `runs-on: self-hosted`, checkout with `fetch-depth: 0`, setup-go 1.25.0, run goreleaser (via `goreleaser/goreleaser-action` or direct binary)
- [x] 4.4 Ensure `GITHUB_TOKEN` is available to the release job for GitHub Release creation
