## MODIFIED Requirements

### Requirement: The dump program prints the command tree and the pins the cli compiles in

`go run ./hack/docskit-dump` SHALL print one `docs.opmodel.dev/cobradump/v1` document of the tree `internal/cmd.NewRootCmd` builds, and `go run ./hack/docskit-dump pins` one `docs.opmodel.dev/pins/v1` document: `library` the version of `github.com/open-platform-model/library` the program links, `core` the exact release the library's `schema.DefaultSchemaModule` names, `opm-operator` the `PinnedOperatorVersion` the cli records beside its pinned operator module (the operator release that module deploys, read from `internal/operator/pin.go` without a registry), each without a leading `v`. Two runs SHALL print the same bytes. Any other argument SHALL exit 1 with `docskit-dump: usage: docskit-dump [pins]` on stderr. The `opm` binary SHALL NOT contain the program or `cobradump`.

#### Scenario: The pins equal the sources the site read

- **WHEN** `go.mod` requires library `v1.0.0-beta.2`, whose `DefaultSchemaModule` is `opmodel.dev/core@v2.0.0-beta.2`, and `internal/operator/pin.go` records `PinnedOperatorVersion` `v1.0.0-beta.5`
- **THEN** `go run ./hack/docskit-dump pins` prints `{"schema": "docs.opmodel.dev/pins/v1", "pins": {"core": "2.0.0-beta.2", "library": "1.0.0-beta.2", "opm-operator": "1.0.0-beta.5"}}`, and the program's test passes

#### Scenario: A library that pins only a core major is refused

- **WHEN** the linked library's `DefaultSchemaModule` is `opmodel.dev/core@v2`
- **THEN** `go run ./hack/docskit-dump pins` exits 1, naming the value

#### Scenario: A wrong argument is refused

- **WHEN** a developer runs `go run ./hack/docskit-dump pin`
- **THEN** it exits 1 with the usage line and prints nothing on stdout
