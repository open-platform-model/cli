## Context

The pin moves by the procedure of the workspace `RELEASING.md`, "Moving the cascade pin". Five references carry one SHA; `task cascade:wiring:check` asserts that. The check script is a byte-identical copy of the file at the pinned SHA, and it is the same bytes at `0f9c6ac` and `6df4000`.

## Goals / Non-Goals

**Goals:** the five references name `6df4000f6cabbf460a5cad6431b6bc9044fa3027`; the main spec cites the README at the new pin.

**Non-Goals:** no change to the wiring-check copy, `wiring-check.yaml`, `pr.yml`, release configuration or the cli's pin file.

## Decisions

- **No new copy of `wiring-check.sh`.** `cmp` against the `.github` object at `6df4000` shows no difference, so the copy stays. Alternative: copy the file again; rejected, because it changes nothing and hides that the file did not move.
- **One pull request, as the previous repin.** Alternative: fold the cli into one pull request with the library and opm-operator; rejected, because the owner merges library first, then the two receivers, each on its own checks.

## Risks / Trade-offs

- A later change to a mirrored `.tasks/cascade/` file makes publish refuse the cli until `.github` records the new hash and the cli pins again → the intended fail-closed state.
