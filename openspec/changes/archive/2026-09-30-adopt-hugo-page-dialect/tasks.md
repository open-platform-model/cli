# Tasks: adopt-hugo-page-dialect

This change has two sections (design.md, "Sections and spike"). Neither is a spike: the recipe was dry-run on a copy of `origin/main` on 2026-09-30, and it passed.

Placeholders. Write every path literally, never as a variable:
- `<wt>` is `/var/home/emil/dev/open-platform-model/cli/.claude/worktrees/adopt-hugo-page-dialect`, the worktree from `orchestration.md` section 7, step 2 (branch `docs/adopt-hugo-page-dialect`).
- `<scratch>` is your scratch directory, outside any repo.

Gates for every section, from `orchestration.md` section 10 (the cli row):
- the dialect lint;
- the page-order diff;
- `task -d <wt> openspec:check`;
- `task -d <wt> vet`;
- `task -d <wt> test:unit`.

These replace the `task lint` and `task test` gates that `openspec instructions apply` serves from `openspec/config.yaml` (`orchestration.md` section 10). No Go file changes, so `task lint` is not needed. Never run `task check` or `task test`: they run the integration and e2e suites against the kind cluster.

## 1. Adopt the page dialect in docs/site

- [x] 1.1 Set up the tools and the baseline.
  - Write the lint and the page-order helper from `orchestration.md` 4.1, byte for byte, to `<scratch>/opm-dialect-lint.sh` and `<scratch>/opm-page-order.sh`.
  - Save the tree before the change: `mkdir -p <scratch>/before`, then `git -C <wt> archive origin/main docs/site | tar -x -C <scratch>/before`, then `sh <scratch>/opm-page-order.sh <scratch>/before/docs/site > <scratch>/order-before.txt`.
  - Run `sh <scratch>/opm-dialect-lint.sh <wt>/docs/site`.
  - Verify:
    - `sha256sum` prints `dae9717af0c43fc3bdc29a9e730fd171fe7541dab35a9625a35efb1682973c6b` for the lint and `edc62591eb019efc23b5c106d3b91e76620bb6e6a7ac47b3bde084c7857b5f65` for the helper.
    - The lint prints exactly five findings, each `<page>:5: sidebar: is Starlight front matter; write weight: N`, one for each page in 1.2, then `opm-dialect-lint: 5 violation(s)`.
  - If it prints anything else:
    - A `sidebar:` finding on another page gets the 1.2 edit: the `sidebar:` line the finding names and the `  order: N` line after it become `weight: N`. Name that page in the report. The checks in 1.2, 1.4, 1.5 and 2.2 then count it with the five.
    - Any other kind of finding: stop, and report it under `deviations`.
- [x] 1.2 Change the order key to `weight`. In each page below, replace the two front-matter lines `sidebar:` and `  order: N` (lines 5 and 6) with the one line `weight: N`. Keep N the same (design.md, "Weight values"). Leave `title`, `description`, `type` and the rest of the file byte-identical.

  | Page under `<wt>/docs/site/` | New line |
  |---|---|
  | `authoring/publish-a-module.md` | `weight: 24` |
  | `diagnostics/publish-refusals.md` | `weight: 26` |
  | `diagnostics/unprovided-contracts.md` | `weight: 27` |
  | `extending/publish-a-catalog.md` | `weight: 22` |
  | `reference/registry-namespaces.md` | `weight: 6` |

  Verify:
  - `grep -rn -e '^sidebar:' -e '^  order:' <wt>/docs/site` prints nothing.
  - `grep -rn '^weight:' <wt>/docs/site` prints the five lines above, plus one line for each page named under 1.1, and nothing else.
- [x] 1.3 Fix the stale path. In `<wt>/docs/site/authoring/publish-a-module.md`, find the planning comment under "Related". It is line 66 before 1.2 and line 65 after. Replace `opmodel.dev/site/content/docs/reference/cli/index.md` with `opmodel.dev/site/content/docs/reference/cli/_index.md`, and change nothing else on the line. Verify: `grep -rn 'index\.md' <wt>/docs/site | grep -v '_index\.md'` prints nothing.
- [x] 1.4 Run the dialect and order gates. Verify:
  - `sh <scratch>/opm-dialect-lint.sh <wt>/docs/site` prints `opm-dialect-lint: OK (...)`.
  - After `sh <scratch>/opm-page-order.sh <wt>/docs/site > <scratch>/order-after.txt`, the command `diff <scratch>/order-before.txt <scratch>/order-after.txt && echo "order unchanged"` prints `order unchanged`.
  - `git -C <wt> diff --stat` lists only the five pages, any page named under 1.1, and this `tasks.md`.
  - `git -C <wt> diff --check` prints nothing.
- [x] 1.5 Run `task -d <wt> openspec:check`, `task -d <wt> vet` and `task -d <wt> test:unit`, and get them green. Then stage the five pages, any page named under 1.1, and `openspec/changes/adopt-hugo-page-dialect/tasks.md` by explicit path, and commit `docs(site): adopt the hugo page dialect`.

## 2. Say the site is built with Hugo and Hextra in RFC 0006

- [x] 2.1 Replace line 9 of `<wt>/docs/rfc/0006-documentation-generation.md`, the line that starts `> **Superseded in part (2026-09-24):**`. The new line is the exact text in design.md, "The RFC banner". Keep the blank lines 8 and 10. Verify:
  - `sed -n 9p <wt>/docs/rfc/0006-documentation-generation.md` prints the new line.
  - `grep -n -i -e astro -e starlight <wt>/docs/rfc/0006-documentation-generation.md` prints nothing.
  - `git -C <wt> diff --stat -- docs/rfc` shows one file, with 1 insertion and 1 deletion.
- [x] 2.2 Run the gates. Verify:
  - The lint still prints `opm-dialect-lint: OK (...)`.
  - The order diff, run as in 1.4, still prints `order unchanged`.
  - `git -C <wt> diff --check` prints nothing.
  - `git -C <wt> diff --stat origin/main` lists only the five pages, any page named under 1.1, `docs/rfc/0006-documentation-generation.md` and files under `openspec/changes/adopt-hugo-page-dialect/`.
  - `task -d <wt> openspec:check`, `task -d <wt> vet` and `task -d <wt> test:unit` are green.

  Then stage `docs/rfc/0006-documentation-generation.md` and `openspec/changes/adopt-hugo-page-dialect/tasks.md` by explicit path, and commit `docs(rfc): say the site is built with hugo and hextra`.

## After section 2: hand-off (`orchestration.md` section 7; not tasks)

These lines are not checkboxes. cli's `openspec/config.yaml` allows no delivery step in this file other than the section commits, so the rest of the worker protocol is `orchestration.md` section 7, steps 6-8. The lines below add only what this change needs on top of it.

- **Step 6: verify, report, then stop and wait.** Read `<wt>/.claude/skills/openspec-verify-change/SKILL.md` and follow it for `adopt-hugo-page-dialect`. Run each `openspec` command as `cd <wt> && openspec ...`. Never use the root `/opsx:verify` router. In the report block:
  - `change`: `S4 cli adopt-hugo-page-dialect`.
  - `sections`: `2/2`.
  - `gates`: the lint, the order diff, `task openspec:check`, `task vet` and `task test:unit`, one line each.
  - `surface`: the names in design.md, "Interface".
- **Step 7: on the supervisor's go.** design.md, "Migration Plan", names the PR title. After any merge of `origin/main` into the branch, and before the push that follows it:
  - Extract the fresh `origin/main` `docs/site` into a new, empty directory under `<scratch>`, and write a new order baseline from it, as 1.1 does.
  - Run the lint. A cli page with a `sidebar:` block that the merge brought in gets the 1.2 edit. Any other finding: stop and report it under `deviations`.
  - Get the lint, the order diff (a new `order-after` against the new baseline), `task -d <wt> openspec:check`, `task -d <wt> vet` and `task -d <wt> test:unit` green.
  - For each converted page, stage only that page and commit `docs(site): adopt the hugo page dialect for <page>`, where `<page>` is the file name without `.md`. Leave the archived `tasks.md` as it is, and tell the supervisor the page and the commit.
