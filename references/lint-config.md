# Lint configuration and moving golangci-lint

Covers the linters and limits in `.golangci.yml`, and the steps to move the golangci-lint version
without breaking the offline config check. It does not cover Go style; that is in `AGENTS.md`.

## Contents

- Highlights
- Where the version lives
- Moving the version
- The offline check
- Pitfalls
- Check before you finish

## Highlights

- `golangci-lint` runs in readonly module download mode.
- Key linters: `errorlint`, `errname`, `gocritic`, `gocyclo`, `gosec`, `revive`, `staticcheck`,
  `tparallel`, and `depguard` (the retired Kubernetes copy paths).
- `nolint` comments name the linter and give the reason.
- `gocyclo` threshold is 15. Refactor before complexity grows.
- Tests relax `dupl`, `errcheck`, `goconst` and `gosec`.
- `.golangci.yml` excludes `examples` from lint and format. It also lists `third_party`, `builtin` and
  `experiments` patterns; none of those directories existed on 2026-10-14.

## Where the version lives

`.golangci-lint-version` is the only place that names the version CI installs. The action reads it
through `version-file`, and no workflow sets `version`. The action's own configuration check stays
off (`verify: false`), because it downloads the schema from golangci-lint.run on every run, which
failed the required `Lint` job twice on 2026-10-08. The committed schema is under
`.github/golangci-lint/`.

## Moving the version

- Patch move (`v2.11.4` to `v2.11.5`): edit `.golangci-lint-version`. The schema is per minor line
  and stays.
- Minor or major move:
  1. Edit `.golangci-lint-version`.
  2. Take `jsonschema/golangci.jsonschema.json` of `github.com/golangci/golangci-lint` at the new
     tag. Save it in `.github/golangci-lint/` as `golangci.vX.Y.jsonschema.json`.
  3. Remove the old schema file.
  4. Rewrite `SHA256SUMS`: `sha256sum golangci.vX.Y.jsonschema.json > SHA256SUMS`, run in that
     directory.
  5. Say in the PR body where the file came from, so the reviewer can compare the checksum.

## The offline check

`task lint:config` runs `.github/scripts/lint-config-check.sh`. CI's `Lint` jobs run the same script.
It checks, with no network, that:

- `.golangci-lint-version`, the committed schema and the workflows' use of the golangci-lint action
  agree;
- the installed golangci-lint is of the same minor line (a patch difference is allowed);
- `golangci-lint config verify --schema <file>` passes for `.golangci.yml`.

The linter runs with its proxy variables set to a closed local port, so a run that reaches for the
network fails at once. `task lint:config:test` is the scenario test of the script.

`task lint:config` needs a local golangci-lint of the same minor line. It uses the hidden `--schema`
flag of `golangci-lint config verify`. If a release drops that flag, the check fails on the bump PR
and needs a new design, not a silent `verify: false`.

## Pitfalls

- Setting `version` in a workflow: the version file is the only source.
- A minor move without the new schema and `SHA256SUMS`: `task lint:config` fails.
- Turning `verify` back on in the action: it needs the network and failed the required job.
- Moving the version with a local golangci-lint of another minor line: the check refuses it.

## Check before you finish

- [ ] `task lint:config` and `task lint:config:test` pass.
- [ ] `task lint` passes.
- [ ] A minor move names the schema source in the PR body.
