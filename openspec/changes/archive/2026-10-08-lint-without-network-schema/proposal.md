## Why

The required `Lint` check failed twice on 2026-10-08 (runs 37772352800 and 37782005229, first attempts) and blocked two pull requests, with no fault in either. The `golangci/golangci-lint-action` step runs `golangci-lint config verify`, which downloads its JSON schema from `https://golangci-lint.run/jsonschema/golangci.v2.11.jsonschema.json` on every run, and the download timed out (`context deadline exceeded`). A required check must not depend on a third-party website being fast.

## What Changes

- The `Lint` job in `pr.yml` and in `ci.yml` turns the action's own configuration check off (`verify: false`, the input the action documents) and runs the same check, `golangci-lint config verify`, against a schema file committed in the repository. The check stays; only the download goes.
- The linter version moves out of the two workflows into one file, `.golangci-lint-version`, which the action reads through its `version-file` input.
- A new script, `.github/scripts/lint-config-check.sh`, runs the offline configuration check and refuses a tree where the version file, the committed schema, its recorded checksum and the workflows' use of the action disagree. `task lint` runs it before the linters, so a contributor sees the same result as CI.
- A scenario test for that script runs in the `Lint` job of `pr.yml` and as `task lint:config:test`.
- `AGENTS.md` says how to move the linter version.

No lint rule changes. No other job changes. No new action, permission, secret or token.

SemVer class: none (PATCH at most after GA). The change is typed `ci`, which release-please hides, so it cuts no release; before GA nothing ships as beta.N+1 from it. Not breaking.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ci-workflow`: the lint requirement names the version file instead of a version literal; two requirements are added, one for the offline configuration check and one for the agreement of version, schema and action pin.
- `pr-workflow`: the lint requirement names the version file instead of a version literal.

## Impact

- `.github/workflows/pr.yml` and `.github/workflows/ci.yml`: the `golangci-lint` step of the `Lint` job, plus one new step each (two in `pr.yml`, which also runs the script's test).
- New files: `.golangci-lint-version`, `.github/golangci-lint/golangci.v2.11.jsonschema.json`, `.github/golangci-lint/SHA256SUMS`, `.github/scripts/lint-config-check.sh`, `.github/scripts/lint-config-check-test.sh`.
- `Taskfile.yml`: `lint` runs the configuration check first; new `lint:config` and `lint:config:test`.
- `AGENTS.md`: the lint command entries and the linter bump procedure.
- The linter binary download from GitHub Releases by the action is unchanged; this change removes only the schema download.
- Other repositories are not changed.
