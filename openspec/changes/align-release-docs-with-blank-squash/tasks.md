# Tasks: align-release-docs-with-blank-squash

One PR, titled `docs(release): describe breaking changes and forced versions under the BLANK squash message`. `docs` is hidden in `release-please-config.json:23`, so merging cuts no release. Workers never push to `main`, tag, edit `release-please-config.json`, or touch repo settings. Items marked **SUPERVISOR** are not worker tasks.

This change edits text only. The local test gate is `task test:unit` plus `go test ./tests/e2e/... -timeout 25m` run with `env -u OPM_CONFIG -u KUBECONFIG` and `HOME` pointed at an empty scratch directory (with `GOMODCACHE`, `GOCACHE`, `GOPATH` and `CUE_CACHE_DIR` exported from the real environment), so the cluster tests skip; `task test:integration` needs the `kind-opm-dev` cluster and is reported as skipped. No Go file changes, so the cluster half adds no evidence.

## Gates (SUPERVISOR ticks)

- [ ] G-workspace (before merge) workspace `main` RELEASING.md still states, in "Owner settings" › "Merge settings", that a breaking change is `!` in the PR title and a forced version is a `release-as` key set by a normal PR and removed by the next. Check: `git -C <workspace> show origin/main:RELEASING.md | sed -n '/^### Merge settings/,/^### Rulesets/p' | grep -n -E 'release-as|BLANK'`.

## 1. Contributor documents

- [ ] 1.1 `AGENTS.md:144-147` ("Release line: beta"): replace "a line change travels as a one-shot `Release-As` footer in the carrier's squash commit message, never as a `release-as` key in `release-please-config.json`" with: a line change is a `release-as` key in `release-please-config.json`, landed by a normal PR and removed by the next PR once that release is cut (release-please re-applies it on every run while it stays); a `Release-As:` footer never reaches `main` under the `BLANK` squash message (workspace RELEASING.md, section "Owner settings").
- [ ] 1.2 `AGENTS.md:357` ("Beta promise"): replace "only as a `feat!` commit whose `BREAKING CHANGE:` footer is the migration note the CHANGELOG shows" with: only as a `!` in the PR title (`feat!:`), which is the squash commit and the CHANGELOG entry; the migration note goes in the PR body, and in the user docs when users need it to upgrade (design.md D5). Leave the rest of the bullet, the GA sentence included, unchanged.
- [ ] 1.3 `AGENTS.md:358` ("Beta skew rule"): replace "never a `Release-As` to `1.1.0-beta.1` or beyond" with "never a `release-as` of `1.1.0-beta.1` or beyond".
- [ ] 1.4 `CONSTITUTION.md:104-113` (Pre-GA note) and `openspec/config.yaml:51-60` (the same note inside `context`, indented two spaces): replace the sentence "A breaking change is still allowed during beta, but only as a `feat!` commit whose `BREAKING CHANGE:` footer is the migration note the CHANGELOG shows." with the 1.2 wording, re-wrapped to the existing line width. Verify the two copies stay identical: `diff <(sed -n '/^Pre-GA note/,/^order\./p' CONSTITUTION.md) <(sed -n '/^  Pre-GA note/,/^  order\./p' openspec/config.yaml | sed 's/^  //')` prints nothing.
- [ ] 1.5 `openspec/config.yaml:225` (apply guidance): keep the rule "no body line starting with word(" and replace its reason "because the squash body reaches release-please" with "because a squash message other than BLANK copies commit bodies into main, where release-please parses such a line as a commit" (design.md D3). Keep the YAML string valid: `openspec instructions apply --change align-release-docs-with-blank-squash --json >/dev/null` exits 0.
- [ ] 1.6 Confirm nothing stale is left: `grep -rn -E "Release-As|BREAKING CHANGE:" AGENTS.md CONSTITUTION.md openspec/config.yaml` prints only the new "never reaches `main`" sentence from 1.1 (and nothing that prescribes a footer); `grep -n "squash body reaches" openspec/config.yaml` prints nothing.
- [ ] 1.7 `openspec validate align-release-docs-with-blank-squash --strict` passes, and `openspec verify` (or the `opsx:verify` skill) finds every task and the `release-workflow` delta satisfied.
- [ ] 1.8 `task fmt`, `task lint`, `task test` (see the note above) and `task openspec:check` green, then commit `docs(release): describe breaking changes and forced versions under the BLANK squash message`.

