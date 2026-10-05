## Why

The cli's cascade references pin `open-platform-model/.github` at `6938f8e`. Three `.github` changes have merged since: `6793c45` (PR 16, name a single code owner), `9991ed1` (PR 17, let all receivers move together while dry) and `0f9c6ac` (PR 19, sync the cli publish mirror and refuse a stale one). PR 19 changes `wiring/lib.sh`, which both `cascade-notify` and `cascade-publish` run, so under the README's "Which repos move when" rule (owner decision 37, "Both actions") one upstream moved first: the library (library PR 204). Its first live publish (library PR 206) and its first live notify (library `v1.0.0-beta.5` release run 37294102028, "Notify downstream" success) on `0f9c6ac` have succeeded, so the cli may move now.

PR 19 adds `mirror_sources` to `wiring/lib.sh`: publish refuses a receiver whose mirrored `.tasks/cascade/` files on `origin/main` no longer hash to the recorded values. The cli's files on `origin/main` (`bd4d1a7c`) match the recorded sha256 for all three (`pins.sh` `3c3f50ed…e3e1`, `classes` `4a0a74ff…e528`, `cascade.sh` `55223391…0387`; the cli has no `.tasks/cascade/lib.sh`), so the new drift refusal passes for the cli.

## What Changes

- **Pin.** Every `.github` reference (`cascade-notify` in `release.yml`, `cascade-receive.yml` and `cascade-publish` in `deps-cascade.yml`, `cascade-gates.yml`, the resolver checkout `ref:` in `cascade-task.yml`) moves to `0f9c6ac2c9b752a79f4874f637ef9955bcf00c13 # .github main`.
- **Wiring check.** `.tasks/cascade/wiring-check.sh` is unchanged between the two SHAs, so the copy stays byte-identical to the file at `0f9c6ac` (proved with `cmp` against the local object and the `gh api` raw content).
- **Nothing else.** The README at `0f9c6ac` changes no caller input and leaves the cli's row in the wiring-check table as it was, so `.tasks/cascade/wiring-check.yaml` and `pr.yml` are unchanged.
- **Spec.** The pinning requirement's source cites the README at `0f9c6ac`.

Release class: none. CI only; no command, flag, output or shipped file changes. PR title `ci(deps): pin the cascade to .github 0f9c6ac`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-cascade-wiring`: the pinning requirement's source moves to the README at `0f9c6ac`; the rule itself is unchanged.

## Impact

- **Files:** `.github/workflows/{release,deps-cascade,cascade-gates,cascade-task}.yml`, the main spec.
- **Risks:** a later cli change to `.tasks/cascade/pins.sh`, `classes` or `cascade.sh` makes publish refuse the cli's plans until `.github` updates its mirror and `mirror_sources` and the cli moves its pin again (the intended fail-closed state).
