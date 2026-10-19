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

The requirement "Ownership guard judges every apply and dry run" of `apply-pruning` is removed here (form c2, split into fourteen requirements). `openspec/specs/deploy/spec.md` cites it in the requirement "Create-namespace creates the namespace only after every check passed", in the words `(capability apply-pruning, "Ownership guard judges every apply and dry run")`. That requirement is in unit cli-B2 and is rewritten there. B2 must write the new name `The ownership guard judges every apply` (or the dry-run requirement `A dry run previews each refusal of the guard` where the text speaks of a dry run) into its rewritten text. Until B2 lands, that one citation is stale by name. Checked with `grep -rn --exclude-dir=archive` on 2026-10-19: it is the only reference outside the rewritten requirements.


### Strength table of cli-B1

Plain sentences that carry an RFC 2119 keyword in the new text and had none in the old text:

| Old sentence | New body | Word added |
| --- | --- | --- |
| `opm operator install` is the exception: it refuses on such a resource | "Operator install refuses a resource adopted by another instance" | "it refuses" reads "it SHALL refuse" |
