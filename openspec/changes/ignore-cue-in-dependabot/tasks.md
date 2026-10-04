## 1. Ignore cuelang.org/go in Dependabot

- [x] 1.1 In `.github/dependabot.yml`, add `- dependency-name: "cuelang.org/go"` to the `ignore:` list of the `gomod` update, preceded by a comment in the style of the `github.com/open-platform-model/*` comment above it: CUE moves only through a library release (the library's pull request runs the CUE check), and reaches the cli through the cascade's library bump, whose `go get` of the library raises it by minimal version selection (kept by `go mod tidy`); cite owner decision j4 (2026-10-03) and workspace RELEASING.md, section "Pin classes". Leave the `github-actions` block and every other key unchanged.
- [x] 1.2 Check the file: it parses as YAML (`yq` or `python3 -c 'import yaml,sys; yaml.safe_load(open(sys.argv[1]))' .github/dependabot.yml`), and `git diff` shows only the added comment and ignore entry.
- [x] 1.3 `task lint` and `task openspec:check` green (no Go code changes, so the unit tests are not affected), then commit `ci(dependabot): leave cuelang.org/go to library releases`.
- [ ] 1.4 Run the opsx:verify skill for ignore-cue-in-dependabot and report the result to the supervisor.

## 2. Archive (rides this PR)

The archive rides the implementing PR, never a push to main. This section runs only after the supervisor's review of section 1, and never in the planning or implementation run.

- [ ] 2.1 `openspec archive ignore-cue-in-dependabot --yes`. This adds "Dependabot leaves cuelang.org/go to library releases" to `openspec/specs/repo-automation/spec.md` and applies the MODIFIED "Dependabot leaves OPM Go modules to the release cascade". In the same commit, update the spec's Purpose line so Dependabot leaves OPM-owned Go modules to the release cascade and `cuelang.org/go` to library releases. Verify: `task openspec:check` is green.
- [ ] 2.2 `task openspec:check` green, then commit `chore(openspec): archive ignore-cue-in-dependabot`. The commit touches only `openspec/`.
