## REMOVED Requirements

### Requirement: The cli publishes one docs bundle that carries its pins
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The repository declares one docs-kit project, `cli`, in `docs-kit.cue`", "The docs bundle carries the authored pages and the generated command reference together and `task docs:bundle` builds it".

### Requirement: The dump program prints the command tree and the pins the cli compiles in
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`go run ./hack/docskit-dump` prints the command tree as one `docs.opmodel.dev/cobradump/v1` document", "`go run ./hack/docskit-dump pins` prints the pins the cli compiles in as one `docs.opmodel.dev/pins/v1` document", "Two runs of the dump program print the same bytes", "Any other argument to the dump program exits 1 with a usage line", "The `opm` binary does not contain the dump program or `cobradump`".

## ADDED Requirements

### Requirement: The repository declares one docs-kit project, `cli`, in `docs-kit.cue`

The repository SHALL declare one docs-kit project, `cli`, in `docs-kit.cue`: placed in a site version's `/docs/` tree, owning `reference/cli/`, versioned from tags with the prefix `v`, with `pins` for `library`, `core` and `opm-operator` printed by `go run ./hack/docskit-dump pins`, built from a `cobra` source whose dump comes from `go run ./hack/docskit-dump` and a `markdown` source over `docs/site`.

#### Scenario: The cli project is declared in docs-kit.cue

- **WHEN** the repository's `docs-kit.cue` is read
- **THEN** it SHALL declare one docs-kit project, `cli`, built from a `cobra` source whose dump comes from `go run ./hack/docskit-dump` and a `markdown` source over `docs/site`

### Requirement: The docs bundle carries the authored pages and the generated command reference together and `task docs:bundle` builds it

The bundle SHALL carry the authored pages under `docs/site/` and the generated command reference together (docs-kit DESIGN decision 20), and its `manifest.json` SHALL record the pins.
`task docs:bundle` SHALL build it into `out/cli/` and `task docs:bundle:check` SHALL build and lint it without publishing.

#### Scenario: The bundle holds both kinds of page and the pins

- **WHEN** `task docs:bundle` runs on a clean checkout
- **THEN** `out/cli/content/` holds `reference/registry-namespaces.md` (authored), `reference/cli/_index.md` and `reference/cli/opm-module.md` (generated), and `out/cli/manifest.json` has `pins` with exactly `library`, `core` and `opm-operator`

#### Scenario: A dev pin is refused

- **WHEN** `go.mod` replaces the library with a local directory
- **THEN** `task docs:bundle:check` fails, naming the library pin `(devel)` as not a release version

### Requirement: `go run ./hack/docskit-dump` prints the command tree as one `docs.opmodel.dev/cobradump/v1` document

`go run ./hack/docskit-dump` SHALL print one `docs.opmodel.dev/cobradump/v1` document of the tree `internal/cmd.NewRootCmd` builds.

#### Scenario: The dump prints the command tree

- **WHEN** `go run ./hack/docskit-dump` runs
- **THEN** it SHALL print one `docs.opmodel.dev/cobradump/v1` document of the tree `internal/cmd.NewRootCmd` builds

### Requirement: `go run ./hack/docskit-dump pins` prints the pins the cli compiles in as one `docs.opmodel.dev/pins/v1` document

`go run ./hack/docskit-dump pins` SHALL print one `docs.opmodel.dev/pins/v1` document: `library` the version of `github.com/open-platform-model/library` the program links, `core` the exact release the library's `schema.DefaultSchemaModule` names, `opm-operator` the `PinnedOperatorVersion` the cli records beside its pinned operator module (the operator release that module deploys, read from `internal/operator/pin.go` without a registry), each without a leading `v`.

#### Scenario: The pins equal the sources the site read

- **WHEN** `go.mod` requires library `v1.0.0-beta.2`, whose `DefaultSchemaModule` is `opmodel.dev/core@v2.0.0-beta.2`, and `internal/operator/pin.go` records `PinnedOperatorVersion` `v1.0.0-beta.5`
- **THEN** `go run ./hack/docskit-dump pins` prints `{"schema": "docs.opmodel.dev/pins/v1", "pins": {"core": "2.0.0-beta.2", "library": "1.0.0-beta.2", "opm-operator": "1.0.0-beta.5"}}`, and the program's test passes

#### Scenario: A library that pins only a core major is refused

- **WHEN** the linked library's `DefaultSchemaModule` is `opmodel.dev/core@v2`
- **THEN** `go run ./hack/docskit-dump pins` exits 1, naming the value

### Requirement: Two runs of the dump program print the same bytes

Two runs SHALL print the same bytes.
(Requirements: `go run ./hack/docskit-dump` prints the command tree as one `docs.opmodel.dev/cobradump/v1` document; `go run ./hack/docskit-dump pins` prints the pins the cli compiles in as one `docs.opmodel.dev/pins/v1` document)

#### Scenario: Two runs print the same bytes

- **WHEN** `go run ./hack/docskit-dump` or `go run ./hack/docskit-dump pins` runs twice
- **THEN** the two runs SHALL print the same bytes

### Requirement: Any other argument to the dump program exits 1 with a usage line

Any other argument SHALL exit 1 with `docskit-dump: usage: docskit-dump [pins]` on stderr.
(Requirements: `go run ./hack/docskit-dump` prints the command tree as one `docs.opmodel.dev/cobradump/v1` document; `go run ./hack/docskit-dump pins` prints the pins the cli compiles in as one `docs.opmodel.dev/pins/v1` document)

#### Scenario: A wrong argument is refused

- **WHEN** a developer runs `go run ./hack/docskit-dump pin`
- **THEN** it exits 1 with the usage line and prints nothing on stdout

### Requirement: The `opm` binary does not contain the dump program or `cobradump`

The `opm` binary SHALL NOT contain the program or `cobradump`.
(Requirements: `go run ./hack/docskit-dump` prints the command tree as one `docs.opmodel.dev/cobradump/v1` document; `go run ./hack/docskit-dump pins` prints the pins the cli compiles in as one `docs.opmodel.dev/pins/v1` document)

#### Scenario: The opm binary holds neither the program nor cobradump

- **WHEN** the `opm` binary is built
- **THEN** it SHALL NOT contain the program or `cobradump`
