# docs-bundle Specification

## Purpose
The cli publishes one signed docs-kit bundle, `cli`, to `ghcr.io/open-platform-model/docs/cli`: the command reference generated from the cobra commands, the authored pages under `docs/site/`, and in its manifest the library, core and opm-operator versions the cli pins. A bundle is published for every release and as `edge` for every push to `main`, and pull requests check it. opmodel.dev anchors a site version on the cli's bundle and pulls exactly the bundles its pins name, so the pins decide which reference the site shows beside the cli's.

## Requirements

### Requirement: The cli publishes one docs bundle that carries its pins

The repository SHALL declare one docs-kit project, `cli`, in `docs-kit.cue`: placed in a site version's `/docs/` tree, owning `reference/cli/`, versioned from tags with the prefix `v`, with `pins` for `library`, `core` and `opm-operator` printed by `go run ./hack/docskit-dump pins`, built from a `cobra` source whose dump comes from `go run ./hack/docskit-dump` and a `markdown` source over `docs/site`. The bundle SHALL carry the authored pages under `docs/site/` and the generated command reference together (docs-kit DESIGN decision 20), and its `manifest.json` SHALL record the pins. `task docs:bundle` SHALL build it into `out/cli/` and `task docs:bundle:check` SHALL build and lint it without publishing.

#### Scenario: The bundle holds both kinds of page and the pins

- **WHEN** `task docs:bundle` runs on a clean checkout
- **THEN** `out/cli/content/` holds `reference/registry-namespaces.md` (authored), `reference/cli/_index.md` and `reference/cli/opm-module.md` (generated), and `out/cli/manifest.json` has `pins` with exactly `library`, `core` and `opm-operator`

#### Scenario: A dev pin is refused

- **WHEN** `go.mod` replaces the library with a local directory
- **THEN** `task docs:bundle:check` fails, naming the library pin `(devel)` as not a release version

### Requirement: The dump program prints the command tree and the pins the cli compiles in

`go run ./hack/docskit-dump` SHALL print one `docs.opmodel.dev/cobradump/v1` document of the tree `internal/cmd.NewRootCmd` builds, and `go run ./hack/docskit-dump pins` one `docs.opmodel.dev/pins/v1` document: `library` the version of `github.com/open-platform-model/library` the program links, `core` the exact release the library's `schema.DefaultSchemaModule` names, `opm-operator` the `PinnedOperatorVersion` the cli installs, each without a leading `v`. Two runs SHALL print the same bytes. Any other argument SHALL exit 1 with `docskit-dump: usage: docskit-dump [pins]` on stderr. The `opm` binary SHALL NOT contain the program or `cobradump`.

#### Scenario: The pins equal the sources the site read

- **WHEN** `go.mod` requires library `v1.0.0-beta.2`, whose `DefaultSchemaModule` is `opmodel.dev/core@v2.0.0-beta.2`, and `PinnedOperatorVersion` is `v1.0.0-beta.5`
- **THEN** `go run ./hack/docskit-dump pins` prints `{"schema": "docs.opmodel.dev/pins/v1", "pins": {"core": "2.0.0-beta.2", "library": "1.0.0-beta.2", "opm-operator": "1.0.0-beta.5"}}`, and the program's test passes

#### Scenario: A library that pins only a core major is refused

- **WHEN** the linked library's `DefaultSchemaModule` is `opmodel.dev/core@v2`
- **THEN** `go run ./hack/docskit-dump pins` exits 1, naming the value

#### Scenario: A wrong argument is refused

- **WHEN** a developer runs `go run ./hack/docskit-dump pin`
- **THEN** it exits 1 with the usage line and prints nothing on stdout

### Requirement: Every cli release publishes its docs bundle

`release.yml` SHALL run a `publish-docs` job that calls docs-kit's `publish.yml` with `project: cli`, `mode: release`, `setup-go: true` and the release's tag (the release-please tag, or the `tag` input on a manual run), only when `goreleaser` succeeded.

#### Scenario: A published release publishes its bundle

- **WHEN** the release PR for `v1.0.0-beta.6` merges and `goreleaser` publishes the release
- **THEN** `publish-docs` publishes `ghcr.io/open-platform-model/docs/cli` for `1.0.0-beta.6`, its manifest carrying that tree's pins

#### Scenario: A release left a draft has no bundle

- **WHEN** `publish-templates` fails, so `goreleaser` refuses to build and the release stays a draft
- **THEN** `publish-docs` is skipped; the manual run that finishes the draft publishes the bundle

### Requirement: Pull requests check the bundle and main publishes edge

`.github/workflows/docs.yml` SHALL run `publish.yml` with `setup-go: true` in `check` mode on every pull request (permissions `contents: read`, `packages: read`), in `edge` mode on every push to `main`, and on `workflow_dispatch` in the `release` or `revision` mode with a `tag` and, for `revision`, a `fix` commit; the publishing jobs SHALL hold only `contents: read`, `packages: write` and `id-token: write`, and the workflow SHALL declare `permissions: {}` at the top.

#### Scenario: A nondeterministic dump fails the check

- **WHEN** a pull request makes a flag default depend on the time of day
- **THEN** the `Docs / check` job fails, naming `go run ./hack/docskit-dump` and docs-kit C14

#### Scenario: A tag without the dump program cannot be backfilled

- **WHEN** the owner dispatches `mode: release` for `v1.0.0-beta.5`
- **THEN** the job fails, because that tree has no `hack/docskit-dump` to print the dump and the pins

### Requirement: The docs-kit release is pinned once per form and the forms agree

`.opm-docs-version` SHALL hold one docs-kit release tag, and every `open-platform-model/docs-kit/.github/workflows/publish.yml@<ref>` under `.github/workflows/` SHALL name that same tag, never a SHA (docs-kit C5, C9). `task docs:pins:check` SHALL refuse a disagreement offline, and `task docs:bundle:check` SHALL run it first.

#### Scenario: A half-moved pin is refused

- **WHEN** `.opm-docs-version` names `v0.4.0` and `release.yml` still names `publish.yml@v0.3.0`
- **THEN** `task docs:pins:check` fails, listing the stale ref

#### Scenario: Agreeing pins pass

- **WHEN** both forms name `v0.4.0`
- **THEN** `task docs:pins:check` passes
