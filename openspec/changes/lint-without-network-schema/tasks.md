## 1. Offline configuration check

- [x] 1.1 Add `.golangci-lint-version` (`v2.11.4`), the schema `.github/golangci-lint/golangci.v2.11.jsonschema.json` (bytes as in design.md) and `.github/golangci-lint/SHA256SUMS`; verify with `sha256sum -c` in that directory
- [x] 1.2 Write `.github/scripts/lint-config-check.sh` per design.md; verify it fails against the unchanged workflows (the behaviour before this change) and that `shellcheck` is clean
- [x] 1.3 Change the `golangci-lint` step in `pr.yml` and `ci.yml` (`version-file`, `verify: false`) and add the "Verify the linter configuration (offline)" step after it in both; verify the script exits 0 and `actionlint` is clean on both files
- [x] 1.4 Write `.github/scripts/lint-config-check-test.sh` (one scenario per refusal in design.md, plus the pass case and an invalid `.golangci.yml`); add the step to the `Lint` job of `pr.yml`; verify it exits 0, and that the script exits 0 in a network namespace with no interfaces
- [x] 1.5 `Taskfile.yml`: `lint` runs the check first; add `lint:config` and `lint:config:test`; verify `task lint` and `task lint:config:test` exit 0
- [x] 1.6 `task lint`, `task openspec:check` and `task cascade:wiring:check` green, then commit `ci(lint): verify the linter config against a committed schema`

## 2. Contributor docs

- [ ] 2.1 `AGENTS.md`: the `task lint` entry, the two new tasks, and how to move the linter version (version file, schema, checksum); verify every named path and task exists
- [ ] 2.2 `task lint` and `task openspec:check` green, then commit `docs: say how to move the golangci-lint version`
