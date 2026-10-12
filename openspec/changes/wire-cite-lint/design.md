## Context

The reusable workflow builds the tool from a checkout of `open-platform-model/enhancements` at its default `enh-ref`. A caller passes only `paths` and `exclude`. The wiring check holds the call to a plain shape: a job with only `name`, `permissions` (exactly `contents: read`), `uses` and `with`, no secrets, and a calling file that does not trigger on `pull_request_target` or `workflow_run`.

## Decisions

- **A new small workflow file, not a job in `pr.yml` or `ci.yml`.** The wiring check allows one file per reference kind. `pr.yml` runs on `pull_request` only and `ci.yml` on every push; the call is wanted on both pull requests and pushes to `main`. A new file with exactly those two triggers holds it. Alternative: a call in each of `pr.yml` and `ci.yml`; rejected, since the check refuses a second file of one kind and `ci.yml` would run it on every branch push.
- **The call job is not a required check.** Making it org-required is out of scope.
- **Exclude in two places, each naming the other.** The Taskfile variable and the workflow `exclude` input carry the same list. Alternative: a shared file read by both; rejected, because a reusable workflow call takes inputs from the caller file and cannot read a file of the repo before the job starts.
- **The task runs `go run ...@e0335b2e`.** Same commit as the workflow default. The task must be moved by hand when `.github` moves its default; the comment says so.
- **The OpenSpec CLI pin stays at 1.12.0.** The ruling: move to 1.14.1 only when `openspec validate --all --strict` passes there.

## Risks / Trade-offs

- A drift between the task's tool commit and the workflow default shows as a local pass and a CI fail, or the reverse. Accepted: the comment in the Taskfile names the default's place.
