## Why

The cli's cascade references pin `open-platform-model/.github` at `7b9ad1b`. Two `.github` changes have merged since: `94a9036` (.github PR 14, keep a background `git gc` from failing the resolver) and `6938f8e` (.github PR 15, verify the wiring check copy and allow declared resolver checkouts). They reach the cli only when it moves its pin (`.github` README, "Pinning and bumps"). PR 15 tightens the canonical wiring check:

- with `--pin-on-main`, after the compare check, it fetches `.github/scripts/cascade/wiring-check.sh` at the pinned SHA and refuses a running copy whose bytes differ (`differs from`);
- the CI workflow's and job's `env` may name only `CUE_*`, `OPM_*`, `REGISTRY` or `IMAGE_NAME`, the CI job has no `container` or `services`, and every step before the wiring step is an action from another repo at a full SHA with only `id`, `name`, `uses` and `with`; `--pin-on-main` also exits 1 when `BASH_ENV` or `ENV` is set;
- every checkout of any `.github` repository is `actions/checkout@<full SHA>` with exactly `repository`, `ref`, `path` and `persist-credentials: false`, no step's `repository` input is an expression, and no `run:` step names `.github` past the pinned checkouts.

## What Changes

- **Pin.** Every `.github` reference (`cascade-notify` in `release.yml`, `cascade-receive.yml` and `cascade-publish` in `deps-cascade.yml`, `cascade-gates.yml`, the resolver checkout `ref:` in `cascade-task.yml`) moves to `6938f8e0247e019cb0c2db13fff5b7b558a6b67d # .github main`.
- **Wiring check.** `.tasks/cascade/wiring-check.sh` becomes the byte-identical copy of `.github/scripts/cascade/wiring-check.sh` at that SHA.
- **Nothing else in the wiring.** The cli already meets the new rules: `pr.yml`'s workflow `env` names only `OPM_REGISTRY` and `CUE_REGISTRY`, the `lint` job has no `env`, `container` or `services`, the three steps before "Verify the cascade wiring" are SHA-pinned `actions/checkout`, `actions/setup-go` and `go-task/setup-task` with only `uses` and `with`, and `cascade-task.yml`'s resolver checkout already has exactly the four allowed inputs. `.tasks/cascade/wiring-check.yaml` needs no new key (`extra-references` is opm-operator's only). No caller input of `cascade-receive.yml`, `cascade-publish`, `cascade-notify` or `cascade-gates.yml` changed between the two SHAs.
- **Docs.** `AGENTS.md`'s release cascade note says `--pin-on-main` also compares the copy with the file at the pinned SHA.

Release class: none. CI, a copied script and docs; no command, flag, output or shipped file changes. PR title `ci(deps): pin the cascade to .github 6938f8e`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-cascade-wiring`: the CI wiring step also compares the copy with the canonical file, and the check now bounds the CI job's env and earlier steps and every `.github` checkout.

## Impact

- **Files:** `.github/workflows/{release,deps-cascade,cascade-gates,cascade-task,pr}.yml`, `.tasks/cascade/wiring-check.sh`, `AGENTS.md`.
- **Risks:** the diff of `.github` between the two SHAs touches neither `cascade-publish` nor `cascade-notify` (only `lib/common.sh`, `lib/release.sh` and the wiring check), so the canary rule's live-run wait does not apply. The required `Lint` job's wiring step now makes a second GitHub API call (the raw content of the script); an API failure fails it.
