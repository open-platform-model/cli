## MODIFIED Requirements

### Requirement: The cli publishes one docs bundle that carries its pins

The repository SHALL declare one docs-kit project, `cli`, in `docs-kit.cue`: placed in a site version's `/docs/` tree, owning `reference/cli/`, versioned from tags with the prefix `v`, with `pins` for `library`, `core` and `opm-controller` printed by `go run ./hack/docskit-dump pins`, built from a `cobra` source whose dump comes from `go run ./hack/docskit-dump` and a `markdown` source over `docs/site`. The bundle SHALL carry the authored pages under `docs/site/` and the generated command reference together (docs-kit DESIGN decision 20), and its `manifest.json` SHALL record the pins. `task docs:bundle` SHALL build it into `out/cli/` and `task docs:bundle:check` SHALL build and lint it without publishing.

#### Scenario: The bundle holds both kinds of page and the pins

- **WHEN** `task docs:bundle` runs on a clean checkout
- **THEN** `out/cli/content/` holds `reference/registry-namespaces.md` (authored), `reference/cli/_index.md` and `reference/cli/opm-module.md` (generated), and `out/cli/manifest.json` has `pins` with exactly `library`, `core` and `opm-controller`

#### Scenario: A dev pin is refused

- **WHEN** `go.mod` replaces the library with a local directory
- **THEN** `task docs:bundle:check` fails, naming the library pin `(devel)` as not a release version

### Requirement: The dump program prints the command tree and the pins the cli compiles in

`go run ./hack/docskit-dump` SHALL print one `docs.opmodel.dev/cobradump/v1` document of the tree `internal/cmd.NewRootCmd` builds, and `go run ./hack/docskit-dump pins` one `docs.opmodel.dev/pins/v1` document: `library` the version of `github.com/open-platform-model/library` the program links, `core` the exact release the library's `schema.DefaultSchemaModule` names, `opm-controller` the `PinnedControllerVersion` the cli records beside its pinned controller module (the controller release that module deploys, read from `internal/controller/pin.go` without a registry), each without a leading `v`. Two runs SHALL print the same bytes. Any other argument SHALL exit 1 with `docskit-dump: usage: docskit-dump [pins]` on stderr. The `opm` binary SHALL NOT contain the program or `cobradump`.

#### Scenario: The pins equal the sources the site read

- **WHEN** `go.mod` requires library `v1.0.0-beta.2`, whose `DefaultSchemaModule` is `opmodel.dev/core@v2.0.0-beta.2`, and `internal/controller/pin.go` records `PinnedControllerVersion` `v1.0.0-beta.5`
- **THEN** `go run ./hack/docskit-dump pins` prints `{"schema": "docs.opmodel.dev/pins/v1", "pins": {"core": "2.0.0-beta.2", "library": "1.0.0-beta.2", "opm-controller": "1.0.0-beta.5"}}`, and the program's test passes

#### Scenario: A library that pins only a core major is refused

- **WHEN** the linked library's `DefaultSchemaModule` is `opmodel.dev/core@v2`
- **THEN** `go run ./hack/docskit-dump pins` exits 1, naming the value

#### Scenario: A wrong argument is refused

- **WHEN** a developer runs `go run ./hack/docskit-dump pin`
- **THEN** it exits 1 with the usage line and prints nothing on stdout
