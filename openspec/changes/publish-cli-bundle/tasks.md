Delivery: one PR per section (proposal.md). Each section has its own gate, named as in docs-kit `docs/orchestration.md`; do not start a section before its gate holds.

## 1. Adopt docs-kit

Gate G2-cli: docs-kit's `add-cobra-extractor`, `add-authored-docs` and `generalize-build-assembly` are released, and the tag `cobradump/v0.1.0` exists. Use the first docs-kit release that carries all three as `vX.Y.Z` below.

- [ ] 1.1 `go.mod`: `go get github.com/open-platform-model/docs-kit/cobradump@v0.1.0`. Verify: `go mod graph` gains only the cobradump node and its edges to cobra and pflag already in the graph; `go.sum` gains only cobradump lines (design.md D2).
- [ ] 1.2 `hack/docskit-dump/main.go` as design.md D1. Verify: `go run ./hack/docskit-dump | head -c 200` starts a `docs.opmodel.dev/cobradump/v1` document; `go run ./hack/docskit-dump pins` prints today's three pins without `v`; `go run ./hack/docskit-dump pin` exits 1 with the usage line; `go build ./cmd/opm` and `go tool nm` show no `cobradump` symbol in `opm`.
- [ ] 1.3 `hack/docskit-dump/main_test.go`: pins equal `go.mod`'s library require, that library's `DefaultSchemaModule` line and `PinnedOperatorVersion`; two dumps are byte-identical. Verify: `go test ./hack/docskit-dump/...` passes, including inside `task test`; confirm `debug.ReadBuildInfo` reports the library dependency in the test binary as it does under `go run`, and record it in design.md D1.
- [ ] 1.4 `.opm-docs-version`: `vX.Y.Z`. `.tasks/opm-docs.sh`: copy catalog_opm's byte for byte. `.gitignore`: `/out/` and `/.bin/`. `Taskfile.yml`: `tools:opm-docs`, `docs:bundle` (`--project cli --out out`), `docs:pins:check`, `docs:bundle:check` as catalog_opm has them; `task check` runs `docs:bundle:check` after `docs:reference:check`.
- [ ] 1.5 `docs-kit.cue` exactly as design.md D3. Verify: `task docs:bundle` writes `out/cli/` with the pins in `manifest.json`; `task docs:bundle:check` passes (it runs the dump twice).
- [ ] 1.6 Parity: diff `out/cli/content/reference/cli/` against `docs/site/reference/cli/` with cmdref's marker comments removed. Verify: no difference beyond those docs-kit's C19 parity record lists; record the commit and the result in design.md. An unlisted difference stops the section.
- [ ] 1.7 `.github/workflows/docs.yml`: catalog_opm's with `project: cli`, `setup-go: true` on every job, tags `vX.Y.Z`, `publish.yml@vX.Y.Z`, the backfill floor in the dispatch comment (design.md D4). `.github/workflows/release.yml`: `publish-docs` after `goreleaser` (design.md D4). Verify: `actionlint` clean; `task docs:pins:check` passes.
- [ ] 1.8 `AGENTS.md`: a "Docs bundles" paragraph under "Documentation And Output Conventions" (PR check, edge on `main`, a bundle per published release; the pins it records and why a cli release needs bundles for them first; preview with `task docs:bundle` or `opm-docs serve`; recover a published release without a bundle with `gh workflow run docs.yml --ref main -f mode=release -f tag=vX.Y.Z`; fix a released page with `mode=revision`, dispatched by hand (cli#282); after the site reads the bundle, help text reaches it only by a release), `hack/docskit-dump/` in "Repository Layout", the tasks in "Core commands".
- [ ] 1.9 `task openspec:check` passes; `task lint` and `task test` green (and `task check`), then commit `ci(docs): publish the cli docs bundle with docs-kit`.

## 2. Publish the first anchor bundle

Gate G2-pins, checked before the release PR merges: core `v2.0.0-beta.1`, library `v1.0.0-beta.1` and opm-operator `v1.0.0-beta.4` have verified backfilled bundles (owner decision 2026-10-03; the siblings `publish-definitions-bundle`, `publish-go-api-bundle` and `publish-crd-bundle` dispatch them), the cli's `main` still pins exactly those (or every version it pins has a bundle), and the local `opm-docs` carries `pull-docs-placement` (gate G2-site). Owner: hold release PR cli#276 (`v1.0.0-beta.6`) until then. This section's deliverable is a publishing operation (the owner's), so its steps are the implementation.

- [ ] 2.1 On the open release PR's head: `task docs:bundle`, then `opm-docs pull --local cli@v1.0=out/cli` with a scratch `bundles.cue` holding opmodel.dev's planned `docs` (cli, core, library, opm-operator) and `versions."v1.0": {anchor: {project: "cli", tag: "1.0"}, pinned: ["library", "core", "opm-operator"]}` (design.md D5). Verify: the pull succeeds anonymously, resolving the three pinned bundles.
- [ ] 2.2 Owner: merge the cli release PR; `publish-docs` publishes `docs/cli` after `goreleaser`. Check `ghcr.io/open-platform-model/docs/cli` is public and linked to `open-platform-model/cli` (change it in the package settings if not).
- [ ] 2.3 Verify: the full, release, minor and major tags with `cosign verify` and docs-kit C9's identity flags; the scratch pull of 2.1 without `--local`, anonymously (gate G2-pins holds).
- [ ] 2.4 Record in design.md D5 the run URL, version, digest, pins and the verification; tell opmodel.dev that G2-pins holds, so its v1.0 switch may merge.
- [ ] 2.5 `task openspec:check` passes; `task lint` and `task test` green, then commit `docs(openspec): record the first cli docs bundle`.

## 3. Retire cmdref

Gate G2-switch: opmodel.dev's v1.0 reads core, cli, library and opm-operator from bundles (docs-kit `docs/orchestration.md`).

- [ ] 3.1 Delete `internal/cmdref/`, `hack/cmdref/` and `docs/site/reference/cli/`.
- [ ] 3.2 `Taskfile.yml`: delete `docs:reference`, `docs:reference:check`, their comment and the line in `task check`. `.github/workflows/ci.yml` and `pr.yml`: delete the `command-reference` jobs. Owner: drop "Command Reference (current)" from the required checks if the ruleset lists it.
- [ ] 3.3 `docs-kit.cue`: the `markdown` source becomes `{kind: "markdown", dir: "docs/site"}`. Verify: `task docs:bundle:check` passes and `out/cli/content/reference/cli/` still holds the section page and one page per top-level command.
- [ ] 3.4 `AGENTS.md`: replace the `internal/cmdref/` layout entry and the `docs:reference:check` mention in "Core commands" (design.md D6). Verify: `grep -rn "cmdref\|docs:reference" --exclude-dir=archive .` finds nothing outside `openspec/changes/`.
- [ ] 3.5 `openspec archive publish-cli-bundle --yes`; then set `openspec/specs/command-reference/spec.md`'s Purpose to the bundle-built reference. Verify: `task openspec:check` passes.
- [ ] 3.6 `task lint` and `task test` green (and `task check`), then commit `ci(docs): retire cmdref and the committed command reference`.
