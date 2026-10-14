# Cascade wiring: secrets, permissions and the SHA pin

Covers how the cascade and release workflows hold the App keys, which permissions they declare, the
five `.github` SHA pins, and the wiring check that enforces them. The task itself is in
the receiver workflow and the deps task. The `.github` README owns the check's design.

## Contents

- Secrets
- The SHA pin
- The wiring check
- Permissions and caches
- Pitfalls
- Check before you finish

## Secrets

- The cascade App key (`CASCADE_APP_PRIVATE_KEY`) is read only in the caller-owned
  `notify-downstream` and `publish` jobs. They declare `environment: cascade` (deploys from `main`
  only) and pass the secret only as the `private-key` input of the SHA-pinned cascade action.
- Those jobs have no checkout or `run:` of their own, and no `env:`, `container:` or `services:`. The
  publish action checks the repo out but never runs it.
- No reusable call passes `secrets:` or `secrets: inherit`.
- The release App key (`RELEASE_APP_PRIVATE_KEY`) is read only by `release.yml`'s `release-please`
  job. It declares `environment: release` (deploys from `main` only) and grants the workflow token
  nothing.

## The SHA pin

Every cascade reference names one `.github` `main` commit SHA, with the comment `# .github main`.
The references are the two actions, the two reusable workflows, and the `ref:` of
`cascade-task.yml`'s resolver checkout.

A `.github` change reaches the cli only through a PR titled `ci(deps): pin the cascade to .github
<sha7>` that moves all five together. Dependabot ignores them.

## The wiring check

The wiring-check script in `.tasks/cascade/` is a byte-identical copy of the one in `.github`
(`.github/scripts/cascade/`) at the pinned SHA. Only the pin PR replaces it. Never edit
the copy; change it in `.github`. The cli's values are in a config file beside it.

It refuses a PR that breaks any of this by mistake:

- a job other than `release-please` that reads or declares the release key's Environment;
- an Actions cache in `release.yml`, `publish-fixtures.yml` or `docs.yml`;
- workflow-level `env` in `release.yml`.

`pr.yml`'s required `Lint` job runs the wiring-check script with `--pin-on-main` with the
job's token. That run also asks the GitHub API two things: that the SHA is on `.github`'s `main`, and
that the copy has exactly the bytes of the file at that SHA. So only SHA-pinned actions may come
before that step, and the workflow and job `env` may name only `CUE_*`, `OPM_*`, `REGISTRY` or
`IMAGE_NAME`.

`task cascade:wiring:check` (also in `task check`) runs it offline.

## Permissions and caches

- Every workflow declares its permissions: `{}` or `contents: read` at workflow level, writes per
  job. The repository default token can then stay read-only.
- A job that publishes (`goreleaser`, `publish-templates`, `publish-fixtures.yml`) sets `cache: false`
  on `setup-go`, because `main`'s Actions cache is written by runs of other code.
- `publish-templates` and `publish-fixtures.yml` run from `main` only, also when dispatched. The
  release jobs check out `refs/tags/<tag>`.
- `.github/CODEOWNERS` covers the workflows, `.tasks/`, `Taskfile.yml`, the release-please files,
  `.cascade-frozen`, `hack/` and `.goreleaser.yml`.

## Pitfalls

- Editing the wiring-check copy in the cli: the byte comparison fails. Change `.github` and send a pin PR.
- Moving only some of the five SHA references: the check fails.
- Adding a workflow-level `env` to `release.yml`: refused.
- Adding an Actions cache to a publishing workflow: refused.
- Passing `secrets: inherit` to a reusable call: refused.
- Putting a non-SHA-pinned action before the wiring-check step in `Lint`.

## Check before you finish

- [ ] `task cascade:wiring:check` passes.
- [ ] A new workflow declares its permissions.
- [ ] A publishing job sets `cache: false` on `setup-go`.
- [ ] No secret reaches a job outside the two named jobs and `release-please`.
