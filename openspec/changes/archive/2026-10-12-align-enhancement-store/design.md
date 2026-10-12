## Context

`design/T2.3.md` sections 6 and 9 of the refactoring-enhancements swarm set the rule: an
enhancement id appears only beside a reason in words, never alone, never as a decision number.
`enh cite-rewrite` does the mechanical part and writes a worklist for the rest.

## Decisions

- **Tool first, hand second.** Run `enh cite-rewrite --write --worklist` once, commit by place
  (code, live specs, archived changes, docs), then clear each worklist row by hand. Reason: a
  reviewer can check the tool's output by sample, and the hand edits stay small.
- **Fixtures stay.** `tests/fixtures/` is published to GHCR and `hack/fixtures.sh check` demands
  a new version for any edit. The tool and the lint skip it with `--exclude`.
- **Live specs by wording only.** The edits keep every requirement and scenario name, so
  `openspec validate --all --strict` still passes and no delta is needed. One exception: the
  `kernel-render` requirement `Local-replacement renders warn` lost its id suffix, because a
  citation in a requirement heading becomes part of the requirement name. The archived delta
  specs that quote the name follow. A live requirement that a quoted title would push past 500
  characters names the enhancement with a short reason instead.
- **No lint task and no CI call here.** The pinned `enh` commits do not exist on a merged
  branch yet. A later task adds both.
- **OpenSpec pin.** The 1.12.0 pin stays unless `openspec validate --all --strict` under 1.14.1
  passes on this branch. The result goes in the report.

## Risks

- A wrong tool edit in a comment changes its meaning. Mitigation: spot-read at least 40 edits,
  and the review samples at least 60 changed lines.
- A code line changes by mistake. Mitigation: the review filters `git diff -U0` to code lines.
- Test messages: the rewrite keeps the meaning; no test asserts on the old text (checked with
  `task test:unit`). `task check` and `task test` also run the integration and e2e suites against
  the shared kind cluster, so this change does not use them.
