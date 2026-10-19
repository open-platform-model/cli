## Why

The cli's cascade references pin `open-platform-model/.github` at `0f9c6ac`. That commit does not accept the repository name `opm-controller`. `.github` PR 20 (`6df4000`, "accept opm-controller beside opm-operator") makes every map in `wiring/lib.sh` accept both names with the same values. The cli must pin it before the `opm-operator` repository is renamed, because the name the cascade reads comes from `GITHUB_REPOSITORY` and flips at the rename whatever commit the caller pins.

PR 20 changes `wiring/lib.sh`, which both `cascade-notify` and `cascade-publish` run. The owner waived the canary rule of the workspace `RELEASING.md`, "Moving the cascade pin", for this one change, so the library, opm-operator and the cli may move in one round; the owner merges the library pull request first.

PR 20 keeps `mirror_sources` for the cli as it was. The cli's files on `origin/main` (`0d0c2e80`) match the recorded sha256 for all three (`pins.sh` `3c3f50ed...e3e1`, `classes` `4a0a74ff...e528`, `cascade.sh` `55223391...0387`; the cli has no `.tasks/cascade/lib.sh`), so the drift refusal passes for the cli.

## What Changes

- **Pin.** Every `.github` reference (`cascade-notify` in `release.yml`, `cascade-receive.yml` and `cascade-publish` in `deps-cascade.yml`, `cascade-gates.yml`, the resolver checkout `ref:` in `cascade-task.yml`) moves to `6df4000f6cabbf460a5cad6431b6bc9044fa3027 # .github main`.
- **Wiring check.** `.tasks/cascade/wiring-check.sh` is unchanged between the two SHAs, so the copy stays byte-identical to the file at `6df4000` (proved with `cmp` against the local `.github` object).
- **Nothing else.** PR 20 changes no caller input and leaves the cli's row in the wiring-check table as it was, so `.tasks/cascade/wiring-check.yaml` and `pr.yml` are unchanged. The cli's own pin file path (`internal/operator/pin.go` or `internal/controller/pin.go`) is not touched here.
- **Spec.** The pinning requirement's source cites the README at `6df4000`.

Release class: none. CI only; no command, flag, output or shipped file changes. PR title `ci(deps): pin the cascade to .github 6df4000`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-cascade-wiring`: the pinning requirement's source moves to the README at `6df4000`; the rule itself is unchanged.

## Impact

- **Files:** `.github/workflows/{release,deps-cascade,cascade-gates,cascade-task}.yml`, the main spec.
- **Risks:** a later cli change to `.tasks/cascade/pins.sh`, `classes` or `cascade.sh` makes publish refuse the cli's plans until `.github` updates its mirror and `mirror_sources` and the cli moves its pin again (the intended fail-closed state).
