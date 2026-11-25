# Evidence: open Version and the identity check

Throwaway checks from the explore of cli issue 326. Nothing here is product code. `go/` is its own
Go module, so the cli's `./...` build, lint and test commands do not load it.

## Hypothesis

Core's `#IdentityPackage` treats an open `Version` as incomplete, not as wrong. The refusal comes
from the cli's check, which validates in concrete mode and filters errors by text.

## Setup

- `go/main.go`: a copy of the `VersionMajor` derivation of core v2.0.0-beta.4
  (`src/identity_package.cue`, `src/types.cue`), validated in plain and in concrete mode. CUE
  v0.17.1, the version in the cli's `go.mod`.
- `open/`: `tests/fixtures/valid/simple-module` with `Version: string` in `identity/identity.cue`.
- `wired/`: `open/` with `metadata.version: id.Version`, as the templates wire it.

## Run

```sh
cd go && go run .
# from the cli repo root, with R the default registry mapping of internal/config/templates.go:
go run ./cmd/opm module vet <evidence>/open --registry "$R"
go run ./cmd/opm module publish <evidence>/open --version 0.1.0 --dry-run --registry "$R"
```

## Outcome

`go run .` prints:

```text
== open
  plain: <nil>
  concrete: [#IP.Version: incomplete value ...] [#IP.VersionMajor: error in call to strings.SplitN: non-concrete value string]
== absent
  plain: <nil>
  concrete: [#IP.Version: field is required but not present] [#IP.VersionMajor: required field missing: Version]
== skew
  plain: #IP.VersionMajor: conflicting values "v0" and "v1"
== badfmt
  plain: #IP.Version: invalid value "abc" (out of bound ...)
```

`opm module vet open` and `opm module publish open --version 0.1.0 --dry-run` both exit 2 with
`the identity package does not conform to core's #IdentityPackage` and `error in call to
strings.SplitN` at `VersionMajor`.

Hypothesis held. Plain validation passes an open and an absent `Version` and still refuses a skew
and a malformed value. The `non-concrete value` error escapes the cli's `incomplete value` text
filter.

The library loader requires a concrete `metadata.version` (`opm/internal/loader/shape.go`,
`RequiredConcreteFields`), so `wired/` would still fail at vet's render once the identity check
passes. This was read from the source, not run, because the identity check stops `wired/` first.
