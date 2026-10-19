## Context

The cli has 215 long requirements in 59 main specs. The rules below follow the swarm guide for this rewrite (design/T2.1.md of the swarm folder, approved 2026-10-19). The numbers in brackets point to that guide's sections. Nothing here changes the meaning of a requirement or scenario, and no code changes.

## Research & Decisions

### Which form rewrites a requirement
**Context**: A requirement over 500 characters can be shortened in four safe ways, and several unsafe ones pass the validator without keeping the rules (guide section 3).
**Explored**: Archive output of each form in scratch copies (guide sections 3 and 5).
**Options considered**:
1. Form a, MODIFIED with a shorter body and more scenarios: keeps the header and every scenario name. Fits examples, lists and formats around one behavior.
2. Form b, MODIFIED plus ADDED: the parent keeps its header and all scenarios, split-off behaviors become ADDED requirements with new scenarios. A live scenario never moves to an ADDED block (OpenSpec refuses it).
3. Form c2, REMOVED plus ADDED under new names: the only form that may rename or re-home scenarios. OpenSpec does not check scenarios here, so the ledger is mandatory.
4. Form d, RENAMED plus MODIFIED under the new name: renames only.
**Decision**: Read the text of each requirement and choose by this table, in this order.

| The excess is | Form |
| --- | --- |
| Examples, lists, formats or reasons around one behavior | a |
| Several behaviors, and every old scenario fits the parent that stays | b |
| Several behaviors, and old scenarios belong to different behaviors | c2 |
| Only the name must change | d, alone |

**Rationale**: Most cli requirements state several rules, so b and c2 are the common remedy and a is second. The kind labels of the inventory are a heuristic and do not decide. A MODIFIED body that stays over 500 passes `validate <change> --strict` but fails `validate --specs --strict` after archive, so every unit measures after archive. Unsafe pairs on one name (MODIFIED with ADDED, MODIFIED with REMOVED, ADDED with REMOVED, RENAMED with REMOVED or ADDED) are never written.

### What counts as shortening
**Context**: The validator counts the body lines between the header and the first `#` line, minus blank lines, fenced lines and `**Key**:` lines. Three loopholes pass the check without removing a rule.
**Decision**: The body itself must get shorter by moving text to scenarios or other requirements. No `**Key**:` line (only `**Reason**:` and `**Migration**:` inside REMOVED), no heading of level 5 or deeper, no fenced block, in a changed body.
**Rationale**: A loophole keeps the long text and defeats the point of the check.

### One change, eight units
**Context**: The repo rules ask for about five sections and a block above that (`openspec/config.yaml` proposal and tasks rules, Principle VIII).
**Options considered**:
1. One change in ten sections (eight batches, the gate extras, the final gate): one archive, one history, one ledger. Exceeds the "about five".
2. Several changes: breaks the single-ledger view, and each change must archive in turn on the same specs.
**Decision**: Option 1. The owner chose one change per repo and several units in sequence on one branch (swarm DECISIONS.md 2026-10-19 09:50).
**Rationale**: Each section still satisfies both invariants of Principle VIII: it ends green under `task openspec:check` (non-strict until the gate section) and closes with its own commit. The sections are uniform batches, not ten different designs. The proposal gate confirms this deviation from the "about five" wording.

## Sections

| Section | Content | Long requirements | Cumulative |
| --- | --- | --- | --- |
| 1 to 8 | cli-B1 to cli-B8 | 28, 28, 28, 28, 28, 28, 28, 19 | 28, 56, 84, 112, 140, 168, 196, 215 |
| 9 | Restore `--strict` in `Taskfile.yml`, `AGENTS.md`, `openspec/config.yaml` | none | 215 |
| 10 | Final gate | none | 215 |

Order of the batches: B1 to B8 in the inventory order, which is by size, largest first. No order is forced by a collision: the cli has no open change (only `archive/`), so no `MODIFIED` of another author can go stale. B1 holds the two largest specs (`operator-lifecycle` 17, `apply-pruning` 11), so the form choices are settled there on the most requirements, and B8, the smallest, runs last. Section 9 runs after every batch, because `--strict` fails until all 215 are rewritten.

Each batch writes its delta files under `specs/<spec>/spec.md` of this change, one per spec, and no delta for a spec outside the batch. A delta file holds only the requirements the batch rewrites.

## The scenario ledger

Each unit appends one subsection per rewritten requirement to this file, under a heading `### Ledger: <spec>, <requirement header>`. A unit that rewrites nothing adds none.

Scenario table, one row per old scenario:

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |

- Forms a, b, d: every old name stays under the same requirement, byte for byte.
- Form c2: every old name must reappear in one of the new requirements, because OpenSpec does not check it. Moved scenarios keep WHEN and THEN byte for byte.

Clause table, when a clause leaves the body: old clause, new place (body clause or scenario THEN line). A clause that moves from a body SHALL to a scenario THEN keeps its SHALL, and only a default or an exact value moves this way. A clause that adds or changes a condition stays in the body. A separate behavior goes to form b or c2.

Other ledger rules for this repo:

- RFC 2119 keywords stay in every body, with the same strength.
- Citations (an enhancement id beside a reason in words, a `Source:` line) stay in the body of the block that carries the rule they support, never in the header. The repo rule is that an enhancement id never stands alone and is never a decision number; a rewrite copies the old wording and adds no new citation. A `Source:` line counts toward 500; `**Source**:` does not, and is a loophole.
- Before a rename, split or c2, grep the repo for the old header text outside `openspec/changes/archive/`. A stale reference is fixed in the same unit.
- Every new ADDED requirement has at least one scenario with a body and states SHALL or MUST. `openspec/config.yaml` asks scenarios to show observable behavior and user-facing outcomes; a scenario added for a new requirement uses only words and values of the old text. A scenario that adds a case absent from the old text is a question for the human, not a rewrite.
- No spec has a placeholder Purpose today (checked on `origin/main`), so no unit writes Purpose text. A c2 or b rewrite does not create a spec, so none appears.

## The gate

Every unit runs the gate in a scratch copy of the folder `openspec/`, never in the worktree. The worktree path is `<repo>/agent-swarm/openspec-requirement-split/worktrees/cli-split-long-requirements`. The pinned CLI is 1.14.1 (`Taskfile.yml` `openspec:install`). The gate block is guide section 6, run in one shell call: archive `main` and the working copy into `$S/base` and `$S/openspec`; count WARNING lines of `validate --all --strict` before; `validate split-long-requirements --strict`; `archive split-long-requirements --yes`; `validate --all --strict` after; fail on any "Archive would refuse" line, on any ERROR item that `main` does not have (compared as a set of type, id and path, needs `jq`), and, for an earlier unit, when the WARNING count did not fall by N. N is the total of requirements rewritten so far by all units, 28 after section 1 and so on (the table above). Exit 0 only after `GATE PASS`.

- Each unit also runs `openspec validate <spec> --type spec --strict --json 2>&1` for its specs and reads that no rewritten requirement has a WARNING or an ERROR.
- Each unit checks every moved or kept scenario byte for byte against `main` with the guide's checklist item 3 script, run in the worktree root once per rewritten requirement. A `DIFF` or `LOST` line is a failure.
- Section 10 is the last unit (`LAST=1`): `validate split-long-requirements --strict` exits 0, `validate --specs --strict` exits 0 after archive in the scratch copy, no "Archive would refuse" line, no new ERROR item. Then, with the extras of section 9 in place, `task openspec:check` runs in the worktree and must exit 0. In the cli no other change is open, so `validate --all --strict` also exits 0 at that point.
- This is the same check that CI runs after section 9 (`validate --all --strict` through `task openspec:check`, jobs "OpenSpec Specs (strict)").

## Risks and trade-offs

- A rewrite can drop a clause without any tool error (form c2 and moved clauses). The mitigations are the ledger, the byte check and the review checklist of the guide, run per unit by the reviewer.
- A long MODIFIED passes `validate <change> --strict` and fails only after archive. The gate catches it; a pre-archive check alone does not.
- A stale reference to a removed header is invisible to OpenSpec. The grep rule above is the only guard.
- Another change that lands on a spec of a later batch before that batch runs would make its MODIFIED blocks stale. A unit re-reads the spec from `main` of its worktree first; the supervisor merges `main` into the branch before a unit if `main` moved.

## Open questions

- Confirm at the proposal gate that ten sections in one change are accepted, as the owner ruled on 2026-10-19 09:50, although the repo wording says "about five".

## Ledger entries of cli-B1 (operator-lifecycle, apply-pruning)

One subsection per rewritten requirement. A clause row says "verbatim" where the sentence of the old body appears unchanged in the new body; other rows say what changed. Old scenario names are all kept; the byte check ran once per rewritten requirement and printed no `DIFF` and no `LOST`.

### Ledger: operator-lifecycle, Opt-in RBAC emission via `--rbac`

Form: c2 (REMOVED, ADDED under new names). New requirements: "Install with `--rbac` creates the `opm-cli-user` role"; "The `--user` and `--group` flags require `--rbac`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| RBAC off by default | "Install with `--rbac` creates the `opm-cli-user` role", "RBAC off by default" | No (byte equal, checked) |
| Role plus binding for a user | "Install with `--rbac` creates the `opm-cli-user` role", "Role plus binding for a user" | No (byte equal, checked) |
| Subject flags require --rbac | "The `--user` and `--group` flags require `--rbac`", "Subject flags require --rbac" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm operator install --rbac` SHALL additionally create a ClusterRole ... | body of "Install with `--rbac` creates the `opm-cli-user` role", verbatim |
| When `--user <U>` or `--group <G>` is supplied alongside `--rbac`, the... | body of "Install with `--rbac` creates the `opm-cli-user` role", verbatim |
| Without `--rbac`, no RBAC objects beyond those of the operator module'... | body of "Install with `--rbac` creates the `opm-cli-user` role", verbatim |
| `--user`/`--group` without `--rbac` SHALL be rejected as a flag-valida... | body of "The `--user` and `--group` flags require `--rbac`", verbatim |

### Ledger: operator-lifecycle, Uninstall refuses while operator cleanup finalizers are armed

Form: c2 (REMOVED, ADDED under new names). New requirements: "Uninstall refuses and names operator instances that carry the cleanup finalizer"; "Uninstall removes only the cleanup finalizer on request".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Refusal names the armed instances | "Uninstall refuses and names operator instances that carry the cleanup finalizer", "Refusal names the armed instances" | No (byte equal, checked) |
| Override strips only the operator's finalizer | "Uninstall removes only the cleanup finalizer on request", "Override strips only the operator's finalizer" | No (byte equal, checked) |
| List failure fails closed | "Uninstall refuses and names operator instances that carry the cleanup finalizer", "List failure fails closed" | No (byte equal, checked) |
| Partial finalizer-removal failure blocks the delete step | "Uninstall removes only the cleanup finalizer on request", "Partial finalizer-removal failure blocks the delete step" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| Before deleting anything, `opm operator uninstall` SHALL list `ModuleI... | body of "Uninstall refuses and names operator instances that carry the cleanup finalizer", verbatim |
| If the cluster-wide list fails (including RBAC denial), the command SH... | body of "Uninstall refuses and names operator instances that carry the cleanup finalizer", verbatim |
| With `--remove-finalizers`, the command SHALL attempt to remove exactl... | reworded, nearest body: "Uninstall removes only the cleanup finalizer on request"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) With `--remove-finalizers`, attempt to remove exactly the cleanup finalizer from every such instance, state the orphaning, proceed only if every armed instance was stripped | body of "Uninstall removes only the cleanup finalizer on request"; "every such instance" reads "every instance that carries it" (same set) |

### Ledger: operator-lifecycle, Catalog resolution precedes every cluster write

Form: b (MODIFIED parent, ADDED split-offs). New requirements: "Catalog resolution failures map to the standard exit codes". The parent "Catalog resolution precedes every cluster write" stays, with a shorter body.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| No selectable release aborts before install | "Catalog resolution precedes every cluster write", "No selectable release aborts before install" | No (byte equal, checked) |
| Unreachable registry aborts before install | "Catalog resolution precedes every cluster write", "Unreachable registry aborts before install" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "Unreadable registry is a connectivity failure" in "Catalog resolution failures map to the standard exit codes"

| Old clause | New place |
| --- | --- |
| When `opm operator install` will seed a Platform, it SHALL resolve the... | body of "Catalog resolution precedes every cluster write", verbatim |
| A resolution that cannot produce a version SHALL abort the command wit... | body of "Catalog resolution precedes every cluster write", verbatim |
| Resolution failures SHALL map to the CLI's standard exit codes: a cata... | reworded, nearest body: "Catalog resolution failures map to the standard exit codes"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Resolution failures map to the standard exit codes (exit 2 with the prerelease flag named, exit 3) | body of "Catalog resolution failures map to the standard exit codes"; the subject reads "Catalog resolution failures", the rest is verbatim |

### Ledger: operator-lifecycle, Install deploys the operator module as a CLI-owned instance in two steps

Form: c2 (REMOVED, ADDED under new names). New requirements: "Install renders the operator module as the instance `opm-operator`"; "Install applies the CRDs before it applies the instance"; "The operator instance records every object of the render"; "Reinstall changes only what the module version changes".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Fresh cluster | "Install applies the CRDs before it applies the instance", "Fresh cluster" | No (byte equal, checked) |
| The module's Namespace is not refused | "The operator instance records every object of the render", "The module's Namespace is not refused" | No (byte equal, checked) |
| Unchanged reinstall | "Reinstall changes only what the module version changes", "Unchanged reinstall" | No (byte equal, checked) |
| Upgrade drops an object the new version no longer renders | "Reinstall changes only what the module version changes", "Upgrade drops an object the new version no longer renders" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "Install creates the render plus the Platform and the role" in "Install renders the operator module as the instance `opm-operator`"

| Old clause | New place |
| --- | --- |
| The instance record SHALL be written only after the CRDs are served. | body of "Install applies the CRDs before it applies the instance", verbatim |
| Every object the module renders, the CRDs and the Namespace included, ... | body of "The operator instance records every object of the render", verbatim |
| Install SHALL NOT create the Namespace outside the render. | body of "The operator instance records every object of the render", verbatim |
| Apart from the render, install SHALL create only the cluster Platform ... | body of "Install renders the operator module as the instance `opm-operator`", verbatim |
| Re-running install with an unchanged module version and unchanged valu... | body of "Reinstall changes only what the module version changes", verbatim |
| Source: enhancement 0021 ("The operator's install artifact is a regist... | body of "Install renders the operator module as the instance `opm-operator`", verbatim |
| `opm operator install` SHALL obtain the operator module `opmodel.dev/m... | reworded, nearest body: "Install applies the CRDs before it applies the instance"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Obtain the module, render it as the instance `opm-operator` | body of "Install renders the operator module as the instance `opm-operator`", words verbatim |
| (detail) Apply the rendered CRDs with SSA as `opm-cli`, wait for `Established=True`, then apply the whole render as a CLI-owned ModuleInstance through the same apply | body of "Install applies the CRDs before it applies the instance", words verbatim |

### Ledger: operator-lifecycle, Install refuses an operator instance record that is not CLI-owned

Form: b (MODIFIED parent, ADDED split-offs). New requirements: "Install does not change the owner of the operator instance record". The parent "Install refuses an operator instance record that is not CLI-owned" stays, with a shorter body.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Operator-owned record | "Install refuses an operator instance record that is not CLI-owned", "Operator-owned record" | No (byte equal, checked) |
| Record without an owner | "Install refuses an operator instance record that is not CLI-owned", "Record without an owner" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "Refusal leaves the owner as it was" in "Install does not change the owner of the operator instance record"

| Old clause | New place |
| --- | --- |
| Before any object changes, install SHALL refuse when the operator's in... | body of "Install refuses an operator instance record that is not CLI-owned", verbatim |
| The refusal SHALL exit 2, name the record, state that its `spec.owner`... | body of "Install refuses an operator instance record that is not CLI-owned", verbatim |
| Install SHALL NOT change the owner itself: the instance apply would ot... | body of "Install does not change the owner of the operator instance record", verbatim |

### Ledger: operator-lifecycle, The CLI pins a default operator module version and the operator it deploys

Form: c2 (REMOVED, ADDED under new names). New requirements: "Each CLI release pins a default operator module version"; "Only `task operator:pin` writes the pin, from the module's own statement".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default install uses the pin | "Each CLI release pins a default operator module version", "Default install uses the pin" | No (byte equal, checked) |
| Pin refresh is one reviewable diff | "Only `task operator:pin` writes the pin, from the module's own statement", "Pin refresh is one reviewable diff" | No (byte equal, checked) |
| A module without a readable operator version is refused by the pin task | "Only `task operator:pin` writes the pin, from the module's own statement", "A module without a readable operator version is refused by the pin task" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| Each CLI release SHALL name one default operator module version and SH... | body of "Each CLI release pins a default operator module version", verbatim |
| The recorded operator version SHALL be the version the module's own so... | body of "Each CLI release pins a default operator module version", verbatim |
| Install without `--version` SHALL install exactly that module version. | body of "Each CLI release pins a default operator module version", verbatim |
| The pin SHALL be produced by `task operator:pin VERSION=<v>` from the ... | body of "Only `task operator:pin` writes the pin, from the module's own statement", verbatim |
| The module is trusted as the configured registry serves it, as every o... | body of "Only `task operator:pin` writes the pin, from the module's own statement", verbatim |
| Source: enhancement 0021 ("The operator's install artifact is a regist... | body of "Each CLI release pins a default operator module version", verbatim |

### Ledger: operator-lifecycle, Install refuses a target version the CLI cannot drive or that disagrees with itself

Form: c2 (REMOVED, ADDED under new names). New requirements: "Install refuses a target operator newer than the CLI"; "Install refuses a target whose render disagrees with its stated operator version"; "Install refuses a target whose ModuleInstance CRD lacks the required fields"; "A newer running operator does not block install".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Operator newer than the CLI | "Install refuses a target operator newer than the CLI", "Operator newer than the CLI" | No (byte equal, checked) |
| Rendered image disagrees with the stated operator version | "Install refuses a target whose render disagrees with its stated operator version", "Rendered image disagrees with the stated operator version" | No (byte equal, checked) |
| Rendered CRD below the floor | "Install refuses a target whose ModuleInstance CRD lacks the required fields", "Rendered CRD below the floor" | No (byte equal, checked) |
| Newer running operator does not block repair | "A newer running operator does not block install", "Newer running operator does not block repair" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| Install SHALL NOT be refused because the operator the cluster Platform... | body of "A newer running operator does not block install", verbatim |
| Source: enhancement 0021 ("The CLI's ceiling on operator versions comp... | body of "Install refuses a target operator newer than the CLI", verbatim |
| Before any object changes, install SHALL refuse a target module versio... | reworded, nearest body: "Install refuses a target operator newer than the CLI"; the subject or a condition is restated so that the body reads alone, no rule changed |
| A target below the recorded module version or the running operator SHA... | reworded, nearest body: "A newer running operator does not block install"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Three refusals, each naming the module version and the rule it failed, exit 2 | one requirement per refusal; each body repeats "with exit 2, naming the module version and the rule it failed"; "naming upgrading the CLI as the fix" stays in the first, "naming the floor" in the third; the skip-with-warning rule stays in the first |
| (detail) A target below the recorded module version or the running operator is not refused by this requirement | body of "A newer running operator does not block install"; "this requirement" reads "the target-version rules of install" |

### Ledger: operator-lifecycle, The operator's settings are recorded instance values that survive reinstall

Form: c2 (REMOVED, ADDED under new names). New requirements: "Install layers values over the values recorded on the operator instance"; "Recorded operator values survive reinstall"; "Install refuses values it cannot use".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A recorded value survives reinstall | "Recorded operator values survive reinstall", "A recorded value survives reinstall" | No (byte equal, checked) |
| Changing one value keeps the others | "Install layers values over the values recorded on the operator instance", "Changing one value keeps the others" | No (byte equal, checked) |
| Recorded value the target rejects | "Install refuses values it cannot use", "Recorded value the target rejects" | No (byte equal, checked) |
| (moved in from another requirement of this spec) CRDs-only with a values file | "Install refuses values it cannot use", "CRDs-only with a values file" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| Install SHALL take values for the operator module from `-f/--values` f... | body of "Install layers values over the values recorded on the operator instance", verbatim |
| `--reset-values` SHALL drop the recorded values for that run. | body of "Install layers values over the values recorded on the operator instance", verbatim |
| Re-running install, for the same or another module version, SHALL keep... | body of "Recorded operator values survive reinstall", verbatim |
| The module's `debugValues` SHALL NOT be used. | body of "Install layers values over the values recorded on the operator instance", verbatim |
| When the merged values are not accepted by the target version's `#conf... | body of "Install refuses values it cannot use", verbatim |
| `-f` and `--reset-values` SHALL be refused with `--crds-only` before a... | body of "Install refuses values it cannot use", verbatim |
| Source: enhancement 0006 ("`spec.values` is the sole authoritative ren... | body of "Install layers values over the values recorded on the operator instance", verbatim |

### Ledger: operator-lifecycle, Platform seeding and the opt-in user role stay install steps

Form: c2 (REMOVED, ADDED under new names). New requirements: "Install seeds the cluster Platform after the operator rolls out"; "An existing Platform is left alone and a denied create is a warning"; "The Platform and the opt-in user role are not part of the render or the inventory"; "`--rbac` applies on the CRDs-only path as on the full one".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Platform is seeded on a full install | "Install seeds the cluster Platform after the operator rolls out", "Platform is seeded on a full install" | No (byte equal, checked) |
| Existing Platform is left untouched | "An existing Platform is left alone and a denied create is a warning", "Existing Platform is left untouched" | No (byte equal, checked) |
| Platform create denied by RBAC | "An existing Platform is left alone and a denied create is a warning", "Platform create denied by RBAC" | No (byte equal, checked) |
| Platform and role are not recorded | "The Platform and the opt-in user role are not part of the render or the inventory", "Platform and role are not recorded" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "`--rbac` on the CRDs-only path" in "`--rbac` applies on the CRDs-only path as on the full one"

| Old clause | New place |
| --- | --- |
| After the operator's rollout, and unless `--skip-platform` or `--crds-... | body of "Install seeds the cluster Platform after the operator rolls out", verbatim |
| An existing Platform SHALL be reported and left untouched, an `Already... | body of "An existing Platform is left alone and a denied create is a warning", verbatim |
| `--rbac` SHALL apply on the CRDs-only path as on the full one. | body of "`--rbac` applies on the CRDs-only path as on the full one", verbatim |
| Neither the Platform nor the `opm-cli-user` role of `--rbac` SHALL be ... | reworded, nearest body: "The Platform and the opt-in user role are not part of the render or the inventory"; the subject or a condition is restated so that the body reads alone, no rule changed |
| Source: enhancement 0006 ("`Platform` carries no owner marker; the ope... | reworded, nearest body: "Install seeds the cluster Platform after the operator rolls out"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Neither the Platform nor the `opm-cli-user` role is part of the render or the inventory; `--rbac` applies on the CRDs-only path | first half in "The Platform and the opt-in user role are not part of the render or the inventory"; `--rbac` clause in "`--rbac` applies on the CRDs-only path as on the full one" |
| (detail) Source: enhancement 0006 (three quotes) and enhancement 0021 | the "owner marker" quote with the Platform create, the "write-if-absent" quote with the existing-Platform rule, the "Status-subresource RBAC" quote with the `--rbac` clause, the 0021 quote with the render/inventory rule |

### Ledger: operator-lifecycle, CRDs-only install applies the CRDs of the same module render

Form: c2 (REMOVED, ADDED under new names). New requirements: "CRDs-only install applies exactly the rendered CRDs"; "CRDs-only install refuses what a full install refuses before it applies a CRD"; "A full install records the CRDs a CRDs-only install applied".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Solo-cluster CRD install | "CRDs-only install applies exactly the rendered CRDs", "Solo-cluster CRD install" | No (byte equal, checked) |
| Full install after CRDs-only | "A full install records the CRDs a CRDs-only install applied", "Full install after CRDs-only" | No (byte equal, checked) |
| CRDs-only with a values file | "Install refuses values it cannot use", "CRDs-only with a values file" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "CRDs-only refuses a target newer than the CLI" in "CRDs-only install refuses what a full install refuses before it applies a CRD"

| Old clause | New place |
| --- | --- |
| It SHALL write no instance record, no workload and no Platform, and SH... | body of "CRDs-only install applies exactly the rendered CRDs", verbatim |
| A later full install of the same module version SHALL record those CRD... | body of "A full install records the CRDs a CRDs-only install applied", verbatim |
| Source: enhancement 0021 ("The operator's install artifact is a regist... | body of "CRDs-only install applies exactly the rendered CRDs", verbatim |
| `opm operator install --crds-only` SHALL render the same module versio... | reworded, nearest body: "CRDs-only install applies exactly the rendered CRDs"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Render the same module version, refuse under the three target rules before applying any CRD, apply exactly the rendered CRDs, wait for `Established` | render, apply and wait in "CRDs-only install applies exactly the rendered CRDs"; the refusal in "CRDs-only install refuses what a full install refuses before it applies a CRD"; words verbatim |

### Ledger: operator-lifecycle, Uninstall deletes the operator instance's recorded inventory

Form: c2 (REMOVED, ADDED under new names). New requirements: "Uninstall without an instance record deletes nothing"; "Uninstall deletes the recorded objects except the CRDs and the Namespace"; "Uninstall leaves behind what the instance delete verdict leaves behind"; "Uninstall deletes nothing outside the inventory and treats an absent object as deleted".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Uninstall after a module install | "Uninstall deletes the recorded objects except the CRDs and the Namespace", "Uninstall after a module install" | No (byte equal, checked) |
| Object an older release installed is removed | "Uninstall deletes the recorded objects except the CRDs and the Namespace", "Object an older release installed is removed" | No (byte equal, checked) |
| No record | "Uninstall without an instance record deletes nothing", "No record" | No (byte equal, checked) |
| Re-running uninstall | "Uninstall deletes nothing outside the inventory and treats an absent object as deleted", "Re-running uninstall" | No (byte equal, checked) |
| Recorded object adopted by another instance is left behind | "Uninstall leaves behind what the instance delete verdict leaves behind", "Recorded object adopted by another instance is left behind" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm operator uninstall` SHALL read the operator's instance record `op... | body of "Uninstall without an instance record deletes nothing", verbatim |
| When none exists, it SHALL delete nothing and exit 2 naming `opm opera... | body of "Uninstall without an instance record deletes nothing", verbatim |
| Otherwise, after the finalizer guard, it SHALL delete every object the... | reworded, nearest body: "Uninstall leaves behind what the instance delete verdict leaves behind"; the subject or a condition is restated so that the body reads alone, no rule changed |
| It SHALL delete no object the inventory does not list, SHALL NOT wait ... | reworded, nearest body: "Uninstall deletes nothing outside the inventory and treats an absent object as deleted"; the subject or a condition is restated so that the body reads alone, no rule changed |
| Source: enhancement 0006 ("`opm operator uninstall` preserves the name... | reworded, nearest body: "Uninstall deletes the recorded objects except the CRDs and the Namespace"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) After the finalizer guard delete every recorded object except CRDs and the Namespace, in descending weight order, leaving behind what the delete verdict leaves behind, then delete the record when no object failed | deletion order and the record in "Uninstall deletes the recorded objects except the CRDs and the Namespace"; the leave-behind clause in "Uninstall leaves behind what the instance delete verdict leaves behind" |
| (detail) Delete no object the inventory does not list, do not wait, treat an absent object as deleted | body of "Uninstall deletes nothing outside the inventory and treats an absent object as deleted", verbatim |
| (detail) Source: enhancement 0006 (preserves the namespace) and enhancement 0021 | 0021 with the no-record rule, 0006 with the deletion rule |

### Ledger: operator-lifecycle, Uninstall keeps the record when a recorded object could not be read

Form: b (MODIFIED parent, ADDED split-offs). New requirements: "Uninstall leaves an unreadable Namespace or CRD behind without failing"; "Re-running uninstall after a read failure deletes the rest". The parent "Uninstall keeps the record when a recorded object could not be read" stays, with a shorter body.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Unreadable recorded object keeps the record | "Uninstall keeps the record when a recorded object could not be read", "Unreadable recorded object keeps the record" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "Unreadable recorded Namespace is left behind" in "Uninstall leaves an unreadable Namespace or CRD behind without failing"
- "Re-running uninstall after the read cause is fixed" in "Re-running uninstall after a read failure deletes the rest"

| Old clause | New place |
| --- | --- |
| When `opm operator uninstall` reads the objects its instance record li... | body of "Uninstall keeps the record when a recorded object could not be read", verbatim |
| A recorded `Namespace` or `CustomResourceDefinition` that could not be... | reworded, nearest body: "Uninstall leaves an unreadable Namespace or CRD behind without failing"; the subject or a condition is restated so that the body reads alone, no rule changed |
| Re-running uninstall after the cause is fixed SHALL delete the remaini... | reworded, nearest body: "Re-running uninstall after a read failure deletes the rest"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) A recorded Namespace or CRD that could not be read is left behind and is not a failure | body of "Uninstall leaves an unreadable Namespace or CRD behind without failing" |
| (detail) Re-running uninstall after the cause is fixed deletes the remaining objects and the record | body of "Re-running uninstall after a read failure deletes the rest" |

### Ledger: operator-lifecycle, Uninstall fails when the instance record cannot be deleted

Form: b (MODIFIED parent, ADDED split-offs). New requirements: "A failed record delete exits by the cause of the failure". The parent "Uninstall fails when the instance record cannot be deleted" stays, with a shorter body.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Forbidden record delete | "Uninstall fails when the instance record cannot be deleted", "Forbidden record delete" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "Record delete times out" in "A failed record delete exits by the cause of the failure"
- "Record delete fails for another reason" in "A failed record delete exits by the cause of the failure"

| Old clause | New place |
| --- | --- |
| When `opm operator uninstall` deleted every recorded object and the de... | body of "Uninstall fails when the instance record cannot be deleted", verbatim |
| The error SHALL name the record and the cause, and SHALL say that the ... | body of "Uninstall fails when the instance record cannot be deleted", verbatim |
| The exit code SHALL be 4 when the API server denied the delete (Forbid... | reworded, nearest body: "A failed record delete exits by the cause of the failure"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Exit code 4, 3 or 1 by the cause of the failed record delete | body of "A failed record delete exits by the cause of the failure" |

### Ledger: operator-lifecycle, Uninstall follows the shared deletion plan

Form: c2 (REMOVED, ADDED under new names). New requirements: "Uninstall takes every action on a recorded object from the shared deletion plan"; "Uninstall acts on an object and on the record only as the plan allows"; "The finalizer guard and the no-record refusal run before the plan is built".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Recorded objects are deleted in the plan's order | "Uninstall takes every action on a recorded object from the shared deletion plan", "Recorded objects are deleted in the plan's order" | No (byte equal, checked) |
| Failed object holds the record | "Uninstall acts on an object and on the record only as the plan allows", "Failed object holds the record" | No (byte equal, checked) |
| Guard refuses before any plan | "The finalizer guard and the no-record refusal run before the plan is built", "Guard refuses before any plan" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm operator uninstall` SHALL take every action on a recorded object ... | body of "Uninstall takes every action on a recorded object from the shared deletion plan", verbatim |
| The output lines and the exit codes of the command SHALL stay as the o... | body of "Uninstall takes every action on a recorded object from the shared deletion plan", verbatim |
| Source: enhancement 0012 ("The library owns the whole deletion sequenc... | body of "Uninstall acts on an object and on the record only as the plan allows", verbatim |
| It SHALL read a recorded object, delete it or leave it in place only w... | reworded, nearest body: "Uninstall acts on an object and on the record only as the plan allows"; the subject or a condition is restated so that the body reads alone, no rule changed |
| The finalizer guard and the no-record refusal SHALL run before the pla... | reworded, nearest body: "The finalizer guard and the no-record refusal run before the plan is built"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Read, delete or leave a recorded object only as the plan says; delete the record only when the release verdict allows | body of "Uninstall acts on an object and on the record only as the plan allows" |
| (detail) The finalizer guard and the no-record refusal run before the plan is built | body of "The finalizer guard and the no-record refusal run before the plan is built" |

### Ledger: operator-lifecycle, Install runs every refusing check before its first write

Form: c2 (REMOVED, ADDED under new names). New requirements: "Install runs every check that can refuse it before it writes"; "Install writes in a fixed order after every check passed"; "The apply guard runs on every install"; "Install passes the guard no admission set and no proof of origin"; "The guard judges an object the record lists as recorded"; "The guard judges an existing object the record does not list"; "Install refuses an object adopted by another instance"; "Every refusal of the guard exits 2 and names the object, the instance and the annotation"; "Install waits out terminating objects before the guard runs"; "A failed read refuses the install in the check that made it"; "A failed read in the wait for terminating objects exits by the read error".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A refused install leaves the CRDs untouched | "Install runs every check that can refuse it before it writes", "A refused install leaves the CRDs untouched" | No (byte equal, checked) |
| Install right after uninstall waits out the terminating objects | "Install waits out terminating objects before the guard runs", "Install right after uninstall waits out the terminating objects" | No (byte equal, checked) |
| Terminating object outlives the budget | "Install waits out terminating objects before the guard runs", "Terminating object outlives the budget" | No (byte equal, checked) |
| An object unreadable from the start exits by the read error | "A failed read in the wait for terminating objects exits by the read error", "An object unreadable from the start exits by the read error" | No (byte equal, checked) |
| A read the apply guard fails exits with the guard code | "A failed read refuses the install in the check that made it", "A read the apply guard fails exits with the guard code" | No (byte equal, checked) |
| Reinstall refuses a foreign object new to the inventory | "The guard judges an existing object the record does not list", "Reinstall refuses a foreign object new to the inventory" | No (byte equal, checked) |
| First install refuses an object of another instance | "The guard judges an existing object the record does not list", "First install refuses an object of another instance" | No (byte equal, checked) |
| Install refuses an object adopted by another instance | "Install refuses an object adopted by another instance", "Install refuses an object adopted by another instance" | No (byte equal, checked) |
| Reinstall with a failed guard read exits with the guard code | "A failed read refuses the install in the check that made it", "Reinstall with a failed guard read exits with the guard code" | No (byte equal, checked) |
| Reinstall over its own recorded objects passes the guard | "The guard judges an object the record lists as recorded", "Reinstall over its own recorded objects passes the guard" | No (byte equal, checked) |
| Recorded objects that carry another identity are applied | "The guard judges an object the record lists as recorded", "Recorded objects that carry another identity are applied" | No (byte equal, checked) |
| Annotated objects that carry another identity are applied | "The guard judges an existing object the record does not list", "Annotated objects that carry another identity are applied" | No (byte equal, checked) |
| Objects that carry another identity, with no record and no annotation, are refused | "Every refusal of the guard exits 2 and names the object, the instance and the annotation", "Objects that carry another identity, with no record and no annotation, are refused" | No (byte equal, checked) |
| Unlabelled CRDs of an earlier manifest install are refused | "Install passes the guard no admission set and no proof of origin", "Unlabelled CRDs of an earlier manifest install are refused" | No (byte equal, checked) |
| CRDs-only refuses unlabelled CRDs | "The apply guard runs on every install", "CRDs-only refuses unlabelled CRDs" | No (byte equal, checked) |
| The instance's own objects without a record pass the guard | "The guard judges an existing object the record does not list", "The instance's own objects without a record pass the guard" | No (byte equal, checked) |
| An unrecorded object with the instance's UUID and no OPM managed-by label is refused | "The guard judges an existing object the record does not list", "An unrecorded object with the instance's UUID and no OPM managed-by label is refused" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "Install writes the CRD step, the instance apply and the Platform seed in order" in "Install writes in a fixed order after every check passed"

| Old clause | New place |
| --- | --- |
| Install SHALL pass the guard no admission set and SHALL run no proof o... | body of "Install passes the guard no admission set and no proof of origin", verbatim |
| An object the record lists is judged as recorded and SHALL pass whatev... | body of "The guard judges an object the record lists as recorded", verbatim |
| An existing object the record does not list SHALL pass when its `opmod... | body of "The guard judges an existing object the record does not list", verbatim |
| Install needs every object its render names, so, unlike any other appl... | body of "Install refuses an object adopted by another instance", verbatim |
| No refusal of install SHALL tell the user to delete or rename an objec... | body of "Every refusal of the guard exits 2 and names the object, the instance and the annotation", verbatim |
| Source: enhancement 0012 ("The apply guard runs on every apply, refusi... | body of "The apply guard runs on every install", verbatim |
| Objects of the render that exist with a deletion timestamp SHALL be wa... | body of "Install waits out terminating objects before the guard runs", verbatim |
| Only after every check passed SHALL install write, in this order: the ... | body of "Install writes in a fixed order after every check passed", verbatim |
| Outside the instance apply and its prune, install SHALL send no delete... | body of "Install writes in a fixed order after every check passed", verbatim |
| An install that refuses SHALL change no object in the cluster. | body of "Install runs every check that can refuse it before it writes", verbatim |
| A read that fails with an error other than NotFound SHALL refuse the i... | body of "A failed read refuses the install in the check that made it", verbatim |
| When the apply guard fails a read, the command SHALL exit 2, the code ... | body of "A failed read refuses the install in the check that made it", verbatim |
| An object that install cannot read from the start is therefore refused... | body of "A failed read in the wait for terminating objects exits by the read error", verbatim |
| Install SHALL run, before it writes any object, every check that can r... | reworded, nearest body: "Install runs every check that can refuse it before it writes"; the subject or a condition is restated so that the body reads alone, no rule changed |
| The guard SHALL run on every install, with or without a record of the ... | reworded, nearest body: "The apply guard runs on every install"; the subject or a condition is restated so that the body reads alone, no rule changed |
| Every refusal of the guard in this phase SHALL exit 2, SHALL name the ... | reworded, nearest body: "Every refusal of the guard exits 2 and names the object, the instance and the annotation"; the subject or a condition is restated so that the body reads alone, no rule changed |
| Two of these checks read every object install applies, in this order: ... | reworded, nearest body: "A failed read refuses the install in the check that made it"; the subject or a condition is restated so that the body reads alone, no rule changed |
| When the wait for terminating objects fails a read, the command SHALL ... | reworded, nearest body: "A failed read in the wait for terminating objects exits by the read error"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) List of checks, with the target rules named in a parenthesis | body of "Install runs every check that can refuse it before it writes"; the parenthesis (the three target rules) is not repeated, because "Install refuses a target operator newer than the CLI", "Install refuses a target whose render disagrees with its stated operator version" and "Install refuses a target whose ModuleInstance CRD lacks the required fields" state them, and the apply guard clause "over every object the render names" is in "The apply guard runs on every install" |
| (detail) The guard runs on every install, with or without a record | body of "The apply guard runs on every install" |
| (detail) Every refusal of the guard in this phase exits 2, names the object, the instance, the annotation and the UUID, leaves the cluster unchanged | body of "Every refusal of the guard exits 2 and names the object, the instance and the annotation"; "in this phase" reads "before the first write" |

### Ledger: operator-lifecycle, Install resolves another module version as the registry serves it

Form: c2 (REMOVED, ADDED under new names). New requirements: "`--version` resolves the operator module through the configured registry"; "Install refuses a selector the registry cannot satisfy before any cluster call"; "A registry mirror is the only registry install needs".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Selecting another version | "`--version` resolves the operator module through the configured registry", "Selecting another version" | No (byte equal, checked) |
| Unserved version | "Install refuses a selector the registry cannot satisfy before any cluster call", "Unserved version" | No (byte equal, checked) |
| Mirror only | "A registry mirror is the only registry install needs", "Mirror only" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm operator install --version <selector>` SHALL resolve the operator... | body of "`--version` resolves the operator module through the configured registry", verbatim |
| Install SHALL report the module version and the operator version it de... | body of "`--version` resolves the operator module through the configured registry", verbatim |
| A selector the registry cannot satisfy, or that is not a version selec... | body of "Install refuses a selector the registry cannot satisfy before any cluster call", verbatim |
| With the CLI's registry mapping pointed at a mirror holding the module... | body of "A registry mirror is the only registry install needs", verbatim |
| Source: enhancement 0021 ("The operator's install artifact is a regist... | body of "`--version` resolves the operator module through the configured registry", verbatim |

### Ledger: operator-lifecycle, The running-operator check finds the operator by its fixed names alone

Form: c2 (REMOVED, ADDED under new names). New requirements: "The running-operator check locates the operator by its fixed names"; "The running-operator check reads no Namespace and reports a missing CRD as not ready"; "The operator counts as running only on a positive finding".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Module-installed operator is found by the same names | "The running-operator check locates the operator by its fixed names", "Module-installed operator is found by the same names" | No (byte equal, checked) |
| Operator older than the fourth CRD is not ready | "The running-operator check reads no Namespace and reports a missing CRD as not ready", "Operator older than the fourth CRD is not ready" | No (byte equal, checked) |
| No operator at all | "The operator counts as running only on a positive finding", "No operator at all" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| Every CLI command that checks for a running operator before it acts SH... | body of "The running-operator check locates the operator by its fixed names", verbatim |
| The operator's objects SHALL be its fixed names, which every install o... | body of "The running-operator check locates the operator by its fixed names", verbatim |
| The `Namespace` is checked only as the `Deployment`'s namespace; the c... | body of "The running-operator check reads no Namespace and reports a missing CRD as not ready"; "the `Deployment`'s namespace" reads "the namespace of the `Deployment` `opm-operator-controller-manager`", the name from the old second sentence |
| Otherwise the check SHALL report the operator as not ready, name each ... | body of "The operator counts as running only on a positive finding", verbatim |
| An operator that lacks one of these CRDs SHALL be reported as not read... | body of "The running-operator check reads no Namespace and reports a missing CRD as not ready", old words "these CRDs"; the parenthesis names the sibling requirement that lists the CRDs (16:55 ruling) |
| The operator SHALL count as running only when every one of these `Cust... | reworded, nearest body: "The operator counts as running only on a positive finding"; the subject or a condition is restated so that the body reads alone, no rule changed |
| A read of any of these objects that fails SHALL count that object as n... | reworded, nearest body: "The operator counts as running only on a positive finding"; the subject or a condition is restated so that the body reads alone, no rule changed |

### Ledger: apply-pruning, Stale resources pruned after successful apply

Form: c2 (REMOVED, ADDED under new names). New requirements: "Stale resources are deleted in reverse weight order after a successful apply"; "A Namespace and a CustomResourceDefinition are never pruned".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Stale resources deleted in reverse weight order | "Stale resources are deleted in reverse weight order after a successful apply", "Stale resources deleted in reverse weight order" | No (byte equal, checked) |
| Already-deleted stale resource | "Stale resources are deleted in reverse weight order after a successful apply", "Already-deleted stale resource" | No (byte equal, checked) |
| Namespace excluded from pruning | "A Namespace and a CustomResourceDefinition are never pruned", "Namespace excluded from pruning" | No (byte equal, checked) |
| CRD excluded from pruning | "A Namespace and a CustomResourceDefinition are never pruned", "CRD excluded from pruning" | No (byte equal, checked) |
| Protected stale resources are reported as left behind | "A Namespace and a CustomResourceDefinition are never pruned", "Protected stale resources are reported as left behind" | No (byte equal, checked) |
| Same kind name in another group is not protected | "A Namespace and a CustomResourceDefinition are never pruned", "Same kind name in another group is not protected" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| After all rendered resources have been successfully applied, stale res... | body of "Stale resources are deleted in reverse weight order after a successful apply", verbatim |
| Deletion SHALL treat 404 as success (resource already gone). | body of "Stale resources are deleted in reverse weight order after a successful apply", verbatim |
| A `Namespace` in the core API group and a `CustomResourceDefinition` i... | body of "A Namespace and a CustomResourceDefinition are never pruned", verbatim |
| Every stale resource of a protected kind SHALL be reported as left beh... | body of "A Namespace and a CustomResourceDefinition are never pruned", verbatim |
| A stale `PersistentVolumeClaim` in the core API group SHALL be pruned ... | reworded, nearest body: "Stale resources are deleted in reverse weight order after a successful apply"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) A stale PersistentVolumeClaim is pruned only with `--delete-data` (pointer) | body of "Stale resources are deleted in reverse weight order after a successful apply"; the pointer names the renamed requirement "A stale PersistentVolumeClaim is kept unless `--delete-data` is set" |

### Ledger: apply-pruning, Unreadable instance record stops the apply

Form: c2 (REMOVED, ADDED under new names). New requirements: "A failed read of the instance record stops the apply before any change"; "A failed record read names its cause"; "A failed record read exits by the answer of the API server"; "A missing namespace is not read for an instance record"; "Operator install does not say nothing changed after its CRD step".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Read error on a real apply | "A failed read of the instance record stops the apply before any change", "Read error on a real apply" | No (byte equal, checked) |
| Read denied | "A failed record read exits by the answer of the API server", "Read denied" | No (byte equal, checked) |
| Read error on a dry run | "A failed read of the instance record stops the apply before any change", "Read error on a dry run" | No (byte equal, checked) |
| Missing record is still a first install | "A failed record read exits by the answer of the API server", "Missing record is still a first install" | No (byte equal, checked) |
| A missing namespace is not read for a record | "A missing namespace is not read for an instance record", "A missing namespace is not read for a record" | No (byte equal, checked) |
| Read denied inside operator install after its CRD step | "Operator install does not say nothing changed after its CRD step", "Read denied inside operator install after its CRD step" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The exit code SHALL be 4 when the API server denied the read (Forbidde... | body of "A failed record read exits by the answer of the API server", verbatim, led by the condition of the first old sentence |
| A NotFound answer SHALL still mean that no record exists. | body of "A failed record read exits by the answer of the API server", verbatim, led by the condition of the first old sentence |
| With `--create-namespace` and a missing namespace the record SHALL NOT... | body of "A missing namespace is not read for an instance record", verbatim |
| `opm operator install` SHALL NOT say that nothing was changed when the... | body of "Operator install does not say nothing changed after its CRD step", verbatim except "the read" reads "the read of the instance's `ModuleInstance` record" (the subject of the first old sentence) |
| When the read of the instance's `ModuleInstance` record fails with any... | reworded, nearest body: "A failed read of the instance record stops the apply before any change"; the subject or a condition is restated so that the body reads alone, no rule changed |
| The system SHALL NOT treat the failed read as a first install: it SHAL... | reworded, nearest body: "A failed read of the instance record stops the apply before any change"; the subject or a condition is restated so that the body reads alone, no rule changed |
| This SHALL hold for a dry run too, which SHALL NOT print a preview bui... | reworded, nearest body: "A failed read of the instance record stops the apply before any change"; the subject or a condition is restated so that the body reads alone, no rule changed |
| The error SHALL name the instance and its namespace, carry the cause o... | body of "A failed record read names its cause", verbatim, led by the condition "When the read of the instance's `ModuleInstance` record fails with any error other than NotFound," copied from the first old sentence; its scenario "A failed record read names the instance and its cause" restates the clause (14:48 ruling, no new case) |
| The error of `opm instance apply` and `opm module apply` SHALL say tha... | reworded, nearest body: "A failed read of the instance record stops the apply before any change"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Stop with an error; do not treat the failed read as a first install; apply, prune and write nothing | body of "A failed read of the instance record stops the apply before any change"; two sentences joined, "it SHALL NOT apply any resource, SHALL NOT prune, and SHALL NOT write the record" reads "they SHALL NOT apply any resource, prune, or write the record" |
| (detail) This holds for a dry run, which prints no preview built without the record | same body, "built" reads "made" |
| (detail) The error says the apply stopped before any change; the command changed nothing | same body; "The error of `opm instance apply` and `opm module apply`" reads "The error" |

### Ledger: apply-pruning, Failed prune keeps the entry and fails the command

Form: c2 (REMOVED, ADDED under new names). New requirements: "A failed stale delete is kept in the record and fails the command"; "A failed stale delete is reported and pruned again on the next apply".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Delete denied for one stale resource | "A failed stale delete is kept in the record and fails the command", "Delete denied for one stale resource" | No (byte equal, checked) |
| Next apply retries the prune | "A failed stale delete is reported and pruned again on the next apply", "Next apply retries the prune" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| When the delete of a stale resource fails with any error other than No... | body of "A failed stale delete is kept in the record and fails the command", verbatim |
| The command SHALL exit 1 and SHALL NOT print the success line. | body of "A failed stale delete is kept in the record and fails the command", verbatim |
| A stale resource that was deleted, or that was already gone, SHALL NOT... | body of "A failed stale delete is kept in the record and fails the command", verbatim |
| On the next apply a kept entry SHALL be stale again and SHALL be prune... | body of "A failed stale delete is reported and pruned again on the next apply", verbatim |
| The output SHALL name each such resource by kind, namespace and name w... | reworded, nearest body: "A failed stale delete is reported and pruned again on the next apply"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) The output names each such resource with its delete error and says the next apply retries | body of "A failed stale delete is reported and pruned again on the next apply"; "each such resource" reads "each stale resource whose delete failed" |

### Ledger: apply-pruning, Apply flow orchestration

Form: c2 (REMOVED, ADDED under new names). New requirements: "The apply flow reads, checks and applies before it prunes"; "The apply flow prunes only after every apply succeeded"; "No step of the apply flow filters the stale set"; "The apply flow writes the record last".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Normal apply with pruning | "The apply flow prunes only after every apply succeeded", "Normal apply with pruning" | No (byte equal, checked) |
| First-time apply | "The apply flow reads, checks and applies before it prunes", "First-time apply" | No (byte equal, checked) |
| Record write follows a failed prune | "The apply flow writes the record last", "Record write follows a failed prune" | No (byte equal, checked) |
| The record takes the new identity after the prune | "The apply flow writes the record last", "The record takes the new identity after the prune" | No (byte equal, checked) |
| The guard runs on a later apply | "The apply flow reads, checks and applies before it prunes", "The guard runs on a later apply" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The apply flow SHALL follow this sequence: | body of "The apply flow reads, checks and applies before it prunes", verbatim |
| No step SHALL filter the stale set after it is computed: the stale set... | body of "No step of the apply flow filters the stale set", verbatim; its scenario restates the clause (14:48 ruling) |
| (1) render resources, (2) compute manifest digest, (3) compute change ... | reworded, nearest body: "The apply flow reads, checks and applies before it prunes"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Steps (1) to (6) of the sequence | body of "The apply flow reads, checks and applies before it prunes", step text verbatim |
| (detail) Steps (7a), (7b) and (9) | body of "The apply flow prunes only after every apply succeeded"; the old sentence "The apply flow SHALL follow this sequence" supplies the keyword and the subject, step numbers and "step 4", "step 7a" as in the old text; the parenthesis "(steps 1 to 6 as listed in requirement ...)" names the sibling that holds steps 1 to 6 (16:55 ruling) |
| (detail) Step (8) | body of "The apply flow writes the record last"; "SHALL" added in front of the verb because every ADDED body needs one (the old sequence was one SHALL sentence) |; "step (7a)" reads "step 7a" as in the old text

### Ledger: apply-pruning, Dry-run prune preview lists left-behind resources

Form: c2 (REMOVED, ADDED under new names). New requirements: "The dry-run preview asks the real prune's verdict and sends no write"; "The dry-run preview lists the resources a real prune would delete or leave behind"; "The dry-run preview lists a PersistentVolumeClaim a real prune would keep"; "The dry-run preview lists a resource the verdict leaves in place on its own line"; "The dry-run preview lists no resource that is gone and flags one it cannot read"; "`--no-prune` turns the dry-run prune preview off".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Preview separates pruned and left-behind resources | "The dry-run preview lists the resources a real prune would delete or leave behind", "Preview separates pruned and left-behind resources" | No (byte equal, checked) |
| No-prune dry run lists nothing | "`--no-prune` turns the dry-run prune preview off", "No-prune dry run lists nothing" | No (byte equal, checked) |
| Preview lists a claim as kept | "The dry-run preview lists a PersistentVolumeClaim a real prune would keep", "Preview lists a claim as kept" | No (byte equal, checked) |
| Preview with delete-data lists a claim as pruned | "The dry-run preview lists a PersistentVolumeClaim a real prune would keep", "Preview with delete-data lists a claim as pruned" | No (byte equal, checked) |
| Preview keeps what the instance does not own | "The dry-run preview lists a resource the verdict leaves in place on its own line", "Preview keeps what the instance does not own" | No (byte equal, checked) |
| Preview names the owning instance | "The dry-run preview lists a resource the verdict leaves in place on its own line", "Preview names the owning instance" | No (byte equal, checked) |
| Preview lets go of an adopted resource | "The dry-run preview asks the real prune's verdict and sends no write", "Preview lets go of an adopted resource" | No (byte equal, checked) |
| Preview leaves out a stale resource that is gone | "The dry-run preview lists no resource that is gone and flags one it cannot read", "Preview leaves out a stale resource that is gone" | No (byte equal, checked) |
| Preview cannot read a stale resource | "The dry-run preview lists no resource that is gone and flags one it cannot read", "Preview cannot read a stale resource" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| It SHALL be listed on its own line with the reason the verdict gives, ... | body of "The dry-run preview lists a resource the verdict leaves in place on its own line", verbatim |
| Neither line SHALL change the exit code. | body of "The dry-run preview lists a resource the verdict leaves in place on its own line", verbatim |
| A stale resource whose live read fails with any error other than NotFo... | body of "The dry-run preview lists no resource that is gone and flags one it cannot read", verbatim |
| The preview SHALL send no delete and no other write. | body of "The dry-run preview asks the real prune's verdict and sends no write", verbatim |
| With `--no-prune`, none of the lists SHALL be printed and no stale res... | body of "`--no-prune` turns the dry-run prune preview off", verbatim |
| When `--dry-run` is set and pruning is enabled, the apply SHALL read e... | reworded, nearest body: "The dry-run preview asks the real prune's verdict and sends no write"; the subject or a condition is restated so that the body reads alone, no rule changed |
| It SHALL list each stale resource the verdict allows a real apply to d... | reworded, nearest body: "The dry-run preview lists the resources a real prune would delete or leave behind"; the subject or a condition is restated so that the body reads alone, no rule changed |
| The preview SHALL use the same protected-kind rule and the same `--del... | reworded, nearest body: "The dry-run preview lists a PersistentVolumeClaim a real prune would keep"; the subject or a condition is restated so that the body reads alone, no rule changed |
| A stale resource the verdict leaves in place SHALL NOT be listed as `w... | reworded, nearest body: "The dry-run preview lists a resource the verdict leaves in place on its own line"; the subject or a condition is restated so that the body reads alone, no rule changed |
| A stale resource that is already gone SHALL be listed on no line.... | reworded, nearest body: "The dry-run preview lists no resource that is gone and flags one it cannot read"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Read each stale resource a real apply would try to prune, ask the same verdict with the same identity | body of "The dry-run preview asks the real prune's verdict and sends no write"; the pointer names the renamed requirement "Prune takes the delete decision from the ownership verdict" |
| (detail) List `would prune`, `left behind`, `kept` | bodies of "The dry-run preview lists the resources a real prune would delete or leave behind" and "The dry-run preview lists a PersistentVolumeClaim a real prune would keep" |
| (detail) Same protected-kind rule and same `--delete-data` rule as the real prune, so nothing is listed as `would prune` that a real apply leaves | split: protected-kind half in the first body above, `--delete-data` half and the "so no stale resource" consequence in the second |

### Ledger: apply-pruning, First install over OPM-managed resources warns

Form: c2 (REMOVED, ADDED under new names). New requirements: "A first install over OPM-managed resources prints one warning"; "The first-install warning says what the apply does and does not stop it"; "The warning of a dry run names the release to apply with first"; "The warning of a real run names the Secret and not an older release"; "The first-install warning is not printed when it does not apply".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Existing managed resources on a first install | "A first install over OPM-managed resources prints one warning", "Existing managed resources on a first install" | No (byte equal, checked) |
| Dry run warns before anything is written | "The warning of a dry run names the release to apply with first", "Dry run warns before anything is written" | No (byte equal, checked) |
| Operator install prints no warning | "The first-install warning is not printed when it does not apply", "Operator install prints no warning" | No (byte equal, checked) |
| Clean first install prints no warning | "The first-install warning is not printed when it does not apply", "Clean first install prints no warning" | No (byte equal, checked) |
| Instance with a record prints no warning | "The first-install warning is not printed when it does not apply", "Instance with a record prints no warning" | No (byte equal, checked) |
| Dry run that previews a refusal prints no warning | "The warning of a dry run names the release to apply with first", "Dry run that previews a refusal prints no warning" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "First-install warning content and effect" in "The first-install warning says what the apply does and does not stop it"
- "Real-run warning names the Secret" in "The warning of a real run names the Secret and not an older release"

| Old clause | New place |
| --- | --- |
| On a first-time apply (no `ModuleInstance` record), when one or more r... | body of "A first install over OPM-managed resources prints one warning", verbatim |
| The warning SHALL NOT change the exit code and SHALL NOT stop the appl... | body of "The first-install warning says what the apply does and does not stop it", verbatim |
| The dry-run warning SHALL name the release to apply with first (`v1.0.... | body of "The warning of a dry run names the release to apply with first", verbatim |
| The warning of a real run SHALL NOT send the user to that release, bec... | body of "The warning of a real run names the Secret and not an older release"; "that release" reads "the release to apply with first (`v1.0.0-beta.10`)", words of the old second paragraph |
| It SHALL name the Secret (`opm.<name>.<id>`), SHALL say to keep it and... | body of "The warning of a real run names the Secret and not an older release", verbatim |
| The apply SHALL print no such warning when no rendered resource exists... | body of "The first-install warning is not printed when it does not apply", verbatim |
| `opm operator install` SHALL NOT print it: install applies the render'... | body of "The first-install warning is not printed when it does not apply", verbatim |
| The warning SHALL give the number of such resources and the number of ... | reworded, nearest body: "The first-install warning says what the apply does and does not stop it"; the subject or a condition is restated so that the body reads alone, no rule changed |
| A dry run SHALL print the warning too, when its ownership guard refuse... | reworded, nearest body: "The warning of a dry run names the release to apply with first"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) The warning gives the counts and says: no record, updates in place and records them, prunes nothing | counts in "A first install over OPM-managed resources prints one warning"; the content and the exit-code rule in "The first-install warning says what the apply does and does not stop it" |
| (detail) A dry run prints the warning too when the guard refuses nothing | body of "The warning of a dry run names the release to apply with first"; the pointer names the renamed requirement "A dry run previews each refusal of the guard" |

### Ledger: apply-pruning, Prune keeps stale PersistentVolumeClaims unless delete-data is set

Form: c2 (REMOVED, ADDED under new names). New requirements: "A stale PersistentVolumeClaim is kept unless `--delete-data` is set"; "A kept PersistentVolumeClaim is listed at the informational level and does not fail the apply"; "A stale PersistentVolumeClaim that is gone is not kept"; "`--delete-data` and `--no-prune` exclude each other"; "`--delete-data` on an operator-managed instance only warns".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Stale claim is kept and stays in the inventory | "A kept PersistentVolumeClaim is listed at the informational level and does not fail the apply", "Stale claim is kept and stays in the inventory" | No (byte equal, checked) |
| Stale claim that is already gone | "A stale PersistentVolumeClaim that is gone is not kept", "Stale claim that is already gone" | No (byte equal, checked) |
| Stale claim that cannot be read | "A stale PersistentVolumeClaim that is gone is not kept", "Stale claim that cannot be read" | No (byte equal, checked) |
| Delete-data prunes the claim | "A stale PersistentVolumeClaim is kept unless `--delete-data` is set", "Delete-data prunes the claim" | No (byte equal, checked) |
| A later apply with the flag removes a claim kept earlier | "A stale PersistentVolumeClaim is kept unless `--delete-data` is set", "A later apply with the flag removes a claim kept earlier" | No (byte equal, checked) |
| Empty render with force keeps claims | "A stale PersistentVolumeClaim is kept unless `--delete-data` is set", "Empty render with force keeps claims" | No (byte equal, checked) |
| Delete-data with no-prune | "`--delete-data` and `--no-prune` exclude each other", "Delete-data with no-prune" | No (byte equal, checked) |
| Delete-data on an operator-managed instance | "`--delete-data` on an operator-managed instance only warns", "Delete-data on an operator-managed instance" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The prune of `opm instance apply` and `opm module apply` SHALL NOT del... | body of "A stale PersistentVolumeClaim is kept unless `--delete-data` is set", verbatim |
| A kept claim SHALL stay in the written inventory, so that it is stale ... | body of "A stale PersistentVolumeClaim is kept unless `--delete-data` is set", verbatim |
| Each kept claim SHALL be listed on its own line, naming its kind, name... | body of "A kept PersistentVolumeClaim is listed at the informational level and does not fail the apply", verbatim |
| A stale claim that is no longer in the cluster (its read answers NotFo... | body of "A stale PersistentVolumeClaim that is gone is not kept", verbatim |
| A stale claim whose read fails in any other way SHALL be kept. | body of "A stale PersistentVolumeClaim that is gone is not kept", verbatim |
| A kept claim SHALL NOT fail the apply: the exit code SHALL be 0 and th... | body of "A kept PersistentVolumeClaim is listed at the informational level and does not fail the apply", verbatim |
| With `--delete-data`, a stale claim SHALL be pruned like any other sta... | body of "A stale PersistentVolumeClaim is kept unless `--delete-data` is set", verbatim |
| `--delete-data` and `--no-prune` SHALL exclude each other: both togeth... | body of "`--delete-data` and `--no-prune` exclude each other", verbatim |
| On an operator-managed instance `--delete-data` SHALL NOT change what ... | body of "`--delete-data` on an operator-managed instance only warns", verbatim |

### Ledger: apply-pruning, Prune asks the ownership verdict for each stale resource

Form: c2 (REMOVED, ADDED under new names). New requirements: "Prune takes the delete decision from the ownership verdict"; "The prune does not delete a stale resource adopted by another instance"; "The verdict is asked with the identity stored in the instance record"; "A resource that fails the verdict is left behind and not kept"; "The prune still deletes under the recorded identity after the identity changes"; "Each prune delete carries a UID precondition"; "A failed live read counts as a failed delete"; "The verdict is not asked for a kept claim".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Stale name taken by a user's object | "A resource that fails the verdict is left behind and not kept", "Stale name taken by a user's object" | No (byte equal, checked) |
| Stale resource now owned by another instance | "Prune takes the delete decision from the ownership verdict", "Stale resource now owned by another instance" | No (byte equal, checked) |
| Stale resource adopted by another instance | "The prune does not delete a stale resource adopted by another instance", "Stale resource adopted by another instance" | No (byte equal, checked) |
| Stale object after the instance identity changed | "The prune still deletes under the recorded identity after the identity changes", "Stale object after the instance identity changed" | No (byte equal, checked) |
| Stale object that carries neither identity | "The verdict is asked with the identity stored in the instance record", "Stale object that carries neither identity" | No (byte equal, checked) |
| Owned stale resource is deleted with a UID precondition | "Each prune delete carries a UID precondition", "Owned stale resource is deleted with a UID precondition" | No (byte equal, checked) |
| Object replaced between the read and the delete | "Each prune delete carries a UID precondition", "Object replaced between the read and the delete" | No (byte equal, checked) |
| Live read denied | "A failed live read counts as a failed delete", "Live read denied" | No (byte equal, checked) |
| Kept claim is not judged | "The verdict is not asked for a kept claim", "Kept claim is not judged" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| Before the prune deletes a stale resource, it SHALL read the live obje... | body of "Prune takes the delete decision from the ownership verdict", verbatim |
| The verdict SHALL be asked with the instance identity stored in the in... | body of "The verdict is asked with the identity stored in the instance record", verbatim |
| An object with no UUID label SHALL be judged without the UUID comparis... | body of "The verdict is asked with the identity stored in the instance record", verbatim |
| A resource that fails the verdict SHALL NOT be deleted: it SHALL be li... | body of "A resource that fails the verdict is left behind and not kept", verbatim |
| The stale set itself SHALL NOT change: it stays every recorded resourc... | body of "A resource that fails the verdict is left behind and not kept", verbatim |
| An instance's identity derives from its module's path without the majo... | body of "The prune still deletes under the recorded identity after the identity changes", verbatim |
| The record is found by name and namespace, so within one record the id... | body of "The prune still deletes under the recorded identity after the identity changes", verbatim |
| On the first apply after such a change the prune SHALL still delete th... | body of "The prune still deletes under the recorded identity after the identity changes", verbatim |
| Each delete SHALL carry a precondition on the UID of the object that w... | body of "Each prune delete carries a UID precondition", verbatim |
| A stale `PersistentVolumeClaim` that is kept SHALL NOT be judged; with... | body of "The verdict is not asked for a kept claim", verbatim |
| The prune SHALL delete the resource only when the live object carries ... | reworded, nearest body: "Prune takes the delete decision from the ownership verdict"; the subject or a condition is restated so that the body reads alone, no rule changed |
| Source: enhancement 0012 ("The library owns the whole deletion sequenc... | reworded, nearest body: "The prune does not delete a stale resource adopted by another instance"; the subject or a condition is restated so that the body reads alone, no rule changed |
| A delete the API server refuses on that precondition, and a live read ... | reworded, nearest body: "A failed live read counts as a failed delete"; the subject or a condition is restated so that the body reads alone, no rule changed |
| The dry-run prune preview SHALL ask the same verdict with the same ide... | reworded, nearest body: "The verdict is not asked for a kept claim"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Delete only when the live object carries an OPM managed-by label, its UUID label does not name another instance, and its adopt annotation does not name another instance | label and UUID condition in "Prune takes the delete decision from the ownership verdict"; adopt condition in "The prune does not delete a stale resource adopted by another instance" (written as "SHALL NOT delete a resource whose live adopt annotation names an instance other than the recorded one"); together they say the same "only when" |
| (detail) Source: enhancement 0012 (three quotes) | quote 1 with the verdict requirement, quote 3 with the adopt requirement, quote 2 with "A resource that fails the verdict is left behind and not kept" |
| (detail) A delete refused on the precondition, and a failed live read, count as a failed delete (pointer) | split into "Each prune delete carries a UID precondition" and "A failed live read counts as a failed delete"; the pointer names the renamed requirement "A failed stale delete is kept in the record and fails the command" |
| (detail) The dry-run preview asks the same verdict (pointer) | body of "The verdict is not asked for a kept claim"; the pointer names "The dry-run preview asks the real prune's verdict and sends no write" |

### Ledger: apply-pruning, Resource adopted by another instance is let go

Form: c2 (REMOVED, ADDED under new names). New requirements: "A resource adopted by another instance is not applied and does not fail the apply"; "The apply warns about a resource let go and writes the inventory without it"; "A dry run skips a resource adopted by another instance"; "A real apply that lets go of every rendered resource says that nothing was applied"; "A dry run where every rendered resource is let go says that nothing would be applied"; "Operator install refuses a resource adopted by another instance".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Recorded resource handed to another instance | "A resource adopted by another instance is not applied and does not fail the apply", "Recorded resource handed to another instance" | No (byte equal, checked) |
| The instance does not take the resource back | "A resource adopted by another instance is not applied and does not fail the apply", "The instance does not take the resource back" | No (byte equal, checked) |
| Annotating the resource back reverses the hand-over | "The apply warns about a resource let go and writes the inventory without it", "Annotating the resource back reverses the hand-over" | No (byte equal, checked) |
| Every rendered resource is adopted elsewhere | "A real apply that lets go of every rendered resource says that nothing was applied", "Every rendered resource is adopted elsewhere" | No (byte equal, checked) |
| Dry run skips an adopted resource | "A dry run skips a resource adopted by another instance", "Dry run skips an adopted resource" | No (byte equal, checked) |
| Dry run where every rendered resource is adopted elsewhere | "A dry run where every rendered resource is let go says that nothing would be applied", "Dry run where every rendered resource is adopted elsewhere" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "Operator install refuses an adopted resource" in "Operator install refuses a resource adopted by another instance"

| Old clause | New place |
| --- | --- |
| When the instance's UUID is known and the live `opmodel.dev/adopt` ann... | body of "A resource adopted by another instance is not applied and does not fail the apply", verbatim |
| This SHALL hold for a resource the instance's record lists, and for an... | body of "A resource adopted by another instance is not applied and does not fail the apply", verbatim |
| The apply SHALL NOT prune or otherwise delete it. | body of "The apply warns about a resource let go and writes the inventory without it", verbatim |
| When every rendered resource is let go, so that the apply applies noth... | body of "A real apply that lets go of every rendered resource says that nothing was applied", verbatim |
| It SHALL print no success line that says resources were applied or are... | body of "A real apply that lets go of every rendered resource says that nothing was applied", verbatim |
| Source: enhancement 0012 ("The apply guard runs on every apply, refusi... | body of "Operator install refuses a resource adopted by another instance", verbatim |
| The apply SHALL print one warning that names the resource and the adop... | reworded, nearest body: "The apply warns about a resource let go and writes the inventory without it"; the subject or a condition is restated so that the body reads alone, no rule changed |
| A dry run SHALL NOT send such a resource to the server either: in plac... | reworded, nearest body: "A dry run skips a resource adopted by another instance"; the first sentence of the old body of "A resource adopted by another instance is not applied and does not fail the apply" supplies the condition ("When the instance's UUID is known and the live `opmodel.dev/adopt` annotation of a rendered resource names another instance"), copied verbatim, and "such a resource" is tied to that requirement by name; the let-go set and the dry-run sentence are the old ones |
| A dry run in the same state SHALL print one closing line that says tha... | reworded, nearest body: "A dry run where every rendered resource is let go says that nothing would be applied"; the subject or a condition is restated so that the body reads alone, no rule changed |
| `opm operator install` is the exception: it refuses on such a resource... | reworded, nearest body: "Operator install refuses a resource adopted by another instance"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) `opm operator install` is the exception: it refuses on such a resource | body of "Operator install refuses a resource adopted by another instance"; "it refuses" reads "it SHALL refuse" so that the body carries a SHALL |

### Ledger: apply-pruning, Prune follows the shared deletion plan

Form: c2 (REMOVED, ADDED under new names). New requirements: "The prune takes every action from the deletion plan shared with the operator"; "The prune acts in the plan's order with the plan's propagation and precondition"; "A failed read or delete does not stop the plan, except a failed discovery request"; "A stale resource counts as pruned when its delete is accepted".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Stale resources are deleted in the plan's order | "The prune acts in the plan's order with the plan's propagation and precondition", "Stale resources are deleted in the plan's order" | No (byte equal, checked) |
| Delete carries what the plan names | "The prune acts in the plan's order with the plan's propagation and precondition", "Delete carries what the plan names" | No (byte equal, checked) |
| One failed delete does not stop the others | "A failed read or delete does not stop the plan, except a failed discovery request", "One failed delete does not stop the others" | No (byte equal, checked) |
| Failed discovery request stops the requests | "A failed read or delete does not stop the plan, except a failed discovery request", "Failed discovery request stops the requests" | No (byte equal, checked) |
| Stale resource held by a finalizer | "A stale resource counts as pruned when its delete is accepted", "Stale resource held by a finalizer" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "The deletion plan holds only the stale entries the prune may delete" in "The prune takes every action from the deletion plan shared with the operator"

| Old clause | New place |
| --- | --- |
| The prune of `opm instance apply` and `opm module apply` SHALL take ev... | body of "The prune takes every action from the deletion plan shared with the operator", verbatim |
| That set SHALL NOT hold a `Namespace`, a `CustomResourceDefinition`, o... | body of "The prune takes every action from the deletion plan shared with the operator", verbatim |
| The prune SHALL read a stale resource, delete it or leave it in place ... | body of "The prune acts in the plan's order with the plan's propagation and precondition", verbatim |
| A resource whose read or delete fails SHALL NOT stop the plan, except ... | body of "A failed read or delete does not stop the plan, except a failed discovery request", verbatim |
| A stale resource SHALL count as pruned when the API server accepts its... | body of "A stale resource counts as pruned when its delete is accepted", verbatim |
| Which entries stay in the written inventory, the output lines and the ... | body of "A stale resource counts as pruned when its delete is accepted", verbatim |
| The dry-run prune preview SHALL NOT be affected. | body of "A stale resource counts as pruned when its delete is accepted", verbatim |
| Source: enhancement 0012 ("The library owns the whole deletion sequenc... | body of "The prune acts in the plan's order with the plan's propagation and precondition", verbatim |

### Ledger: apply-pruning, Ownership guard judges every apply and dry run

Form: c2 (REMOVED, ADDED under new names). New requirements: "The ownership guard judges every apply"; "The ownership guard refuses an unlisted resource that is foreign or owned by another instance"; "The ownership guard refuses a terminating resource whether or not the record lists it"; "A refused apply fails before its first write and reports every refused resource"; "A refusal on a first apply explains the earlier identity"; "The adopt annotation of this instance lets a resource pass the ownership refusals"; "The CLI sets no adopt annotation and offers no flag to bypass the guard"; "An unreadable rendered resource fails the apply before its first write"; "A dry run previews each refusal of the guard"; "A dry run stops where the real run stops"; "A dry run fails on an unreadable resource as the real run does"; "A dry run of an operator-managed instance does not run the guard"; "The guard takes no admission set"; "A guard refusal in operator install keeps that command's exit code and wording".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Foreign resource refused on a first apply | "The CLI sets no adopt annotation and offers no flag to bypass the guard", "Foreign resource refused on a first apply" | No (byte equal, checked) |
| Foreign resource refused on a later apply | "The ownership guard refuses an unlisted resource that is foreign or owned by another instance", "Foreign resource refused on a later apply" | No (byte equal, checked) |
| Resource of another instance refused | "The ownership guard refuses an unlisted resource that is foreign or owned by another instance", "Resource of another instance refused" | No (byte equal, checked) |
| Recorded resource is applied whatever its UUID label | "The ownership guard refuses an unlisted resource that is foreign or owned by another instance", "Recorded resource is applied whatever its UUID label" | No (byte equal, checked) |
| Terminating resource refused on every apply | "The ownership guard refuses a terminating resource whether or not the record lists it", "Terminating resource refused on every apply" | No (byte equal, checked) |
| Adopt annotation lets the instance take a resource | "The adopt annotation of this instance lets a resource pass the ownership refusals", "Adopt annotation lets the instance take a resource" | No (byte equal, checked) |
| Adopt annotation does not lift the terminating refusal | "The ownership guard refuses a terminating resource whether or not the record lists it", "Adopt annotation does not lift the terminating refusal" | No (byte equal, checked) |
| First install over the instance's own resources under an earlier identity | "A refusal on a first apply explains the earlier identity", "First install over the instance's own resources under an earlier identity" | No (byte equal, checked) |
| Every refused resource is reported | "A refused apply fails before its first write and reports every refused resource", "Every refused resource is reported" | No (byte equal, checked) |
| Unreadable resource refused on a later apply | "An unreadable rendered resource fails the apply before its first write", "Unreadable resource refused on a later apply" | No (byte equal, checked) |
| Refusal with create-namespace changes nothing | "A refused apply fails before its first write and reports every refused resource", "Refusal with create-namespace changes nothing" | No (byte equal, checked) |
| Absent resource passes | "The ownership guard judges every apply", "Absent resource passes" | No (byte equal, checked) |
| Dry run previews a refusal and exits 1 | "A dry run stops where the real run stops", "Dry run previews a refusal and exits 1" | No (byte equal, checked) |
| Dry run names the owner of a refused resource | "A dry run previews each refusal of the guard", "Dry run names the owner of a refused resource" | No (byte equal, checked) |
| Dry run lists every refused resource | "A dry run previews each refusal of the guard", "Dry run lists every refused resource" | No (byte equal, checked) |
| Dry run with an unreadable resource | "A dry run fails on an unreadable resource as the real run does", "Dry run with an unreadable resource" | No (byte equal, checked) |
| Dry run with nothing to refuse | "A dry run previews each refusal of the guard", "Dry run with nothing to refuse" | No (byte equal, checked) |
| Operator install lifts no refusal | "The guard takes no admission set", "Operator install lifts no refusal" | No (byte equal, checked) |

New scenarios (each uses only words and values of the old body of this requirement):

- "Dry run of an operator-managed instance skips the guard" in "A dry run of an operator-managed instance does not run the guard"
- "Guard refusal in the check phase of operator install" in "A guard refusal in operator install keeps that command's exit code and wording"

| Old clause | New place |
| --- | --- |
| On every apply in CLI-executor mode, the first as well as every later ... | body of "The ownership guard judges every apply", verbatim |
| An existing resource that the instance's `ModuleInstance` record does ... | body of "The ownership guard refuses an unlisted resource that is foreign or owned by another instance", verbatim |
| An existing resource with a `deletionTimestamp` SHALL be refused wheth... | body of "The ownership guard refuses a terminating resource whether or not the record lists it", verbatim |
| When any resource is refused, the apply SHALL fail before its first wr... | body of "A refused apply fails before its first write and reports every refused resource", verbatim |
| When the instance has no record and a resource is refused because it c... | body of "A refusal on a first apply explains the earlier identity", verbatim |
| The one override SHALL be the adopt annotation: an existing resource w... | body of "The adopt annotation of this instance lets a resource pass the ownership refusals", verbatim |
| The CLI SHALL NOT set the annotation and SHALL offer no flag that sets... | body of "The CLI sets no adopt annotation and offers no flag to bypass the guard", verbatim |
| Every ownership refusal SHALL name the annotation and the UUID to set,... | body of "The CLI sets no adopt annotation and offers no flag to bypass the guard", verbatim |
| Source: enhancement 0012 ("The apply guard runs on every apply, refusi... | body of "The ownership guard refuses an unlisted resource that is foreign or owned by another instance", verbatim |
| If a rendered resource cannot be read (the read fails with any error o... | body of "An unreadable rendered resource fails the apply before its first write", verbatim |
| With `--create-namespace` and a missing namespace, the guard SHALL ski... | body of "An unreadable rendered resource fails the apply before its first write", verbatim |
| A dry run SHALL run the same guard, with the same reads, and the same ... | body of "A dry run previews each refusal of the guard", verbatim |
| For each resource the real run would refuse, the dry run SHALL print o... | body of "A dry run previews each refusal of the guard", verbatim |
| On a first apply it SHALL add the same earlier-identity line as the re... | body of "A dry run stops where the real run stops", verbatim |
| When a rendered resource cannot be read, the dry run SHALL fail with t... | body of "A dry run fails on an unreadable resource as the real run does", verbatim |
| A dry run of an operator-managed instance SHALL NOT run the guard: the... | body of "A dry run of an operator-managed instance does not run the guard", verbatim |
| The guard SHALL take no admission set: no caller, `opm operator instal... | body of "The guard takes no admission set", verbatim |
| An existing resource SHALL pass only when the record of the instance l... | body of "The guard takes no admission set", verbatim |
| When the guard refuses in the check phase of `opm operator install`, t... | body of "A guard refusal in operator install keeps that command's exit code and wording", verbatim |
| When it refuses inside that command's instance apply, after the instal... | body of "A guard refusal in operator install keeps that command's exit code and wording", verbatim |
| A resource that does not exist SHALL pass.... | reworded, nearest body: "The ownership guard judges every apply"; the subject or a condition is restated so that the body reads alone, no rule changed |
| Source: enhancement 0012 ("The library owns the whole deletion sequenc... | reworded, nearest body: "The ownership guard refuses an unlisted resource that is foreign or owned by another instance"; the subject or a condition is restated so that the body reads alone, no rule changed |
| The annotation SHALL NOT override the refusal of a terminating resourc... | reworded, nearest body: "The ownership guard refuses a terminating resource whether or not the record lists it"; the subject or a condition is restated so that the body reads alone, no rule changed |
| A NotFound answer, or a kind the cluster does not serve yet, SHALL mea... | reworded, nearest body: "The ownership guard judges every apply"; the subject or a condition is restated so that the body reads alone, no rule changed |
| The dry run SHALL then stop where the real run stops: it SHALL send no... | reworded, nearest body: "A dry run stops where the real run stops"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) A resource that does not exist passes; NotFound or an unserved kind means it does not exist | body of "The ownership guard judges every apply" (the NotFound sentence joined to it) |
| (detail) Source: enhancement 0012 (two quotes, twice) | quote 1 with "A refused apply fails before its first write and reports every refused resource", quote 3 with the unlisted-resource refusal and the adopt-annotation requirement |
| (detail) The adopt annotation does not override the refusal of a terminating resource | moved next to the terminating refusal: body of "The ownership guard refuses a terminating resource whether or not the record lists it" |
| (detail) A NotFound answer or an unserved kind means the resource does not exist | body of "The ownership guard judges every apply", same sentence |
### Cross-unit reference left for unit cli-B2

The requirement "Ownership guard judges every apply and dry run" of `apply-pruning` is removed in unit cli-B1 (form c2, split into fourteen requirements). `openspec/specs/deploy/spec.md` cited it in the requirement "Create-namespace creates the namespace only after every check passed", in the words `(capability apply-pruning, "Ownership guard judges every apply and dry run")`. Unit cli-B2 resolved this: the first new requirement of that split, "Create-namespace creates a missing namespace only after every check that can refuse the apply passed", cites `(capability `apply-pruning`, "The ownership guard judges every apply")` (swarm DECISIONS.md 2026-10-19 15:05). Checked with `grep -rn --exclude-dir=archive` on 2026-10-19: no other reference outside the rewritten requirements.

### Strength table of cli-B1

Plain sentences that carry an RFC 2119 keyword in the new text and had none in the old text:

| Old sentence | New body | Word added |
| --- | --- | --- |
| `opm operator install` is the exception: it refuses on such a resource | "Operator install refuses a resource adopted by another instance" | "it refuses" reads "it SHALL refuse" |


## Ledger entries of cli-B2 (deploy, platform-resolution, artifact-publishing)

One subsection per rewritten requirement, 28 in all. Every new body is built from whole sentences of the old body, copied unchanged (swarm DECISIONS.md 2026-10-19 15:50). A clause row says "verbatim" for a sentence that appears unchanged. A row that names a substitution lists each changed word pair; a lowered first letter after a copied condition is named as such. A list marker `- ` of an old bullet is dropped where its sentence opens a body or follows a sentence that does not end in a colon. Old scenario names are all kept; the byte check ran once per rewritten requirement and printed no `DIFF` and no `LOST`.

Form per requirement: the names below are cited by other requirements, so they keep their header as form b parents (MODIFIED, all old scenarios in place, split-offs ADDED with restatement scenarios): in `platform-resolution` "Acquisition mirrors the operator", "The cluster arm reads the effective registry", "Module commands render against the module's deps", "Platform source precedence by command", "Renders fall back to their own deps"; in `deploy` "Instance delete re-checks live ownership before each delete" and "Instance delete follows the shared deletion plan" (both cited by `operator-lifecycle`). The other 21 are form c2.

Exceptions to the one-sentence rule, all with every word copied from the old requirement:

- Four old sentences are longer than 500 characters and are cut inside the sentence, each part verbatim, the subject repeated: `artifact-publishing` "One pipeline, two entry points" (562), "Gates and refusal catalog" (769, the gate list is cut at its semicolons), "The compatibility gate" (531); `deploy` "Instance delete keeps PersistentVolumeClaims unless delete-data is set" (610). The supervisor is asked to rule on this in the report.
- A condition or subject copied from the same old requirement leads a body where the old sentence relied on a sibling (rows "the part from ... on"). Where the old content cannot fit, a body names a sibling by its exact header (the 16:55 extension): "These, as listed in requirement ...", "one of the checks listed in requirement ...", "no such field (the field of requirement ...)", "In every other case than the one of requirement ...".
- Cites of renamed requirements are updated to the new header: "Instance delete keeps PersistentVolumeClaims unless delete-data is set" and "Instance delete waits for the deleted resources when wait is set" in `deploy`, "Ownership guard judges every apply and dry run" (see the section above).
- A `Source:` line is repeated whole on each split where the old text does not tie it to a clause. Where it does not fit, the quote that supports the block stays and the other quote stays in the block that carries its rule ("Instance delete re-checks live ownership before each delete" and its split).

### Ledger: deploy, Instance delete never deletes CRDs or Namespaces

Form: c2 (REMOVED, ADDED under new names). New requirements: "Instance delete never deletes a Namespace or a CustomResourceDefinition", "Instance delete lists a protected resource as left behind", "A left-behind resource does not fail the delete".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Namespace tracked by the instance is left behind | "A left-behind resource does not fail the delete", "Namespace tracked by the instance is left behind" | No (byte equal, checked) |
| CRD tracked by the instance is left behind | "Instance delete lists a protected resource as left behind", "CRD tracked by the instance is left behind" | No (byte equal, checked) |
| Dry run reports left-behind resources | "Instance delete lists a protected resource as left behind", "Dry run reports left-behind resources" | No (byte equal, checked) |
| (new) | "Instance delete never deletes a Namespace or a CustomResourceDefinition", "No flag overrides the protection" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm instance delete` of a CLI-owned instance SHALL NOT delete a `Name... | body of "Instance delete never deletes a Namespace or a CustomResourceDefinition", verbatim; body of "Instance delete lists a protected resource as left behind", verbatim; body of "A left-behind resource does not fail the delete", verbatim |
| There SHALL be no flag that overrides this. | body of "Instance delete never deletes a Namespace or a CustomResourceDefinition", verbatim |
| Each such resource SHALL be listed as left behind, naming its kind, na... | body of "Instance delete lists a protected resource as left behind", verbatim |
| A left-behind resource SHALL NOT count as a failure: the remaining tra... | body of "A left-behind resource does not fail the delete", verbatim |
| The closing line SHALL state how many resources were left behind and h... | body of "A left-behind resource does not fail the delete", verbatim |

Lengths (old, new): old 699; new "Instance delete never deletes a Namespace or a CustomResourceDefinition" 257; "Instance delete lists a protected resource as left behind" 382; "A left-behind resource does not fail the delete" 486.

### Ledger: deploy, Instance delete re-checks live ownership before each delete

Form: b (MODIFIED parent with its header and all scenarios, ADDED splits). New requirements: "keep", "A tracked resource is deleted only when its live labels and annotation show the instance owns it", "An object with no UUID label is judged without the UUID comparison", "A resource that fails the ownership check is left behind and does not fail the delete", "Each delete carries a precondition on the UID of the object that was read", "A resource that is already gone is no failure and any other read error is".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Resource no longer managed by OPM is left behind | "Instance delete re-checks live ownership before each delete", "Resource no longer managed by OPM is left behind" | No (byte equal, checked) |
| Resource owned by another instance is left behind | "Instance delete re-checks live ownership before each delete", "Resource owned by another instance is left behind" | No (byte equal, checked) |
| Resource without a UUID label is deleted on managed-by alone | "Instance delete re-checks live ownership before each delete", "Resource without a UUID label is deleted on managed-by alone" | No (byte equal, checked) |
| Resource gone before its delete | "Instance delete re-checks live ownership before each delete", "Resource gone before its delete" | No (byte equal, checked) |
| Resource gone between its re-read and its delete | "Instance delete re-checks live ownership before each delete", "Resource gone between its re-read and its delete" | No (byte equal, checked) |
| Instance without a recorded UUID deletes on managed-by alone | "Instance delete re-checks live ownership before each delete", "Instance without a recorded UUID deletes on managed-by alone" | No (byte equal, checked) |
| Read error keeps the ModuleInstance | "Instance delete re-checks live ownership before each delete", "Read error keeps the ModuleInstance" | No (byte equal, checked) |
| Resource adopted by another instance is left behind | "Instance delete re-checks live ownership before each delete", "Resource adopted by another instance is left behind" | No (byte equal, checked) |
| Delete carries a UID precondition | "Instance delete re-checks live ownership before each delete", "Delete carries a UID precondition" | No (byte equal, checked) |
| Object replaced between its re-read and its delete | "Instance delete re-checks live ownership before each delete", "Object replaced between its re-read and its delete" | No (byte equal, checked) |
| (new) | "A tracked resource is deleted only when its live labels and annotation show the instance owns it", "Ownership conditions for the delete" | New restatement scenario: only words of the old clause, no new case |
| (new) | "An object with no UUID label is judged without the UUID comparison", "No UUID comparison without a UUID" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A resource that fails the ownership check is left behind and does not fail the delete", "A resource that fails the check is listed with the reason" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Each delete carries a precondition on the UID of the object that was read", "A refused precondition is a failure" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A resource that is already gone is no failure and any other read error is", "Already gone counts neither as deleted nor as a failure" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| Before deleting each tracked resource, `opm instance delete` SHALL rea... | body of "Instance delete re-checks live ownership before each delete", verbatim; body of "An object with no UUID label is judged without the UUID comparison", the part from "Before deleting each tracked r..." on, verbatim; body of "A resource that fails the ownership check is left behind and does not fail the delete", the part from "Before deleting each tracked r..." on, verbatim; body of "Each delete carries a precondition on the UID of the object that was read", the part from "Before deleting each tracked r..." on, verbatim; body of "A resource that is already gone is no failure and any other read error is", the part from "Before deleting each tracked r..." on, verbatim |
| It SHALL delete the resource only when the live `app.kubernetes.io/man... | body of "A tracked resource is deleted only when its live labels and annotation show the instance owns it", with "It SHALL delete the resource" reads "`opm instance delete` SHALL delete each tracked resource"; body of "An object with no UUID label is judged without the UUID comparison", with "It SHALL" reads "SHALL"; body of "A resource that fails the ownership check is left behind and does not fail the delete", with "It SHALL" reads "SHALL" |
| An object with no UUID label SHALL be judged without the UUID comparis... | body of "An object with no UUID label is judged without the UUID comparison", verbatim |
| A resource that fails the check SHALL be left behind and listed with t... | body of "A resource that fails the ownership check is left behind and does not fail the delete", verbatim |
| Each delete SHALL carry a precondition on the UID of the object that w... | body of "Each delete carries a precondition on the UID of the object that was read", verbatim |
| A resource that is already gone, whether its re-read or its delete cal... | body of "A resource that is already gone is no failure and any other read error is", verbatim |
| Any other read error SHALL count as a failure for that resource, so th... | body of "A resource that is already gone is no failure and any other read error is", verbatim |
| Source: enhancement 0012 ("The library owns the whole deletion sequenc... | body of "Instance delete re-checks live ownership before each delete", the part from "Source: enhancement 0012 (" on, verbatim; body of "A tracked resource is deleted only when its live labels and annotation show the instance owns it", the part from "Source: enhancement 0012 (" on, verbatim; body of "A tracked resource is deleted only when its live labels and annotation show the instance owns it", the part from ""The apply guard runs" on, verbatim |

Lengths (old, new): old 1645; new "Instance delete re-checks live ownership before each delete" 401; "A tracked resource is deleted only when its live labels and annotation show the instance owns it" 497; "An object with no UUID label is judged without the UUID comparison" 458; "A resource that fails the ownership check is left behind and does not fail the delete" 498; "Each delete carries a precondition on the UID of the object that was read" 307; "A resource that is already gone is no failure and any other read error is" 359.

### Ledger: deploy, Apply stages CustomResourceDefinitions and Namespaces before the rest

Form: c2 (REMOVED, ADDED under new names). New requirements: "The apply goes in two stages in ascending kind-class weight", "The first stage holds every CustomResourceDefinition and every Namespace", "After the first stage the apply waits until the CustomResourceDefinitions are established", "A wait that does not complete fails the apply", "A resource that fails to apply does not stop the rest".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A CRD and its custom resource apply in one run | "After the first stage the apply waits until the CustomResourceDefinitions are established", "A CRD and its custom resource apply in one run" | No (byte equal, checked) |
| A Namespace is applied before the objects in it | "The first stage holds every CustomResourceDefinition and every Namespace", "A Namespace is applied before the objects in it" | No (byte equal, checked) |
| The second stage follows weight order | "The apply goes in two stages in ascending kind-class weight", "The second stage follows weight order" | No (byte equal, checked) |
| A CRD that never becomes established fails the apply | "A wait that does not complete fails the apply", "A CRD that never becomes established fails the apply" | No (byte equal, checked) |
| (new) | "A resource that fails to apply does not stop the rest", "A failed resource is reported and the stages go on" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm instance apply` and `opm module apply` SHALL apply the rendered r... | body of "The apply goes in two stages in ascending kind-class weight", verbatim; body of "The first stage holds every CustomResourceDefinition and every Namespace", the part from "`opm instance apply` and `opm ..." on, verbatim; body of "A resource that fails to apply does not stop the rest", the part from "`opm instance apply` and `opm ..." on, verbatim |
| The first stage SHALL hold every `CustomResourceDefinition` (group `ap... | body of "The first stage holds every CustomResourceDefinition and every Namespace", verbatim |
| Outside a dry run, after the first stage the command SHALL wait until ... | body of "After the first stage the apply waits until the CustomResourceDefinitions are established", verbatim; body of "A wait that does not complete fails the apply", the part from "Outside a dry run, after the f..." on, verbatim |
| When the wait does not complete, the command SHALL fail naming the Cus... | body of "A wait that does not complete fails the apply", verbatim |
| A resource that fails to apply SHALL be reported and SHALL NOT stop th... | body of "A resource that fails to apply does not stop the rest", verbatim |

Lengths (old, new): old 1203; new "The apply goes in two stages in ascending kind-class weight" 375; "The first stage holds every CustomResourceDefinition and every Namespace" 265; "After the first stage the apply waits until the CustomResourceDefinitions are established" 345; "A wait that does not complete fails the apply" 366; "A resource that fails to apply does not stop the rest" 217.

### Ledger: deploy, A dry run skips a custom resource whose CRD the same apply creates

Form: c2 (REMOVED, ADDED under new names). New requirements: "A dry run skips a custom resource whose CRD the same apply creates and warns", "A dry run sends a custom resource whose CRD already exists", "A dry run does not wait for any CustomResourceDefinition", "A custom resource in a namespace the same apply creates is warned about once".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A dry run with a new CRD skips its custom resource | "A dry run skips a custom resource whose CRD the same apply creates and warns", "A dry run with a new CRD skips its custom resource" | No (byte equal, checked) |
| A dry run with an existing CRD sends its custom resource | "A dry run sends a custom resource whose CRD already exists", "A dry run with an existing CRD sends its custom resource" | No (byte equal, checked) |
| (new) | "A dry run does not wait for any CustomResourceDefinition", "A dry run does not wait" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A custom resource in a namespace the same apply creates is warned about once", "One warning for a custom resource in a new namespace" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| On `--dry-run`, `opm instance apply` and `opm module apply` SHALL NOT ... | body of "A dry run skips a custom resource whose CRD the same apply creates and warns", verbatim; body of "A dry run sends a custom resource whose CRD already exists", the part from "On `--dry-run`, `opm instance ..." on, verbatim; body of "A dry run does not wait for any CustomResourceDefinition", the part from "On `--dry-run`, `opm instance ..." on, verbatim; body of "A custom resource in a namespace the same apply creates is warned about once", the part from "On `--dry-run`, `opm instance ..." on, verbatim |
| The command SHALL log a warning naming the skipped resource and its Cu... | body of "A dry run skips a custom resource whose CRD the same apply creates and warns", verbatim |
| A dry run SHALL NOT wait for any CustomResourceDefinition. | body of "A dry run does not wait for any CustomResourceDefinition", first letter lowered after the condition, otherwise verbatim |
| A custom resource whose CustomResourceDefinition already exists on the... | body of "A dry run sends a custom resource whose CRD already exists", first letter lowered after the condition, otherwise verbatim |
| A custom resource that is also in a namespace the same apply creates S... | body of "A custom resource in a namespace the same apply creates is warned about once", first letter lowered after the condition, otherwise verbatim |

Lengths (old, new): old 686; new "A dry run skips a custom resource whose CRD the same apply creates and warns" 393; "A dry run sends a custom resource whose CRD already exists" 163; "A dry run does not wait for any CustomResourceDefinition" 119; "A custom resource in a namespace the same apply creates is warned about once" 191.

### Ledger: deploy, Instance delete of an instance that deploys the operator is guarded

Form: c2 (REMOVED, ADDED under new names). New requirements: "An instance deploys the operator when one of three signals holds", "Instance delete does not delete an instance that deploys the operator", "The guard runs before any object is deleted, on both branches and on a dry run", "An operator-owned target is refused with exit code 2", "The guard lists ModuleInstance resources cluster-wide when the target is not operator-owned", "An armed instance blocks the delete with exit code 2", "A failed cluster-wide list fails closed", "Instance delete offers no flag that removes finalizers", "The yes flag and its deprecated alias do not bypass either refusal", "Every other instance's delete is unchanged by the guard".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Refused while an instance waits on the operator's cleanup | "An armed instance blocks the delete with exit code 2", "Refused while an instance waits on the operator's cleanup" | No (byte equal, checked) |
| Operator-owned instance of the operator is refused | "An operator-owned target is refused with exit code 2", "Operator-owned instance of the operator is refused" | No (byte equal, checked) |
| Dry run reports the same refusal | "The guard runs before any object is deleted, on both branches and on a dry run", "Dry run reports the same refusal" | No (byte equal, checked) |
| The target itself carries the finalizer | "An armed instance blocks the delete with exit code 2", "The target itself carries the finalizer" | No (byte equal, checked) |
| The operator module under another name is guarded | "An instance deploys the operator when one of three signals holds", "The operator module under another name is guarded" | No (byte equal, checked) |
| A record holding the operator's CRDs is guarded | "An instance deploys the operator when one of three signals holds", "A record holding the operator's CRDs is guarded" | No (byte equal, checked) |
| Look-alike names are not the operator | "An instance deploys the operator when one of three signals holds", "Look-alike names are not the operator" | No (byte equal, checked) |
| No armed instance, no refusal | "An armed instance blocks the delete with exit code 2", "No armed instance, no refusal" | No (byte equal, checked) |
| List failure fails closed | "A failed cluster-wide list fails closed", "List failure fails closed" | No (byte equal, checked) |
| Other instances are not guarded | "Every other instance's delete is unchanged by the guard", "Other instances are not guarded" | No (byte equal, checked) |
| The deprecated alias does not bypass the guard | "The yes flag and its deprecated alias do not bypass either refusal", "The deprecated alias does not bypass the guard" | No (byte equal, checked) |
| (new) | "Instance delete does not delete an instance that deploys the operator", "The guard names the operator's instance" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The guard lists ModuleInstance resources cluster-wide when the target is not operator-owned", "The guard lists every ModuleInstance" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Instance delete offers no flag that removes finalizers", "No finalizer flag and no owner write" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm instance delete` SHALL NOT delete an instance that deploys the op... | body of "Instance delete does not delete an instance that deploys the operator", verbatim; body of "The yes flag and its deprecated alias do not bypass either refusal", verbatim |
| An instance deploys the operator when any one of these holds, read fro... | body of "An instance deploys the operator when one of three signals holds", with "any one of these holds" reads "any one of these SHALL hold" |
| These are the same signals by which the operator recognises the instan... | body of "Instance delete does not delete an instance that deploys the operator", with "These are the same signals" reads "These, as listed in requirement "An instance deploys the operator when one of three signals holds", are the same signals" |
| The guard SHALL run after the target instance is resolved and before a... | body of "The guard runs before any object is deleted, on both branches and on a dry run", verbatim |
| - When the target is operator-owned, the command SHALL refuse with exi... | body of "An operator-owned target is refused with exit code 2", verbatim |
| The operator never reconciles, finalizes or prunes the instance that d... | body of "An operator-owned target is refused with exit code 2", verbatim |
| - Otherwise the command SHALL list `ModuleInstance` resources cluster-... | body of "The guard lists ModuleInstance resources cluster-wide when the target is not operator-owned", with "Otherwise the command" reads "When the target is not operator-owned, the command"; body of "An armed instance blocks the delete with exit code 2", with "Otherwise the command" reads "When the target is not operator-owned, the command"; body of "A failed cluster-wide list fails closed", with "Otherwise the command" reads "When the target is not operator-owned, the command" |
| When any carries `opmodel.dev/cleanup`, the command SHALL refuse with ... | body of "An armed instance blocks the delete with exit code 2", verbatim |
| Deleting the operator while it still owes cleanup leaves every armed i... | body of "The guard lists ModuleInstance resources cluster-wide when the target is not operator-owned", verbatim |
| When the cluster-wide list fails, the command SHALL fail closed with t... | body of "A failed cluster-wide list fails closed", verbatim |
| `opm instance delete` SHALL offer no flag that removes finalizers and ... | body of "Instance delete offers no flag that removes finalizers", verbatim |
| `--yes` SHALL keep its one meaning, skipping the confirmation prompt, ... | body of "The yes flag and its deprecated alias do not bypass either refusal", verbatim |
| Every other instance's delete SHALL be unchanged by this requirement. | body of "Every other instance's delete is unchanged by the guard", verbatim |

Lengths (old, new): old 2453; new "An instance deploys the operator when one of three signals holds" 416; "Instance delete does not delete an instance that deploys the operator" 451; "The guard runs before any object is deleted, on both branches and on a dry run" 201; "An operator-owned target is refused with exit code 2" 355; "The guard lists ModuleInstance resources cluster-wide when the target is not operator-owned" 236; "An armed instance blocks the delete with exit code 2" 459; "A failed cluster-wide list fails closed" 287; "Instance delete offers no flag that removes finalizers" 99; "The yes flag and its deprecated alias do not bypass either refusal" 408; "Every other instance's delete is unchanged by the guard" 69.

### Ledger: deploy, Instance delete keeps the ModuleInstance when a tracked resource could not be read

Form: c2 (REMOVED, ADDED under new names). New requirements: "An unreadable tracked resource counts as a failure and keeps the ModuleInstance", "The output says the ModuleInstance was kept and a dry run counts the resource as not checked", "An unreadable Namespace or CustomResourceDefinition is left behind and is no failure", "The operator-owned delete path is not affected by discovery read errors".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Discovery read error keeps the ModuleInstance | "An unreadable tracked resource counts as a failure and keeps the ModuleInstance", "Discovery read error keeps the ModuleInstance" | No (byte equal, checked) |
| Dry run counts an unreadable resource as not checked | "The output says the ModuleInstance was kept and a dry run counts the resource as not checked", "Dry run counts an unreadable resource as not checked" | No (byte equal, checked) |
| Unreadable Namespace is left behind, not a failure | "An unreadable Namespace or CustomResourceDefinition is left behind and is no failure", "Unreadable Namespace is left behind, not a failure" | No (byte equal, checked) |
| Operator-owned delete ignores discovery read errors | "The operator-owned delete path is not affected by discovery read errors", "Operator-owned delete ignores discovery read errors" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| When `opm instance delete` deletes a CLI-owned instance and a tracked ... | body of "An unreadable tracked resource counts as a failure and keeps the ModuleInstance", verbatim; body of "The output says the ModuleInstance was kept and a dry run counts the resource as not checked", the part from "When `opm instance delete` del..." on, verbatim; body of "An unreadable Namespace or CustomResourceDefinition is left behind and is no failure", the part from "When `opm instance delete` del..." on, verbatim; body of "The operator-owned delete path is not affected by discovery read errors", verbatim |
| On a real run the output SHALL say that the `ModuleInstance` was kept ... | body of "The output says the ModuleInstance was kept and a dry run counts the resource as not checked", first letter lowered after the condition, otherwise verbatim |
| A dry run SHALL count the resource among those it could not check. | body of "The output says the ModuleInstance was kept and a dry run counts the resource as not checked", verbatim |
| A tracked `Namespace` in the core group or `CustomResourceDefinition` ... | body of "An unreadable Namespace or CustomResourceDefinition is left behind and is no failure", first letter lowered after the condition, otherwise verbatim |
| The operator-owned delete path SHALL NOT be affected. | body of "The operator-owned delete path is not affected by discovery read errors", verbatim |

Lengths (old, new): old 880; new "An unreadable tracked resource counts as a failure and keeps the ModuleInstance" 441; "The output says the ModuleInstance was kept and a dry run counts the resource as not checked" 332; "An unreadable Namespace or CustomResourceDefinition is left behind and is no failure" 385; "The operator-owned delete path is not affected by discovery read errors" 495.

### Ledger: deploy, A dry run skips a namespaced object whose namespace the same apply creates

Form: c2 (REMOVED, ADDED under new names). New requirements: "A dry run skips a namespaced object whose namespace the apply would create", "A skipped object is warned about, counted and no error", "A dry run sends every other object as usual".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A dry run with a new Namespace skips the objects in it | "A dry run skips a namespaced object whose namespace the apply would create", "A dry run with a new Namespace skips the objects in it" | No (byte equal, checked) |
| A dry run with --create-namespace skips the objects in the new instance namespace | "A dry run skips a namespaced object whose namespace the apply would create", "A dry run with --create-namespace skips the objects in the new instance namespace" | No (byte equal, checked) |
| A dry run in an existing namespace sends everything | "A dry run sends every other object as usual", "A dry run in an existing namespace sends everything" | No (byte equal, checked) |
| A real apply skips nothing | "A dry run sends every other object as usual", "A real apply skips nothing" | No (byte equal, checked) |
| (new) | "A skipped object is warned about, counted and no error", "A skipped object gets a warning and a count" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| On `--dry-run`, `opm instance apply` and `opm module apply` SHALL NOT ... | body of "A dry run skips a namespaced object whose namespace the apply would create", verbatim; body of "A skipped object is warned about, counted and no error", the part from "On `--dry-run`, `opm instance ..." on, verbatim; body of "A dry run sends every other object as usual", the part from "On `--dry-run`, `opm instance ..." on, verbatim |
| A dry run persists neither, so the server's namespace admission would ... | body of "A dry run skips a namespaced object whose namespace the apply would create", verbatim |
| The command SHALL log a warning naming the skipped object and its name... | body of "A skipped object is warned about, counted and no error", verbatim |
| A namespaced object in a namespace that already exists, and every clus... | body of "A dry run sends every other object as usual", verbatim |
| Without `--create-namespace`, a missing instance namespace that no ren... | body of "A dry run sends every other object as usual", verbatim |
| Outside a dry run nothing is skipped. | body of "A dry run sends every other object as usual", verbatim |

Lengths (old, new): old 996; new "A dry run skips a namespaced object whose namespace the apply would create" 423; "A skipped object is warned about, counted and no error" 380; "A dry run sends every other object as usual" 461.

### Ledger: deploy, Create-namespace creates the namespace only after every check passed

Form: c2 (REMOVED, ADDED under new names). New requirements: "Create-namespace creates a missing namespace only after every check that can refuse the apply passed", "A check that refuses the apply leaves the cluster unchanged and creates no namespace", "While the namespace is missing the apply sends no read into it", "The status-permission check runs unchanged with a missing namespace", "A failed read or create of the namespace stops the apply with the exit code of its cause", "A namespace that exists at the create stops the apply with exit code 1", "A dry run creates no namespace and reports that it would".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A refused first install creates no namespace | "A check that refuses the apply leaves the cluster unchanged and creates no namespace", "A refused first install creates no namespace" | No (byte equal, checked) |
| An untracked cluster-scoped resource refuses before the namespace is created | "A check that refuses the apply leaves the cluster unchanged and creates no namespace", "An untracked cluster-scoped resource refuses before the namespace is created" | No (byte equal, checked) |
| A failed cluster gate creates no namespace | "A check that refuses the apply leaves the cluster unchanged and creates no namespace", "A failed cluster gate creates no namespace" | No (byte equal, checked) |
| A denied status permission creates no namespace | "A check that refuses the apply leaves the cluster unchanged and creates no namespace", "A denied status permission creates no namespace" | No (byte equal, checked) |
| A missing namespace is a first install and is not read | "While the namespace is missing the apply sends no read into it", "A missing namespace is a first install and is not read" | No (byte equal, checked) |
| A caller without cluster-wide read is not refused by the missing namespace | "While the namespace is missing the apply sends no read into it", "A caller without cluster-wide read is not refused by the missing namespace" | No (byte equal, checked) |
| An existing namespace is left alone | "Create-namespace creates a missing namespace only after every check that can refuse the apply passed", "An existing namespace is left alone" | No (byte equal, checked) |
| The namespace read fails | "A failed read or create of the namespace stops the apply with the exit code of its cause", "The namespace read fails" | No (byte equal, checked) |
| The namespace create fails | "A failed read or create of the namespace stops the apply with the exit code of its cause", "The namespace create fails" | No (byte equal, checked) |
| The namespace appears during the checks | "A namespace that exists at the create stops the apply with exit code 1", "The namespace appears during the checks" | No (byte equal, checked) |
| (new) | "The status-permission check runs unchanged with a missing namespace", "The status-permission check asks about the namespace by name" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A dry run creates no namespace and reports that it would", "A dry run reports first that the namespace would be created" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| With `--create-namespace`, `opm instance apply` and `opm module apply`... | body of "Create-namespace creates a missing namespace only after every check that can refuse the apply passed", with "(capability `apply-pruning`, "Ownership guard judges every apply and dry run")" reads "(capability `apply-pruning`, "The ownership guard judges every apply")"; body of "The status-permission check runs unchanged with a missing namespace", the part from "With `--create-namespace`, `op..." on, verbatim; body of "A failed read or create of the namespace stops the apply with the exit code of its cause", the part from "With `--create-namespace`, `op..." on, verbatim; body of "A namespace that exists at the create stops the apply with exit code 1", the part from "With `--create-namespace`, `op..." on, verbatim; body of "A dry run creates no namespace and reports that it would", the part from "With `--create-namespace`, `op..." on, verbatim |
| The namespace SHALL be created before the first rendered resource is a... | body of "Create-namespace creates a missing namespace only after every check that can refuse the apply passed", verbatim |
| An apply that one of these checks refuses SHALL NOT create the namespa... | body of "A check that refuses the apply leaves the cluster unchanged and creates no namespace", with "one of these checks" reads "one of the checks listed in requirement "Create-namespace creates a missing namespace only after every check that can refuse the apply passed"" |
| While the namespace is missing, the apply SHALL treat it as holding no... | body of "While the namespace is missing the apply sends no read into it", verbatim |
| Rendered cluster-scoped objects and objects in other namespaces SHALL ... | body of "While the namespace is missing the apply sends no read into it", verbatim |
| The status-permission check SHALL run unchanged; it asks about the nam... | body of "The status-permission check runs unchanged with a missing namespace", verbatim |
| A failure of the first read (whether the namespace exists) SHALL stop ... | body of "A failed read or create of the namespace stops the apply with the exit code of its cause", verbatim |
| A failure of the create SHALL stop the apply with the exit code of its... | body of "A failed read or create of the namespace stops the apply with the exit code of its cause", verbatim |
| When the namespace turns out to exist at the create although the first... | body of "A namespace that exists at the create stops the apply with exit code 1", verbatim |
| A dry run SHALL create nothing and SHALL report first that the namespa... | body of "A dry run creates no namespace and reports that it would", verbatim |
| (note) | "Create-namespace creates a missing namespace only after every check that can refuse the apply passed": The cite of the removed name of apply-pruning is updated as DECISIONS.md 2026-10-19 15:05 rules. |

Lengths (old, new): old 1663; new "Create-namespace creates a missing namespace only after every check that can refuse the apply passed" 472; "A check that refuses the apply leaves the cluster unchanged and creates no namespace" 234; "While the namespace is missing the apply sends no read into it" 357; "The status-permission check runs unchanged with a missing namespace" 242; "A failed read or create of the namespace stops the apply with the exit code of its cause" 381; "A namespace that exists at the create stops the apply with exit code 1" 377; "A dry run creates no namespace and reports that it would" 217.

### Ledger: deploy, Instance delete fails when the ModuleInstance record cannot be deleted

Form: c2 (REMOVED, ADDED under new names). New requirements: "A failed record delete prints no success line and exits non-zero", "A failed record delete exits by the answer of the API server", "A re-run after the cause is fixed deletes the record", "A dry run is not affected by a failed record delete".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Forbidden record delete | "A failed record delete prints no success line and exits non-zero", "Forbidden record delete" | No (byte equal, checked) |
| Record delete fails with a server error | "A failed record delete exits by the answer of the API server", "Record delete fails with a server error" | No (byte equal, checked) |
| Re-run after the cause is fixed | "A re-run after the cause is fixed deletes the record", "Re-run after the cause is fixed" | No (byte equal, checked) |
| (new) | "A dry run is not affected by a failed record delete", "A dry run deletes no record" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| When `opm instance delete` deletes a CLI-owned instance, every tracked... | body of "A failed record delete prints no success line and exits non-zero", verbatim; body of "A failed record delete exits by the answer of the API server", the part from "When `opm instance delete` del..." on, verbatim; body of "A re-run after the cause is fixed deletes the record", the part from "When `opm instance delete` del..." on, verbatim; body of "A dry run is not affected by a failed record delete", verbatim |
| The output SHALL name the record and the error, SHALL say that the tra... | body of "A failed record delete prints no success line and exits non-zero", verbatim |
| The exit code SHALL be 4 when the API server denied the delete (Forbid... | body of "A failed record delete exits by the answer of the API server", first letter lowered after the condition, otherwise verbatim |
| A NotFound answer SHALL count as deleted. | body of "A failed record delete exits by the answer of the API server", verbatim |
| Re-running the command after the cause is fixed SHALL delete the recor... | body of "A re-run after the cause is fixed deletes the record", first letter lowered after the condition, otherwise verbatim |
| A dry run deletes no record and SHALL NOT be affected. | body of "A dry run is not affected by a failed record delete", verbatim |

Lengths (old, new): old 785; new "A failed record delete prints no success line and exits non-zero" 418; "A failed record delete exits by the answer of the API server" 412; "A re-run after the cause is fixed deletes the record" 268; "A dry run is not affected by a failed record delete" 307.

### Ledger: deploy, Instance delete keeps PersistentVolumeClaims unless delete-data is set

Form: c2 (REMOVED, ADDED under new names). New requirements: "Instance delete keeps a tracked PersistentVolumeClaim unless `--delete-data` is set", "Each kept claim is listed on its own line at the informational level", "A kept claim does not count as a failure", "The closing output counts the kept claims and names the command that deletes them", "With --delete-data a tracked claim is deleted under the live-ownership check", "The kept-claim rule covers the core group only and no StatefulSet template claim", "--delete-data on an operator-managed instance warns and changes nothing", "The kept-claim rule of the CLI is not claimed for an operator-managed instance", "An operator-managed delete that may have kept claims does not claim a full prune", "The closing output of an operator-managed delete that may have kept claims names them", "Every other operator-managed delete closes as a prune of the tracked resources".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Claim is kept by default | "Instance delete keeps a tracked PersistentVolumeClaim unless `--delete-data` is set", "Claim is kept by default" | No (byte equal, checked) |
| Delete-data deletes the claim | "With --delete-data a tracked claim is deleted under the live-ownership check", "Delete-data deletes the claim" | No (byte equal, checked) |
| Dry run lists the claim as kept | "Each kept claim is listed on its own line at the informational level", "Dry run lists the claim as kept" | No (byte equal, checked) |
| Unreadable claim is kept | "A kept claim does not count as a failure", "Unreadable claim is kept" | No (byte equal, checked) |
| Operator-managed instance | "--delete-data on an operator-managed instance warns and changes nothing", "Operator-managed instance" | No (byte equal, checked) |
| Operator-managed delete that may have kept claims closes without claiming either outcome | "The closing output of an operator-managed delete that may have kept claims names them", "Operator-managed delete that may have kept claims closes without claiming either outcome" | No (byte equal, checked) |
| Operator-managed delete with the Delete policy closes as a full prune | "Every other operator-managed delete closes as a prune of the tracked resources", "Operator-managed delete with the Delete policy closes as a full prune" | No (byte equal, checked) |
| Operator-managed delete on an operator without the field closes as a full prune | "Every other operator-managed delete closes as a prune of the tracked resources", "Operator-managed delete on an operator without the field closes as a full prune" | No (byte equal, checked) |
| (new) | "The closing output counts the kept claims and names the command that deletes them", "The closing output names the claims kept" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The kept-claim rule covers the core group only and no StatefulSet template claim", "A claim of another group is deleted like any other resource" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The kept-claim rule of the CLI is not claimed for an operator-managed instance", "The operator decides what it removes" | New restatement scenario: only words of the old clause, no new case |
| (new) | "An operator-managed delete that may have kept claims does not claim a full prune", "No claim of a full prune and no claim of claims in the cluster" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm instance delete` of a CLI-owned instance SHALL NOT delete a track... | body of "Instance delete keeps a tracked PersistentVolumeClaim unless `--delete-data` is set", verbatim; body of "Each kept claim is listed on its own line at the informational level", verbatim; body of "A kept claim does not count as a failure", verbatim; body of "The closing output counts the kept claims and names the command that deletes them", verbatim; body of "With --delete-data a tracked claim is deleted under the live-ownership check", verbatim; body of "The kept-claim rule covers the core group only and no StatefulSet template claim", verbatim; body of "The kept-claim rule of the CLI is not claimed for an operator-managed instance", verbatim |
| Each kept claim SHALL be listed on its own line, naming its kind, name... | body of "Each kept claim is listed on its own line at the informational level", verbatim |
| A kept claim SHALL NOT count as a failure: the other tracked resources... | body of "A kept claim does not count as a failure", verbatim |
| After the delete the claim is in no inventory, so no later `opm` comma... | body of "Instance delete keeps a tracked PersistentVolumeClaim unless `--delete-data` is set", verbatim |
| The closing output SHALL state how many claims were kept, SHALL print ... | body of "The closing output counts the kept claims and names the command that deletes them", verbatim |
| With `--delete-data`, a tracked claim SHALL be deleted under the same ... | body of "With --delete-data a tracked claim is deleted under the live-ownership check", verbatim |
| The kept-claim rule SHALL apply to the core group only: a kind of the ... | body of "The kept-claim rule covers the core group only and no StatefulSet template claim", verbatim |
| The command SHALL NOT handle claims that a StatefulSet created from `v... | body of "The kept-claim rule covers the core group only and no StatefulSet template claim", verbatim |
| On an operator-managed instance `--delete-data` SHALL NOT change what ... | body of "--delete-data on an operator-managed instance warns and changes nothing", verbatim |
| The kept-claim rule of the CLI SHALL NOT be claimed for an operator-ma... | body of "The kept-claim rule of the CLI is not claimed for an operator-managed instance", verbatim |
| When the delete of an operator-managed instance with `spec.prune` set ... | body of "An operator-managed delete that may have kept claims does not claim a full prune", the part from "When the delete of an operator..." on, verbatim; body of "An operator-managed delete that may have kept claims does not claim a full prune", the part from "the closing output SHALL NOT s..." on, verbatim; body of "The closing output of an operator-managed delete that may have kept claims names them", the part from "When the delete of an operator..." on, verbatim; body of "The closing output of an operator-managed delete that may have kept claims names them", the part from "it SHALL name each such claim" on, verbatim; "it SHALL name" reads "the closing output SHALL name" |
| In every other case the closing output SHALL report the prune of the t... | body of "Every other operator-managed delete closes as a prune of the tracked resources", with "In every other case" reads "In every other case than the one of requirement "An operator-managed delete that may have kept claims does not claim a full prune"" |

Lengths (old, new): old 2409; new "Instance delete keeps a tracked PersistentVolumeClaim unless `--delete-data` is set" 234; "Each kept claim is listed on its own line at the informational level" 361; "A kept claim does not count as a failure" 319; "The closing output counts the kept claims and names the command that deletes them" 321; "With --delete-data a tracked claim is deleted under the live-ownership check" 273; "The kept-claim rule covers the core group only and no StatefulSet template claim" 435; "--delete-data on an operator-managed instance warns and changes nothing" 377; "The kept-claim rule of the CLI is not claimed for an operator-managed instance" 273; "An operator-managed delete that may have kept claims does not claim a full prune" 404; "The closing output of an operator-managed delete that may have kept claims names them" 494; "Every other operator-managed delete closes as a prune of the tracked resources" 208.

### Ledger: deploy, Instance delete confirmation names the claims that delete-data deletes

Form: c2 (REMOVED, ADDED under new names). New requirements: "Instance delete reads the instance record before it asks for confirmation", "The prompt names the claims that delete-data deletes", "Without delete-data the prompt says PersistentVolumeClaims are kept", "The prompt for an operator-managed instance says so and says what the operator does", "The prompt for an operator-managed instance without prune says the operator leaves the tracked resources running", "The prompt for an operator-managed instance with prune set and no claim says nothing about claims", "With prune set and a tracked claim the prompt says one of four things decided by the CRD", "The prompt says the operator deletes the claims and their data when the CRD has no policy field or the policy is Delete", "The prompt says the operator keeps the claims when the CRD has the field and the policy is Keep, absent or another value", "The prompt does not say whether claims are kept when the CRD has no readable schema", "The prompt shows the value of spec.dataPolicy as it is", "Every check that can refuse the delete runs before the prompt".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Prompt lists the claims | "The prompt names the claims that delete-data deletes", "Prompt lists the claims" | No (byte equal, checked) |
| Prompt without delete-data | "Without delete-data the prompt says PersistentVolumeClaims are kept", "Prompt without delete-data" | No (byte equal, checked) |
| Prompt for an operator-managed instance with prune set | "The prompt says the operator deletes the claims and their data when the CRD has no policy field or the policy is Delete", "Prompt for an operator-managed instance with prune set" | No (byte equal, checked) |
| Prompt on an operator without the field | "The prompt says the operator deletes the claims and their data when the CRD has no policy field or the policy is Delete", "Prompt on an operator without the field" | No (byte equal, checked) |
| Prompt for an operator-managed instance with prune set and no data policy | "The prompt says the operator keeps the claims when the CRD has the field and the policy is Keep, absent or another value", "Prompt for an operator-managed instance with prune set and no data policy" | No (byte equal, checked) |
| Prompt when the CRD cannot be read | "The prompt does not say whether claims are kept when the CRD has no readable schema", "Prompt when the CRD cannot be read" | No (byte equal, checked) |
| Refusal comes before the question | "Every check that can refuse the delete runs before the prompt", "Refusal comes before the question" | No (byte equal, checked) |
| Prompt for an unknown data policy | "The prompt shows the value of spec.dataPolicy as it is", "Prompt for an unknown data policy" | No (byte equal, checked) |
| Prompt for an operator-managed instance without prune | "The prompt for an operator-managed instance without prune says the operator leaves the tracked resources running", "Prompt for an operator-managed instance without prune" | No (byte equal, checked) |
| Prompt for an instance that tracks no claim | "The prompt for an operator-managed instance with prune set and no claim says nothing about claims", "Prompt for an instance that tracks no claim" | No (byte equal, checked) |
| Delete-data note comes before the question | "The prompt for an operator-managed instance says so and says what the operator does", "Delete-data note comes before the question" | No (byte equal, checked) |
| Missing instance is reported before the prompt | "Instance delete reads the instance record before it asks for confirmation", "Missing instance is reported before the prompt" | No (byte equal, checked) |
| (new) | "With prune set and a tracked claim the prompt says one of four things decided by the CRD", "Same outcome in a dry run and with --yes" | New restatement scenario: only words of the old clause, no new case |
| (new) | "With prune set and a tracked claim the prompt says one of four things decided by the CRD", "The prompt depends on the CRD" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm instance delete` SHALL read the instance record before it asks fo... | body of "Instance delete reads the instance record before it asks for confirmation", verbatim |
| When `--delete-data` is set, the instance is CLI-owned and its invento... | body of "The prompt names the claims that delete-data deletes", verbatim |
| Without `--delete-data` the prompt for a CLI-owned instance SHALL say ... | body of "Without delete-data the prompt says PersistentVolumeClaims are kept", verbatim |
| For an operator-managed instance the prompt SHALL be the same with or ... | body of "The prompt for an operator-managed instance says so and says what the operator does", verbatim; body of "The prompt for an operator-managed instance without prune says the operator leaves the tracked resources running", verbatim; body of "The prompt for an operator-managed instance with prune set and no claim says nothing about claims", verbatim |
| When `spec.prune` is not set, it SHALL say that the operator leaves th... | body of "The prompt for an operator-managed instance without prune says the operator leaves the tracked resources running", verbatim |
| When `spec.prune` is set and the inventory tracks no PersistentVolumeC... | body of "The prompt for an operator-managed instance with prune set and no claim says nothing about claims", verbatim |
| When `spec.prune` is set and the inventory tracks at least one Persist... | body of "With prune set and a tracked claim the prompt says one of four things decided by the CRD", verbatim; body of "The prompt says the operator keeps the claims when the CRD has the field and the policy is Keep, absent or another value", the part from "When `spec.prune` is set and t..." on, verbatim; body of "The prompt does not say whether claims are kept when the CRD has no readable schema", the part from "When `spec.prune` is set and t..." on, verbatim |
| The command SHALL take that from the CRD its operator-readiness check ... | body of "With prune set and a tracked claim the prompt says one of four things decided by the CRD", verbatim |
| When the CRD has no such field, the operator does not know it: the pro... | body of "The prompt says the operator deletes the claims and their data when the CRD has no policy field or the policy is Delete", with "no such field" reads "no such field (the field of requirement "With prune set and a tracked claim the prompt says one of four things decided by the CRD")" |
| When `spec.dataPolicy` is `Delete`, it SHALL say the same and name the... | body of "The prompt says the operator deletes the claims and their data when the CRD has no policy field or the policy is Delete", verbatim |
| When the CRD has the field and `spec.dataPolicy` is `Keep`, absent, or... | body of "The prompt says the operator keeps the claims when the CRD has the field and the policy is Keep, absent or another value", with "it SHALL say that the operator deletes" reads "the prompt SHALL say that the operator deletes" |
| When the CRD has no schema that can be read and `spec.dataPolicy` is n... | body of "The prompt does not say whether claims are kept when the CRD has no readable schema", with "it SHALL NOT say that claims" reads "the prompt SHALL NOT say that claims" |
| The prompt SHALL show the value of `spec.dataPolicy` as it is, or say ... | body of "The prompt shows the value of spec.dataPolicy as it is", verbatim |
| The dry run, and the run with `--yes`, SHALL state the same outcome fo... | body of "With prune set and a tracked claim the prompt says one of four things decided by the CRD", verbatim |
| An instance that has no record SHALL be reported as not found before a... | body of "Instance delete reads the instance record before it asks for confirmation", verbatim |
| Every check that can refuse the delete SHALL run before the prompt: th... | body of "Every check that can refuse the delete runs before the prompt", verbatim |
| A refused delete SHALL NOT ask for confirmation. | body of "Every check that can refuse the delete runs before the prompt", verbatim |

Lengths (old, new): old 2802; new "Instance delete reads the instance record before it asks for confirmation" 166; "The prompt names the claims that delete-data deletes" 242; "Without delete-data the prompt says PersistentVolumeClaims are kept" 107; "The prompt for an operator-managed instance says so and says what the operator does" 174; "The prompt for an operator-managed instance without prune says the operator leaves the tracked resources running" 273; "The prompt for an operator-managed instance with prune set and no claim says nothing about claims" 363; "With prune set and a tracked claim the prompt says one of four things decided by the CRD" 440; "The prompt says the operator deletes the claims and their data when the CRD has no policy field or the policy is Delete" 424; "The prompt says the operator keeps the claims when the CRD has the field and the policy is Keep, absent or another value" 451; "The prompt does not say whether claims are kept when the CRD has no readable schema" 422; "The prompt shows the value of spec.dataPolicy as it is" 179; "Every check that can refuse the delete runs before the prompt" 288.

### Ledger: deploy, Instance delete follows the shared deletion plan

Form: b (MODIFIED parent with its header and all scenarios, ADDED splits). New requirements: "keep", "A recorded resource that the command did not find is not in the deletion plan", "The command acts on a tracked resource only as the deletion plan names", "The deletion plan does not hold a PersistentVolumeClaim that the command keeps", "The ModuleInstance record is deleted only when the plan's release verdict allows it", "A dry run follows the same plan and deletes nothing", "The output lines and exit codes stay as the other requirements state them".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Resources are deleted in the plan's order | "Instance delete follows the shared deletion plan", "Resources are deleted in the plan's order" | No (byte equal, checked) |
| Delete carries what the plan names | "Instance delete follows the shared deletion plan", "Delete carries what the plan names" | No (byte equal, checked) |
| Failed resource holds the record | "Instance delete follows the shared deletion plan", "Failed resource holds the record" | No (byte equal, checked) |
| Left-behind resources do not hold the record | "Instance delete follows the shared deletion plan", "Left-behind resources do not hold the record" | No (byte equal, checked) |
| Kept claim is outside the plan | "Instance delete follows the shared deletion plan", "Kept claim is outside the plan" | No (byte equal, checked) |
| Resource that appears after the instance was read | "Instance delete follows the shared deletion plan", "Resource that appears after the instance was read" | No (byte equal, checked) |
| Dry run follows the plan without deleting | "Instance delete follows the shared deletion plan", "Dry run follows the plan without deleting" | No (byte equal, checked) |
| (new) | "A recorded resource that the command did not find is not in the deletion plan", "A resource not found is not in the plan" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The command acts on a tracked resource only as the deletion plan names", "Actions and delete options come from the plan" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The deletion plan does not hold a PersistentVolumeClaim that the command keeps", "A kept claim is outside the plan" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The ModuleInstance record is deleted only when the plan's release verdict allows it", "The release verdict holds the record" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A dry run follows the same plan and deletes nothing", "A dry run sends no delete" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The output lines and exit codes stay as the other requirements state them", "Output and exit codes are unchanged" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm instance delete` of a CLI-owned instance SHALL take every action ... | body of "Instance delete follows the shared deletion plan", verbatim; body of "A recorded resource that the command did not find is not in the deletion plan", verbatim; body of "The command acts on a tracked resource only as the deletion plan names", the part from "`opm instance delete` of a CLI..." on, verbatim; body of "The deletion plan does not hold a PersistentVolumeClaim that the command keeps", the part from "`opm instance delete` of a CLI..." on, verbatim; body of "The ModuleInstance record is deleted only when the plan's release verdict allows it", the part from "`opm instance delete` of a CLI..." on, verbatim; body of "A dry run follows the same plan and deletes nothing", the part from "`opm instance delete` of a CLI..." on, verbatim; body of "The output lines and exit codes stay as the other requirements state them", the part from "`opm instance delete` of a CLI..." on, verbatim |
| A recorded resource that the command did not find when it read the ins... | body of "A recorded resource that the command did not find is not in the deletion plan", verbatim |
| It SHALL read a tracked resource, delete it or leave it in place only ... | body of "The command acts on a tracked resource only as the deletion plan names", verbatim |
| The plan SHALL NOT hold a `PersistentVolumeClaim` that the command kee... | body of "The deletion plan does not hold a PersistentVolumeClaim that the command keeps", with "("Instance delete keeps PersistentVolumeClaims unless delete-data is set")" reads "("Instance delete keeps a tracked PersistentVolumeClaim unless `--delete-data` is set")" |
| The command SHALL delete the `ModuleInstance` record only when the pla... | body of "The ModuleInstance record is deleted only when the plan's release verdict allows it", verbatim |
| A dry run SHALL follow the same plan, SHALL send no delete, and SHALL ... | body of "A dry run follows the same plan and deletes nothing", verbatim |
| The output lines, their wording and the exit codes of the command SHAL... | body of "The output lines and exit codes stay as the other requirements state them", verbatim |
| The operator-owned delete path SHALL NOT be affected. | body of "The output lines and exit codes stay as the other requirements state them", verbatim |
| Source: enhancement 0012 ("The library owns the whole deletion sequenc... | body of "Instance delete follows the shared deletion plan", verbatim |

Lengths (old, new): old 1424; new "Instance delete follows the shared deletion plan" 456; "A recorded resource that the command did not find is not in the deletion plan" 465; "The command acts on a tracked resource only as the deletion plan names" 362; "The deletion plan does not hold a PersistentVolumeClaim that the command keeps" 310; "The ModuleInstance record is deleted only when the plan's release verdict allows it" 332; "A dry run follows the same plan and deletes nothing" 241; "The output lines and exit codes stay as the other requirements state them" 336.

### Ledger: deploy, Instance delete reports a resource as deleted when its delete is accepted

Form: c2 (REMOVED, ADDED under new names). New requirements: "Instance delete reports a resource as deleted when the API server accepts its delete", "Without --wait an accepted delete is no failure although the resource still exists", "With --wait the command also waits for the deleted resources to disappear".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Resource held by a finalizer | "Without --wait an accepted delete is no failure although the resource still exists", "Resource held by a finalizer" | No (byte equal, checked) |
| Accepted delete of a resource with dependents | "Instance delete reports a resource as deleted when the API server accepts its delete", "Accepted delete of a resource with dependents" | No (byte equal, checked) |
| (new) | "With --wait the command also waits for the deleted resources to disappear", "The wait is bounded" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm instance delete` of a CLI-owned instance SHALL report a tracked r... | body of "Instance delete reports a resource as deleted when the API server accepts its delete", verbatim |
| Each delete uses Foreground propagation, so a deleted resource can sti... | body of "Instance delete reports a resource as deleted when the API server accepts its delete", verbatim |
| Without `--wait`, a resource whose delete was accepted SHALL NOT count... | body of "Without --wait an accepted delete is no failure although the resource still exists", verbatim |
| This requirement describes when a resource counts as deleted. | body of "Instance delete reports a resource as deleted when the API server accepts its delete", verbatim |
| With `--wait` the command also waits, bounded, for the deleted resourc... | body of "With --wait the command also waits for the deleted resources to disappear", with "the command also waits" reads "the command SHALL also wait"; "("Instance delete waits for the deleted resources when wait is set")" reads "("Instance delete with `--wait` waits for the deleted resources to disappear")" |

Lengths (old, new): old 784; new "Instance delete reports a resource as deleted when the API server accepts its delete" 402; "Without --wait an accepted delete is no failure although the resource still exists" 225; "With --wait the command also waits for the deleted resources to disappear" 170.

### Ledger: deploy, Instance delete waits for the deleted resources when wait is set

Form: c2 (REMOVED, ADDED under new names). New requirements: "Instance delete accepts --wait and without it nothing changes", "Instance delete with `--wait` waits for the deleted resources to disappear", "A resource is gone when its read returns NotFound or its UID differs", "While it waits the command prints how many deleted resources are still terminating", "The wait skips a resource the command did not delete", "A timeout lists the resources still terminating and keeps the ModuleInstance record", "--wait has no effect on a dry run, after a failed delete or on an operator-managed instance".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Deleted resources go away within the timeout | "Instance delete with `--wait` waits for the deleted resources to disappear", "Deleted resources go away within the timeout" | No (byte equal, checked) |
| Resource held by a finalizer times out | "A timeout lists the resources still terminating and keeps the ModuleInstance record", "Resource held by a finalizer times out" | No (byte equal, checked) |
| Name taken by a new object | "A resource is gone when its read returns NotFound or its UID differs", "Name taken by a new object" | No (byte equal, checked) |
| Kept claim is not waited for | "The wait skips a resource the command did not delete", "Kept claim is not waited for" | No (byte equal, checked) |
| Resource left behind is not waited for | "The wait skips a resource the command did not delete", "Resource left behind is not waited for" | No (byte equal, checked) |
| Re-run after a timeout | "A timeout lists the resources still terminating and keeps the ModuleInstance record", "Re-run after a timeout" | No (byte equal, checked) |
| Dry run does not wait | "--wait has no effect on a dry run, after a failed delete or on an operator-managed instance", "Dry run does not wait" | No (byte equal, checked) |
| Operator-managed instance | "--wait has no effect on a dry run, after a failed delete or on an operator-managed instance", "Operator-managed instance" | No (byte equal, checked) |
| Without the flag nothing changes | "Instance delete accepts --wait and without it nothing changes", "Without the flag nothing changes" | No (byte equal, checked) |
| (new) | "While it waits the command prints how many deleted resources are still terminating", "Progress while waiting" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm instance delete` SHALL accept `--wait`. | body of "Instance delete accepts --wait and without it nothing changes", verbatim |
| With `--wait`, on a real run of a CLI-owned instance in which no track... | body of "Instance delete with `--wait` waits for the deleted resources to disappear", verbatim; body of "While it waits the command prints how many deleted resources are still terminating", the part from "With `--wait`, on a real run o..." on, verbatim; body of "The wait skips a resource the command did not delete", the part from "With `--wait`, on a real run o..." on, verbatim; body of "A timeout lists the resources still terminating and keeps the ModuleInstance record", the part from "With `--wait`, on a real run o..." on, verbatim |
| A resource is gone when its read returns NotFound, or when the live ob... | body of "A resource is gone when its read returns NotFound or its UID differs", verbatim |
| A read that fails with another error SHALL NOT count as gone. | body of "A resource is gone when its read returns NotFound or its UID differs", verbatim |
| While it waits, the command SHALL print a line that says how many of t... | body of "While it waits the command prints how many deleted resources are still terminating", first letter lowered after the condition, otherwise verbatim |
| The command SHALL NOT wait for a resource it did not delete: a kept `P... | body of "The wait skips a resource the command did not delete", first letter lowered after the condition, otherwise verbatim |
| When every such resource is gone, the command SHALL delete the `Module... | body of "Instance delete with `--wait` waits for the deleted resources to disappear", verbatim |
| When the timeout passes first, the command SHALL list each resource th... | body of "A timeout lists the resources still terminating and keeps the ModuleInstance record", first letter lowered after the condition, otherwise verbatim |
| With `--dry-run`, and in a run in which a tracked resource failed, `--... | body of "--wait has no effect on a dry run, after a failed delete or on an operator-managed instance", verbatim |
| For an operator-managed instance `--wait` SHALL have no effect: the co... | body of "--wait has no effect on a dry run, after a failed delete or on an operator-managed instance", verbatim |
| Without `--wait`, the output and the exit code of the command SHALL be... | body of "Instance delete accepts --wait and without it nothing changes", verbatim |

Lengths (old, new): old 1665; new "Instance delete accepts --wait and without it nothing changes" 152; "Instance delete with `--wait` waits for the deleted resources to disappear" 471; "A resource is gone when its read returns NotFound or its UID differs" 212; "While it waits the command prints how many deleted resources are still terminating" 272; "The wait skips a resource the command did not delete" 245; "A timeout lists the resources still terminating and keeps the ModuleInstance record" 317; "--wait has no effect on a dry run, after a failed delete or on an operator-managed instance" 260.

### Ledger: platform-resolution, Registry-resolved catalog subscription

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI resolves a catalog subscription version from the registry within one major", "Release and prerelease are the two selection modes", "Development builds are never selected", "Versions that are not valid SemVer are ignored", "Resolution refuses rather than substitute a version from the other mode", "Resolution infers nothing".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Stable release selected by default | "Release and prerelease are the two selection modes", "Stable release selected by default" | No (byte equal, checked) |
| Prerelease-only history refuses under the default | "Resolution refuses rather than substitute a version from the other mode", "Prerelease-only history refuses under the default" | No (byte equal, checked) |
| Prerelease mode ignores development builds | "Development builds are never selected", "Prerelease mode ignores development builds" | No (byte equal, checked) |
| Development builds are never the answer | "Development builds are never selected", "Development builds are never the answer" | No (byte equal, checked) |
| Empty history refuses | "Resolution refuses rather than substitute a version from the other mode", "Empty history refuses" | No (byte equal, checked) |
| Registry failure is not a refusal | "Resolution refuses rather than substitute a version from the other mode", "Registry failure is not a refusal" | No (byte equal, checked) |
| (new) | "The CLI resolves a catalog subscription version from the registry within one major", "Resolution is scoped to the major of the path" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Versions that are not valid SemVer are ignored", "An invalid version is not a candidate" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Resolution infers nothing", "No inference of maturity, major or literal" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The CLI SHALL be able to resolve the version a Platform subscription p... | body of "The CLI resolves a catalog subscription version from the registry within one major", verbatim |
| Resolution SHALL be scoped to the major carried by that path, so a bui... | body of "The CLI resolves a catalog subscription version from the registry within one major", verbatim |
| Two selection modes exist, and the caller chooses between them explici... | body of "Release and prerelease are the two selection modes", with "exist," reads "SHALL exist," |
| - **Release (default)**: the highest published version whose SemVer pr... | body of "Release and prerelease are the two selection modes", verbatim |
| - **Prerelease (opt-in)**: the highest published version whose prerele... | body of "Release and prerelease are the two selection modes", verbatim |
| Development builds SHALL NEVER be selected in either mode. | body of "Development builds are never selected", verbatim |
| A development build is identified by a prerelease part whose first ide... | body of "Development builds are never selected", verbatim |
| Versions the registry returns that are not valid SemVer SHALL be ignor... | body of "Versions that are not valid SemVer are ignored", verbatim |
| When no published version satisfies the active mode, resolution SHALL ... | body of "Resolution refuses rather than substitute a version from the other mode", verbatim |
| The refusal SHALL name the module path queried, the registry consulted... | body of "Resolution refuses rather than substitute a version from the other mode", verbatim |
| Resolution SHALL NOT infer maturity, widen the major, or fall back to ... | body of "Resolution infers nothing", verbatim |

Lengths (old, new): old 1358; new "The CLI resolves a catalog subscription version from the registry within one major" 332; "Release and prerelease are the two selection modes" 291; "Development builds are never selected" 266; "Versions that are not valid SemVer are ignored" 107; "Resolution refuses rather than substitute a version from the other mode" 271; "Resolution infers nothing" 92.

### Ledger: platform-resolution, Acquisition mirrors the operator

Form: b (MODIFIED parent with its header and all scenarios, ADDED splits). New requirements: "keep", "The registry entries of a generated platform are the effective registry", "Every platform source is acquired through the kernel's directory acquisition", "The CLI neither synthesizes nor persists a platform value", "A generated platform module lives in the OPM home cache under its content hash", "The CLI does not publish or write the generated module".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| CR generates the operator's module | "Acquisition mirrors the operator", "CR generates the operator's module" | No (byte equal, checked) |
| An active claim's catalog is generated | "Acquisition mirrors the operator", "An active claim's catalog is generated" | No (byte equal, checked) |
| Unchanged CR reuses the cache | "Acquisition mirrors the operator", "Unchanged CR reuses the cache" | No (byte equal, checked) |
| Unpublished pin fails at acquisition | "Acquisition mirrors the operator", "Unpublished pin fails at acquisition" | No (byte equal, checked) |
| Same failure surface for every source | "Acquisition mirrors the operator", "Same failure surface for every source" | No (byte equal, checked) |
| (new) | "The registry entries of a generated platform are the effective registry", "The effective registry feeds the entries" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Every platform source is acquired through the kernel's directory acquisition", "A bad pin fails at acquisition for every source" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The CLI neither synthesizes nor persists a platform value", "No synthesized or persisted platform value" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A generated platform module lives in the OPM home cache under its content hash", "The cache directory is named by the content hash" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The CLI does not publish or write the generated module", "The generated module is derived state" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| A cluster `Platform` CR SHALL be turned into a platform module exactly... | body of "Acquisition mirrors the operator", verbatim; body of "The registry entries of a generated platform are the effective registry", the part from "A cluster `Platform` CR SHALL ..." on, verbatim; body of "A generated platform module lives in the OPM home cache under its content hash", the part from "A cluster `Platform` CR SHALL ..." on, verbatim |
| The registry entries SHALL be the effective registry the operator reco... | body of "The registry entries of a generated platform are the effective registry", verbatim |
| Every platform source SHALL then be acquired through the kernel's dire... | body of "Every platform source is acquired through the kernel's directory acquisition", verbatim |
| The CLI MUST NOT synthesize a platform value from typed inputs and MUS... | body of "The CLI neither synthesizes nor persists a platform value", verbatim |
| The generated module SHALL live under the OPM home cache at `cache/pla... | body of "A generated platform module lives in the OPM home cache under its content hash", verbatim |
| The CLI MUST NOT publish the generated module, write it to the cluster... | body of "The CLI does not publish or write the generated module", verbatim |

Lengths (old, new): old 1579; new "Acquisition mirrors the operator" 445; "The registry entries of a generated platform are the effective registry" 413; "Every platform source is acquired through the kernel's directory acquisition" 252; "The CLI neither synthesizes nor persists a platform value" 109; "A generated platform module lives in the OPM home cache under its content hash" 394; "The CLI does not publish or write the generated module" 153.

### Ledger: platform-resolution, The cluster arm reads the effective registry

Form: b (MODIFIED parent with its header and all scenarios, ADDED splits). New requirements: "keep", "Resolution warns and never silently substitutes when the status is stale or the Platform is not Ready", "The write-if-absent seeding of a missing Platform is unchanged".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Effective registry wins over the spec | "The cluster arm reads the effective registry", "Effective registry wins over the spec" | No (byte equal, checked) |
| No recorded registry falls back to the spec | "The cluster arm reads the effective registry", "No recorded registry falls back to the spec" | No (byte equal, checked) |
| A stale status warns | "The cluster arm reads the effective registry", "A stale status warns" | No (byte equal, checked) |
| A refused Platform renders against the last good package | "The cluster arm reads the effective registry", "A refused Platform renders against the last good package" | No (byte equal, checked) |
| (new) | "Resolution warns and never silently substitutes when the status is stale or the Platform is not Ready", "A warning does not change the generated package" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The write-if-absent seeding of a missing Platform is unchanged", "Seeding happens only when no CR exists" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| When the cluster Platform's status carries a resolved registry, resolu... | body of "The cluster arm reads the effective registry", verbatim |
| When it carries none, resolution SHALL generate from the spec and SHAL... | body of "The cluster arm reads the effective registry", verbatim |
| Resolution SHALL warn, never silently substitute, in two cases: when `... | body of "Resolution warns and never silently substitutes when the status is stale or the Platform is not Ready", verbatim |
| Neither warning changes which package is generated. | body of "Resolution warns and never silently substitutes when the status is stale or the Platform is not Ready", verbatim |
| The write-if-absent seeding of a missing Platform is unchanged: it hap... | body of "The write-if-absent seeding of a missing Platform is unchanged", with "is unchanged:" reads "SHALL be unchanged:" |

Lengths (old, new): old 796; new "The cluster arm reads the effective registry" 243; "Resolution warns and never silently substitutes when the status is stale or the Platform is not Ready" 410; "The write-if-absent seeding of a missing Platform is unchanged" 147.

### Ledger: platform-resolution, Module commands render against the module's deps

Form: b (MODIFIED parent with its header and all scenarios, ADDED splits). New requirements: "keep", "The generated platform carries one enabled registry entry per catalog dependency", "The dependency list of the generated platform is the closure of the pins", "The generated platform carries the module's replacements", "The deps are read from the module's acquired source", "The generated platform is cached under its content hash", "The provenance line names the module's deps", "The skew policy is neither named nor applied to the module's deps", "A module with no catalog dependency renders against an empty registry", "A pin that is not published fails at generation", "A refusal for unresolved demands adds a hint that names the platform flag", "The same generator serves every other render".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Catalog pins come from the module | "Module commands render against the module's deps", "Catalog pins come from the module" | No (byte equal, checked) |
| A catalog's own dependencies are closed over | "Module commands render against the module's deps", "A catalog's own dependencies are closed over" | No (byte equal, checked) |
| No local default platform is needed | "Module commands render against the module's deps", "No local default platform is needed" | No (byte equal, checked) |
| The flag overrides the deps | "Module commands render against the module's deps", "The flag overrides the deps" | No (byte equal, checked) |
| Deploying commands keep the platform | "Module commands render against the module's deps", "Deploying commands keep the platform" | No (byte equal, checked) |
| A local catalog checkout is rendered | "Module commands render against the module's deps", "A local catalog checkout is rendered" | No (byte equal, checked) |
| A published module renders against its own deps | "Module commands render against the module's deps", "A published module renders against its own deps" | No (byte equal, checked) |
| A module without a catalog dependency renders against an empty registry | "Module commands render against the module's deps", "A module without a catalog dependency renders against an empty registry" | No (byte equal, checked) |
| Core never falls below the kernel's verified release | "Module commands render against the module's deps", "Core never falls below the kernel's verified release" | No (byte equal, checked) |
| A provider-fulfilled demand names the platform flag | "Module commands render against the module's deps", "A provider-fulfilled demand names the platform flag" | No (byte equal, checked) |
| Unchanged deps reuse the cache | "Module commands render against the module's deps", "Unchanged deps reuse the cache" | No (byte equal, checked) |
| (new) | "The generated platform carries one enabled registry entry per catalog dependency", "Registry entries and the core pin" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The dependency list of the generated platform is the closure of the pins", "Dependency list is a closure" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The generated platform carries the module's replacements", "Replacements are carried" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The deps are read from the module's acquired source", "Deps come from the acquired source" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The generated platform is cached under its content hash", "Cache named by the content hash" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The provenance line names the module's deps", "Provenance names the deps" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The skew policy is neither named nor applied to the module's deps", "No skew policy for the deps" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A module with no catalog dependency renders against an empty registry", "Empty registry for a module without catalog dependency" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A pin that is not published fails at generation", "Unpublished pin fails at generation" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A refusal for unresolved demands adds a hint that names the platform flag", "Hint on an unresolved demand" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The same generator serves every other render", "One generator for every render" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| When `opm module build` or `opm module vet` runs without `--platform`,... | body of "Module commands render against the module's deps", verbatim; body of "The generated platform carries one enabled registry entry per catalog dependency", the part from "When `opm module build` or `op..." on, verbatim; body of "The dependency list of the generated platform is the closure of the pins", the part from "When `opm module build` or `op..." on, verbatim; body of "The generated platform carries the module's replacements", the part from "When `opm module build` or `op..." on, verbatim; body of "The deps are read from the module's acquired source", the part from "When `opm module build` or `op..." on, verbatim; body of "The generated platform is cached under its content hash", the part from "When `opm module build` or `op..." on, verbatim; body of "The provenance line names the module's deps", the part from "When `opm module build` or `op..." on, verbatim; body of "The skew policy is neither named nor applied to the module's deps", the part from "When `opm module build` or `op..." on, verbatim; body of "A module with no catalog dependency renders against an empty registry", the part from "When `opm module build` or `op..." on, verbatim; body of "A pin that is not published fails at generation", the part from "When `opm module build` or `op..." on, verbatim; body of "A refusal for unresolved demands adds a hint that names the platform flag", verbatim; body of "The same generator serves every other render", verbatim |
| Source: cli issue 229. | body of "Module commands render against the module's deps", verbatim; body of "The generated platform carries one enabled registry entry per catalog dependency", verbatim; body of "The dependency list of the generated platform is the closure of the pins", verbatim; body of "The generated platform carries the module's replacements", verbatim; body of "The deps are read from the module's acquired source", verbatim; body of "The generated platform is cached under its content hash", verbatim; body of "The provenance line names the module's deps", verbatim; body of "The skew policy is neither named nor applied to the module's deps", verbatim; body of "A module with no catalog dependency renders against an empty registry", verbatim; body of "A pin that is not published fails at generation", verbatim; body of "A refusal for unresolved demands adds a hint that names the platform flag", verbatim |
| - The generated platform SHALL carry one enabled `#registry` entry per... | body of "The generated platform carries one enabled registry entry per catalog dependency", first letter lowered after the condition, otherwise verbatim |
| - Its dependency list SHALL be the minimum-version-selection closure o... | body of "The dependency list of the generated platform is the closure of the pins", first letter lowered after the condition, otherwise verbatim |
| - Each `replaceWith` in the module's `cue.mod/local-module.cue` whose ... | body of "The generated platform carries the module's replacements", first letter lowered after the condition, otherwise verbatim |
| The closure SHALL read a directory-replaced module's requirements from... | body of "The dependency list of the generated platform is the closure of the pins", verbatim |
| - The deps SHALL be read from the module's acquired source, so a modul... | body of "The deps are read from the module's acquired source", first letter lowered after the condition, otherwise verbatim |
| A published module has no local module context, so nothing is carried ... | body of "The deps are read from the module's acquired source", verbatim |
| - The generated module SHALL be cached under `cache/platforms/` beside... | body of "The generated platform is cached under its content hash", first letter lowered after the condition, otherwise verbatim |
| - The provenance line SHALL name the source as the module's deps and l... | body of "The provenance line names the module's deps", first letter lowered after the condition, otherwise verbatim |
| The skew policy SHALL NOT be named or applied: every pinned path is at... | body of "The skew policy is neither named nor applied to the module's deps", first letter lowered after the condition, otherwise verbatim |
| - A module with no `opmodel.dev/catalogs/*` dependency SHALL render ag... | body of "A module with no catalog dependency renders against an empty registry", first letter lowered after the condition, otherwise verbatim |
| - A pin that is not published SHALL fail at generation naming the modu... | body of "A pin that is not published fails at generation", first letter lowered after the condition, otherwise verbatim |
| - When a render against this source is refused for unresolved demands,... | body of "A refusal for unresolved demands adds a hint that names the platform flag", verbatim |
| - The same generator serves the deps fallback of every other render (s... | body of "The same generator serves every other render", with "serves" reads "SHALL serve" |

Lengths (old, new): old 2420; new "Module commands render against the module's deps" 206; "The generated platform carries one enabled registry entry per catalog dependency" 441; "The dependency list of the generated platform is the closure of the pins" 422; "The generated platform carries the module's replacements" 324; "The deps are read from the module's acquired source" 334; "The generated platform is cached under its content hash" 269; "The provenance line names the module's deps" 196; "The skew policy is neither named nor applied to the module's deps" 203; "A module with no catalog dependency renders against an empty registry" 367; "A pin that is not published fails at generation" 182; "A refusal for unresolved demands adds a hint that names the platform flag" 388; "The same generator serves every other render" 299.

### Ledger: platform-resolution, Platform source precedence by command

Form: b (MODIFIED parent with its header and all scenarios, ADDED splits). New requirements: "keep", "Instance commands and module apply resolve the flag, then the cluster Platform, then their own deps", "Platform check resolves its argument, then the flag, then the cluster Platform", "Every platform source resolves to a platform module directory", "Every command that renders reports the platform source it resolved", "A platform argument that is a file or an empty directory fails with the expected shape", "The --provider flag does not exist".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Flag wins | "Platform source precedence by command", "Flag wins" | No (byte equal, checked) |
| Cluster CR used when no flag | "Platform source precedence by command", "Cluster CR used when no flag" | No (byte equal, checked) |
| Absent Platform falls back to the deps | "Platform source precedence by command", "Absent Platform falls back to the deps" | No (byte equal, checked) |
| Module commands never read the cluster | "Platform source precedence by command", "Module commands never read the cluster" | No (byte equal, checked) |
| A platform file is refused | "Platform source precedence by command", "A platform file is refused" | No (byte equal, checked) |
| platform check with no source refuses | "Platform source precedence by command", "platform check with no source refuses" | No (byte equal, checked) |
| (new) | "Instance commands and module apply resolve the flag, then the cluster Platform, then their own deps", "Precedence of the instance commands" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Platform check resolves its argument, then the flag, then the cluster Platform", "Precedence of platform check" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Every platform source resolves to a platform module directory", "Every source is a module directory" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Every command that renders reports the platform source it resolved", "The resolved source is reported" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A platform argument that is a file or an empty directory fails with the expected shape", "A directory without a platform module is refused" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The --provider flag does not exist", "No --provider flag" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The CLI SHALL resolve the platform for every render by precedence, and... | body of "Platform source precedence by command", verbatim; body of "Instance commands and module apply resolve the flag, then the cluster Platform, then their own deps", verbatim; body of "Platform check resolves its argument, then the flag, then the cluster Platform", verbatim |
| - `opm module build` and `opm module vet` resolve `--platform <dir>` (... | body of "Platform source precedence by command", verbatim |
| - `opm instance build`, `opm instance vet`, `opm instance diff`, `opm ... | body of "Instance commands and module apply resolve the flag, then the cluster Platform, then their own deps", verbatim |
| - `opm platform check` resolves its `[dir]` argument (highest) > `--pl... | body of "Platform check resolves its argument, then the flag, then the cluster Platform", verbatim |
| Every source resolves to a platform module directory the kernel acquir... | body of "Every platform source resolves to a platform module directory", with "Every source resolves to" reads "Every source SHALL resolve to" |
| Every command that renders SHALL report which platform source it resol... | body of "Every command that renders reports the platform source it resolved", verbatim |
| A `--platform` argument that is a file, or a directory holding no plat... | body of "A platform argument that is a file or an empty directory fails with the expected shape", verbatim |
| The `--provider` flag SHALL NOT exist (superseded by `--platform`, enh... | body of "The --provider flag does not exist", verbatim |

Lengths (old, new): old 1859; new "Platform source precedence by command" 337; "Instance commands and module apply resolve the flag, then the cluster Platform, then their own deps" 437; "Platform check resolves its argument, then the flag, then the cluster Platform" 313; "Every platform source resolves to a platform module directory" 415; "Every command that renders reports the platform source it resolved" 224; "A platform argument that is a file or an empty directory fails with the expected shape" 186; "The --provider flag does not exist" 228.

### Ledger: platform-resolution, Only operator install seeds the cluster Platform

Form: c2 (REMOVED, ADDED under new names). New requirements: "Operator install seeds the cluster Platform with a plain create", "A forbidden create of the seeded Platform degrades to a warning", "Operator install reports the provenance it resolved and no other source", "No render-bearing command seeds a Platform".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Install seeds an absent Platform | "Operator install seeds the cluster Platform with a plain create", "Install seeds an absent Platform" | No (byte equal, checked) |
| Concurrent create tolerated | "Operator install seeds the cluster Platform with a plain create", "Concurrent create tolerated" | No (byte equal, checked) |
| RBAC-denied create degrades | "A forbidden create of the seeded Platform degrades to a warning", "RBAC-denied create degrades" | No (byte equal, checked) |
| Install reports its own provenance | "Operator install reports the provenance it resolved and no other source", "Install reports its own provenance" | No (byte equal, checked) |
| Apply never seeds a Platform | "No render-bearing command seeds a Platform", "Apply never seeds a Platform" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm operator install` SHALL seed the singleton `cluster` Platform wit... | body of "Operator install seeds the cluster Platform with a plain create", verbatim; body of "A forbidden create of the seeded Platform degrades to a warning", the part from "`opm operator install` SHALL s..." on, verbatim |
| The CLI MUST NOT use server-side apply or update for this write and MU... | body of "Operator install seeds the cluster Platform with a plain create", verbatim |
| A forbidden create SHALL degrade to a warning. | body of "A forbidden create of the seeded Platform degrades to a warning", verbatim |
| The reported provenance SHALL name the catalog coordinate and version ... | body of "Operator install reports the provenance it resolved and no other source", verbatim |
| No render-bearing command SHALL create, update or seed a `Platform`: `... | body of "No render-bearing command seeds a Platform", verbatim |

Lengths (old, new): old 843; new "Operator install seeds the cluster Platform with a plain create" 498; "A forbidden create of the seeded Platform degrades to a warning" 160; "Operator install reports the provenance it resolved and no other source" 128; "No render-bearing command seeds a Platform" 168.

### Ledger: platform-resolution, Renders fall back to their own deps

Form: b (MODIFIED parent with its header and all scenarios, ADDED splits). New requirements: "keep", "An instance command reads its pins and replacements from the instance package", "Module apply reads its pins from the module's acquired source", "The skew policy is not applied to the deps fallback", "An instance package with no catalog dependency renders against an empty registry".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Instance build with no cluster renders against its pins | "Renders fall back to their own deps", "Instance build with no cluster renders against its pins" | No (byte equal, checked) |
| module apply on a cluster without a Platform | "Renders fall back to their own deps", "module apply on a cluster without a Platform" | No (byte equal, checked) |
| (new) | "An instance command reads its pins and replacements from the instance package", "Pins of an instance command" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Module apply reads its pins from the module's acquired source", "Pins of module apply" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The skew policy is not applied to the deps fallback", "No skew policy for the fallback" | New restatement scenario: only words of the old clause, no new case |
| (new) | "An instance package with no catalog dependency renders against an empty registry", "Empty registry for an instance package" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| When `opm instance build`, `opm instance vet`, `opm instance diff`, `o... | body of "Renders fall back to their own deps", verbatim; body of "An instance command reads its pins and replacements from the instance package", the part from "When `opm instance build`, `op..." on, verbatim; body of "Module apply reads its pins from the module's acquired source", the part from "When `opm instance build`, `op..." on, verbatim; body of "The skew policy is not applied to the deps fallback", the part from "When `opm instance build`, `op..." on, verbatim; body of "An instance package with no catalog dependency renders against an empty registry", the part from "When `opm instance build`, `op..." on, verbatim |
| - For an instance command, the pins SHALL be read from the instance pa... | body of "An instance command reads its pins and replacements from the instance package", first letter lowered after the condition, otherwise verbatim |
| - For `opm module apply`, the pins SHALL be read from the module's acq... | body of "Module apply reads its pins from the module's acquired source", first letter lowered after the condition, otherwise verbatim |
| - The skew policy SHALL NOT be applied to this source. | body of "The skew policy is not applied to the deps fallback", first letter lowered after the condition, otherwise verbatim |
| - An instance package whose `cue.mod/module.cue` lists no `opmodel.dev... | body of "An instance package with no catalog dependency renders against an empty registry", first letter lowered after the condition, otherwise verbatim |

Lengths (old, new): old 1094; new "Renders fall back to their own deps" 470; "An instance command reads its pins and replacements from the instance package" 365; "Module apply reads its pins from the module's acquired source" 310; "The skew policy is not applied to the deps fallback" 239; "An instance package with no catalog dependency renders against an empty registry" 446.

### Ledger: platform-resolution, Instance build and vet probe the cluster without depending on it

Form: c2 (REMOVED, ADDED under new names). New requirements: "Instance build and vet try the cluster Platform before the deps", "Offline instance build and vet never contact a cluster", "Instance build and vet resolve the deps silently when no kubeconfig context resolves", "Instance build and vet warn and resolve the deps when the Platform is absent or forbidden", "Instance build and vet warn and resolve the deps when the cluster does not answer in time", "Instance diff, instance apply and module apply keep failing on an unreachable cluster".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| offline never contacts the cluster | "Offline instance build and vet never contact a cluster", "offline never contacts the cluster" | No (byte equal, checked) |
| Unreachable cluster degrades to the deps | "Instance build and vet warn and resolve the deps when the cluster does not answer in time", "Unreachable cluster degrades to the deps" | No (byte equal, checked) |
| Kubeconfig flags select the cluster | "Instance build and vet try the cluster Platform before the deps", "Kubeconfig flags select the cluster" | No (byte equal, checked) |
| (new) | "Instance build and vet resolve the deps silently when no kubeconfig context resolves", "No kubeconfig context resolves the deps without a warning" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Instance build and vet warn and resolve the deps when the Platform is absent or forbidden", "Absent or forbidden Platform warns and resolves the deps" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Instance diff, instance apply and module apply keep failing on an unreachable cluster", "Deploying commands need the cluster" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm instance build` and `opm instance vet` SHALL try the cluster `Pla... | body of "Instance build and vet try the cluster Platform before the deps", verbatim; body of "Offline instance build and vet never contact a cluster", verbatim; body of "Instance build and vet resolve the deps silently when no kubeconfig context resolves", verbatim; body of "Instance build and vet warn and resolve the deps when the Platform is absent or forbidden", verbatim; body of "Instance build and vet warn and resolve the deps when the cluster does not answer in time", verbatim |
| - With `--offline`, the CLI SHALL NOT resolve a kubeconfig or contact ... | body of "Offline instance build and vet never contact a cluster", verbatim |
| - With no resolvable kubeconfig context, the CLI SHALL resolve the dep... | body of "Instance build and vet resolve the deps silently when no kubeconfig context resolves", verbatim |
| - When the `Platform` CR is absent (including a cluster without the `P... | body of "Instance build and vet warn and resolve the deps when the Platform is absent or forbidden", verbatim |
| - When the API server does not answer within a bounded wait (at most 1... | body of "Instance build and vet warn and resolve the deps when the cluster does not answer in time", verbatim |
| - `opm instance diff`, `opm instance apply` and `opm module apply` kee... | body of "Instance diff, instance apply and module apply keep failing on an unreachable cluster", with "keep failing" reads "SHALL keep failing" |

Lengths (old, new): old 918; new "Instance build and vet try the cluster Platform before the deps" 136; "Offline instance build and vet never contact a cluster" 267; "Instance build and vet resolve the deps silently when no kubeconfig context resolves" 226; "Instance build and vet warn and resolve the deps when the Platform is absent or forbidden" 343; "Instance build and vet warn and resolve the deps when the cluster does not answer in time" 335; "Instance diff, instance apply and module apply keep failing on an unreachable cluster" 150.

### Ledger: platform-resolution, Operator install renders against the operator module's own pins

Form: c2 (REMOVED, ADDED under new names). New requirements: "Operator install renders the operator module against a platform of its own pins", "Operator install renders the same objects whatever the cluster Platform holds", "Operator install names the module's deps as the provenance source", "The operator render narrows the platform-source precedence for the operator's own instance only".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Seeded Platform on another catalog release | "Operator install renders the same objects whatever the cluster Platform holds", "Seeded Platform on another catalog release" | No (byte equal, checked) |
| Stalled Platform does not block repair | "Operator install renders the operator module against a platform of its own pins", "Stalled Platform does not block repair" | No (byte equal, checked) |
| (new) | "Operator install names the module's deps as the provenance source", "Provenance of the operator render" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The operator render narrows the platform-source precedence for the operator's own instance only", "The narrowing is for the operator's own instance only" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm operator install`, the CRDs-only form included, SHALL render the ... | body of "Operator install renders the operator module against a platform of its own pins", verbatim; body of "Operator install renders the same objects whatever the cluster Platform holds", the part from "`opm operator install`, the CR..." on, verbatim; body of "Operator install names the module's deps as the provenance source", the part from "`opm operator install`, the CR..." on, verbatim; body of "The operator render narrows the platform-source precedence for the operator's own instance only", verbatim |
| Every other command keeps its precedence under "Platform source preced... | body of "Operator install renders the operator module against a platform of its own pins", verbatim |
| The same module version and values SHALL render the same objects wheth... | body of "Operator install renders the same objects whatever the cluster Platform holds", verbatim |
| The provenance line SHALL name the module's deps as the source. | body of "Operator install names the module's deps as the provenance source", verbatim |
| This narrows the platform-source precedence of enhancement 0006 for th... | body of "The operator render narrows the platform-source precedence for the operator's own instance only", verbatim |
| Source: enhancement 0021 ("The operator's install artifact is a regist... | body of "Operator install renders the operator module against a platform of its own pins", verbatim; body of "Operator install renders the same objects whatever the cluster Platform holds", verbatim; body of "Operator install names the module's deps as the provenance source", verbatim |

Lengths (old, new): old 911; new "Operator install renders the operator module against a platform of its own pins" 453; "Operator install renders the same objects whatever the cluster Platform holds" 419; "Operator install names the module's deps as the provenance source" 258; "The operator render narrows the platform-source precedence for the operator's own instance only" 426.

### Ledger: artifact-publishing, One pipeline, two entry points

Form: c2 (REMOVED, ADDED under new names). New requirements: "Publish loads the artifact tree and validates its identity package by unification", "Publish derives the registry coordinates, runs the gates and pushes", "What is published is exactly the committed tree", "Module publish loads the tree through the kernel's module loader".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Clean artifact publishes at its declared version | "Publish derives the registry coordinates, runs the gates and pushes", "Clean artifact publishes at its declared version" | No (byte equal, checked) |
| Identity is validated by unification | "Publish loads the artifact tree and validates its identity package by unification", "Identity is validated by unification" | No (byte equal, checked) |
| A tree that does not load surfaces the load error | "Publish loads the artifact tree and validates its identity package by unification", "A tree that does not load surfaces the load error" | No (byte equal, checked) |
| A module the kernel refuses is refused by publish | "Module publish loads the tree through the kernel's module loader", "A module the kernel refuses is refused by publish" | No (byte equal, checked) |
| (new) | "What is published is exactly the committed tree", "Published bytes are the committed tree" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm module publish [dir]` and `opm catalog publish [dir]` SHALL share... | body of "Publish loads the artifact tree and validates its identity package by unification", the part from "`opm module publish [dir]`" on, verbatim; body of "Publish derives the registry coordinates, runs the gates and pushes", the part from "`opm module publish [dir]`" on, verbatim; body of "Publish derives the registry coordinates, runs the gates and pushes", the part from "derive the registry coordinate..." on, verbatim |
| What is published SHALL be exactly the committed tree - no copied buil... | body of "What is published is exactly the committed tree", verbatim |
| For a module, publish SHALL additionally load the tree through the ker... | body of "Module publish loads the tree through the kernel's module loader", verbatim |
| Publish SHALL NOT judge a module loadable by any test the kernel does ... | body of "Module publish loads the tree through the kernel's module loader", verbatim |
| (note) | "Publish loads the artifact tree and validates its identity package by unification": The first sentence is longer than 500 characters, so it is cut at its list of steps: this block holds the subject and the first two steps, the next block the subject and the last three steps. |

Lengths (old, new): old 984; new "Publish loads the artifact tree and validates its identity package by unification" 291; "Publish derives the registry coordinates, runs the gates and pushes" 361; "What is published is exactly the committed tree" 122; "Module publish loads the tree through the kernel's module loader" 298.

### Ledger: artifact-publishing, Gates and refusal catalog

Form: c2 (REMOVED, ADDED under new names). New requirements: "Publish accumulates every refusal and refuses with exit code 2", "Each refusal follows one fixed shape", "One cause yields one refusal", "The loadability, identity and derivation gates", "The cue.mod, source, tag and package-name gates", "The domain, local-override and already-published gates".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Multiple defects reported in one pass | "Publish accumulates every refusal and refuses with exit code 2", "Multiple defects reported in one pass" | No (byte equal, checked) |
| Kernel refusal is one refusal | "One cause yields one refusal", "Kernel refusal is one refusal" | No (byte equal, checked) |
| Already published is always a refusal | "The domain, local-override and already-published gates", "Already published is always a refusal" | No (byte equal, checked) |
| Unowned domains are not policed | "The domain, local-override and already-published gates", "Unowned domains are not policed" | No (byte equal, checked) |
| (new) | "Each refusal follows one fixed shape", "A refusal follows the fixed shape" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The loadability, identity and derivation gates", "The loadability and identity gates run" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The cue.mod, source, tag and package-name gates", "The cue.mod and tag gates run" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| Publish SHALL evaluate every evaluable gate, accumulate all refusals i... | body of "Publish accumulates every refusal and refuses with exit code 2", verbatim |
| Each refusal SHALL follow the fixed shape condition → evidence → conse... | body of "Each refusal follows one fixed shape", verbatim |
| One cause SHALL yield one refusal. | body of "One cause yields one refusal", verbatim |
| The gates: kernel loadability (modules only: the kernel's module loade... | body of "The loadability, identity and derivation gates", the part from "The gates:" on, verbatim; "The gates:" reads "The gates SHALL be:"; body of "The cue.mod, source, tag and package-name gates", the part from "The gates:" on, verbatim; "The gates:" reads "The gates SHALL be:"; body of "The cue.mod, source, tag and package-name gates", the part from "`cue.mod` `module:`" on, verbatim; body of "The domain, local-override and already-published gates", the part from "The gates:" on, verbatim; "The gates:" reads "The gates SHALL be:"; body of "The domain, local-override and already-published gates", the part from "namespace and kind-segment che..." on, verbatim |
| (note) | "The loadability, identity and derivation gates": The gate list is one sentence of 769 characters, so it is cut at the semicolons between gates; each block holds "The gates:" and a contiguous run of its items. |

Lengths (old, new): old 1175; new "Publish accumulates every refusal and refuses with exit code 2" 132; "Each refusal follows one fixed shape" 237; "One cause yields one refusal" 34; "The loadability, identity and derivation gates" 359; "The cue.mod, source, tag and package-name gates" 333; "The domain, local-override and already-published gates" 124.

### Ledger: artifact-publishing, Exit codes

Form: c2 (REMOVED, ADDED under new names). New requirements: "Publish exits 0, 2, 4, 3 or 1 by outcome", "A failed already-published lookup does not hide locally derived refusals", "A failed registry operation with no HTTP response reads registry unreachable", "A 401 or 403 answer reads as refused credentials and exits 4", "Any other failure answer reads registry operation failed", "A 403 answer to the already-published lookup is not a failure", "A refusing token endpoint is a refused credential", "Publish with no registry configured exits 2".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Connectivity is not a refusal | "A failed registry operation with no HTTP response reads registry unreachable", "Connectivity is not a refusal" | No (byte equal, checked) |
| A refused credential is not called unreachable | "A 401 or 403 answer reads as refused credentials and exits 4", "A refused credential is not called unreachable" | No (byte equal, checked) |
| A refusing token endpoint is not called unreachable | "A refusing token endpoint is a refused credential", "A refusing token endpoint is not called unreachable" | No (byte equal, checked) |
| A forbidden lookup reads as not published | "A 403 answer to the already-published lookup is not a failure", "A forbidden lookup reads as not published" | No (byte equal, checked) |
| Another registry answer is not called unreachable | "Any other failure answer reads registry operation failed", "Another registry answer is not called unreachable" | No (byte equal, checked) |
| No registry is configured | "Publish with no registry configured exits 2", "No registry is configured" | No (byte equal, checked) |
| (new) | "Publish exits 0, 2, 4, 3 or 1 by outcome", "Exit code by outcome" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A failed already-published lookup does not hide locally derived refusals", "Locally derived refusals print before the registry error" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| Publish SHALL exit 0 on success, 2 on any refusal, 4 when a registry r... | body of "Publish exits 0, 2, 4, 3 or 1 by outcome", verbatim; body of "A 403 answer to the already-published lookup is not a failure", verbatim |
| No consumer contract is made on distinguishing refusal causes by code. | body of "Publish exits 0, 2, 4, 3 or 1 by outcome", verbatim |
| When the already-published lookup is what failed, the plan and every l... | body of "A failed already-published lookup does not hide locally derived refusals", verbatim |
| A failed registry operation SHALL be named by its cause, and the messa... | body of "A failed registry operation with no HTTP response reads registry unreachable", verbatim; body of "A 401 or 403 answer reads as refused credentials and exits 4", verbatim; body of "Any other failure answer reads registry operation failed", verbatim |
| - no HTTP response at all (a refused connection, a DNS or TLS failure,... | body of "A failed registry operation with no HTTP response reads registry unreachable", verbatim |
| - a 401 or 403 answer that reaches the cli as such SHALL exit 4, SHALL... | body of "A 401 or 403 answer reads as refused credentials and exits 4", verbatim |
| - any other failure answer SHALL read "registry operation failed" and ... | body of "Any other failure answer reads registry operation failed", verbatim |
| A 403 answer to the already-published lookup is not such a failure: th... | body of "A 403 answer to the already-published lookup is not a failure", verbatim |
| A registry that hands out bearer tokens and whose token endpoint refus... | body of "A refusing token endpoint is a refused credential", with "is a refused credential" reads "SHALL be a refused credential" |
| A 403 from the token endpoint on the lookup or the core-schema fetch r... | body of "A refusing token endpoint is a refused credential", verbatim |
| When the core-schema fetch fails and no registry is configured (no `--... | body of "Publish with no registry configured exits 2", verbatim |

Lengths (old, new): old 2236; new "Publish exits 0, 2, 4, 3 or 1 by outcome" 334; "A failed already-published lookup does not hide locally derived refusals" 292; "A failed registry operation with no HTTP response reads registry unreachable" 277; "A 401 or 403 answer reads as refused credentials and exits 4" 433; "Any other failure answer reads registry operation failed" 260; "A 403 answer to the already-published lookup is not a failure" 455; "A refusing token endpoint is a refused credential" 466; "Publish with no registry configured exits 2" 304.

### Ledger: artifact-publishing, The compatibility gate

Form: c2 (REMOVED, ADDED under new names). New requirements: "Catalog publish compares each beta and GA member under the additive-only rule", "The compatibility baseline is the last published build that shipped the member", "A violation refuses as refusal 9", "A beta or GA name and apiVersion key is a permanent claim", "A dev build is not judged and is never a predecessor", "A release prerelease tag exempts beta and GA members until the first stable tag".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Field removal refused against the true predecessor | "A violation refuses as refusal 9", "Field removal refused against the true predecessor" | No (byte equal, checked) |
| Prerelease predecessors count | "The compatibility baseline is the last published build that shipped the member", "Prerelease predecessors count" | No (byte equal, checked) |
| Remove-then-readd refused | "A beta or GA name and apiVersion key is a permanent claim", "Remove-then-readd refused" | No (byte equal, checked) |
| apiVersion bump escapes | "The compatibility baseline is the last published build that shipped the member", "apiVersion bump escapes" | No (byte equal, checked) |
| Dev build is not judged | "A dev build is not judged and is never a predecessor", "Dev build is not judged" | No (byte equal, checked) |
| Dev build is never the baseline | "A dev build is not judged and is never a predecessor", "Dev build is never the baseline" | No (byte equal, checked) |
| Release prereleases stay in the window | "The compatibility baseline is the last published build that shipped the member", "Release prereleases stay in the window" | No (byte equal, checked) |
| Prerelease module line is exempt, visibly | "A release prerelease tag exempts beta and GA members until the first stable tag", "Prerelease module line is exempt, visibly" | No (byte equal, checked) |
| First stable tag arms the gate | "A release prerelease tag exempts beta and GA members until the first stable tag", "First stable tag arms the gate" | No (byte equal, checked) |
| (new) | "Catalog publish compares each beta and GA member under the additive-only rule", "Each beta and GA member is compared" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm catalog publish` SHALL, for every member of the tree at beta or G... | body of "Catalog publish compares each beta and GA member under the additive-only rule", the part from "`opm catalog publish` SHALL" on, verbatim; body of "Catalog publish compares each beta and GA member under the additive-only rule", the part from "under the additive-only rule v..." on, verbatim; body of "The compatibility baseline is the last published build that shipped the member", the part from "`opm catalog publish` SHALL" on, verbatim; body of "The compatibility baseline is the last published build that shipped the member", the part from "for every member of the tree a..." on, verbatim; body of "The compatibility baseline is the last published build that shipped the member", the part from "compare the member against the..." on, verbatim |
| A member no published build has carried SHALL pass. | body of "The compatibility baseline is the last published build that shipped the member", verbatim |
| Violations SHALL refuse as refusal 9: a header naming the member, its ... | body of "A violation refuses as refusal 9", verbatim |
| The gate makes a beta-or-GA name-and-apiVersion key a permanent claim ... | body of "A beta or GA name and apiVersion key is a permanent claim", verbatim |
| A dev build, an effective tag whose prerelease segment carries a `dev`... | body of "A dev build is not judged and is never a predecessor", verbatim |
| The plan SHALL report the skip visibly on the `compat gate` row as `de... | body of "A dev build is not judged and is never a predecessor", verbatim |
| While the effective tag is a release prerelease (`-alpha.N`, `-beta.N`... | body of "A release prerelease tag exempts beta and GA members until the first stable tag", verbatim |
| The exemption SHALL end at the first stable tag of the major: a stable... | body of "A release prerelease tag exempts beta and GA members until the first stable tag", verbatim |
| (note) | "Catalog publish compares each beta and GA member under the additive-only rule": The first sentence is 531 characters, so it is cut: this block holds the sentence without the clause that starts at the dash, the next block holds the subject, the scope and that clause. |

Lengths (old, new): old 1702; new "Catalog publish compares each beta and GA member under the additive-only rule" 344; "The compatibility baseline is the last published build that shipped the member" 481; "A violation refuses as refusal 9" 160; "A beta or GA name and apiVersion key is a permanent claim" 167; "A dev build is not judged and is never a predecessor" 348; "A release prerelease tag exempts beta and GA members until the first stable tag" 440.

### Ledger: artifact-publishing, Connectivity across the member walk

Form: c2 (REMOVED, ADDED under new names). New requirements: "A transport failure or a missing dependency aborts the walk as a connectivity error", "A package absent at a published version is the scan's negative signal", "The CLI takes fetch decisions from the library's typed classification".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Mid-walk failure renders no verdict | "A transport failure or a missing dependency aborts the walk as a connectivity error", "Mid-walk failure renders no verdict" | No (byte equal, checked) |
| Package absent at a published version | "A package absent at a published version is the scan's negative signal", "Package absent at a published version" | No (byte equal, checked) |
| A predecessor's dependency is not held | "A transport failure or a missing dependency aborts the walk as a connectivity error", "A predecessor's dependency is not held" | No (byte equal, checked) |
| (new) | "The CLI takes fetch decisions from the library's typed classification", "Decisions come from the typed fetch classification" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| A transport failure during predecessor enumeration or package loading ... | body of "A transport failure or a missing dependency aborts the walk as a connectivity error", verbatim; body of "The CLI takes fetch decisions from the library's typed classification", verbatim |
| A package absent at a given published version is the scan's negative s... | body of "A package absent at a published version is the scan's negative signal", verbatim |
| Absent SHALL include a probed version the registry does not hold and a... | body of "A package absent at a published version is the scan's negative signal", verbatim |
| A dependency of the predecessor build that the registry does not hold ... | body of "A transport failure or a missing dependency aborts the walk as a connectivity error", verbatim |
| Source: enhancement 0011 ("`opm catalog publish` refuses a build that ... | body of "A transport failure or a missing dependency aborts the walk as a connectivity error", verbatim; body of "A package absent at a published version is the scan's negative signal", verbatim |
| The CLI SHALL take these decisions from the library's typed fetch clas... | body of "The CLI takes fetch decisions from the library's typed classification", verbatim |
| Source: enhancement 0021 ("A beta line reaches GA only when its exit c... | body of "The CLI takes fetch decisions from the library's typed classification", verbatim |

Lengths (old, new): old 876; new "A transport failure or a missing dependency aborts the walk as a connectivity error" 390; "A package absent at a published version is the scan's negative signal" 331; "The CLI takes fetch decisions from the library's typed classification" 428.

### Strength table of cli-B2

Plain sentences that carry an RFC 2119 keyword in the new text and had none in the old text:

| Old sentence | New body | Word added |
| --- | --- | --- |
| An instance deploys the operator when any one of these holds, read from its record without rendering: it is na... | "An instance deploys the operator when one of three signals holds" | "any one of these holds" reads "any one of these SHALL hold" |
| With `--wait` the command also waits, bounded, for the deleted resources to disappear ("Instance delete waits ... | "With --wait the command also waits for the deleted resources to disappear" | "the command also waits" reads "the command SHALL also wait" |
| Two selection modes exist, and the caller chooses between them explicitly: | "Release and prerelease are the two selection modes" | "exist," reads "SHALL exist," |
| The write-if-absent seeding of a missing Platform is unchanged: it happens only when no CR exists, and it seed... | "The write-if-absent seeding of a missing Platform is unchanged" | "is unchanged:" reads "SHALL be unchanged:" |
| - The same generator serves the deps fallback of every other render (see "Renders fall back to their own deps"... | "The same generator serves every other render" | "serves" reads "SHALL serve" |
| Every source resolves to a platform module directory the kernel acquires; the CR source is generated into one ... | "Every platform source resolves to a platform module directory" | "Every source resolves to" reads "Every source SHALL resolve to" |
| - `opm instance diff`, `opm instance apply` and `opm module apply` keep failing on an unreachable cluster: the... | "Instance diff, instance apply and module apply keep failing on an unreachable cluster" | "keep failing" reads "SHALL keep failing" |
| The gates: kernel loadability (modules only: the kernel's module loader accepts the tree; its refusal is surfa... | "The loadability, identity and derivation gates" | "The gates:" reads "The gates SHALL be:" |
| The gates: kernel loadability (modules only: the kernel's module loader accepts the tree; its refusal is surfa... | "The cue.mod, source, tag and package-name gates" | "The gates:" reads "The gates SHALL be:" |
| The gates: kernel loadability (modules only: the kernel's module loader accepts the tree; its refusal is surfa... | "The domain, local-override and already-published gates" | "The gates:" reads "The gates SHALL be:" |
| A registry that hands out bearer tokens and whose token endpoint refuses the caller is a refused credential li... | "A refusing token endpoint is a refused credential" | "is a refused credential" reads "SHALL be a refused credential" |
