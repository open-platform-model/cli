## Context

See proposal.md for the failure. Facts read from source on 2026-10-08:

- `golangci/golangci-lint-action` at `ba0d7d2ec06a0ea1cb5fa41b2e4a3ab91d21278a` (v9.3.0), `src/run.ts`: `runVerify` returns at once when the `verify` input is false; otherwise it runs `<bin> config verify`. `run` calls `core.addPath(dirname(binPath))` after the install, so later steps of the job find `golangci-lint` on `PATH`. `src/version.ts`: with no `version` input and no golangci-lint line in `go.mod`, the `version-file` input is read; a file not named `.tool-versions` is taken whole, trimmed, with or without a leading `v`.
- golangci-lint v2.11.4, `pkg/commands/config_verify.go`: `config verify` takes the schema location from the hidden flag `--schema` when set, else builds the `golangci-lint.run` URL from the build version. The compiler has a `file` loader and an `https` loader. The schema holds no remote `$ref`.
- The schema the website serves for v2.11, `jsonschema/golangci.jsonschema.json` at tag `v2.11.4` of `golangci/golangci-lint` (commit `8f3b0c7ed018e57905fbd873c697e0b1ede605a5`) and `jsonschema/golangci.v2.11.jsonschema.json` on that repository's default branch are the same 169695 bytes, SHA-256 `985af311f9448d5b0964c3eda502204326dcf35d8f757192684cddc9b6615676`. The file named `golangci.v2.11.jsonschema.json` does not exist at the tag itself.

## Goals / Non-Goals

**Goals:**

- No schema download in a `Lint` job, with the configuration check kept.
- One place for the linter version; a failing check when schema, version or action pin drift.
- The same check on a laptop (`task lint`) as in CI.

**Non-Goals:**

- The download of the linter binary by the action (GitHub Releases, unchanged).
- Lint rules, other jobs, other repositories, a rollup job or timeouts for the workflow.
- Verifying the committed schema against upstream at run time; that would need the network again.

## Research & Decisions

### How to keep the configuration check without the download

**Context**: the action offers one switch, `verify`. Nothing in its README at the pinned SHA lets it use a local schema.
**Explored**: the action source and the linter source named above; runs of `golangci-lint config verify` (2.11.3) in a network namespace with no interfaces (`unshare -rn`) and with `HTTPS_PROXY` set to a closed local port.
**Options considered**:
1. `verify: false` and nothing else. Smallest, but drops the check: a mistyped key in `.golangci.yml` is then ignored in silence.
2. `verify: false`, and a `run:` step with `golangci-lint config verify --schema <committed file>`. Keeps the check, no network. Costs a committed 170 kB file that must follow the linter's minor line, and relies on a hidden flag.
3. Keep `verify: true` and retry the step. Still depends on the website; a retry hides the cause.
4. Serve the schema from a local HTTP server in the job. The linter builds the URL itself when `--schema` is unset, so this needs the same flag and more parts.
**Decision**: option 2.
**Rationale**: observed results: without network and without `--schema` the command exits 3 on the schema load; with `--schema` and the committed file it exits 0 for the repository's `.golangci.yml` and exits 3 with the schema violation for a config with an unknown key. The hidden flag is the linter's own mechanism; if a later version drops it, the check fails loudly on the bump pull request, never in silence.

### Where the version lives

**Options considered**:
1. Keep `version:` in both workflows and compare them in the check.
2. One file `.golangci-lint-version`, read by the action's `version-file` input and by the check.
**Decision**: option 2. One writer, and the action documents the input. The check refuses a `version` input, which would win over the file.

### What the check compares

The script `.github/scripts/lint-config-check.sh` MUST fail when:

- `.golangci-lint-version` is not `vX.Y.Z`;
- `.github/golangci-lint/golangci.vX.Y.jsonschema.json` is missing, or another `*.jsonschema.json` sits beside it;
- the schema does not match the one line in `.github/golangci-lint/SHA256SUMS`;
- no workflow uses the action (the check would be empty), or a use lacks `verify: false` or `version-file: .golangci-lint-version`, sets `version`, or is not SHA-pinned; or two uses pin different SHAs; or a workflow that uses the action does not run the script (the action name is matched in any letter case, quoted or not);
- `go.mod` names golangci-lint (the action reads a version there before the version file);
- no `golangci-lint` is on `PATH`;
- the installed `golangci-lint` is of another `X.Y` than the file names (a patch difference is allowed: the schema is per minor line, and a laptop may lag a patch);
- `golangci-lint config verify --schema <file>` fails.

It runs the linter with `HTTPS_PROXY` and `HTTP_PROXY` set to a closed local port and `NO_PROXY` empty, so a regression that goes back to the network fails at once instead of passing while the website is up. The workflow text is read with `awk` and `grep`, not `yq`, so the script needs nothing the job does not already have.

```text
lint-config-check.sh [--root DIR]     exit 0 ok, 1 a refusal (one line each on stderr), 2 usage
```

The checksum does not prove where the schema came from; it catches an edited or reformatted file and gives a reviewer one value to compare with upstream. The source and the procedure are in the header of the script and in `AGENTS.md`.

### Step order in the job

The action step (install and lint) runs first, then the configuration check, because the action is what installs the binary. A second action call with `install-only` before the check was considered and not taken: two calls of one action for an ordering nicety. Effect: when lint itself fails, the configuration check does not run in that job run; it runs once lint passes, and the job is red either way.

## Risks / Trade-offs

- [The schema goes stale when the linter moves a minor line] -> the check fails on the bump pull request and names the missing file.
- [`--schema` is a hidden flag] -> a removal fails the check on the bump; fallback is option 1 with the loss stated.
- [The schema is a 170 kB third-party file in the tree] -> data only, never executed; checksum recorded; `.github/` is code-owned.
- [Dependabot moves the action SHA] -> it edits both workflows in one pull request; the check refuses a split.
- [A contributor's local linter is of another minor line] -> `task lint` now fails early with a message naming both versions; before, it ran with whatever was installed.
