## Why

The cli builds on `cuelang.org/go` twice: through the library, and directly (`go.mod:7`, `v0.17.1`; `internal/cuemod` drives `cuelang.org/go/cmd/cue/cmd`). The library is where a CUE bump is tested against the kernel: its own pull request runs the closedness canary. A Dependabot pull request in the cli that moves `cuelang.org/go` alone skips that check and lets the cli build against a CUE version no library release was tested with.

The owner decided this in the kernel beta.1 plan walkthrough (decision j4, 2026-10-03): "cuelang.org/go moves only through a library release: Dependabot in cli and operator ignores it, and the rule is documented in RELEASING.md." This change is the cli part. The operator and the workspace RELEASING.md rule are separate changes.

At origin/main (`5180cad1`) `.github/dependabot.yml` ignores only `github.com/open-platform-model/*` in its `gomod` block, so Dependabot still proposes `cuelang.org/go` bumps.

## What Changes

- `.github/dependabot.yml`: the `gomod` update also ignores `cuelang.org/go`, with a comment in the same style as the existing `github.com/open-platform-model/*` ignore. The comment says that CUE moves only through a library release and reaches the cli through the cascade's library bump, whose `go get` of the library raises `cuelang.org/go` by minimal version selection (kept by `go mod tidy`).
- Main spec `repo-automation`: one ADDED requirement, "Dependabot leaves cuelang.org/go to library releases", and one MODIFIED requirement, "Dependabot leaves OPM Go modules to the release cascade", whose sentence about other Go modules now excepts `cuelang.org/go` (both scenarios kept). The archive commit also updates the spec's Purpose line to name `cuelang.org/go`.

Not in this change:

- the opm-operator ignore (its own change, after opm-operator#225);
- the workspace RELEASING.md rule (its own change in the workspace repo);
- the CUE CLI version pins, the literal `go install cuelang.org/go/cmd/cue@v0.17.1` installs in `cascade-task.yml`, `ci.yml`, `e2e-cluster.yml`, `pr.yml` (three times) and `publish-fixtures.yml`, which are not Go module requires and which Dependabot does not manage (`pr.yml:159` and `release.yml:246` already derive the version from `go.mod`);
- any change to how `task deps:cascade` moves the library.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `repo-automation`: Dependabot also leaves `cuelang.org/go` alone.

## Impact

- Files: `.github/dependabot.yml` only. No Go code, command, flag or output changes.
- Release class: `ci`. No release is cut; SemVer impact none (after GA as well).
- Open pull requests: none of the open Dependabot pull requests (cli#208, #222, #251, #252, #273) moves `cuelang.org/go`, so none is closed by this change.
