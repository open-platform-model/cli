## Context

The cli has 215 long requirements in 59 main specs. The rules below set out how each requirement is rewritten and checked. Nothing here changes the meaning of a requirement or scenario, and no code changes.

## Research & Decisions

### Which form rewrites a requirement
**Context**: A requirement over 500 characters can be shortened in four safe ways, and several unsafe ones pass the validator without keeping the rules (see the table below).
**Explored**: Archive output of each form in scratch copies (see the table below).
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
**Decision**: Option 1. One change per repo, with several units in sequence on one branch.
**Rationale**: Each section still satisfies both invariants of Principle VIII: it ends green under `task openspec:check` (non-strict until the gate section) and closes with its own commit. The sections are uniform batches, not ten different designs. The ten sections deviate from the "about five" wording of the repo rules.

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
- Every new ADDED requirement has at least one scenario with a body and states SHALL or MUST. `openspec/config.yaml` asks scenarios to show observable behavior and user-facing outcomes; a scenario added for a new requirement uses only words and values of the old text. A scenario that adds a case absent from the old text is not a rewrite and needs a decision.
- No spec has a placeholder Purpose today (checked on `origin/main`), so no unit writes Purpose text. A c2 or b rewrite does not create a spec, so none appears.

## The gate

Every unit runs the gate in a scratch copy of the folder `openspec/`, never in the worktree. The pinned CLI is 1.14.1 (`Taskfile.yml` `openspec:install`). The gate block runs in one shell call: archive `main` and the working copy into `$S/base` and `$S/openspec`; count WARNING lines of `validate --all --strict` before; `validate split-long-requirements --strict`; `archive split-long-requirements --yes`; `validate --all --strict` after; fail on any "Archive would refuse" line, on any ERROR item that `main` does not have (compared as a set of type, id and path, needs `jq`), and, for an earlier unit, when the WARNING count did not fall by N. N is the total of requirements rewritten so far by all units, 28 after section 1 and so on (the table above). Exit 0 only after `GATE PASS`.

- Each unit also runs `openspec validate <spec> --type spec --strict --json 2>&1` for its specs and reads that no rewritten requirement has a WARNING or an ERROR.
- Each unit checks every moved or kept scenario byte for byte against `main` with a script that prints `same`, `new`, `DIFF` or `LOST` for each scenario (old text from `origin/main`, new text from the delta), run in the repo root once per rewritten requirement. A `DIFF` or `LOST` line is a failure.
- Section 10 is the last unit (`LAST=1`): `validate split-long-requirements --strict` exits 0, `validate --specs --strict` exits 0 after archive in the scratch copy, no "Archive would refuse" line, no new ERROR item. Then, with the extras of section 9 in place, `task openspec:check` runs in the worktree and must exit 0. In the cli no other change is open, so `validate --all --strict` also exits 0 at that point.
- This is the same check that CI runs after section 9 (`validate --all --strict` through `task openspec:check`, jobs "OpenSpec Specs (strict)").

## Risks and trade-offs

- A rewrite can drop a clause without any tool error (form c2 and moved clauses). The mitigations are the ledger, the byte check and the review checklist, run per unit by a reviewer.
- A long MODIFIED passes `validate <change> --strict` and fails only after archive. The gate catches it; a pre-archive check alone does not.
- A stale reference to a removed header is invisible to OpenSpec. The grep rule above is the only guard.
- Another change that lands on a spec of a later batch before that batch runs would make its MODIFIED blocks stale. A unit re-reads the spec from `origin/main` first; `main` is merged into the branch before a unit if `main` moved.

## Open questions

None.

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
| An operator that lacks one of these CRDs SHALL be reported as not read... | body of "The running-operator check reads no Namespace and reports a missing CRD as not ready", old words "these CRDs"; the parenthesis names the sibling requirement that lists the CRDs |
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
| The error SHALL name the instance and its namespace, carry the cause o... | body of "A failed record read names its cause", verbatim, led by the condition "When the read of the instance's `ModuleInstance` record fails with any error other than NotFound," copied from the first old sentence; its scenario "A failed record read names the instance and its cause" restates the clause (no new case) |
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
| No step SHALL filter the stale set after it is computed: the stale set... | body of "No step of the apply flow filters the stale set", verbatim; its scenario restates the clause |
| (1) render resources, (2) compute manifest digest, (3) compute change ... | reworded, nearest body: "The apply flow reads, checks and applies before it prunes"; the subject or a condition is restated so that the body reads alone, no rule changed |
| (detail) Steps (1) to (6) of the sequence | body of "The apply flow reads, checks and applies before it prunes", step text verbatim |
| (detail) Steps (7a), (7b) and (9) | body of "The apply flow prunes only after every apply succeeded"; the old sentence "The apply flow SHALL follow this sequence" supplies the keyword and the subject, step numbers and "step 4", "step 7a" as in the old text; the parenthesis "(steps 1 to 6 as listed in requirement ...)" names the sibling that holds steps 1 to 6 |
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

The requirement "Ownership guard judges every apply and dry run" of `apply-pruning` is removed in unit cli-B1 (form c2, split into fourteen requirements). `openspec/specs/deploy/spec.md` cited it in the requirement "Create-namespace creates the namespace only after every check passed", in the words `(capability apply-pruning, "Ownership guard judges every apply and dry run")`. Unit cli-B2 resolved this: the first new requirement of that split, "Create-namespace creates a missing namespace only after every check that can refuse the apply passed", cites `(capability `apply-pruning`, "The ownership guard judges every apply")`. Checked with `grep -rn --exclude-dir=archive` on 2026-10-19: no other reference outside the rewritten requirements.

## Ledger entries of cli-B2 (deploy, platform-resolution, artifact-publishing)

One subsection per rewritten requirement, 28 in all. Every new body is built from whole sentences of the old body, copied unchanged. A clause row says "verbatim" for a sentence that appears unchanged. A row that names a substitution lists each changed word pair; a lowered first letter after a copied condition is named as such. A list marker `- ` of an old bullet is dropped where its sentence opens a body or follows a sentence that does not end in a colon. Old scenario names are all kept; the byte check ran once per rewritten requirement and printed no `DIFF` and no `LOST`.

Form per requirement: the names below are cited by other requirements, so they keep their header as form b parents (MODIFIED, all old scenarios in place, split-offs ADDED with restatement scenarios): in `platform-resolution` "Acquisition mirrors the operator", "The cluster arm reads the effective registry", "Module commands render against the module's deps", "Platform source precedence by command", "Renders fall back to their own deps"; in `deploy` "Instance delete re-checks live ownership before each delete" and "Instance delete follows the shared deletion plan" (both cited by `operator-lifecycle`). The other 21 are form c2.

Exceptions to the one-sentence rule, all with every word copied from the old requirement:

- Three old sentences are longer than 500 characters and are cut inside the sentence, each part verbatim, the subject repeated: `artifact-publishing` "One pipeline, two entry points" (562) and "The compatibility gate" (531); `deploy` "Instance delete keeps PersistentVolumeClaims unless delete-data is set" (610). The gate list of "Gates and refusal catalog" (769) stays whole as the THEN line of the scenario "The ten gates of the gate list", and the body names that scenario.
- A condition or subject copied from the same old requirement leads a body where the old sentence relied on a sibling (rows "the part from ... on"). Where the old content cannot fit, a body names a sibling by its exact header (the named-sibling extension): "These, as listed in requirement ...", "one of the checks listed in requirement ...", "no such field (the field of requirement ...)", "In every other case than the one of requirement ...".
- Cites of renamed requirements are updated to the new header: "Instance delete keeps PersistentVolumeClaims unless delete-data is set" and "Instance delete waits for the deleted resources when wait is set" in `deploy`, "Ownership guard judges every apply and dry run" (see the section above).
- A `Source:` line is repeated whole on each split block where it fits within 500 characters. Where it did not fit, unit cli-B2 first carried a short line `Source: enhancement 0012.` or `Source: enhancement 0021.`. Unit cli-B3 put the supporting title back on the four blocks that had it: `deploy` "An object with no UUID label is judged without the UUID comparison", "A resource that fails the ownership check is left behind and does not fail the delete" and the new "The closing line of a delete gives the number of resources left behind" (the deletion sequence title), "A recorded resource that the command did not find is not in the deletion plan" (the deletion sequence title); `platform-resolution` "The operator render narrows the platform-source precedence for the operator's own instance only" (the install artifact title). The three `deploy` bodies were cut so that they fit with the title, as the ledgers below say. "Instance delete re-checks live ownership before each delete" keeps one quote in the parent and the other in the ownership block. No split block is left without a `Source:` line where its old requirement had one.
- A subject copied from the same old requirement stands beside "the command" in twelve `deploy` bodies that name no command otherwise: "the command (`opm instance apply` and `opm module apply`)" in "After the first stage the apply waits until the CustomResourceDefinitions are established" and "A wait that does not complete fails the apply"; "the command (`opm instance delete`)" in the bodies of the guard and `--wait` blocks and in "--delete-data on an operator-managed instance warns and changes nothing". No word is dropped.
- "A resource that fails the ownership check is left behind and does not fail the delete" and "The closing line of a delete gives the number of resources left behind" name the conditions by their requirement ("only as listed in requirement ..."), because the three conditions and the sentence do not fit 500 characters with the `Source:` line. The old sentence "A resource that fails the check SHALL be left behind and listed with the reason, and SHALL NOT count as a failure; the closing line SHALL give the number of resources left behind." is cut at its semicolon into two requirements, with the full `Source:` line on each; the second part starts with a capital letter ("The closing line").
- "An object with no UUID label is judged without the UUID comparison" opens with the pointer sentence of its sibling "A resource that fails the ownership check is left behind and does not fail the delete" and carries the deletion sequence title, because the guard title does not fit with the pointer.
- "The closure reads a directory-replaced module's requirements from that directory" (new, `platform-resolution`) holds the old sentence "The closure SHALL read a directory-replaced module's requirements ..." with the copied condition and names the closure requirement; "The dependency list of the generated platform is the closure of the pins" names the requirement that lists the generated platform and its pins, because its subject "Its" and "those pins" came from the bullets before.
- "Any other failure answer reads registry operation failed" names the two requirements whose answers "any other" excludes.

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

Form: b (MODIFIED parent with its header and all scenarios, ADDED splits). New requirements: "keep", "A tracked resource is deleted only when its live labels and annotation show the instance owns it", "An object with no UUID label is judged without the UUID comparison", "A resource that fails the ownership check is left behind and does not fail the delete", "The closing line of a delete gives the number of resources left behind", "Each delete carries a precondition on the UID of the object that was read", "A resource that is already gone is no failure and any other read error is".

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
| (new) | "The closing line of a delete gives the number of resources left behind", "The closing line gives the number of resources left behind" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Each delete carries a precondition on the UID of the object that was read", "A refused precondition is a failure" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A resource that is already gone is no failure and any other read error is", "Already gone counts neither as deleted nor as a failure" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| Before deleting each tracked resource, `opm instance delete` SHALL rea... | body of "Instance delete re-checks live ownership before each delete", verbatim; body of "An object with no UUID label is judged without the UUID comparison", the first sentence replaced by "Before deleting each tracked resource, `opm instance delete` SHALL delete the resource only as listed in requirement ..."; body of "A resource that fails the ownership check is left behind and does not fail the delete" and of "The closing line of a delete gives the number of resources left behind", the first sentence replaced by "Before deleting each tracked resource, `opm instance delete` SHALL delete the resource only as listed in requirement ..."; body of "Each delete carries a precondition on the UID of the object that was read", the part from "Before deleting each tracked r..." on, verbatim; body of "A resource that is already gone is no failure and any other read error is", the part from "Before deleting each tracked r..." on, verbatim |
| It SHALL delete the resource only when the live `app.kubernetes.io/man... | body of "A tracked resource is deleted only when its live labels and annotation show the instance owns it", with "It SHALL delete the resource" reads "`opm instance delete` SHALL delete each tracked resource"; body of "An object with no UUID label is judged without the UUID comparison", with "It SHALL" reads "SHALL" |
| An object with no UUID label SHALL be judged without the UUID comparis... | body of "An object with no UUID label is judged without the UUID comparison", verbatim |
| A resource that fails the check SHALL be left behind and listed with t... | body of "A resource that fails the ownership check is left behind and does not fail the delete", verbatim as far as "SHALL NOT count as a failure", with the semicolon replaced by a period; body of "The closing line of a delete gives the number of resources left behind", the part after the semicolon, "the closing line" reads "The closing line" |
| Each delete SHALL carry a precondition on the UID of the object that w... | body of "Each delete carries a precondition on the UID of the object that was read", verbatim |
| A resource that is already gone, whether its re-read or its delete cal... | body of "A resource that is already gone is no failure and any other read error is", verbatim |
| Any other read error SHALL count as a failure for that resource, so th... | body of "A resource that is already gone is no failure and any other read error is", verbatim |
| Source: enhancement 0012 ("The library owns the whole deletion sequenc... | body of "Instance delete re-checks live ownership before each delete", the part from "Source: enhancement 0012 (" on, verbatim; body of "A tracked resource is deleted only when its live labels and annotation show the instance owns it", the part from "Source: enhancement 0012 (" on, verbatim; body of "A tracked resource is deleted only when its live labels and annotation show the instance owns it", the part from ""The apply guard runs" on, verbatim; body of "An object with no UUID label is judged without the UUID comparison", body of "A resource that fails the ownership check is left behind and does not fail the delete" and of "The closing line of a delete gives the number of resources left behind", the deletion sequence title, verbatim; body of "Each delete carries a precondition on the UID of the object that was read", the part from "Source: enhancement 0012 (" on, verbatim; body of "A resource that is already gone is no failure and any other read error is", the part from "Source: enhancement 0012 (" on, verbatim |

Lengths (old, new): old 1645; new "Instance delete re-checks live ownership before each delete" 401; "A tracked resource is deleted only when its live labels and annotation show the instance owns it" 497; "An object with no UUID label is judged without the UUID comparison" 487; "A resource that fails the ownership check is left behind and does not fail the delete" 462; "The closing line of a delete gives the number of resources left behind" 412; "Each delete carries a precondition on the UID of the object that was read" 438; "A resource that is already gone is no failure and any other read error is" 490.

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

Lengths (old, new): old 1203; new "The apply goes in two stages in ascending kind-class weight" 375; "The first stage holds every CustomResourceDefinition and every Namespace" 265; "After the first stage the apply waits until the CustomResourceDefinitions are established" 391; "A wait that does not complete fails the apply" 412; "A resource that fails to apply does not stop the rest" 217.

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
| `opm instance delete` SHALL NOT delete an instance that deploys the op... | body of "Instance delete does not delete an instance that deploys the operator", verbatim; body of "The guard runs before any object is deleted, on both branches and on a dry run", the part from "`opm instance delete` SHALL NO..." on, verbatim; body of "Instance delete offers no flag that removes finalizers", the part from "`opm instance delete` SHALL NO..." on, verbatim; body of "The yes flag and its deprecated alias do not bypass either refusal", verbatim; body of "Every other instance's delete is unchanged by the guard", the part from "`opm instance delete` SHALL NO..." on, verbatim |
| An instance deploys the operator when any one of these holds, read fro... | body of "An instance deploys the operator when one of three signals holds", with "any one of these holds" reads "any one of these SHALL hold" |
| These are the same signals by which the operator recognises the instan... | body of "Instance delete does not delete an instance that deploys the operator", with "These are the same signals" reads "These, as listed in requirement "An instance deploys the operator when one of three signals holds", are the same signals" |
| The guard SHALL run after the target instance is resolved and before a... | body of "The guard runs before any object is deleted, on both branches and on a dry run", verbatim |
| - When the target is operator-owned, the command SHALL refuse with exi... | body of "An operator-owned target is refused with exit code 2", with "When the target is operator-owned" reads "For an instance that deploys the operator, when the target is operator-owned" |
| The operator never reconciles, finalizes or prunes the instance that d... | body of "An operator-owned target is refused with exit code 2", verbatim |
| - Otherwise the command SHALL list `ModuleInstance` resources cluster-... | body of "The guard lists ModuleInstance resources cluster-wide when the target is not operator-owned", with "Otherwise the command" reads "For an instance that deploys the operator that is not operator-owned, the command"; body of "A failed cluster-wide list fails closed", with "Otherwise the command" reads "For an instance that deploys the operator that is not operator-owned, the command" |
| When any carries `opmodel.dev/cleanup`, the command SHALL refuse with ... | body of "An armed instance blocks the delete with exit code 2", with "When any carries `opmodel.dev/cleanup`" reads "For an instance that deploys the operator that is not operator-owned, when any `ModuleInstance` in the cluster carries `opmodel.dev/cleanup`" |
| Deleting the operator while it still owes cleanup leaves every armed i... | body of "The guard lists ModuleInstance resources cluster-wide when the target is not operator-owned", verbatim |
| When the cluster-wide list fails, the command SHALL fail closed with t... | body of "A failed cluster-wide list fails closed", verbatim |
| `opm instance delete` SHALL offer no flag that removes finalizers and ... | body of "Instance delete offers no flag that removes finalizers", verbatim |
| `--yes` SHALL keep its one meaning, skipping the confirmation prompt, ... | body of "The yes flag and its deprecated alias do not bypass either refusal", verbatim |
| Every other instance's delete SHALL be unchanged by this requirement. | body of "Every other instance's delete is unchanged by the guard", verbatim |

Lengths (old, new): old 2453; new "An instance deploys the operator when one of three signals holds" 416; "Instance delete does not delete an instance that deploys the operator" 451; "The guard runs before any object is deleted, on both branches and on a dry run" 408; "An operator-owned target is refused with exit code 2" 422; "The guard lists ModuleInstance resources cluster-wide when the target is not operator-owned" 291; "An armed instance blocks the delete with exit code 2" 458; "A failed cluster-wide list fails closed" 342; "Instance delete offers no flag that removes finalizers" 306; "The yes flag and its deprecated alias do not bypass either refusal" 408; "Every other instance's delete is unchanged by the guard" 276.

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
| With `--create-namespace`, `opm instance apply` and `opm module apply`... | body of "Create-namespace creates a missing namespace only after every check that can refuse the apply passed", with "(capability `apply-pruning`, "Ownership guard judges every apply and dry run")" reads "(capability `apply-pruning`, "The ownership guard judges every apply")"; body of "While the namespace is missing the apply sends no read into it", the part from "With `--create-namespace`, `op..." on, verbatim; body of "The status-permission check runs unchanged with a missing namespace", the part from "With `--create-namespace`, `op..." on, verbatim; body of "A failed read or create of the namespace stops the apply with the exit code of its cause", the part from "With `--create-namespace`, `op..." on, verbatim; body of "A namespace that exists at the create stops the apply with exit code 1", the part from "With `--create-namespace`, `op..." on, verbatim; body of "A dry run creates no namespace and reports that it would", the part from "With `--create-namespace`, `op..." on, verbatim |
| The namespace SHALL be created before the first rendered resource is a... | body of "Create-namespace creates a missing namespace only after every check that can refuse the apply passed", verbatim |
| An apply that one of these checks refuses SHALL NOT create the namespa... | body of "A check that refuses the apply leaves the cluster unchanged and creates no namespace", with "one of these checks" reads "one of the checks listed in requirement "Create-namespace creates a missing namespace only after every check that can refuse the apply passed"" |
| While the namespace is missing, the apply SHALL treat it as holding no... | body of "While the namespace is missing the apply sends no read into it", verbatim |
| Rendered cluster-scoped objects and objects in other namespaces SHALL ... | body of "While the namespace is missing the apply sends no read into it", verbatim |
| The status-permission check SHALL run unchanged; it asks about the nam... | body of "The status-permission check runs unchanged with a missing namespace", verbatim |
| A failure of the first read (whether the namespace exists) SHALL stop ... | body of "A failed read or create of the namespace stops the apply with the exit code of its cause", verbatim |
| A failure of the create SHALL stop the apply with the exit code of its... | body of "A failed read or create of the namespace stops the apply with the exit code of its cause", verbatim |
| When the namespace turns out to exist at the create although the first... | body of "A namespace that exists at the create stops the apply with exit code 1", verbatim |
| A dry run SHALL create nothing and SHALL report first that the namespa... | body of "A dry run creates no namespace and reports that it would", verbatim |
| (note) | "Create-namespace creates a missing namespace only after every check that can refuse the apply passed": The cite of the removed name of apply-pruning is updated. |

Lengths (old, new): old 1663; new "Create-namespace creates a missing namespace only after every check that can refuse the apply passed" 472; "A check that refuses the apply leaves the cluster unchanged and creates no namespace" 234; "While the namespace is missing the apply sends no read into it" 484; "The status-permission check runs unchanged with a missing namespace" 242; "A failed read or create of the namespace stops the apply with the exit code of its cause" 381; "A namespace that exists at the create stops the apply with exit code 1" 377; "A dry run creates no namespace and reports that it would" 217.

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

Lengths (old, new): old 2409; new "Instance delete keeps a tracked PersistentVolumeClaim unless `--delete-data` is set" 234; "Each kept claim is listed on its own line at the informational level" 361; "A kept claim does not count as a failure" 319; "The closing output counts the kept claims and names the command that deletes them" 321; "With --delete-data a tracked claim is deleted under the live-ownership check" 273; "The kept-claim rule covers the core group only and no StatefulSet template claim" 435; "--delete-data on an operator-managed instance warns and changes nothing" 401; "The kept-claim rule of the CLI is not claimed for an operator-managed instance" 273; "An operator-managed delete that may have kept claims does not claim a full prune" 404; "The closing output of an operator-managed delete that may have kept claims names them" 494; "Every other operator-managed delete closes as a prune of the tracked resources" 208.

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
| When `spec.prune` is set and the inventory tracks at least one Persist... | body of "With prune set and a tracked claim the prompt says one of four things decided by the CRD", verbatim; body of "The prompt says the operator deletes the claims and their data when the CRD has no policy field or the policy is Delete", the part from "When `spec.prune` is set and t..." on, verbatim; body of "The prompt says the operator keeps the claims when the CRD has the field and the policy is Keep, absent or another value", the part from "When `spec.prune` is set and t..." on, verbatim; body of "The prompt does not say whether claims are kept when the CRD has no readable schema", the part from "When `spec.prune` is set and t..." on, verbatim |
| The command SHALL take that from the CRD its operator-readiness check ... | body of "With prune set and a tracked claim the prompt says one of four things decided by the CRD", verbatim |
| When the CRD has no such field, the operator does not know it: the pro... | body of "The prompt says the operator deletes the claims and their data when the CRD has no policy field or the policy is Delete", with "When the CRD has no such field" reads "When the installed `ModuleInstance` CRD has no `spec.dataPolicy` field" |
| When `spec.dataPolicy` is `Delete`, it SHALL say the same and name the... | body of "The prompt says the operator deletes the claims and their data when the CRD has no policy field or the policy is Delete", verbatim |
| When the CRD has the field and `spec.dataPolicy` is `Keep`, absent, or... | body of "The prompt says the operator keeps the claims when the CRD has the field and the policy is Keep, absent or another value", with "it SHALL say that the operator deletes" reads "the prompt SHALL say that the operator deletes" |
| When the CRD has no schema that can be read and `spec.dataPolicy` is n... | body of "The prompt does not say whether claims are kept when the CRD has no readable schema", with "it SHALL NOT say that claims" reads "the prompt SHALL NOT say that claims" |
| The prompt SHALL show the value of `spec.dataPolicy` as it is, or say ... | body of "The prompt shows the value of spec.dataPolicy as it is", verbatim |
| The dry run, and the run with `--yes`, SHALL state the same outcome fo... | body of "With prune set and a tracked claim the prompt says one of four things decided by the CRD", verbatim |
| An instance that has no record SHALL be reported as not found before a... | body of "Instance delete reads the instance record before it asks for confirmation", verbatim |
| Every check that can refuse the delete SHALL run before the prompt: th... | body of "Every check that can refuse the delete runs before the prompt", verbatim |
| A refused delete SHALL NOT ask for confirmation. | body of "Every check that can refuse the delete runs before the prompt", verbatim |

Lengths (old, new): old 2802; new "Instance delete reads the instance record before it asks for confirmation" 166; "The prompt names the claims that delete-data deletes" 242; "Without delete-data the prompt says PersistentVolumeClaims are kept" 107; "The prompt for an operator-managed instance says so and says what the operator does" 174; "The prompt for an operator-managed instance without prune says the operator leaves the tracked resources running" 273; "The prompt for an operator-managed instance with prune set and no claim says nothing about claims" 363; "With prune set and a tracked claim the prompt says one of four things decided by the CRD" 440; "The prompt says the operator deletes the claims and their data when the CRD has no policy field or the policy is Delete" 473; "The prompt says the operator keeps the claims when the CRD has the field and the policy is Keep, absent or another value" 451; "The prompt does not say whether claims are kept when the CRD has no readable schema" 422; "The prompt shows the value of spec.dataPolicy as it is" 179; "Every check that can refuse the delete runs before the prompt" 288.

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
| `opm instance delete` of a CLI-owned instance SHALL take every action ... | body of "Instance delete follows the shared deletion plan", verbatim; body of "A recorded resource that the command did not find is not in the deletion plan", the first clause up to "the operator" with a period added (the same lead as the bodies below; the "built from" clause stays in the parent); body of "The command acts on a tracked resource only as the deletion plan names", the part from "`opm instance delete` of a CLI..." on, verbatim; body of "The deletion plan does not hold a PersistentVolumeClaim that the command keeps", the part from "`opm instance delete` of a CLI..." on, verbatim; body of "The ModuleInstance record is deleted only when the plan's release verdict allows it", the part from "`opm instance delete` of a CLI..." on, verbatim; body of "A dry run follows the same plan and deletes nothing", the part from "`opm instance delete` of a CLI..." on, verbatim; body of "The output lines and exit codes stay as the other requirements state them", the part from "`opm instance delete` of a CLI..." on, verbatim |
| A recorded resource that the command did not find when it read the ins... | body of "A recorded resource that the command did not find is not in the deletion plan", verbatim |
| It SHALL read a tracked resource, delete it or leave it in place only ... | body of "The command acts on a tracked resource only as the deletion plan names", verbatim |
| The plan SHALL NOT hold a `PersistentVolumeClaim` that the command kee... | body of "The deletion plan does not hold a PersistentVolumeClaim that the command keeps", with "("Instance delete keeps PersistentVolumeClaims unless delete-data is set")" reads "("Instance delete keeps a tracked PersistentVolumeClaim unless `--delete-data` is set")" |
| The command SHALL delete the `ModuleInstance` record only when the pla... | body of "The ModuleInstance record is deleted only when the plan's release verdict allows it", verbatim |
| A dry run SHALL follow the same plan, SHALL send no delete, and SHALL ... | body of "A dry run follows the same plan and deletes nothing", verbatim |
| The output lines, their wording and the exit codes of the command SHAL... | body of "The output lines and exit codes stay as the other requirements state them", verbatim |
| The operator-owned delete path SHALL NOT be affected. | body of "The output lines and exit codes stay as the other requirements state them", verbatim |
| Source: enhancement 0012 ("The library owns the whole deletion sequenc... | body of "Instance delete follows the shared deletion plan", verbatim; body of "The command acts on a tracked resource only as the deletion plan names", verbatim; body of "The deletion plan does not hold a PersistentVolumeClaim that the command keeps", verbatim; body of "The ModuleInstance record is deleted only when the plan's release verdict allows it", verbatim; body of "A dry run follows the same plan and deletes nothing", verbatim; body of "The output lines and exit codes stay as the other requirements state them", verbatim |

Lengths (old, new): old 1424; new "Instance delete follows the shared deletion plan" 456; "A recorded resource that the command did not find is not in the deletion plan" 419; "The command acts on a tracked resource only as the deletion plan names" 493; "The deletion plan does not hold a PersistentVolumeClaim that the command keeps" 441; "The ModuleInstance record is deleted only when the plan's release verdict allows it" 463; "A dry run follows the same plan and deletes nothing" 372; "The output lines and exit codes stay as the other requirements state them" 467.

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

Lengths (old, new): old 784; new "Instance delete reports a resource as deleted when the API server accepts its delete" 402; "Without --wait an accepted delete is no failure although the resource still exists" 225; "With --wait the command also waits for the deleted resources to disappear" 194.

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

Lengths (old, new): old 1665; new "Instance delete accepts --wait and without it nothing changes" 152; "Instance delete with `--wait` waits for the deleted resources to disappear" 495; "A resource is gone when its read returns NotFound or its UID differs" 212; "While it waits the command prints how many deleted resources are still terminating" 296; "The wait skips a resource the command did not delete" 269; "A timeout lists the resources still terminating and keeps the ModuleInstance record" 341; "--wait has no effect on a dry run, after a failed delete or on an operator-managed instance" 284.

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

Form: b (MODIFIED parent with its header and all scenarios, ADDED splits). New requirements: "keep", "The generated platform carries one enabled registry entry per catalog dependency", "The dependency list of the generated platform is the closure of the pins", "The closure reads a directory-replaced module's requirements from that directory", "The generated platform carries the module's replacements", "The deps are read from the module's acquired source", "The generated platform is cached under its content hash", "The provenance line names the module's deps", "The skew policy is neither named nor applied to the module's deps", "A module with no catalog dependency renders against an empty registry", "A pin that is not published fails at generation", "A refusal for unresolved demands adds a hint that names the platform flag", "The same generator serves every other render".

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
| (new) | "The closure reads a directory-replaced module's requirements from that directory", "A directory-replaced module is read from its directory" | New restatement scenario: only words of the old clause, no new case |
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
| Source: cli issue 229. | body of "Module commands render against the module's deps", verbatim; body of "The generated platform carries one enabled registry entry per catalog dependency", verbatim; body of "The dependency list of the generated platform is the closure of the pins", verbatim; body of "The generated platform carries the module's replacements", verbatim; body of "The deps are read from the module's acquired source", verbatim; body of "The generated platform is cached under its content hash", verbatim; body of "The provenance line names the module's deps", verbatim; body of "The skew policy is neither named nor applied to the module's deps", verbatim; body of "A module with no catalog dependency renders against an empty registry", verbatim; body of "A pin that is not published fails at generation", verbatim; body of "A refusal for unresolved demands adds a hint that names the platform flag", verbatim; body of "The same generator serves every other render", verbatim |
| - The generated platform SHALL carry one enabled `#registry` entry per... | body of "The generated platform carries one enabled registry entry per catalog dependency", first letter lowered after the condition, otherwise verbatim |
| - Its dependency list SHALL be the minimum-version-selection closure o... | body of "The dependency list of the generated platform is the closure of the pins", first letter lowered after the condition, otherwise verbatim |
| - Each `replaceWith` in the module's `cue.mod/local-module.cue` whose ... | body of "The generated platform carries the module's replacements", first letter lowered after the condition, otherwise verbatim |
| The closure SHALL read a directory-replaced module's requirements from... | body of "The closure reads a directory-replaced module's requirements from that directory", with the condition "When `opm module build` or `opm module vet` runs without `--platform`," copied in and "The closure" lowered; the sentence "The closure is that of requirement ..." names the closure requirement |
| - The deps SHALL be read from the module's acquired source, so a modul... | body of "The deps are read from the module's acquired source", first letter lowered after the condition, otherwise verbatim |
| A published module has no local module context, so nothing is carried ... | body of "The deps are read from the module's acquired source", verbatim |
| - The generated module SHALL be cached under `cache/platforms/` beside... | body of "The generated platform is cached under its content hash", first letter lowered after the condition, otherwise verbatim |
| - The provenance line SHALL name the source as the module's deps and l... | body of "The provenance line names the module's deps", first letter lowered after the condition, otherwise verbatim |
| The skew policy SHALL NOT be named or applied: every pinned path is at... | body of "The skew policy is neither named nor applied to the module's deps", first letter lowered after the condition, otherwise verbatim |
| - A module with no `opmodel.dev/catalogs/*` dependency SHALL render ag... | body of "A module with no catalog dependency renders against an empty registry", first letter lowered after the condition, otherwise verbatim |
| - A pin that is not published SHALL fail at generation naming the modu... | body of "A pin that is not published fails at generation", first letter lowered after the condition, otherwise verbatim |
| - When a render against this source is refused for unresolved demands,... | body of "A refusal for unresolved demands adds a hint that names the platform flag", verbatim |
| - The same generator serves the deps fallback of every other render (s... | body of "The same generator serves every other render", with "serves" reads "SHALL serve" |

Lengths (old, new): old 2420; new "Module commands render against the module's deps" 206; "The generated platform carries one enabled registry entry per catalog dependency" 441; "The dependency list of the generated platform is the closure of the pins" 437 (its sentence "The generated platform and its pins are as listed in requirement ..." names the pins); "The closure reads a directory-replaced module's requirements from that directory" 338; "The generated platform carries the module's replacements" 324; "The deps are read from the module's acquired source" 334; "The generated platform is cached under its content hash" 269; "The provenance line names the module's deps" 196; "The skew policy is neither named nor applied to the module's deps" 203; "A module with no catalog dependency renders against an empty registry" 367; "A pin that is not published fails at generation" 182; "A refusal for unresolved demands adds a hint that names the platform flag" 388; "The same generator serves every other render" 322.

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
| `opm operator install`, the CRDs-only form included, SHALL render the ... | body of "Operator install renders the operator module against a platform of its own pins", verbatim; body of "Operator install renders the same objects whatever the cluster Platform holds", the part from "`opm operator install`, the CR..." on, verbatim; body of "Operator install names the module's deps as the provenance source", the part from "`opm operator install`, the CR..." on, verbatim; body of "The operator render narrows the platform-source precedence for the operator's own instance only", the first clause "`opm operator install`, the CRDs-only form included, SHALL render the operator module" followed by "as listed in requirement "Operator install renders the operator module against a platform of its own pins"" |
| Every other command keeps its precedence under "Platform source preced... | body of "Operator install renders the operator module against a platform of its own pins", verbatim |
| The same module version and values SHALL render the same objects wheth... | body of "Operator install renders the same objects whatever the cluster Platform holds", verbatim |
| The provenance line SHALL name the module's deps as the source. | body of "Operator install names the module's deps as the provenance source", verbatim |
| This narrows the platform-source precedence of enhancement 0006 for th... | body of "The operator render narrows the platform-source precedence for the operator's own instance only", verbatim |
| Source: enhancement 0021 ("The operator's install artifact is a regist... | body of "Operator install renders the operator module against a platform of its own pins", verbatim; body of "Operator install renders the same objects whatever the cluster Platform holds", verbatim; body of "Operator install names the module's deps as the provenance source", verbatim; body of "The operator render narrows the platform-source precedence for the operator's own instance only", verbatim |

Lengths (old, new): old 911; new "Operator install renders the operator module against a platform of its own pins" 453; "Operator install renders the same objects whatever the cluster Platform holds" 419; "Operator install names the module's deps as the provenance source" 258; "The operator render narrows the platform-source precedence for the operator's own instance only" 470.

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

Form: c2 (REMOVED, ADDED under new names). New requirements: "Publish accumulates every refusal and refuses with exit code 2", "Each refusal follows one fixed shape", "One cause yields one refusal", "Publish evaluates the gates of the gate list".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Multiple defects reported in one pass | "Publish accumulates every refusal and refuses with exit code 2", "Multiple defects reported in one pass" | No (byte equal, checked) |
| Kernel refusal is one refusal | "One cause yields one refusal", "Kernel refusal is one refusal" | No (byte equal, checked) |
| Already published is always a refusal | "Publish evaluates the gates of the gate list", "Already published is always a refusal" | No (byte equal, checked) |
| Unowned domains are not policed | "Publish evaluates the gates of the gate list", "Unowned domains are not policed" | No (byte equal, checked) |
| (new) | "Each refusal follows one fixed shape", "A refusal follows the fixed shape" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Publish evaluates the gates of the gate list", "The ten gates of the gate list" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| Publish SHALL evaluate every evaluable gate, accumulate all refusals i... | body of "Publish accumulates every refusal and refuses with exit code 2", verbatim |
| Each refusal SHALL follow the fixed shape condition → evidence → conse... | body of "Each refusal follows one fixed shape", verbatim |
| One cause SHALL yield one refusal. | body of "One cause yields one refusal", verbatim |
| The gates: kernel loadability (modules only: the kernel's module loade... |  |
| (added words) | in "Publish evaluates the gates of the gate list": "The gates SHALL be those listed in scenario "The ten gates of the gate list".". The gate list is one sentence of 769 characters. It stays whole as the THEN line of a scenario; the body names that scenario. The sentence reads "The gates SHALL be:" there, with the keyword added. |

Lengths (old, new): old 1175; new "Publish accumulates every refusal and refuses with exit code 2" 132; "Each refusal follows one fixed shape" 237; "One cause yields one refusal" 34; "Publish evaluates the gates of the gate list" 77.

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

Lengths (old, new): old 2236; new "Publish exits 0, 2, 4, 3 or 1 by outcome" 334; "A failed already-published lookup does not hide locally derived refusals" 292; "A failed registry operation with no HTTP response reads registry unreachable" 277; "A 401 or 403 answer reads as refused credentials and exits 4" 433; "Any other failure answer reads registry operation failed" 488; "A 403 answer to the already-published lookup is not a failure" 455; "A refusing token endpoint is a refused credential" 466; "Publish with no registry configured exits 2" 304.

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
| A transport failure during predecessor enumeration or package loading ... | body of "A transport failure or a missing dependency aborts the walk as a connectivity error", verbatim |
| A package absent at a given published version is the scan's negative s... | body of "A package absent at a published version is the scan's negative signal", verbatim |
| Absent SHALL include a probed version the registry does not hold and a... | body of "A package absent at a published version is the scan's negative signal", verbatim |
| A dependency of the predecessor build that the registry does not hold ... | body of "A transport failure or a missing dependency aborts the walk as a connectivity error", verbatim |
| Source: enhancement 0011 ("`opm catalog publish` refuses a build that ... | body of "A transport failure or a missing dependency aborts the walk as a connectivity error", verbatim; body of "A package absent at a published version is the scan's negative signal", verbatim |
| The CLI SHALL take these decisions from the library's typed fetch clas... | body of "The CLI takes fetch decisions from the library's typed classification", with "these decisions" reads "the decisions of requirements "A transport failure or a missing dependency aborts the walk as a connectivity error" and "A package absent at a published version is the scan's negative signal"" |
| Source: enhancement 0021 ("A beta line reaches GA only when its exit c... | body of "The CLI takes fetch decisions from the library's typed classification", verbatim |

Lengths (old, new): old 876; new "A transport failure or a missing dependency aborts the walk as a connectivity error" 390; "A package absent at a published version is the scan's negative signal" 331; "The CLI takes fetch decisions from the library's typed classification" 441.

## Ledger entries of cli-B3 (release-workflow, kernel-render, deps-cascade, instance-init, authoring-commands)

One subsection per rewritten requirement, 28 in all. Every new body is built from whole sentences of the old body, copied unchanged. A clause row names where a sentence of the old body appears; the list "Words changed or added" under a requirement names each substitution, as the rewrite rules allow: a pronoun ("It", "That job", "Its") read as the subject copied from the same old requirement, a lowered or raised first letter, a closing period where a list marker or a semicolon stood, and an RFC 2119 keyword where the old sentence had none (also listed in the strength table). All 28 are form c2, because the old scenarios of each requirement belong to different new behaviors; no other change of the repo cites an old header (checked with `grep -rn --exclude-dir=archive` over the repo on 2026-10-19), except where this section says so. Old scenario names are all kept; the byte check ran once per rewritten requirement and printed no `DIFF` and no `LOST`.

Rules applied beyond unit cli-B2:

- A `Source:` line is repeated whole on each split block. Three blocks of `instance-init` carry a shortened line that drops whole titles only and keeps the id: "The initValues source", "On success init reports the resolution, the values source, the files and a closing vet line" and "Init exit codes are those of module init" (each keeps the titles that support its rule). The `deps-cascade` sentence "The source is workspace RELEASING.md, sections ..." is not a `Source:` line; it is repeated, unchanged, at the end of every block of "The cascade task moves the cli's upstream pins to published versions", the file list of "The catalog and core pins cover a fixed set of cue.mod/module.cue files" included, which is cut after `internal/instinit/testdata/initvalues` so that both parts hold it.
- A sentence over 500 characters, or a sentence that cannot hold its `Source:` line, is cut at a list or clause boundary, each part verbatim, the subject or condition repeated: `release-workflow` "Template modules publish on release" (the gate list, 555), "A breaking change is marked by a bang in the pull request title" (the footer sentence, 568), "Downstream repos are notified after a release is published" (the caller-owned list); `kernel-render` "All renders go through the library kernel" (the identity check, 546) and "A colliding platform is refused naming the entries to disable" (the first sentence with its `Source:` line); `deps-cascade` "Title and body tasks describe the cascade diff" (the `pins.sh` list, 552) and "The cascade task moves the cli's upstream pins to published versions" (the file list of the consistent set); `instance-init` "Init writes all or nothing, and only where a standalone package belongs" (the two conditions).
- Where "Otherwise" or "Both warnings" had its antecedent in a sibling, the body names that sibling by its exact header ("In every other case than ..."): `release-workflow` "The goreleaser job checks out the tag and runs goreleaser", `kernel-render` two blocks, `instance-init` three blocks.
- `instance-init` "The values.cue file declares one values field" cited the removed header "Values start from the author's intended source" and now cites "values.cue is filled from the first applicable source and the report names it" (grep over the repo outside `openspec/changes/archive/` on 2026-10-19 found no other reference).
- `release-workflow` "The script runs the identity, layout, tidy, vet and zip gates on every template tree" cites the `pr-workflow` requirement "Template publish gates run dry-run" by name, as the old text did. That header is in the unit cli-B7 batch; unit cli-B7 must keep the cite true if it renames that header.

### Ledger: release-workflow, Binaries publish only when a release was created

Form: c2 (REMOVED, ADDED under new names). New requirements: "The goreleaser job runs only for a created release or a manual run", "The goreleaser job needs a draft release", "The goreleaser job fails without a release for the tag", "A failed draft lookup is no published release", "The goreleaser job needs publish-templates to succeed", "The goreleaser job runs in a concurrency group per tag", "The goreleaser job checks out the tag and runs goreleaser", "A manual run recovers a draft release and never changes a published one".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Skipped on a non-release push | "The goreleaser job runs only for a created release or a manual run", "Skipped on a non-release push" | No (byte equal, checked) |
| Runs on the release tag | "The goreleaser job checks out the tag and runs goreleaser", "Runs on the release tag" | No (byte equal, checked) |
| Manual recovery builds an existing tag | "A manual run recovers a draft release and never changes a published one", "Manual recovery builds an existing tag" | No (byte equal, checked) |
| Manual recovery refuses a published release | "The goreleaser job needs a draft release", "Manual recovery refuses a published release" | No (byte equal, checked) |
| Manual recovery for a tag without a release | "The goreleaser job fails without a release for the tag", "Manual recovery for a tag without a release" | No (byte equal, checked) |
| Template failure keeps the release a draft | "The goreleaser job needs publish-templates to succeed", "Template failure keeps the release a draft" | No (byte equal, checked) |
| (new) | "A failed draft lookup is no published release", "A failed lookup is not reported as published" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The goreleaser job runs in a concurrency group per tag", "The job runs in a concurrency group per tag" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The `goreleaser` job SHALL `need` release-please and publish-templates... | body of "The goreleaser job runs only for a created release or a manual run", verbatim |
| Before building it SHALL read the draft state of the GitHub Release fo... | body of "The goreleaser job needs a draft release", the part of this sentence it holds, verbatim or as the words changed below say |
| When the release is published it SHALL fail with a message that the re... | body of "The goreleaser job needs a draft release", the part of this sentence it holds, verbatim or as the words changed below say |
| When no GitHub Release exists for the tag it SHALL fail with a message... | body of "The goreleaser job fails without a release for the tag", the part of this sentence it holds, verbatim or as the words changed below say |
| When the lookup itself fails for any other reason it SHALL fail with t... | body of "A failed draft lookup is no published release", the part of this sentence it holds, verbatim or as the words changed below say |
| After the draft check and before building, it SHALL fail unless the pu... | body of "The goreleaser job needs publish-templates to succeed", the part of this sentence it holds, verbatim or as the words changed below say |
| It SHALL run in a concurrency group per tag that never cancels a runni... | body of "The goreleaser job runs in a concurrency group per tag", the part of this sentence it holds, verbatim or as the words changed below say |
| Otherwise it SHALL check out `tag_name`, or the `tag` input on a manua... | body of "The goreleaser job checks out the tag and runs goreleaser", the part of this sentence it holds, verbatim or as the words changed below say |
| The manual run is the recovery for a draft release whose tag exists wi... | body of "A manual run recovers a draft release and never changes a published one", verbatim |
| Source: enhancement 0021 ("Release tags are immutable"). | body of "The goreleaser job runs only for a created release or a manual run", verbatim; body of "The goreleaser job needs a draft release", verbatim; body of "The goreleaser job fails without a release for the tag", verbatim; body of "A failed draft lookup is no published release", verbatim; body of "The goreleaser job needs publish-templates to succeed", verbatim; body of "The goreleaser job runs in a concurrency group per tag", verbatim; body of "The goreleaser job checks out the tag and runs goreleaser", verbatim; body of "A manual run recovers a draft release and never changes a published one", verbatim |

Words changed or added:
- "The goreleaser job needs a draft release": "Before building it SHALL read the draft state of the GitHub Release for the tag and SHALL ..." reads "Before building, the `goreleaser` job SHALL read the draft state of the GitHub Release for..." ("it" reads "the `goreleaser` job")
- "The goreleaser job needs a draft release": "When the release is published it SHALL fail with a message that the release is already pub..." reads "When the release is published the `goreleaser` job SHALL fail with a message that the rele..." ("it" reads "the `goreleaser` job")
- "The goreleaser job fails without a release for the tag": "When no GitHub Release exists for the tag it SHALL fail with a message that names the tag ..." reads "When no GitHub Release exists for the tag the `goreleaser` job SHALL fail with a message t..." ("it" reads "the `goreleaser` job")
- "A failed draft lookup is no published release": "Before building it SHALL read the draft state of the GitHub Release for the tag and SHALL ..." reads "Before building, the `goreleaser` job SHALL read the draft state of the GitHub Release for..." (the first clause of the sentence up to "and", with a period; "it" reads "the `goreleaser` job")
- "A failed draft lookup is no published release": "When the lookup itself fails for any other reason it SHALL fail with that error and SHALL ..." reads "When the lookup itself fails for any other reason the `goreleaser` job SHALL fail with tha..." ("it" reads "the `goreleaser` job")
- "The goreleaser job needs publish-templates to succeed": "After the draft check and before building, it SHALL fail unless the publish-templates job ..." reads "After the draft check and before building, the `goreleaser` job SHALL fail unless the publ..." ("it" reads "the `goreleaser` job")
- "The goreleaser job runs in a concurrency group per tag": "It SHALL run in a concurrency group per tag that never cancels a running job." reads "The `goreleaser` job SHALL run in a concurrency group per tag that never cancels a running..." ("It" reads "The `goreleaser` job")
- "The goreleaser job checks out the tag and runs goreleaser": "Otherwise it SHALL check out `tag_name`, or the `tag` input on a manual run, with `fetch-d..." reads "In every other case than those of requirements "The goreleaser job needs a draft release",..." ("Otherwise it" reads the quoted case)

Lengths (old, new): old 1585; new "The goreleaser job runs only for a created release or a manual run" 378; "The goreleaser job needs a draft release" 370; "The goreleaser job fails without a release for the tag" 245; "A failed draft lookup is no published release" 301; "The goreleaser job needs publish-templates to succeed" 279; "The goreleaser job runs in a concurrency group per tag" 152; "The goreleaser job checks out the tag and runs goreleaser" 478; "A manual run recovers a draft release and never changes a published one" 249.

### Ledger: release-workflow, Goreleaser produces per-platform archives, checksums and changelog

Form: c2 (REMOVED, ADDED under new names). New requirements: "Goreleaser builds five platform archives and attaches them with the checksums", "Goreleaser groups the changelog by commit type", "Goreleaser attaches to release-please's draft release and publishes it last", "The goreleaser configuration sets no draft, no renamed release and no replaced draft", "Goreleaser sets the prerelease flag from the tag", "Goreleaser keeps the release notes release-please wrote".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Five archives and checksums attached | "Goreleaser builds five platform archives and attaches them with the checksums", "Five archives and checksums attached" | No (byte equal, checked) |
| Changelog grouped by type | "Goreleaser groups the changelog by commit type", "Changelog grouped by type" | No (byte equal, checked) |
| Beta tag flagged as a pre-release | "Goreleaser sets the prerelease flag from the tag", "Beta tag flagged as a pre-release" | No (byte equal, checked) |
| Release published only with all assets | "Goreleaser attaches to release-please's draft release and publishes it last", "Release published only with all assets" | No (byte equal, checked) |
| (new) | "The goreleaser configuration sets no draft, no renamed release and no replaced draft", "The configuration leaves the draft, name and replace keys unset" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Goreleaser keeps the release notes release-please wrote", "Release-please notes are kept" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| Goreleaser SHALL build `opm` for linux/amd64, linux/arm64, darwin/amd6... | body of "Goreleaser builds five platform archives and attaches them with the checksums", verbatim |
| Its changelog SHALL group commits into Features, Bug Fixes, Performanc... | body of "Goreleaser groups the changelog by commit type", the part of this sentence it holds, verbatim or as the words changed below say |
| Goreleaser SHALL attach to release-please's draft release (`release.us... | body of "Goreleaser attaches to release-please's draft release and publishes it last", verbatim |
| The configuration SHALL NOT set `release.draft`, a `release.name_templ... | body of "The goreleaser configuration sets no draft, no renamed release and no replaced draft", the part of this sentence it holds, verbatim or as the words changed below say |
| Goreleaser SHALL set the GitHub Release's prerelease flag from the tag... | body of "Goreleaser sets the prerelease flag from the tag", verbatim |
| It SHALL keep the release notes release-please already wrote on the ex... | body of "Goreleaser keeps the release notes release-please wrote", the part of this sentence it holds, verbatim or as the words changed below say |
| Source: enhancement 0021 ("Release tags are immutable"). | body of "Goreleaser builds five platform archives and attaches them with the checksums", verbatim; body of "Goreleaser groups the changelog by commit type", verbatim; body of "Goreleaser attaches to release-please's draft release and publishes it last", verbatim; body of "The goreleaser configuration sets no draft, no renamed release and no replaced draft", verbatim; body of "Goreleaser sets the prerelease flag from the tag", verbatim; body of "Goreleaser keeps the release notes release-please wrote", verbatim |

Words changed or added:
- "Goreleaser groups the changelog by commit type": "Its changelog SHALL group commits into Features, Bug Fixes, Performance, Refactoring and O..." reads "The changelog of Goreleaser SHALL group commits into Features, Bug Fixes, Performance, Ref..." ("Its changelog" reads "The changelog of Goreleaser")
- "The goreleaser configuration sets no draft, no renamed release and no replaced draft": "The configuration SHALL NOT set `release.draft`, a `release.name_template` that departs fr..." reads "The goreleaser configuration SHALL NOT set `release.draft`, a `release.name_template` that..." (subject made explicit)
- "Goreleaser keeps the release notes release-please wrote": "It SHALL keep the release notes release-please already wrote on the existing GitHub Releas..." reads "Goreleaser SHALL keep the release notes release-please already wrote on the existing GitHu..." ("It" reads "Goreleaser")

Lengths (old, new): old 1342; new "Goreleaser builds five platform archives and attaches them with the checksums" 352; "Goreleaser groups the changelog by commit type" 236; "Goreleaser attaches to release-please's draft release and publishes it last" 395; "The goreleaser configuration sets no draft, no renamed release and no replaced draft" 206; "Goreleaser sets the prerelease flag from the tag" 277; "Goreleaser keeps the release notes release-please wrote" 189.

### Ledger: release-workflow, Template modules publish on release

Form: c2 (REMOVED, ADDED under new names). New requirements: "The publish-templates job runs the publish script before goreleaser builds", "The script runs the identity, layout, tidy, vet and zip gates on every template tree", "The script runs the dry-run publish gates, the version order and the content check on every template tree", "A failed gate publishes nothing and otherwise the script acts on each gate verdict", "A version published between the gates and the push is not skipped unchecked", "A release with no template changed publishes nothing", "A change to a template's files bumps its identity Version".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| No template version bumped | "A release with no template changed publishes nothing", "No template version bumped" | No (byte equal, checked) |
| Template fails a gate | "A failed gate publishes nothing and otherwise the script acts on each gate verdict", "Template fails a gate" | No (byte equal, checked) |
| Bumped template publishes | "A failed gate publishes nothing and otherwise the script acts on each gate verdict", "Bumped template publishes" | No (byte equal, checked) |
| A template that does not vet blocks every template publish | "The script runs the identity, layout, tidy, vet and zip gates on every template tree", "A template that does not vet blocks every template publish" | No (byte equal, checked) |
| Changed template at a published version fails before any publish | "The script runs the dry-run publish gates, the version order and the content check on every template tree", "Changed template at a published version fails before any publish" | No (byte equal, checked) |
| Re-run after a partial publish completes | "A release with no template changed publishes nothing", "Re-run after a partial publish completes" | No (byte equal, checked) |
| A version published during the run is not skipped | "A version published between the gates and the push is not skipped unchecked", "A version published during the run is not skipped" | No (byte equal, checked) |
| A template whose zip omits a file publishes nothing | "The script runs the identity, layout, tidy, vet and zip gates on every template tree", "A template whose zip omits a file publishes nothing" | No (byte equal, checked) |
| A prerelease or older template version publishes nothing | "The script runs the dry-run publish gates, the version order and the content check on every template tree", "A prerelease or older template version publishes nothing" | No (byte equal, checked) |
| (new) | "The publish-templates job runs the publish script before goreleaser builds", "The job finishes before goreleaser builds" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A change to a template's files bumps its identity Version", "A changed template without a bump fails the release" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The `publish-templates` job SHALL `need` release-please, run only when... | body of "The publish-templates job runs the publish script before goreleaser builds", verbatim |
| It SHALL finish before the goreleaser job builds. | body of "The publish-templates job runs the publish script before goreleaser builds", verbatim |
| Before it publishes any template, the script SHALL run every gate of t... | body of "The script runs the dry-run publish gates, the version order and the content check on every template tree", the part of this sentence it holds, verbatim or as the words changed below say |
| If any template fails a gate, or a fetch, listing, verification or com... | body of "A failed gate publishes nothing and otherwise the script acts on each gate verdict", verbatim |
| Otherwise it SHALL act on each template's gate verdict without reading... | body of "A failed gate publishes nothing and otherwise the script acts on each gate verdict", verbatim |
| A version that becomes published between the gates and the push SHALL ... | body of "A version published between the gates and the push is not skipped unchecked", verbatim |
| A release with no template changed publishes nothing and succeeds, and... | body of "A release with no template changed publishes nothing", with the keyword words changed as the list below says |
| A change to a template's files, its `cue.mod` pins included, SHALL the... | body of "A change to a template's files bumps its identity Version", verbatim |

Words changed or added:
- "The script runs the identity, layout, tidy, vet and zip gates on every template tree": "Before it publishes any template, the script SHALL run every gate of the pr-workflow requi..." reads "Before the `publish-templates` job publishes any template, the script SHALL run every gate..." (the first two list items of the gate list; the final semicolon reads a period; the "it" of the intro reads "the `publish-templates` job")
- "The script runs the dry-run publish gates, the version order and the content check on every template tree": "Before it publishes any template, the script SHALL run every gate of the pr-workflow requi..." reads "Before the `publish-templates` job publishes any template, the script SHALL run every gate..." (the last two list items of the gate list; the "it" of the intro reads "the `publish-templates` job")
- "A release with no template changed publishes nothing": "A release with no template changed publishes nothing and succeeds, and a re-run after a pa..." reads "A release with no template changed SHALL publish nothing and succeed, and a re-run after a..." ("publishes" reads "SHALL publish", "succeeds" reads "succeed", "skips" reads "SHALL skip")
- "A failed gate publishes nothing and otherwise the script acts on each gate verdict": "...the script SHALL publish nothing and fail." reads "...the script (`.github/scripts/publish-templates.sh`) SHALL publish nothing and fail." (the script named as the same old requirement names it)
- "A version published between the gates and the push is not skipped unchecked": "...SHALL make `opm module publish` refuse and the job fail, never be skipped unchecked." reads "...SHALL make `opm module publish` refuse and the job (`publish-templates`) fail, never be skipped unchecked." (the job named as the same old requirement names it)

Lengths (old, new): old 1912; new "The publish-templates job runs the publish script before goreleaser builds" 474; "The script runs the identity, layout, tidy, vet and zip gates on every template tree" 328; "The script runs the dry-run publish gates, the version order and the content check on every template tree" 469; "A failed gate publishes nothing and otherwise the script acts on each gate verdict" 381; "A version published between the gates and the push is not skipped unchecked" 170; "A release with no template changed publishes nothing" 150; "A change to a template's files bumps its identity Version" 240.

### Ledger: release-workflow, Release-please runs on pushes to main

Form: c2 (REMOVED, ADDED under new names). New requirements: "The release workflow triggers on push to main and on manual dispatch with a tag", "On a push the release-please job runs first as the opm-release-please App", "The release-please configuration sets draft and force-tag-creation", "The release-please job exposes releases_created and tag_name as outputs", "On a manual run the release-please job is skipped".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Release-please on a push without a merged release PR | "On a push the release-please job runs first as the opm-release-please App", "Release-please on a push without a merged release PR" | No (byte equal, checked) |
| Release-please on a merged release PR | "The release-please configuration sets draft and force-tag-creation", "Release-please on a merged release PR" | No (byte equal, checked) |
| Releases without a release-as key stay on the beta line | "On a push the release-please job runs first as the opm-release-please App", "Releases without a release-as key stay on the beta line" | No (byte equal, checked) |
| (new) | "The release workflow triggers on push to main and on manual dispatch with a tag", "Triggers of the release workflow" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The release-please job exposes releases_created and tag_name as outputs", "Outputs for the downstream jobs" | New restatement scenario: only words of the old clause, no new case |
| (new) | "On a manual run the release-please job is skipped", "Manual run skips release-please" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The release workflow SHALL trigger on `push` to `main` and on `workflo... | body of "The release workflow triggers on push to main and on manual dispatch with a tag", verbatim |
| On a push, the `release-please` job SHALL run first, authenticated as ... | body of "On a push the release-please job runs first as the opm-release-please App", verbatim |
| The configuration SHALL set `draft: true` and `force-tag-creation: tru... | body of "The release-please configuration sets draft and force-tag-creation", the part of this sentence it holds, verbatim or as the words changed below say |
| The job SHALL expose `releases_created` and `tag_name` as outputs for ... | body of "The release-please job exposes releases_created and tag_name as outputs", the part of this sentence it holds, verbatim or as the words changed below say |
| On a manual run the `release-please` job SHALL be skipped. | body of "On a manual run the release-please job is skipped", verbatim |
| Source: enhancement 0021 ("Release tags are immutable"). | body of "The release workflow triggers on push to main and on manual dispatch with a tag", verbatim; body of "On a push the release-please job runs first as the opm-release-please App", verbatim; body of "The release-please configuration sets draft and force-tag-creation", verbatim; body of "The release-please job exposes releases_created and tag_name as outputs", verbatim; body of "On a manual run the release-please job is skipped", verbatim |

Words changed or added:
- "The release-please configuration sets draft and force-tag-creation": "The configuration SHALL set `draft: true` and `force-tag-creation: true`, so release-pleas..." reads "The configuration of `release-please-config.json` SHALL set `draft: true` and `force-tag-c..." ("The configuration" reads "The configuration of `release-please-config.json`")
- "The release-please job exposes releases_created and tag_name as outputs": "The job SHALL expose `releases_created` and `tag_name` as outputs for the downstream jobs...." reads "The `release-please` job SHALL expose `releases_created` and `tag_name` as outputs for the..." ("The job" reads "The `release-please` job")

Lengths (old, new): old 881; new "The release workflow triggers on push to main and on manual dispatch with a tag" 194; "On a push the release-please job runs first as the opm-release-please App" 375; "The release-please configuration sets draft and force-tag-creation" 306; "The release-please job exposes releases_created and tag_name as outputs" 164; "On a manual run the release-please job is skipped" 115.

### Ledger: release-workflow, A version line moves only through a release-as key

Form: c2 (REMOVED, ADDED under new names). New requirements: "A forced version is a release-as value set by a normal pull request", "The release-as key takes effect only with a releasable commit", "The release-as key is removed before any later release PR merges", "A Release-As footer is not relied on", "A BEGIN_COMMIT_OVERRIDE block is not used".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A release-as key forces the line change | "A forced version is a release-as value set by a normal pull request", "A release-as key forces the line change" | No (byte equal, checked) |
| A key pull request of a hidden type alone opens no release PR | "The release-as key takes effect only with a releasable commit", "A key pull request of a hidden type alone opens no release PR" | No (byte equal, checked) |
| The key is removed once its release is cut | "The release-as key is removed before any later release PR merges", "The key is removed once its release is cut" | No (byte equal, checked) |
| A footer forces nothing | "A Release-As footer is not relied on", "A footer forces nothing" | No (byte equal, checked) |
| (new) | "A BEGIN_COMMIT_OVERRIDE block is not used", "An override block is not used" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| A forced version, such as a move to a new version line (for example `1... | body of "A forced version is a release-as value set by a normal pull request", verbatim |
| The key SHALL take effect only together with a releasable commit (a vi... | body of "The release-as key takes effect only with a releasable commit", the part of this sentence it holds, verbatim or as the words changed below say |
| Because release-please re-applies a configured `release-as` on every r... | body of "The release-as key is removed before any later release PR merges", the part of this sentence it holds, verbatim or as the words changed below say |
| Outside the span from the key pull request to the pull request that re... | body of "The release-as key is removed before any later release PR merges", verbatim |
| A `Release-As:` footer in a pull request body or a branch commit messa... | body of "A Release-As footer is not relied on", verbatim |
| A `BEGIN_COMMIT_OVERRIDE` block in a pull request body SHALL NOT be us... | body of "A BEGIN_COMMIT_OVERRIDE block is not used", verbatim |
| Source: workspace RELEASING.md, section "Owner settings" (owner decisi... | body of "A forced version is a release-as value set by a normal pull request", verbatim; body of "The release-as key takes effect only with a releasable commit", verbatim; body of "The release-as key is removed before any later release PR merges", verbatim; body of "A Release-As footer is not relied on", verbatim; body of "A BEGIN_COMMIT_OVERRIDE block is not used", verbatim |

Words changed or added:
- "The release-as key takes effect only with a releasable commit": "The key SHALL take effect only together with a releasable commit (a visible type: `feat`, ..." reads "The `release-as` key SHALL take effect only together with a releasable commit (a visible t..." ("The key" reads "The `release-as` key")
- "The release-as key is removed before any later release PR merges": "Because release-please re-applies a configured `release-as` on every run, the key SHALL be..." reads "Because release-please re-applies a configured `release-as` on every run, the `release-as`..." ("the key" reads "the `release-as` key")

Lengths (old, new): old 1267; new "A forced version is a release-as value set by a normal pull request" 302; "The release-as key takes effect only with a releasable commit" 486; "The release-as key is removed before any later release PR merges" 366; "A Release-As footer is not relied on" 238; "A BEGIN_COMMIT_OVERRIDE block is not used" 241.

### Ledger: release-workflow, A breaking change is marked by a bang in the pull request title

Form: c2 (REMOVED, ADDED under new names). New requirements: "A breaking change is marked by a bang in the pull request title or the one commit subject", "A BREAKING CHANGE footer is not relied on", "The migration note goes in the pull request body", "The release pull request CHANGELOG is edited by hand and the edit is redone", "A BEGIN_COMMIT_OVERRIDE block is not used for a breaking change".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Breaking change marked in the title | "A breaking change is marked by a bang in the pull request title or the one commit subject", "Breaking change marked in the title" | No (byte equal, checked) |
| A breaking footer alone marks nothing | "A BREAKING CHANGE footer is not relied on", "A breaking footer alone marks nothing" | No (byte equal, checked) |
| (new) | "The migration note goes in the pull request body", "Migration note placement" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The release pull request CHANGELOG is edited by hand and the edit is redone", "A hand edit of the release pull request's changelog" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A BEGIN_COMMIT_OVERRIDE block is not used for a breaking change", "An override block marks nothing" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| A breaking change SHALL be marked by `!` in the pull request title (`f... | body of "A breaking change is marked by a bang in the pull request title or the one commit subject", verbatim |
| Until that owner setting lands, a one-commit pull request squashes und... | body of "A breaking change is marked by a bang in the pull request title or the one commit subject", verbatim |
| A `BREAKING CHANGE:` footer in a pull request body or a branch commit ... | body of "A BREAKING CHANGE footer is not relied on", the part of this sentence it holds, verbatim or as the words changed below say; body of "The release pull request CHANGELOG is edited by hand and the edit is redone", the part of this sentence it holds, verbatim or as the words changed below say |
| A `BEGIN_COMMIT_OVERRIDE` block in a pull request body SHALL NOT be us... | body of "A BEGIN_COMMIT_OVERRIDE block is not used for a breaking change", verbatim |
| Source: workspace RELEASING.md, section "Owner settings" (owner decisi... | body of "A breaking change is marked by a bang in the pull request title or the one commit subject", verbatim; body of "A BREAKING CHANGE footer is not relied on", verbatim; body of "The migration note goes in the pull request body", verbatim; body of "The release pull request CHANGELOG is edited by hand and the edit is redone", verbatim; body of "A BEGIN_COMMIT_OVERRIDE block is not used for a breaking change", verbatim |

Words changed or added:
- "A BREAKING CHANGE footer is not relied on": "A `BREAKING CHANGE:` footer in a pull request body or a branch commit message SHALL NOT be..." reads "A `BREAKING CHANGE:` footer in a pull request body or a branch commit message SHALL NOT be..." (the first clause of the sentence up to the semicolon; the semicolon reads a period)
- "The migration note goes in the pull request body": "A `BREAKING CHANGE:` footer in a pull request body or a branch commit message SHALL NOT be..." reads "The migration note of a breaking change SHALL go in the pull request body, and in the user..." ("the migration note goes" reads "The migration note of a breaking change SHALL go"; the trailing comma reads a period)
- "The release pull request CHANGELOG is edited by hand and the edit is redone": "A `BREAKING CHANGE:` footer in a pull request body or a branch commit message SHALL NOT be..." reads "The release pull request's `CHANGELOG.md` SHALL be edited by hand to carry the migration n..." ("and" dropped before "the release" and "the edit"; "it" reads "the migration note"; the two clauses stand as two sentences)

Lengths (old, new): old 1098; new "A breaking change is marked by a bang in the pull request title or the one commit subject" 395; "A BREAKING CHANGE footer is not relied on" 243; "The migration note goes in the pull request body" 212; "The release pull request CHANGELOG is edited by hand and the edit is redone" 402; "A BEGIN_COMMIT_OVERRIDE block is not used for a breaking change" 219.

### Ledger: release-workflow, Downstream repos are notified after a release is published

Form: c2 (REMOVED, ADDED under new names). New requirements: "The release workflow ends with a notify-downstream job", "The notify job runs only after goreleaser succeeded, on main, unless notifying is switched off", "The notify job runs on a manual recovery run and does not depend on the docs bundle job", "The notify job is caller-owned with a fixed runner, Environment, timeout and permission", "The notify job has exactly one step, the shared cascade-notify action", "The cascade-notify action mints the dispatch token and chooses the targets".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Release run notifies after publishing | "The release workflow ends with a notify-downstream job", "Release run notifies after publishing" | No (byte equal, checked) |
| A release left a draft notifies nothing | "The notify job runs only after goreleaser succeeded, on main, unless notifying is switched off", "A release left a draft notifies nothing" | No (byte equal, checked) |
| Manual recovery that finishes a draft notifies once | "The notify job runs on a manual recovery run and does not depend on the docs bundle job", "Manual recovery that finishes a draft notifies once" | No (byte equal, checked) |
| Manual run on a published release does not notify again | "The notify job runs on a manual recovery run and does not depend on the docs bundle job", "Manual run on a published release does not notify again" | No (byte equal, checked) |
| Manual run from another branch does not notify | "The notify job runs only after goreleaser succeeded, on main, unless notifying is switched off", "Manual run from another branch does not notify" | No (byte equal, checked) |
| A push without a release does not notify | "The notify job runs only after goreleaser succeeded, on main, unless notifying is switched off", "A push without a release does not notify" | No (byte equal, checked) |
| Notify switched off | "The notify job runs only after goreleaser succeeded, on main, unless notifying is switched off", "Notify switched off" | No (byte equal, checked) |
| A failed docs bundle does not suppress the notify | "The notify job runs on a manual recovery run and does not depend on the docs bundle job", "A failed docs bundle does not suppress the notify" | No (byte equal, checked) |
| (new) | "The notify job is caller-owned with a fixed runner, Environment, timeout and permission", "The notify job is caller-owned" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The notify job has exactly one step, the shared cascade-notify action", "The notify job has one step" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The cascade-notify action mints the dispatch token and chooses the targets", "The action chooses the targets" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `.github/workflows/release.yml` SHALL have a `notify-downstream` job, ... | body of "The release workflow ends with a notify-downstream job", verbatim |
| It SHALL run only when goreleaser succeeded, the run's ref is `main`, ... | body of "The notify job runs only after goreleaser succeeded, on main, unless notifying is switched off", the part of this sentence it holds, verbatim or as the words changed below say |
| It SHALL run on a release run and on a manual recovery run alike, beca... | body of "The notify job runs on a manual recovery run and does not depend on the docs bundle job", the part of this sentence it holds, verbatim or as the words changed below say |
| It SHALL NOT depend on the docs bundle job. | body of "The notify job runs on a manual recovery run and does not depend on the docs bundle job", the part of this sentence it holds, verbatim or as the words changed below say |
| It is a caller-owned job: it SHALL run on `ubuntu-latest`, declare `en... | body of "The notify job has exactly one step, the shared cascade-notify action", the part of this sentence it holds, verbatim or as the words changed below say |
| The action mints the dispatch token and chooses the targets: catalog_o... | body of "The cascade-notify action mints the dispatch token and chooses the targets", the part of this sentence it holds, verbatim or as the words changed below say |
| Source: workspace RELEASING.md, sections "Notify after publish" and "S... | body of "The release workflow ends with a notify-downstream job", verbatim; body of "The notify job runs only after goreleaser succeeded, on main, unless notifying is switched off", verbatim; body of "The notify job runs on a manual recovery run and does not depend on the docs bundle job", verbatim; body of "The notify job is caller-owned with a fixed runner, Environment, timeout and permission", verbatim; body of "The notify job has exactly one step, the shared cascade-notify action", verbatim; body of "The cascade-notify action mints the dispatch token and chooses the targets", verbatim |

Words changed or added:
- "The notify job runs only after goreleaser succeeded, on main, unless notifying is switched off": "It SHALL run only when goreleaser succeeded, the run's ref is `main`, and the repository v..." reads "The `notify-downstream` job SHALL run only when goreleaser succeeded, the run's ref is `ma..." ("It" reads "The `notify-downstream` job")
- "The notify job runs on a manual recovery run and does not depend on the docs bundle job": "It SHALL run on a release run and on a manual recovery run alike, because release-please i..." reads "The `notify-downstream` job SHALL run on a release run and on a manual recovery run alike,..." ("It" reads "The `notify-downstream` job")
- "The notify job runs on a manual recovery run and does not depend on the docs bundle job": "It SHALL NOT depend on the docs bundle job." reads "The `notify-downstream` job SHALL NOT depend on the docs bundle job." ("It" reads "The `notify-downstream` job")
- "The notify job is caller-owned with a fixed runner, Environment, timeout and permission": "It is a caller-owned job: it SHALL run on `ubuntu-latest`, declare `environment: cascade`,..." reads "The `notify-downstream` job is a caller-owned job: it SHALL run on `ubuntu-latest`, declar..." (the first part of the sentence up to the list boundary before "and have exactly one step"; the end reads a period)
- "The notify job has exactly one step, the shared cascade-notify action": "It is a caller-owned job: it SHALL run on `ubuntu-latest`, declare `environment: cascade`,..." reads "The notify-downstream job SHALL have exactly one step, the shared `open-platform-model/.gi..." (the second part of the sentence from the list boundary; the subject "It is a caller-owned job: it SHALL" reads "The notify-downstream job SHALL", "and have" reads "have")
- "The cascade-notify action mints the dispatch token and chooses the targets": "The action mints the dispatch token and chooses the targets: catalog_opm and opm-operator,..." reads "The `cascade-notify` action SHALL mint the dispatch token and choose the targets: catalog_..." ("The action" reads "The `cascade-notify` action"; "mints" reads "SHALL mint", "chooses" reads "choose")

Lengths (old, new): old 1148; new "The release workflow ends with a notify-downstream job" 261; "The notify job runs only after goreleaser succeeded, on main, unless notifying is switched off" 291; "The notify job runs on a manual recovery run and does not depend on the docs bundle job" 350; "The notify job is caller-owned with a fixed runner, Environment, timeout and permission" 308; "The notify job has exactly one step, the shared cascade-notify action" 500; "The cascade-notify action mints the dispatch token and chooses the targets" 272.

### Ledger: release-workflow, Only the release-please job reads the release App key, inside the release Environment

Form: c2 (REMOVED, ADDED under new names). New requirements: "Only the release-please job reads the release App key", "The release Environment is declared by the release-please job alone, on pushes to main", "The release-please job grants the GITHUB_TOKEN no permission".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A push to main mints the token inside the Environment | "The release Environment is declared by the release-please job alone, on pushes to main", "A push to main mints the token inside the Environment" | No (byte equal, checked) |
| A manual run does not reach the key | "Only the release-please job reads the release App key", "A manual run does not reach the key" | No (byte equal, checked) |
| (new) | "The release-please job grants the GITHUB_TOKEN no permission", "The job grants the GITHUB_TOKEN nothing" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The `release-please` job of `release.yml` SHALL be the only job in the... | body of "Only the release-please job reads the release App key", verbatim |
| That job SHALL declare `environment: release`, the Environment whose d... | body of "The release Environment is declared by the release-please job alone, on pushes to main", the part of this sentence it holds, verbatim or as the words changed below say |
| It SHALL grant the `GITHUB_TOKEN` no permission (`permissions: {}`), b... | body of "The release-please job grants the GITHUB_TOKEN no permission", the part of this sentence it holds, verbatim or as the words changed below say |
| No other job SHALL declare the `release` Environment. | body of "The release Environment is declared by the release-please job alone, on pushes to main", verbatim |
| Source: owner decision 29 (2026-10-04); security pass finding GOV-2. | body of "Only the release-please job reads the release App key", verbatim; body of "The release Environment is declared by the release-please job alone, on pushes to main", verbatim; body of "The release-please job grants the GITHUB_TOKEN no permission", verbatim |

Words changed or added:
- "The release Environment is declared by the release-please job alone, on pushes to main": "That job SHALL declare `environment: release`, the Environment whose deployment branch pol..." reads "The `release-please` job SHALL declare `environment: release`, the Environment whose deplo..." ("That job" reads "The `release-please` job")
- "The release-please job grants the GITHUB_TOKEN no permission": "It SHALL grant the `GITHUB_TOKEN` no permission (`permissions: {}`), because it acts only ..." reads "The `release-please` job SHALL grant the `GITHUB_TOKEN` no permission (`permissions: {}`),..." ("It" reads "The `release-please` job")

Lengths (old, new): old 645; new "Only the release-please job reads the release App key" 284; "The release Environment is declared by the release-please job alone, on pushes to main" 332; "The release-please job grants the GITHUB_TOKEN no permission" 203.

### Ledger: kernel-render, All renders go through the library kernel

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI renders instances through one Kernel.Render call", "The CLI carries no matching, execution or finalization of its own and imports no opm-operator package", "The CLI checks the compiled objects for shared apply identities", "A shared apply identity is refused as a validation failure", "The refusal of a shared identity prints the library header line and each shared identity", "The CLI words the advisory facts of the kernel as warnings", "The kernel-derived warnings name the same facts as before".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Instance apply renders via the kernel | "The CLI renders instances through one Kernel.Render call", "Instance apply renders via the kernel" | No (byte equal, checked) |
| No CLI-side match implementation | "The CLI carries no matching, execution or finalization of its own and imports no opm-operator package", "No CLI-side match implementation" | No (byte equal, checked) |
| Render warnings reach the user | "The CLI words the advisory facts of the kernel as warnings", "Render warnings reach the user" | No (byte equal, checked) |
| Unhandled optional trait warns | "The CLI words the advisory facts of the kernel as warnings", "Unhandled optional trait warns" | No (byte equal, checked) |
| Skew refusal is a validation failure | "The kernel-derived warnings name the same facts as before", "Skew refusal is a validation failure" | No (byte equal, checked) |
| Typed render causes keep their exit codes | "The CLI renders instances through one Kernel.Render call", "Typed render causes keep their exit codes" | No (byte equal, checked) |
| Two registrations in one module are refused by every render | "A shared apply identity is refused as a validation failure", "Two registrations in one module are refused by every render" | No (byte equal, checked) |
| Distinct identities render as before | "The CLI checks the compiled objects for shared apply identities", "Distinct identities render as before" | No (byte equal, checked) |
| (new) | "The refusal of a shared identity prints the library header line and each shared identity", "The refusal prints each shared identity" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The CLI SHALL render instances exclusively through `github.com/open-pl... | body of "The CLI renders instances through one Kernel.Render call", verbatim |
| The CLI MUST NOT carry its own component-matching, transformer-executi... | body of "The CLI carries no matching, execution or finalization of its own and imports no opm-operator package", verbatim |
| The rendered resource set is the render result's compiled objects. | body of "The CLI renders instances through one Kernel.Render call", verbatim |
| After a successful render and before any resource, digest or output is... | body of "The CLI checks the compiled objects for shared apply identities", the part of this sentence it holds, verbatim or as the words changed below say; body of "A shared apply identity is refused as a validation failure", the part of this sentence it holds, verbatim or as the words changed below say |
| The refusal SHALL print the library's header line under the render-fai... | body of "The refusal of a shared identity prints the library header line and each shared identity", the part of this sentence it holds, verbatim or as the words changed below say |
| The kernel reports advisory facts as structured rows and attaches no m... | body of "The CLI words the advisory facts of the kernel as warnings", with "reports" and "attaches" read as the list below says |
| The CLI SHALL word them itself and surface them to the user as warning... | body of "The CLI words the advisory facts of the kernel as warnings", verbatim |
| Both warnings SHALL name the same facts the kernel previously named — ... | body of "The kernel-derived warnings name the same facts as before", the part of this sentence it holds, verbatim or as the words changed below say |

Words changed or added:
- "The CLI checks the compiled objects for shared apply identities": "After a successful render and before any resource, digest or output is built, the CLI SHAL..." reads "After a successful render and before any resource, digest or output is built, the CLI SHAL..." (the first clause of the sentence up to the helper, with a period; the clause "and SHALL refuse" follows in another requirement)
- "A shared apply identity is refused as a validation failure": "After a successful render and before any resource, digest or output is built, the CLI SHAL..." reads "After a successful render and before any resource, digest or output is built, the CLI SHAL..." (the condition of the sentence repeated, then the clause "SHALL refuse the render ..." to the end; "and" dropped)
- "The refusal of a shared identity prints the library header line and each shared identity": "The refusal SHALL print the library's header line under the render-failed header and each ..." reads "The refusal of requirement "A shared apply identity is refused as a validation failure" SH..." ("The refusal" reads the quoted requirement)
- "The CLI words the advisory facts of the kernel as warnings": "The kernel reports advisory facts as structured rows and attaches no message strings to a ..." reads "The kernel SHALL report advisory facts as structured rows and SHALL attach no message stri..." ("reports" reads "SHALL report", "attaches" reads "SHALL attach")
- "The kernel-derived warnings name the same facts as before": "Both warnings SHALL name the same facts the kernel previously named — for skew, the path a..." reads "Both warnings of the requirement "The CLI words the advisory facts of the kernel as warnin..." ("Both warnings" reads the quoted requirement)

Lengths (old, new): old 2018; new "The CLI renders instances through one Kernel.Render call" 404; "The CLI carries no matching, execution or finalization of its own and imports no opm-operator package" 375; "The CLI checks the compiled objects for shared apply identities" 245; "A shared apply identity is refused as a validation failure" 382; "The refusal of a shared identity prints the library header line and each shared identity" 242; "The CLI words the advisory facts of the kernel as warnings" 424; "The kernel-derived warnings name the same facts as before" 187.

### Ledger: kernel-render, CLI entry points map onto kernel entry points

Form: c2 (REMOVED, ADDED under new names). New requirements: "Each CLI loading path uses its kernel acquire verb", "No acquire call passes per-call load options", "The CLI uses no raw-value loading tier and builds no artifact from a value of its own", "One kernel is constructed per command invocation", "The CLI does not fill values into an evaluated instance from Go".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Module build synthesizes through the kernel | "Each CLI loading path uses its kernel acquire verb", "Module build synthesizes through the kernel" | No (byte equal, checked) |
| Registry mapping is supplied once | "No acquire call passes per-call load options", "Registry mapping is supplied once" | No (byte equal, checked) |
| Single kernel per invocation | "One kernel is constructed per command invocation", "Single kernel per invocation" | No (byte equal, checked) |
| Instance file with extra values | "Each CLI loading path uses its kernel acquire verb", "Instance file with extra values" | No (byte equal, checked) |
| Publish gate acquires through the kernel | "Each CLI loading path uses its kernel acquire verb", "Publish gate acquires through the kernel" | No (byte equal, checked) |
| (new) | "The CLI uses no raw-value loading tier and builds no artifact from a value of its own", "A caller reads the acquired artifact" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The CLI does not fill values into an evaluated instance from Go", "No value is filled from Go" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| Each CLI loading path SHALL use the corresponding kernel acquire verb:... | body of "Each CLI loading path uses its kernel acquire verb", verbatim |
| No acquire call SHALL pass per-call load options: the registry mapping... | body of "No acquire call passes per-call load options", verbatim |
| The CLI MUST NOT reach for a raw-value loading tier or construct an ar... | body of "The CLI uses no raw-value loading tier and builds no artifact from a value of its own", verbatim |
| One `kernel.Kernel` SHALL be constructed per command invocation and th... | body of "One kernel is constructed per command invocation", verbatim |
| The CLI MUST NOT fill values into an evaluated instance from Go. | body of "The CLI does not fill values into an evaluated instance from Go", verbatim |

Lengths (old, new): old 1122; new "Each CLI loading path uses its kernel acquire verb" 438; "No acquire call passes per-call load options" 187; "The CLI uses no raw-value loading tier and builds no artifact from a value of its own" 202; "One kernel is constructed per command invocation" 227; "The CLI does not fill values into an evaluated instance from Go" 64.

### Ledger: kernel-render, Render digests are kernel-derived and operator-parity

Form: c2 (REMOVED, ADDED under new names). New requirements: "The last applied render digest is the library's shared render digest", "The digest leaves out the value of the managed-by label", "A registry-gated check verifies that the CLI and operator render digests are identical", "Evaluator-version skew applies to a future cross-binary comparison".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Parity for the same inputs | "A registry-gated check verifies that the CLI and operator render digests are identical", "Parity for the same inputs" | No (byte equal, checked) |
| Skew reported explicitly (cross-binary check, slice C3) | "Evaluator-version skew applies to a future cross-binary comparison", "Skew reported explicitly (cross-binary check, slice C3)" | No (byte equal, checked) |
| The runtime name does not move the digest | "The digest leaves out the value of the managed-by label", "The runtime name does not move the digest" | No (byte equal, checked) |
| (new) | "The last applied render digest is the library's shared render digest", "The digest is computed by the shared function" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `status.lastAppliedRenderDigest` SHALL be computed over the kernel's r... | body of "The last applied render digest is the library's shared render digest", verbatim |
| The digest leaves the value of the `app.kubernetes.io/managed-by` labe... | body of "The digest leaves out the value of the managed-by label", the part of this sentence it holds, verbatim or as the words changed below say |
| A registry-gated integration check SHALL verify that the CLI's local-d... | body of "A registry-gated check verifies that the CLI and operator render digests are identical", verbatim |
| Evaluator-version skew reporting applies to a future cross-binary comp... | body of "Evaluator-version skew applies to a future cross-binary comparison", the part of this sentence it holds, verbatim or as the words changed below say |

Words changed or added:
- "The digest leaves out the value of the managed-by label": "The digest leaves the value of the `app.kubernetes.io/managed-by` label out, so the CLI (`..." reads "`status.lastAppliedRenderDigest` SHALL leave the value of the `app.kubernetes.io/managed-b..." ("The digest leaves" reads "`status.lastAppliedRenderDigest` SHALL leave")
- "Evaluator-version skew applies to a future cross-binary comparison": "Evaluator-version skew reporting applies to a future cross-binary comparison, where the CL..." reads "Evaluator-version skew reporting SHALL apply to a future cross-binary comparison, where th..." ("applies" reads "SHALL apply")

Lengths (old, new): old 1212; new "The last applied render digest is the library's shared render digest" 190; "The digest leaves out the value of the managed-by label" 403; "A registry-gated check verifies that the CLI and operator render digests are identical" 416; "Evaluator-version skew applies to a future cross-binary comparison" 231.

### Ledger: kernel-render, Local-replacement renders warn

Form: c2 (REMOVED, ADDED under new names). New requirements: "Every render enables the kernel's local replacements", "A platform generated from the render's own deps carries the module's replacements", "The kernel replacement rows are worded as warnings, one per honoured replacement", "A module redirect of a path the platform names is reported inert", "The warnings do not block the render and a context without the file produces none", "Both render entry points enable the local replacements and word the row warnings", "Both render entry points perform the local-replacement warnings", "The provenance signal keeps its file-presence rule".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Replaced dependency warns | "The kernel replacement rows are worded as warnings, one per honoured replacement", "Replaced dependency warns" | No (byte equal, checked) |
| Platform replacement of its catalog is honoured and warns | "Every render enables the kernel's local replacements", "Platform replacement of its catalog is honoured and warns" | No (byte equal, checked) |
| Module-side replacement of a platform path is reported inert | "A module redirect of a path the platform names is reported inert", "Module-side replacement of a platform path is reported inert" | No (byte equal, checked) |
| Clean context stays silent | "The warnings do not block the render and a context without the file produces none", "Clean context stays silent" | No (byte equal, checked) |
| A malformed local file is a validation failure | "Every render enables the kernel's local replacements", "A malformed local file is a validation failure" | No (byte equal, checked) |
| Module deps platform honours the module's replacement | "A platform generated from the render's own deps carries the module's replacements", "Module deps platform honours the module's replacement" | No (byte equal, checked) |
| Instance deps platform honours the instance package's replacement | "A platform generated from the render's own deps carries the module's replacements", "Instance deps platform honours the instance package's replacement" | No (byte equal, checked) |
| (new) | "Both render entry points enable the local replacements and word the row warnings", "Both entry points enable the replacements and word the row warnings" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Both render entry points perform the local-replacement warnings", "Both entry points perform the replacement warnings" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The provenance signal keeps its file-presence rule", "The provenance signal follows the file" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| Every render SHALL enable the kernel's local replacements, so a `cue.m... | body of "Every render enables the kernel's local replacements", verbatim |
| When the platform is generated from the render's own deps (`opm module... | body of "A platform generated from the render's own deps carries the module's replacements", the part of this sentence it holds, verbatim or as the words changed below say |
| The CLI SHALL word the kernel's replacement rows as warnings, one per ... | body of "The kernel replacement rows are worded as warnings, one per honoured replacement", verbatim |
| For each `replaceWith` in the module context's own `cue.mod/local-modu... | body of "A module redirect of a path the platform names is reported inert", verbatim |
| The warnings SHALL NOT block the render or alter its output. | body of "The warnings do not block the render and a context without the file produces none", the part of this sentence it holds, verbatim or as the words changed below say |
| A context without the file SHALL produce no such warning. | body of "The warnings do not block the render and a context without the file produces none", with "the file" read as the list below says |
| Both render entry points SHALL perform this. | body of "Both render entry points enable the local replacements and word the row warnings" and body of "Both render entry points perform the local-replacement warnings", verbatim, with the words added below |
| The provenance signal (`SourceLocal`, the `module-instance.opmodel.dev... | body of "The provenance signal keeps its file-presence rule", the part of this sentence it holds, verbatim or as the words changed below say |
| Source: enhancement 0010 (a moved-ahead dev checkout is accepted and w... | body of "Every render enables the kernel's local replacements", verbatim; body of "A platform generated from the render's own deps carries the module's replacements", verbatim; body of "The kernel replacement rows are worded as warnings, one per honoured replacement", verbatim; body of "A module redirect of a path the platform names is reported inert", verbatim; body of "The warnings do not block the render and a context without the file produces none", verbatim; body of "Both render entry points enable the local replacements and word the row warnings", verbatim; body of "Both render entry points perform the local-replacement warnings", verbatim; body of "The provenance signal keeps its file-presence rule", verbatim |

Words changed or added:
- "A platform generated from the render's own deps carries the module's replacements": "When the platform is generated from the render's own deps (`opm module build` and `opm mod..." reads "When the platform is generated from the render's own deps (`opm module build` and `opm mod..." ("are carried" reads "SHALL be carried")
- "The warnings do not block the render and a context without the file produces none": "The warnings SHALL NOT block the render or alter its output." reads "The local-replacement warnings SHALL NOT block the render or alter its output." ("The warnings" reads "The local-replacement warnings")
- "The warnings do not block the render and a context without the file produces none": "A context without the file SHALL produce no such warning." reads "A context without the `cue.mod/local-module.cue` file SHALL produce no such warning." ("the file" reads "the `cue.mod/local-module.cue` file")
- "Both render entry points enable the local replacements and word the row warnings": "Both render entry points SHALL perform this." reads "Both render entry points SHALL perform this, as the requirements "Every render enables the kernel's local replacements", "A platform generated from the render's own deps carries the module's replacements" and "The kernel replacement rows are worded as warnings, one per honoured replacement" state it." (the exact headers of the same spec's requirements that hold what "this" points at)
- "Both render entry points perform the local-replacement warnings": "Both render entry points SHALL perform this." reads "Both render entry points SHALL perform this, as the requirements "A module redirect of a path the platform names is reported inert" and "The warnings do not block the render and a context without the file produces none" state it." (the exact headers of the same spec's requirements that hold what "this" points at)
- "The provenance signal keeps its file-presence rule": "The provenance signal (`SourceLocal`, the `module-instance.opmodel.dev/source: local` anno..." reads "The provenance signal (`SourceLocal`, the `module-instance.opmodel.dev/source: local` anno..." ("keeps" reads "SHALL keep")

Lengths (old, new): old 1728; new "Every render enables the kernel's local replacements" 486; "A platform generated from the render's own deps carries the module's replacements" 433; "The kernel replacement rows are worded as warnings, one per honoured replacement" 432; "A module redirect of a path the platform names is reported inert" 339; "The warnings do not block the render and a context without the file produces none" 247; "Both render entry points enable the local replacements and word the row warnings" 385; "The provenance signal keeps its file-presence rule" 216.

### Ledger: kernel-render, Unprovided provider-fulfilled demands are skipped on request

Form: c2 (REMOVED, ADDED under new names). New requirements: "Render-bearing commands accept --skip-unprovided and pass it to the kernel", "Without --skip-unprovided a render behaves as before", "With --skip-unprovided a skipped trait is worded as a warning naming the alternatives", "With --skip-unprovided an omitted component is worded once as a warning", "The skipped-demand warnings go to the log stream", "A refusal for unprovided demands adds a hint naming the three ways out", "Instance apply and module apply refuse --skip-unprovided for an operator-managed instance".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default render still refuses | "Without --skip-unprovided a render behaves as before", "Default render still refuses" | No (byte equal, checked) |
| A skipped trait renders the rest | "With --skip-unprovided a skipped trait is worded as a warning naming the alternatives", "A skipped trait renders the rest" | No (byte equal, checked) |
| A skipped resource drops its component | "With --skip-unprovided an omitted component is worded once as a warning", "A skipped resource drops its component" | No (byte equal, checked) |
| A catalog-fulfilled gap still refuses | "Render-bearing commands accept --skip-unprovided and pass it to the kernel", "A catalog-fulfilled gap still refuses" | No (byte equal, checked) |
| A cluster without the provider names the flag | "A refusal for unprovided demands adds a hint naming the three ways out", "A cluster without the provider names the flag" | No (byte equal, checked) |
| An operator-managed instance refuses the flag | "Instance apply and module apply refuse --skip-unprovided for an operator-managed instance", "An operator-managed instance refuses the flag" | No (byte equal, checked) |
| (new) | "The skipped-demand warnings go to the log stream", "Warnings stay out of the manifest" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| Every render-bearing command (`opm module build`, `module vet`, `modul... | body of "Render-bearing commands accept --skip-unprovided and pass it to the kernel", verbatim |
| The switch is the only thing the flag does: which demands are skippabl... | body of "Render-bearing commands accept --skip-unprovided and pass it to the kernel", the part of this sentence it holds, verbatim or as the words changed below say |
| Without the flag, a render SHALL behave exactly as before this change. | body of "Without --skip-unprovided a render behaves as before", verbatim, with the flag named as the words added below say |
| With the flag, the CLI SHALL print one warning per skipped demand afte... | body of "With --skip-unprovided a skipped trait is worded as a warning naming the alternatives", verbatim, with the flag named as the words added below say; body of "With --skip-unprovided an omitted component is worded once as a warning", verbatim, with the flag named as the words added below say |
| A skipped trait SHALL be worded as `component "<c>": skipped provider-... | body of "With --skip-unprovided a skipped trait is worded as a warning naming the alternatives", verbatim |
| A component the kernel omitted SHALL be worded once as `component "<c>... | body of "With --skip-unprovided an omitted component is worded once as a warning", verbatim |
| When the kernel reports alternatives, the warning SHALL name them. | body of "With --skip-unprovided a skipped trait is worded as a warning naming the alternatives", verbatim |
| The skipped-demand warnings SHALL go to the log stream, never to the r... | body of "The skipped-demand warnings go to the log stream", verbatim |
| When a render is refused for unresolved demands and at least one of th... | body of "A refusal for unprovided demands adds a hint naming the three ways out", verbatim |
| `opm instance apply` and `opm module apply` SHALL refuse `--skip-unpro... | body of "Instance apply and module apply refuse --skip-unprovided for an operator-managed instance", verbatim |

Words changed or added:
- "Render-bearing commands accept --skip-unprovided and pass it to the kernel": "The switch is the only thing the flag does: which demands are skippable, and what a skip d..." reads "The switch SHALL be the only thing the flag does: which demands are skippable, and what a ..." ("is" reads "SHALL be")
- "Without --skip-unprovided a render behaves as before": "Without the flag, a render SHALL behave exactly as before this change." reads "Without the flag (`--skip-unprovided`), a render SHALL behave exactly as before this change." (the flag named as the same old requirement names it)
- "With --skip-unprovided a skipped trait is worded as a warning naming the alternatives" and "With --skip-unprovided an omitted component is worded once as a warning": "With the flag, the CLI SHALL print one warning per skipped demand after the render." reads "With the flag (`--skip-unprovided`), the CLI SHALL print one warning per skipped demand after the render." (the flag named as the same old requirement names it)

Lengths (old, new): old 1674; new "Render-bearing commands accept --skip-unprovided and pass it to the kernel" 451; "Without --skip-unprovided a render behaves as before" 92; "With --skip-unprovided a skipped trait is worded as a warning naming the alternatives" 299; "With --skip-unprovided an omitted component is worded once as a warning" 311; "The skipped-demand warnings go to the log stream" 105; "A refusal for unprovided demands adds a hint naming the three ways out" 332; "Instance apply and module apply refuse --skip-unprovided for an operator-managed instance" 224.

### Ledger: kernel-render, A platform pinning an older core is refused as a re-pin

Form: c2 (REMOVED, ADDED under new names). New requirements: "A platform pinning an older core fails as a validation failure with the library message", "A --platform directory with an older core gets a hint with the re-pin command", "The CLI does not fall back or retry after an older-core refusal", "A platform the CLI generates is never refused for its core".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| An older-core --platform directory is refused with the re-pin command | "A --platform directory with an older core gets a hint with the re-pin command", "An older-core --platform directory is refused with the re-pin command" | No (byte equal, checked) |
| A generated platform is never refused for its core | "A platform the CLI generates is never refused for its core", "A generated platform is never refused for its core" | No (byte equal, checked) |
| (new) | "A platform pinning an older core fails as a validation failure with the library message", "The older-core refusal prints the library message" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The CLI does not fall back or retry after an older-core refusal", "No fallback after the refusal" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| When the kernel refuses a render because the platform module pins a co... | body of "A platform pinning an older core fails as a validation failure with the library message", verbatim |
| When the platform is a directory the user named with `--platform`, the... | body of "A --platform directory with an older core gets a hint with the re-pin command", the part of this sentence it holds, verbatim or as the words changed below say |
| The CLI SHALL NOT fall back to another platform source, retry against ... | body of "The CLI does not fall back or retry after an older-core refusal", the part of this sentence it holds, verbatim or as the words changed below say |
| A platform the CLI generates (from the cluster Platform or from the re... | body of "A platform the CLI generates is never refused for its core", the part of this sentence it holds, verbatim or as the words changed below say |

Words changed or added:
- "A --platform directory with an older core gets a hint with the re-pin command": "When the platform is a directory the user named with `--platform`, the command SHALL add a..." reads "In the refusal of requirement "A platform pinning an older core fails as a validation fail..." (the refusal condition named by its requirement; the first letter is lowered)
- "The CLI does not fall back or retry after an older-core refusal": "The CLI SHALL NOT fall back to another platform source, retry against a generated platform..." reads "In the refusal of requirement "A platform pinning an older core fails as a validation fail..." (the refusal condition named by its requirement)
- "A platform the CLI generates is never refused for its core": "A platform the CLI generates (from the cluster Platform or from the render's own dependenc..." reads "A platform the CLI generates (from the cluster Platform or from the render's own dependenc..." ("pins" reads "SHALL pin"; "this way" reads the quoted refusal)

Lengths (old, new): old 1003; new "A platform pinning an older core fails as a validation failure with the library message" 382; "A --platform directory with an older core gets a hint with the re-pin command" 450; "The CLI does not fall back or retry after an older-core refusal" 247; "A platform the CLI generates is never refused for its core" 272.

### Ledger: kernel-render, A colliding platform is refused naming the entries to disable

Form: c2 (REMOVED, ADDED under new names). New requirements: "A colliding platform fails as a validation failure whatever the instance", "A collision refusal lists one row per colliding contract before every other row", "An unresolved demand on a colliding key names the collision", "A platform that is not routable with no explaining row fails with the kernel message", "The CLI does not compute collisions of its own".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A colliding --platform directory is refused with its rows | "A collision refusal lists one row per colliding contract before every other row", "A colliding --platform directory is refused with its rows" | No (byte equal, checked) |
| Skipping unprovided demands does not skip a collision | "A colliding platform fails as a validation failure whatever the instance", "Skipping unprovided demands does not skip a collision" | No (byte equal, checked) |
| An unresolved demand on a colliding key names the collision | "An unresolved demand on a colliding key names the collision", "An unresolved demand on a colliding key names the collision" | No (byte equal, checked) |
| (new) | "A platform that is not routable with no explaining row fails with the kernel message", "A platform not routable with no row" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The CLI does not compute collisions of its own", "The CLI trusts the kernel for collisions" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| When the kernel refuses a render because more than one enabled registr... | body of "A colliding platform fails as a validation failure whatever the instance", the part of this sentence it holds, verbatim or as the words changed below say; body of "A collision refusal lists one row per colliding contract before every other row", the part of this sentence it holds, verbatim or as the words changed below say |
| The refusal applies whatever the instance and whether or not `--skip-u... | body of "A colliding platform fails as a validation failure whatever the instance", verbatim |
| An unresolved demand on a colliding key SHALL be worded as defined by ... | body of "An unresolved demand on a colliding key names the collision", verbatim |
| When the kernel refuses a platform as not routable with no row explain... | body of "A platform that is not routable with no explaining row fails with the kernel message", verbatim |
| The CLI SHALL NOT compute collisions of its own. | body of "The CLI does not compute collisions of its own", verbatim |
| Source: enhancement 0026 (interim safety net until side-by-side majors... | body of "A colliding platform fails as a validation failure whatever the instance", verbatim; body of "A collision refusal lists one row per colliding contract before every other row", verbatim; body of "An unresolved demand on a colliding key names the collision", verbatim; body of "A platform that is not routable with no explaining row fails with the kernel message", verbatim; body of "The CLI does not compute collisions of its own", verbatim |

Words changed or added:
- "A colliding platform fails as a validation failure whatever the instance": "When the kernel refuses a render because more than one enabled registry entry of the platf..." reads "When the kernel refuses a render because more than one enabled registry entry of the platf..." (the first part of the sentence up to the render-failed header, with a period)
- "A collision refusal lists one row per colliding contract before every other row": "When the kernel refuses a render because more than one enabled registry entry of the platf..." reads "When the kernel refuses a render because more than one enabled registry entry of the platf..." (the condition and the clause "every render-bearing command SHALL fail as a validation failure" repeated, then the part from "followed by one row" on; the part about the message under the header stays in the other requirement)

Lengths (old, new): old 1064; new "A colliding platform fails as a validation failure whatever the instance" 408; "A collision refusal lists one row per colliding contract before every other row" 480; "An unresolved demand on a colliding key names the collision" 298; "A platform that is not routable with no explaining row fails with the kernel message" 228; "The CLI does not compute collisions of its own" 121.

### Ledger: deps-cascade, The cascade task moves the cli's upstream pins to published versions

Form: c2 (REMOVED, ADDED under new names). New requirements: "The cascade task moves pins in the working tree only", "The library pin moves to the newest version the Go proxy serves in its major", "The operator module pin moves only through hack/operator-pin", "The operator module target is the newest release whose operator version is not above the cli's own", "The catalog and core pins cover a fixed set of cue.mod/module.cue files", "The catalog and core pins also cover the remaining cue.mod/module.cue files of the fixed set", "The catalog pin moves to the newest published catalog in its major", "The core pin moves to the core that the file's resulting catalog pins", "The kind platform catalog is raised to hack/platform's catalog", "No pin moves backwards, past a hold, or holds a catalog back by a lower core hold", "No pin moves to a version that is tagged but not yet published", "A pin listed in .cascade-frozen does not change", "cue mod get names only the moved keys and runs only where a pin moved", "Third-party pins are not named and a raised one is reported as a warning", "The task uses its own registry mapping".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Older pins move to the published set | "The cascade task moves pins in the working tree only", "Older pins move to the published set" | No (byte equal, checked) |
| Core already ahead of the catalog's pin stays | "The core pin moves to the core that the file's resulting catalog pins", "Core already ahead of the catalog's pin stays" | No (byte equal, checked) |
| A kind catalog ahead of hack/platform stays | "The kind platform catalog is raised to hack/platform's catalog", "A kind catalog ahead of hack/platform stays" | No (byte equal, checked) |
| A hold on core holds the catalog | "No pin moves backwards, past a hold, or holds a catalog back by a lower core hold", "A hold on core holds the catalog" | No (byte equal, checked) |
| An unpublished newest tag leaves the pin | "No pin moves to a version that is tagged but not yet published", "An unpublished newest tag leaves the pin" | No (byte equal, checked) |
| A frozen file is left byte-unchanged | "A pin listed in .cascade-frozen does not change", "A frozen file is left byte-unchanged" | No (byte equal, checked) |
| The third-party k8s pin is never named | "Third-party pins are not named and a raised one is reported as a warning", "The third-party k8s pin is never named" | No (byte equal, checked) |
| (new) | "The cascade task moves pins in the working tree only", "The task edits the tree and nothing else" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The library pin moves to the newest version the Go proxy serves in its major", "The library pin moves" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The operator module pin moves only through hack/operator-pin", "The operator pin moves through the program" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The operator module target is the newest release whose operator version is not above the cli's own", "The target stays within the cli MAJOR.MINOR" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The catalog and core pins cover a fixed set of cue.mod/module.cue files", "The set of files" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The catalog and core pins also cover the remaining cue.mod/module.cue files of the fixed set", "The remaining files of the set" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The catalog pin moves to the newest published catalog in its major", "The catalog pin moves" | New restatement scenario: only words of the old clause, no new case |
| (new) | "cue mod get names only the moved keys and runs only where a pin moved", "cue mod get and tidy run only where a pin moved" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The task uses its own registry mapping", "The mapping holds whatever the environment sets" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `task -x deps:cascade` SHALL move the following pins in the working tr... | body of "The cascade task moves pins in the working tree only", verbatim; body of "The library pin moves to the newest version the Go proxy serves in its major", verbatim; body of "The operator module pin moves only through hack/operator-pin", verbatim; body of "The operator module target is the newest release whose operator version is not above the cli's own", verbatim; body of "The catalog pin moves to the newest published catalog in its major", verbatim; body of "The core pin moves to the core that the file's resulting catalog pins", verbatim; body of "The kind platform catalog is raised to hack/platform's catalog", verbatim |
| It SHALL NOT commit, branch or push. | body of "The cascade task moves pins in the working tree only", verbatim |
| **library.** `github.com/open-platform-model/library` in `go.mod` SHAL... | body of "The library pin moves to the newest version the Go proxy serves in its major", verbatim |
| **opm-operator module.** The operator module pin in `internal/operator... | body of "The operator module pin moves only through hack/operator-pin", verbatim |
| The target is the newest published release of `opmodel.dev/modules/opm... | body of "The operator module target is the newest release whose operator version is not above the cli's own", verbatim |
| **Catalog and core as a consistent set.** This covers the `cue.mod/mod... | body of "The catalog and core pins cover a fixed set of cue.mod/module.cue files" and body of "The catalog and core pins also cover the remaining cue.mod/module.cue files of the fixed set", the part of this sentence each holds, verbatim or as the words changed below say; body of "The catalog pin moves to the newest published catalog in its major", the part of this sentence it holds, verbatim or as the words changed below say; body of "The core pin moves to the core that the file's resulting catalog pins", the part of this sentence it holds, verbatim or as the words changed below say |
| `opmodel.dev/catalogs/opm@v4` SHALL move to the newest published catal... | body of "The catalog pin moves to the newest published catalog in its major", verbatim |
| `opmodel.dev/core@v2` SHALL move to the core version that the file's r... | body of "The core pin moves to the core that the file's resulting catalog pins", verbatim |
| A file without a catalog SHALL use `templates/minimal`'s resulting cat... | body of "The core pin moves to the core that the file's resulting catalog pins", verbatim |
| **Kind Platform.** The bare `version:` under `opmodel.dev/catalogs/opm... | body of "The kind platform catalog is raised to hack/platform's catalog", verbatim |
| A version above it SHALL stay, with a warning, since no pin moves back... | body of "The kind platform catalog is raised to hack/platform's catalog", verbatim |
| How every move behaves: No pin SHALL move backwards. | body of "No pin moves backwards, past a hold, or holds a catalog back by a lower core hold", verbatim |
| No pin SHALL move past an in-date `.cascade-hold` `max`. | body of "No pin moves backwards, past a hold, or holds a catalog back by a lower core hold", verbatim |
| When a hold on core is below the core a newer catalog pins, the catalo... | body of "No pin moves backwards, past a hold, or holds a catalog back by a lower core hold", verbatim |
| No pin SHALL move to a version that is tagged but not yet published. | body of "No pin moves to a version that is tagged but not yet published", verbatim |
| No pin listed for a file in `.cascade-frozen` SHALL change in that fil... | body of "A pin listed in .cascade-frozen does not change", verbatim |
| If `cue mod tidy` raises such a pin, the task SHALL fail, naming the f... | body of "A pin listed in .cascade-frozen does not change", verbatim |
| `cue mod get` SHALL name only the moved `opmodel.dev` keys, at exact v... | body of "cue mod get names only the moved keys and runs only where a pin moved", verbatim |
| It and `cue mod tidy` SHALL run only in a module where a pin moved. | body of "cue mod get names only the moved keys and runs only where a pin moved", verbatim |
| Third-party pins, including `cue.dev/x/k8s.io@v0`, SHALL NOT be named. | body of "Third-party pins are not named and a raised one is reported as a warning", verbatim |
| A third-party pin that `tidy` or `go mod tidy` raised SHALL be reporte... | body of "Third-party pins are not named and a raised one is reported as a warning", verbatim |
| The task SHALL use its own registry mapping, `testing.opmodel.dev=ghcr... | body of "The task uses its own registry mapping", verbatim, with the command named as the words added below say |
| The source is workspace RELEASING.md, sections "The cascade" and "What... | body of "The cascade task moves pins in the working tree only", verbatim; body of "The catalog and core pins cover a fixed set of cue.mod/module.cue files", verbatim; body of "The catalog and core pins also cover the remaining cue.mod/module.cue files of the fixed set", verbatim; body of "The library pin moves to the newest version the Go proxy serves in its major", verbatim; body of "The operator module pin moves only through hack/operator-pin", verbatim; body of "The operator module target is the newest release whose operator version is not above the cli's own", verbatim; body of "The catalog pin moves to the newest published catalog in its major", verbatim; body of "The core pin moves to the core that the file's resulting catalog pins", verbatim; body of "The kind platform catalog is raised to hack/platform's catalog", verbatim; body of "No pin moves backwards, past a hold, or holds a catalog back by a lower core hold", verbatim; body of "No pin moves to a version that is tagged but not yet published", verbatim; body of "A pin listed in .cascade-frozen does not change", verbatim; body of "cue mod get names only the moved keys and runs only where a pin moved", verbatim; body of "Third-party pins are not named and a raised one is reported as a warning", verbatim; body of "The task uses its own registry mapping", verbatim |

Words changed or added:
- "The operator module target is the newest release whose operator version is not above the cli's own": "The target is the newest published release of `opmodel.dev/modules/opm_operator` in the pi..." reads "**opm-operator module.** The target is the newest published release of `opmodel.dev/module..." (the list label copied in front of the sentence)
- "The catalog and core pins cover a fixed set of cue.mod/module.cue files": "**Catalog and core as a consistent set.** This covers the `cue.mod/module.cue` files of th..." reads "**Catalog and core as a consistent set.** The consistent set SHALL cover the `cue.mod/modu..." ("This covers" reads "The consistent set SHALL cover")
- "The catalog and core pins also cover the remaining cue.mod/module.cue files of the fixed set": "**Catalog and core as a consistent set.** This covers the `cue.mod/module.cue` files of th..." reads "**Catalog and core as a consistent set.** The consistent set SHALL cover the `cue.mod/module.cue` files of `internal/workflow/render/testdata/skip-unprovided`, ..." ("This covers" reads "The consistent set SHALL cover"; the file list is cut after `internal/instinit/testdata/initvalues`, each part verbatim, the subject repeated)
- "The task uses its own registry mapping", "The cascade task does not turn a failure into a fallback version" and "A version is written only when it is greater than the file version": "The task SHALL ..." reads "The task (`task -x deps:cascade`) SHALL ..." (the command named as the same old requirement names it)
- "The new version is one patch above the merge base's, or the merge base's when unpublished": "The new version SHALL be one patch above the merge base's declared version when that version is published." reads "The new version of the three templates (`opmodel.dev/templates/<name>@v1`) and the podinfo fixture (`testing.opmodel.dev/modules/cli/podinfo@v0`) SHALL be one patch above ..." (the subject of the first sentence of the same old requirement, copied)
- "The offline set also checks a core ahead of its catalog and a hold on core", "The full set also checks a second catalog move on a branch pinning the unpublished fixture", "The offline set runs as a step of the Lint job in pr.yml" and "The full set runs in cascade-task.yml and is not a required check": "The offline set SHALL ..." and "The full set SHALL ..." read "The offline set of `task -x deps:cascade:test` SHALL ..." and "The full set of `task -x deps:cascade:test` SHALL ..." (the command named as the same old requirement names it)
- "The catalog pin moves to the newest published catalog in its major": "`opmodel.dev/catalogs/opm@v4` SHALL move to the newest published catalog in its major, res..." reads "**Catalog and core as a consistent set.** `opmodel.dev/catalogs/opm@v4` SHALL move to the ..." (the list label copied in front of the sentence)
- "The core pin moves to the core that the file's resulting catalog pins": "`opmodel.dev/core@v2` SHALL move to the core version that the file's resulting catalog pin..." reads "**Catalog and core as a consistent set.** `opmodel.dev/core@v2` SHALL move to the core ver..." (the list label copied in front of the sentence)

Lengths (old, new): old 3012; new "The cascade task moves pins in the working tree only" 211; "The library pin moves to the newest version the Go proxy serves in its major" 376; "The operator module pin moves only through hack/operator-pin" 444; "The operator module target is the newest release whose operator version is not above the cli's own" 482; "The catalog and core pins cover a fixed set of cue.mod/module.cue files" 364; "The catalog pin moves to the newest published catalog in its major" 351; "The core pin moves to the core that the file's resulting catalog pins" 385; "The kind platform catalog is raised to hack/platform's catalog" 434; "No pin moves backwards, past a hold, or holds a catalog back by a lower core hold" 309; "No pin moves to a version that is tagged but not yet published" 164; "A pin listed in .cascade-frozen does not change" 255; "cue mod get names only the moved keys and runs only where a pin moved" 242; "Third-party pins are not named and a raised one is reported as a warning" 252; "The task uses its own registry mapping" 350.

### Ledger: deps-cascade, The cascade task resolves before it edits and reports by exit code

Form: c2 (REMOVED, ADDED under new names). New requirements: "A dirty working tree is refused with exit 1", "The frozen and hold files are validated before anything else", "Every target-deciding resolver call runs before the first file edit", "A failure after the first edit ends the task with an error code", "Exit 0 means the working tree changed and exit 3 means it did not", "The cascade task does not turn a failure into a fallback version".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Nothing to move | "Exit 0 means the working tree changed and exit 3 means it did not", "Nothing to move" | No (byte equal, checked) |
| A resolver error leaves the tree clean | "Every target-deciding resolver call runs before the first file edit", "A resolver error leaves the tree clean" | No (byte equal, checked) |
| A dirty tree is refused | "A dirty working tree is refused with exit 1", "A dirty tree is refused" | No (byte equal, checked) |
| (new) | "The frozen and hold files are validated before anything else", "The files are validated first" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A failure after the first edit ends the task with an error code", "A late failure may leave a partly edited tree" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The cascade task does not turn a failure into a fallback version", "No fallback version" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `task -x deps:cascade` SHALL refuse with exit 1 when the working tree ... | body of "A dirty working tree is refused with exit 1", verbatim |
| It SHALL validate `.cascade-frozen` and `.cascade-hold` through the re... | body of "The frozen and hold files are validated before anything else", the part of this sentence it holds, verbatim or as the words changed below say |
| Order of work: Every resolver call that decides a target SHALL run bef... | body of "Every target-deciding resolver call runs before the first file edit", verbatim |
| A failing target-deciding call, or any answer other than "move" or "st... | body of "Every target-deciding resolver call runs before the first file edit", verbatim |
| A failure after the first edit (a tool, or a resolver call that decide... | body of "A failure after the first edit ends the task with an error code", verbatim |
| The tree may then be partly edited; callers discard it. | body of "A failure after the first edit ends the task with an error code", verbatim |
| Exit codes: 0 when the working tree changed; 3 when it did not; any ot... | body of "Exit 0 means the working tree changed and exit 3 means it did not", the part of this sentence it holds, verbatim or as the words changed below say |
| The task SHALL NOT turn a failure into a fallback version. | body of "The cascade task does not turn a failure into a fallback version", verbatim, with the command named as the words added below say |
| Source: workspace RELEASING.md, section "The cascade", "The receiver". | body of "A dirty working tree is refused with exit 1", verbatim; body of "The frozen and hold files are validated before anything else", verbatim; body of "Every target-deciding resolver call runs before the first file edit", verbatim; body of "A failure after the first edit ends the task with an error code", verbatim; body of "Exit 0 means the working tree changed and exit 3 means it did not", verbatim; body of "The cascade task does not turn a failure into a fallback version", verbatim |

Words changed or added:
- "The frozen and hold files are validated before anything else": "It SHALL validate `.cascade-frozen` and `.cascade-hold` through the resolver's `check-file..." reads "`task -x deps:cascade` SHALL validate `.cascade-frozen` and `.cascade-hold` through the re..." ("It" reads "`task -x deps:cascade`")
- "Exit 0 means the working tree changed and exit 3 means it did not": "Exit codes: 0 when the working tree changed; 3 when it did not; any other code on error." reads "Exit codes SHALL be: 0 when the working tree changed; 3 when it did not; any other code on..." ("Exit codes:" reads "Exit codes SHALL be:")

Lengths (old, new): old 963; new "A dirty working tree is refused with exit 1" 213; "The frozen and hold files are validated before anything else" 205; "Every target-deciding resolver call runs before the first file edit" 324; "A failure after the first edit ends the task with an error code" 291; "Exit 0 means the working tree changed and exit 3 means it did not" 168; "The cascade task does not turn a failure into a fallback version" 154.

### Ledger: deps-cascade, Template and fixture versions advance once per pull request

Form: c2 (REMOVED, ADDED under new names). New requirements: "Templates and the podinfo fixture get a new version when their tree differs from the merge base", "The new version is one patch above the merge base's, or the merge base's when unpublished", "A second run in the same pull request advances no version again", "A version is written only when it is greater than the file version", "The podinfo pin in the two consumers follows the fixture version without cue mod get or tidy", "A consumer pinning an unpublished podinfo version is pointed at the merge base version for cue mod get and tidy".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| First run advances each changed module once | "Templates and the podinfo fixture get a new version when their tree differs from the merge base", "First run advances each changed module once" | No (byte equal, checked) |
| Second run does not advance again | "A second run in the same pull request advances no version again", "Second run does not advance again" | No (byte equal, checked) |
| A consumer pinning the unpublished fixture still moves | "A consumer pinning an unpublished podinfo version is pointed at the merge base version for cue mod get and tidy", "A consumer pinning the unpublished fixture still moves" | No (byte equal, checked) |
| (new) | "The new version is one patch above the merge base's, or the merge base's when unpublished", "The target version" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A version is written only when it is greater than the file version", "A higher file version stays" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The podinfo pin in the two consumers follows the fixture version without cue mod get or tidy", "The consumers follow the fixture" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The three templates (`opmodel.dev/templates/<name>@v1`) and the podinf... | body of "Templates and the podinfo fixture get a new version when their tree differs from the merge base", verbatim |
| The new version SHALL be one patch above the merge base's declared ver... | body of "The new version is one patch above the merge base's, or the merge base's when unpublished", verbatim, with the subject named as the words added below say |
| It SHALL be the merge base's version when that version is not yet publ... | body of "The new version is one patch above the merge base's, or the merge base's when unpublished", verbatim |
| A second run in the same pull request SHALL NOT advance any version ag... | body of "A second run in the same pull request advances no version again", verbatim |
| The task SHALL write a version only when it is greater than the file's... | body of "A version is written only when it is greater than the file version", verbatim, with the command named as the words added below say |
| A file version above the target SHALL stay, with a warning. | body of "A version is written only when it is greater than the file version", verbatim |
| The podinfo pin in `examples/cue.mod/module.cue` and `tests/e2e/testda... | body of "The podinfo pin in the two consumers follows the fixture version without cue mod get or tidy", verbatim |
| No `cue mod get` or `cue mod tidy` SHALL follow that rewrite. | body of "The podinfo pin in the two consumers follows the fixture version without cue mod get or tidy", verbatim |
| When a consumer's catalog or core moves while it pins a podinfo versio... | body of "A consumer pinning an unpublished podinfo version is pointed at the merge base version for cue mod get and tidy", verbatim |
| Source: workspace RELEASING.md, section "The cascade", "The receiver" ... | body of "Templates and the podinfo fixture get a new version when their tree differs from the merge base", verbatim; body of "The new version is one patch above the merge base's, or the merge base's when unpublished", verbatim; body of "A second run in the same pull request advances no version again", verbatim; body of "A version is written only when it is greater than the file version", verbatim; body of "The podinfo pin in the two consumers follows the fixture version without cue mod get or tidy", verbatim; body of "A consumer pinning an unpublished podinfo version is pointed at the merge base version for cue mod get and tidy", verbatim |

Lengths (old, new): old 1341; new "Templates and the podinfo fixture get a new version when their tree differs from the merge base" 424; "The new version is one patch above the merge base's, or the merge base's when unpublished" 424; "A second run in the same pull request advances no version again" 185; "A version is written only when it is greater than the file version" 267; "The podinfo pin in the two consumers follows the fixture version without cue mod get or tidy" 395; "A consumer pinning an unpublished podinfo version is pointed at the merge base version for cue mod get and tidy" 343.

### Ledger: deps-cascade, Title and body tasks describe the cascade diff

Form: c2 (REMOVED, ADDED under new names). New requirements: "The title task prints the resolver title using the path-class map", "The body task prints the resolver body for the same diff", "The pins script reports the library and operator pins", "The pins script reports the catalog and core pins", "Each pin of the pins script is of class shipped with no label".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A shipped move titles as fix(deps) | "The title task prints the resolver title using the path-class map", "A shipped move titles as fix(deps)" | No (byte equal, checked) |
| Test-only paths title as test(fixtures) | "The title task prints the resolver title using the path-class map", "Test-only paths title as test(fixtures)" | No (byte equal, checked) |
| The body lists each moved pin | "The body task prints the resolver body for the same diff", "The body lists each moved pin" | No (byte equal, checked) |
| A module-only pin move is reported | "The pins script reports the library and operator pins", "A module-only pin move is reported" | No (byte equal, checked) |
| (new) | "The pins script reports the catalog and core pins", "Catalog and core are reported" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Each pin of the pins script is of class shipped with no label", "Class of a reported pin" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `task -x deps:cascade:title` SHALL print the shared resolver's one-lin... | body of "The title task prints the resolver title using the path-class map", verbatim; body of "The body task prints the resolver body for the same diff", verbatim |
| It SHALL use the path-class map `.tasks/cascade/classes`, which classe... | body of "The title task prints the resolver title using the path-class map", verbatim |
| `task -x deps:cascade:body` SHALL print the resolver's body for the sa... | body of "The body task prints the resolver body for the same diff", verbatim |
| `.tasks/cascade/pins.sh <ref>` SHALL report five pins, each `v`-prefix... | body of "The pins script reports the library and operator pins", the part of this sentence it holds, verbatim or as the words changed below say; body of "The pins script reports the catalog and core pins", the part of this sentence it holds, verbatim or as the words changed below say; body of "Each pin of the pins script is of class shipped with no label", the part of this sentence it holds, verbatim or as the words changed below say |
| Each pin SHALL be of class `shipped`, with no label. | body of "Each pin of the pins script is of class shipped with no label", verbatim |
| Source: workspace RELEASING.md, section "The cascade", "Title from dif... | body of "The title task prints the resolver title using the path-class map", verbatim; body of "The body task prints the resolver body for the same diff", verbatim; body of "The pins script reports the library and operator pins", verbatim; body of "The pins script reports the catalog and core pins", verbatim; body of "Each pin of the pins script is of class shipped with no label", verbatim |

Words changed or added:
- "The pins script reports the library and operator pins": "`.tasks/cascade/pins.sh <ref>` SHALL report five pins, each `v`-prefixed: `github.com/open..." reads "`.tasks/cascade/pins.sh <ref>` SHALL report five pins, each `v`-prefixed: `github.com/open..." (the sentence cut at the list boundary after the third item; the final semicolon reads a period)
- "The pins script reports the catalog and core pins": "`.tasks/cascade/pins.sh <ref>` SHALL report five pins, each `v`-prefixed: `github.com/open..." reads "`.tasks/cascade/pins.sh <ref>` SHALL report five pins, each `v`-prefixed: `opmodel.dev/cat..." (the sentence cut at the list boundary: the lead, then the last two items)
- "Each pin of the pins script is of class shipped with no label": "`.tasks/cascade/pins.sh <ref>` SHALL report five pins, each `v`-prefixed: `github.com/open..." reads "`.tasks/cascade/pins.sh <ref>` SHALL report five pins, each `v`-prefixed." (the lead of the sentence with a period)

Lengths (old, new): old 1161; new "The title task prints the resolver title using the path-class map" 454; "The body task prints the resolver body for the same diff" 297; "The pins script reports the library and operator pins" 490; "The pins script reports the catalog and core pins" 295; "Each pin of the pins script is of class shipped with no label" 206.

### Ledger: deps-cascade, The cascade task is tested offline in required CI and fully in a network job

Form: c2 (REMOVED, ADDED under new names). New requirements: "The cascade test runs the task against the stub resolver in sandbox copies", "The offline set runs without registry or proxy access and checks fixed items", "The full set adds the older-pins, frozen-file and title and body scenarios", "The offline set also checks a core ahead of its catalog and a hold on core", "The full set also checks a second catalog move on a branch pinning the unpublished fixture", "The setup lowers the library to a version made from the tree and a failed setup fails its scenario", "The offline set runs as a step of the Lint job in pr.yml", "The full set runs in cascade-task.yml and is not a required check".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Offline set runs on every pull request | "The offline set runs as a step of the Lint job in pr.yml", "Offline set runs on every pull request" | No (byte equal, checked) |
| The checkout is never modified by the test | "The cascade test runs the task against the stub resolver in sandbox copies", "The checkout is never modified by the test" | No (byte equal, checked) |
| The full set passes when the cli uses the newest library's API | "The setup lowers the library to a version made from the tree and a failed setup fails its scenario", "The full set passes when the cli uses the newest library's API" | No (byte equal, checked) |
| A stale older row fails the offline set | "The offline set runs without registry or proxy access and checks fixed items", "A stale older row fails the offline set" | No (byte equal, checked) |
| (new) | "The full set adds the older-pins, frozen-file and title and body scenarios", "The full set adds scenarios" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The offline set also checks a core ahead of its catalog and a hold on core", "Offline checks on core" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The full set also checks a second catalog move on a branch pinning the unpublished fixture", "Full check on a second move" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The full set runs in cascade-task.yml and is not a required check", "The network job runs the full set" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `task -x deps:cascade:test` SHALL run the cascade task against the stu... | body of "The cascade test runs the task against the stub resolver in sandbox copies", verbatim |
| Its offline set, selected by `CASCADE_TEST_SET=offline`, SHALL run wit... | body of "The offline set runs without registry or proxy access and checks fixed items", the part of this sentence it holds, verbatim or as the words changed below say |
| It SHALL check: the stub checksum; that `pins.sh` reads the worktree a... | body of "The offline set runs without registry or proxy access and checks fixed items", verbatim |
| The full set SHALL add: the older-pins scenario, including its second,... | body of "The full set adds the older-pins, frozen-file and title and body scenarios", the part of this sentence it holds, verbatim or as the words changed below say |
| The offline set SHALL also check that a core ahead of its catalog's pi... | body of "The offline set also checks a core ahead of its catalog and a hold on core", verbatim, with the command named as the words added below say |
| The full set SHALL also check a second catalog move on a branch whose ... | body of "The full set also checks a second catalog move on a branch pinning the unpublished fixture", verbatim, with the command named as the words added below say |
| The setup of the older-pins, frozen-file and second-move scenarios SHA... | body of "The setup lowers the library to a version made from the tree and a failed setup fails its scenario", verbatim |
| A setup that fails SHALL fail its scenario with the failed step and th... | body of "The setup lowers the library to a version made from the tree and a failed setup fails its scenario", verbatim |
| The offline set SHALL run as a step of the `Lint` job in `.github/work... | body of "The offline set runs as a step of the Lint job in pr.yml", verbatim, with the command named as the words added below say |
| The full set SHALL run in `.github/workflows/cascade-task.yml`, job `C... | body of "The full set runs in cascade-task.yml and is not a required check", verbatim, with the command named as the words added below say |

Words changed or added:
- "The offline set runs without registry or proxy access and checks fixed items": "Its offline set, selected by `CASCADE_TEST_SET=offline`, SHALL run without registry or pro..." reads "The offline set of `task -x deps:cascade:test`, selected by `CASCADE_TEST_SET=offline`, SH..." ("Its offline set" reads "The offline set of `task -x deps:cascade:test`")
- "The full set adds the older-pins, frozen-file and title and body scenarios": "The full set SHALL add: the older-pins scenario, including its second, idempotent run; the..." reads "The full set of `task -x deps:cascade:test` SHALL add: the older-pins scenario, including ..." ("The full set" reads "The full set of `task -x deps:cascade:test`")

Lengths (old, new): old 1740; new "The cascade test runs the task against the stub resolver in sandbox copies" 142; "The offline set runs without registry or proxy access and checks fixed items" 492; "The full set adds the older-pins, frozen-file and title and body scenarios" 226; "The offline set also checks a core ahead of its catalog and a hold on core" 167; "The full set also checks a second catalog move on a branch pinning the unpublished fixture" 147; "The setup lowers the library to a version made from the tree and a failed setup fails its scenario" 368; "The offline set runs as a step of the Lint job in pr.yml" 200; "The full set runs in cascade-task.yml and is not a required check" 155.

### Ledger: deps-cascade, The cascade task leaves non-cascade files alone and runs no dependency code

Form: c2 (REMOVED, ADDED under new names). New requirements: "The cascade task edits none of the frozen, release, workflow, docs-version or fixture files", "The cascade task does not publish, seed a registry or push", "The cascade task runs no program that links a moved Go dependency", "The cascade task appends a warning without failing in two cases".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Release and settings files stay untouched by a full run | "The cascade task edits none of the frozen, release, workflow, docs-version or fixture files", "Release and settings files stay untouched by a full run" | No (byte equal, checked) |
| A library move runs no library code | "The cascade task runs no program that links a moved Go dependency", "A library move runs no library code" | No (byte equal, checked) |
| Merge mode builds opm from the merge base | "The cascade task runs no program that links a moved Go dependency", "Merge mode builds opm from the merge base" | No (byte equal, checked) |
| A newer language version is a warning | "The cascade task appends a warning without failing in two cases", "A newer language version is a warning" | No (byte equal, checked) |
| (new) | "The cascade task does not publish, seed a registry or push", "No publish or push" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `task deps:cascade` SHALL NOT edit any of these: `.cascade-frozen` or ... | body of "The cascade task edits none of the frozen, release, workflow, docs-version or fixture files", verbatim |
| It SHALL NOT publish, seed a real registry or push. | body of "The cascade task does not publish, seed a registry or push", the part of this sentence it holds, verbatim or as the words changed below say |
| It SHALL NOT build or run a program that links a moved Go dependency: ... | body of "The cascade task runs no program that links a moved Go dependency", the part of this sentence it holds, verbatim or as the words changed below say |
| It SHALL append a warning to the cascade warnings file, without failin... | body of "The cascade task appends a warning without failing in two cases", the part of this sentence it holds, verbatim or as the words changed below say |

Words changed or added:
- "The cascade task does not publish, seed a registry or push": "It SHALL NOT publish, seed a real registry or push." reads "`task deps:cascade` SHALL NOT publish, seed a real registry or push." ("It" reads "`task deps:cascade`")
- "The cascade task runs no program that links a moved Go dependency": "It SHALL NOT build or run a program that links a moved Go dependency: the only Go programs..." reads "`task deps:cascade` SHALL NOT build or run a program that links a moved Go dependency: the..." ("It" reads "`task deps:cascade`")
- "The cascade task appends a warning without failing in two cases": "It SHALL append a warning to the cascade warnings file, without failing, in each of these ..." reads "`task deps:cascade` SHALL append a warning to the cascade warnings file, without failing, ..." ("It" reads "`task deps:cascade`")

Lengths (old, new): old 999; new "The cascade task edits none of the frozen, release, workflow, docs-version or fixture files" 309; "The cascade task does not publish, seed a registry or push" 68; "The cascade task runs no program that links a moved Go dependency" 350; "The cascade task appends a warning without failing in two cases" 304.

### Ledger: instance-init, Init writes a complete standalone instance package

Form: c2 (REMOVED, ADDED under new names). New requirements: "Init writes exactly three files forming a standalone CUE module", "The cue.mod/module.cue file declares the package's module path, language version and dependencies", "The module file is exactly what opm module tidy leaves", "The instance.cue file declares the package, the imports and the module instance", "The values.cue file declares one values field", "The package's own module path defaults to a local placeholder and is never looked up".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Package for a published module | "Init writes exactly three files forming a standalone CUE module", "Package for a published module" | No (byte equal, checked) |
| Builds with no step in between | "Init writes exactly three files forming a standalone CUE module", "Builds with no step in between" | No (byte equal, checked) |
| Module path override | "The package's own module path defaults to a local placeholder and is never looked up", "Module path override" | No (byte equal, checked) |
| (new) | "The cue.mod/module.cue file declares the package's module path, language version and dependencies", "The module file declares the closure" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The module file is exactly what opm module tidy leaves", "A tidy check of the module file passes" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The instance.cue file declares the package, the imports and the module instance", "The instance file" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The values.cue file declares one values field", "The values file" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm instance init` SHALL write exactly three files into the target di... | body of "Init writes exactly three files forming a standalone CUE module", verbatim |
| Every line derives from the acquired module; nothing is copied from a ... | body of "Init writes exactly three files forming a standalone CUE module", verbatim |
| Source: enhancement 0016 ("A new CLI command scaffolds an on-disk inst... | body of "Init writes exactly three files forming a standalone CUE module", verbatim |
| `cue.mod/module.cue` declares the package's own module path, a languag... | body of "The cue.mod/module.cue file declares the package's module path, language version and dependencies", the part of this sentence it holds, verbatim or as the words changed below say |
| The file SHALL be exactly what `opm module tidy` leaves, so a tidy che... | body of "The module file is exactly what opm module tidy leaves", the part of this sentence it holds, verbatim or as the words changed below say |
| Source: enhancement 0016 ("The generated module file pins the module e... | body of "The cue.mod/module.cue file declares the package's module path, language version and dependencies", verbatim; body of "The module file is exactly what opm module tidy leaves", verbatim; body of "The package's own module path defaults to a local placeholder and is never looked up", verbatim |
| `instance.cue` declares `package instance`, imports core at the module... | body of "The instance.cue file declares the package, the imports and the module instance", the part of this sentence it holds, verbatim or as the words changed below say |
| `values.cue` declares `package instance` and one `values` field, fille... | body of "The values.cue file declares one values field", with "declares" and the cited header read as the list below says |
| The package's own module path SHALL default to `instance.local/<instan... | body of "The package's own module path defaults to a local placeholder and is never looked up", verbatim |
| `--module-path` (string, default empty) overrides it with any valid ma... | body of "The package's own module path defaults to a local placeholder and is never looked up", verbatim |
| The path is never looked up in a registry. | body of "The package's own module path defaults to a local placeholder and is never looked up", verbatim |
| Source: enhancement 0016 ("The generated module file pins the module e... | body of "The cue.mod/module.cue file declares the package's module path, language version and dependencies", verbatim; body of "The module file is exactly what opm module tidy leaves", verbatim; body of "The package's own module path defaults to a local placeholder and is never looked up", verbatim |

Words changed or added:
- "The cue.mod/module.cue file declares the package's module path, language version and dependencies": "`cue.mod/module.cue` declares the package's own module path, a language version, and depen..." reads "`cue.mod/module.cue` SHALL declare the package's own module path, a language version, and ..." ("declares" reads "SHALL declare")
- "The module file is exactly what opm module tidy leaves": "The file SHALL be exactly what `opm module tidy` leaves, so a tidy check of it passes." reads "The `cue.mod/module.cue` file SHALL be exactly what `opm module tidy` leaves, so a tidy ch..." ("The file" reads "The `cue.mod/module.cue` file")
- "The instance.cue file declares the package, the imports and the module instance": "`instance.cue` declares `package instance`, imports core at the module's core major and th..." reads "`instance.cue` SHALL declare `package instance`, imports core at the module's core major a..." ("declares" reads "SHALL declare")
- "The values.cue file declares one values field": "`values.cue` declares `package instance` and one `values` field, filled as "Values start f..." reads "`values.cue` SHALL declare `package instance` and one `values` field, filled as "values.cu..." ("declares" reads "SHALL declare"; the cite of the renamed requirement reads its new header "values.cue is filled from the first applicable source and the report names it")

Lengths (old, new): old 1642; new "Init writes exactly three files forming a standalone CUE module" 380; "The cue.mod/module.cue file declares the package's module path, language version and dependencies" 422; "The module file is exactly what opm module tidy leaves" 266; "The instance.cue file declares the package, the imports and the module instance" 238; "The values.cue file declares one values field" 169; "The package's own module path defaults to a local placeholder and is never looked up" 382.

### Ledger: instance-init, Init arguments mirror module init

Form: c2 (REMOVED, ADDED under new names). New requirements: "Init takes the positionals and flags of module init", "With one positional a value with a slash or a dot is the module path", "Naming the module path twice is refused", "The module path is major-free", "The instance name and the namespace match the core name type and --dir defaults to the instance name", "A missing input is prompted on a terminal and refused without one", "A prompted value passes the same checks as the flag it replaces".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Positional module path | "With one positional a value with a slash or a dot is the module path", "Positional module path" | No (byte equal, checked) |
| Module path named twice | "Naming the module path twice is refused", "Module path named twice" | No (byte equal, checked) |
| Major suffix refused | "The module path is major-free", "Major suffix refused" | No (byte equal, checked) |
| Missing namespace without a terminal | "A missing input is prompted on a terminal and refused without one", "Missing namespace without a terminal" | No (byte equal, checked) |
| Missing module path without a terminal | "A missing input is prompted on a terminal and refused without one", "Missing module path without a terminal" | No (byte equal, checked) |
| Missing module path on a terminal | "A missing input is prompted on a terminal and refused without one", "Missing module path on a terminal" | No (byte equal, checked) |
| Invalid instance name | "The instance name and the namespace match the core name type and --dir defaults to the instance name", "Invalid instance name" | No (byte equal, checked) |
| (new) | "Init takes the positionals and flags of module init", "Positionals and flags" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A prompted value passes the same checks as the flag it replaces", "A prompted value is checked" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The command SHALL take `[instance-name] [module-path]` positionals and... | body of "Init takes the positionals and flags of module init", verbatim |
| With one positional, a value containing `/` or `.` is the module path,... | body of "With one positional a value with a slash or a dot is the module path", the part of this sentence it holds, verbatim or as the words changed below say |
| `--from` is another spelling of the module path; naming the module pat... | body of "Naming the module path twice is refused", verbatim |
| The module path SHALL be major-free: a path carrying a major suffix is... | body of "The module path is major-free", verbatim |
| The instance name and the namespace SHALL each match the core name typ... | body of "The instance name and the namespace match the core name type and --dir defaults to the instance name", verbatim |
| `--dir` SHALL default to the instance name. | body of "The instance name and the namespace match the core name type and --dir defaults to the instance name", verbatim |
| When the instance name, the module path or the namespace is missing an... | body of "A missing input is prompted on a terminal and refused without one", verbatim |
| A prompted value passes the same checks as the flag or positional it r... | body of "A prompted value passes the same checks as the flag it replaces", the part of this sentence it holds, verbatim or as the words changed below say |
| Source: enhancement 0016 (a published module floats to the highest cor... | body of "Init takes the positionals and flags of module init", verbatim; body of "With one positional a value with a slash or a dot is the module path", verbatim; body of "Naming the module path twice is refused", verbatim; body of "The module path is major-free", verbatim; body of "The instance name and the namespace match the core name type and --dir defaults to the instance name", verbatim; body of "A missing input is prompted on a terminal and refused without one", verbatim; body of "A prompted value passes the same checks as the flag it replaces", verbatim |

Words changed or added:
- "With one positional a value with a slash or a dot is the module path": "With one positional, a value containing `/` or `.` is the module path, and anything else i..." reads "With one positional, a value containing `/` or `.` SHALL be the module path, and anything ..." ("is" reads "SHALL be" twice)
- "A prompted value passes the same checks as the flag it replaces": "A prompted value passes the same checks as the flag or positional it replaces, before any ..." reads "A prompted value SHALL pass the same checks as the flag or positional it replaces, before ..." ("passes" reads "SHALL pass")
- "Init takes the positionals and flags of module init": "The command SHALL take ..." reads "The command (`opm instance init`) SHALL take ..." (the command named as the scenarios of the same old requirement name it)

Lengths (old, new): old 1342; new "Init takes the positionals and flags of module init" 310; "With one positional a value with a slash or a dot is the module path" 213; "Naming the module path twice is refused" 217; "The module path is major-free" 230; "The instance name and the namespace match the core name type and --dir defaults to the instance name" 346; "A missing input is prompted on a terminal and refused without one" 403; "A prompted value passes the same checks as the flag it replaces" 201.

### Ledger: instance-init, Values start from the author's intended source

Form: c2 (REMOVED, ADDED under new names). New requirements: "values.cue is filled from the first applicable source and the report names it", "The initValues source", "No debugValues content reaches the package", "The debugValues source", "A debugValues source makes the report warn the user to review the file", "With no usable source values.cue holds an empty struct and the report warns", "A concrete non-struct source is rendered verbatim", "An empty struct source is still named and carries the empty-file warning", "values.cue opens with a comment naming its source".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| debugValues used and flagged | "A debugValues source makes the report warn the user to review the file", "debugValues used and flagged" | No (byte equal, checked) |
| initValues takes precedence | "No debugValues content reaches the package", "initValues takes precedence" | No (byte equal, checked) |
| Non-concrete initValues | "The initValues source", "Non-concrete initValues" | No (byte equal, checked) |
| No usable source | "With no usable source values.cue holds an empty struct and the report warns", "No usable source" | No (byte equal, checked) |
| Partly concrete debugValues | "The debugValues source", "Partly concrete debugValues" | No (byte equal, checked) |
| Empty debugValues | "An empty struct source is still named and carries the empty-file warning", "Empty debugValues" | No (byte equal, checked) |
| (new) | "values.cue is filled from the first applicable source and the report names it", "The report names the source" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A concrete non-struct source is rendered verbatim", "Non-struct source verbatim" | New restatement scenario: only words of the old clause, no new case |
| (new) | "values.cue opens with a comment naming its source", "The comment names the source" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `values.cue` SHALL be filled from the first applicable source in this ... | body of "values.cue is filled from the first applicable source and the report names it", the part of this sentence it holds, verbatim or as the words changed below say; body of "The initValues source", the part of this sentence it holds, verbatim or as the words changed below say; body of "No debugValues content reaches the package", the part of this sentence it holds, verbatim or as the words changed below say; body of "The debugValues source", the part of this sentence it holds, verbatim or as the words changed below say; body of "A debugValues source makes the report warn the user to review the file", the part of this sentence it holds, verbatim or as the words changed below say; body of "With no usable source values.cue holds an empty struct and the report warns", the part of this sentence it holds, verbatim or as the words changed below say |
| The module's `initValues`, when the module declares it. | body of "The initValues source", verbatim; body of "No debugValues content reaches the package", verbatim |
| Its content is rendered even when non-concrete: a defaulted field appe... | body of "The initValues source", verbatim |
| No `debugValues` content SHALL reach the package. | body of "No debugValues content reaches the package", verbatim |
| Source: enhancement 0016 ("`#Module` gains an optional field carrying ... | body of "No debugValues content reaches the package", the part of this sentence it holds, verbatim or as the words changed below say |
| Otherwise the module's `debugValues`, when the whole value is concrete... | body of "The debugValues source", the part of this sentence it holds, verbatim or as the words changed below say; body of "A debugValues source makes the report warn the user to review the file", the part of this sentence it holds, verbatim or as the words changed below say |
| A `debugValues` with any field left without a value (a bare type, an u... | body of "The debugValues source", verbatim |
| The report SHALL warn the user to review the file before deploying. | body of "A debugValues source makes the report warn the user to review the file", verbatim |
| Source: enhancement 0016 ("`debugValues` is the default template sourc... | body of "The debugValues source", the part of this sentence it holds, verbatim or as the words changed below say; body of "A debugValues source makes the report warn the user to review the file", the part of this sentence it holds, verbatim or as the words changed below say |
| Otherwise `values: {}`. | body of "With no usable source values.cue holds an empty struct and the report warns", "Otherwise" reads the quoted cases |
| The report SHALL warn that the values file is empty and point at `opm ... | body of "With no usable source values.cue holds an empty struct and the report warns", verbatim |
| Source: enhancement 0016 ("With no usable template source, init writes... | body of "With no usable source values.cue holds an empty struct and the report warns", verbatim; body of "A concrete non-struct source is rendered verbatim", verbatim; body of "An empty struct source is still named and carries the empty-file warning", verbatim; body of "values.cue opens with a comment naming its source", verbatim |
| A concrete non-struct source SHALL be rendered verbatim. | body of "A concrete non-struct source is rendered verbatim", verbatim |
| When the chosen source renders as an empty struct, the report SHALL st... | body of "An empty struct source is still named and carries the empty-file warning", verbatim |
| `values.cue` SHALL open with a comment naming the source it was filled... | body of "values.cue opens with a comment naming its source", verbatim |
| Source: enhancement 0016 ("With no usable template source, init writes... | body of "With no usable source values.cue holds an empty struct and the report warns", verbatim; body of "A concrete non-struct source is rendered verbatim", verbatim; body of "An empty struct source is still named and carries the empty-file warning", verbatim; body of "values.cue opens with a comment naming its source", verbatim |

Words changed or added:
- "values.cue is filled from the first applicable source and the report names it": "`values.cue` SHALL be filled from the first applicable source in this order, and the repor..." reads "`values.cue` SHALL be filled from the first applicable source in this order, and the repor..." (the sentence without its final colon and list number, with a period; the parenthesis "(requirements "The initValues source", "The debugValues source" and "With no usable source values.cue holds an empty struct and the report warns")" is added after "in this order": the exact headers of the same spec's requirements that hold the order)
- "The initValues source": "`values.cue` SHALL be filled from the first applicable source in this order, and the repor..." reads "`values.cue` SHALL be filled from the first applicable source in this order:" (the sentence cut before ", and the report" and closed with its colon)
- "The initValues source": "Source: enhancement 0016 ("`#Module` gains an optional field carrying the author-intended ..." reads "Source: enhancement 0016 ("The field is `initValues?: _`, open and optional, with no schem..." (whole title dropped: only the title that supports this rule stays)
- "No debugValues content reaches the package": "`values.cue` SHALL be filled from the first applicable source in this order, and the repor..." reads "`values.cue` SHALL be filled from the first applicable source in this order:" (the sentence cut before ", and the report" and closed with its colon)
- "No debugValues content reaches the package": "Source: enhancement 0016 ("`#Module` gains an optional field carrying the author-intended ..." reads "Source: enhancement 0016 ("`#Module` gains an optional field carrying the author-intended ..." (the trailing list number dropped)
- "The debugValues source": "Otherwise the module's `debugValues`, when the whole value is concrete once defaults are a..." reads "`values.cue` SHALL be filled from the first applicable source in this order: in every othe..." ("Otherwise" reads the quoted case, after the cut lead; the first letter of the rung is lowered)
- "The debugValues source": "Source: enhancement 0016 ("`debugValues` is the default template source for the generated ..." reads "Source: enhancement 0016 ("`debugValues` is the default template source for the generated ..." (the trailing list number dropped)
- "A debugValues source makes the report warn the user to review the file": "Otherwise the module's `debugValues`, when the whole value is concrete once defaults are a..." reads "`values.cue` SHALL be filled from the first applicable source in this order: in every othe..." ("Otherwise" reads the quoted case, after the cut lead; the first letter of the rung is lowered)
- "A debugValues source makes the report warn the user to review the file": "Source: enhancement 0016 ("`debugValues` is the default template source for the generated ..." reads "Source: enhancement 0016 ("`debugValues` is the default template source for the generated ..." (the trailing list number dropped)
- "With no usable source values.cue holds an empty struct and the report warns": "Otherwise `values: {}`." reads "`values.cue` SHALL be filled from the first applicable source in this order: in every othe..." ("Otherwise" reads the quoted cases, after the cut lead)


Lengths (old, new): old 1757; new "values.cue is filled from the first applicable source and the report names it" 266; "The initValues source" 431; "No debugValues content reaches the package" 423; "The debugValues source" 489; "A debugValues source makes the report warn the user to review the file" 411; "With no usable source values.cue holds an empty struct and the report warns" 459; "A concrete non-struct source is rendered verbatim" 153; "An empty struct source is still named and carries the empty-file warning" 282; "values.cue opens with a comment naming its source" 173.

### Ledger: instance-init, Init writes all or nothing, and only where a standalone package belongs

Form: c2 (REMOVED, ADDED under new names). New requirements: "An existing target directory is refused with exit 2 before any registry access", "A target inside an existing CUE module is refused with exit 2 before any registry access", "The package is written completely or not at all".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Existing directory refused | "An existing target directory is refused with exit 2 before any registry access", "Existing directory refused" | No (byte equal, checked) |
| Inside a CUE module refused | "A target inside an existing CUE module is refused with exit 2 before any registry access", "Inside a CUE module refused" | No (byte equal, checked) |
| Failure leaves nothing behind | "The package is written completely or not at all", "Failure leaves nothing behind" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The command SHALL refuse with exit 2, before any registry access, when... | body of "An existing target directory is refused with exit 2 before any registry access", the part of this sentence it holds, verbatim or as the words changed below say; body of "A target inside an existing CUE module is refused with exit 2 before any registry access", the part of this sentence it holds, verbatim or as the words changed below say |
| The package SHALL be written completely or not at all: on any failure ... | body of "The package is written completely or not at all", verbatim |
| Source: enhancement 0016 ("`opm instance init` mirrors `opm module ini... | body of "An existing target directory is refused with exit 2 before any registry access", verbatim; body of "A target inside an existing CUE module is refused with exit 2 before any registry access", verbatim; body of "The package is written completely or not at all", verbatim |

Words changed or added:
- "An existing target directory is refused with exit 2 before any registry access": "The command SHALL refuse with exit 2, before any registry access, when the target director..." reads "The command (`opm instance init`) SHALL refuse with exit 2, before any registry access, when the target director..." (the first condition, up to its parenthesis, with a period; the command named as the Source line of the same old requirement names it)
- "A target inside an existing CUE module is refused with exit 2 before any registry access": "The command SHALL refuse with exit 2, before any registry access, when the target director..." reads "The command (`opm instance init`) SHALL refuse with exit 2, before any registry access, when the target director..." (the lead repeated, then the second condition; "or" dropped; the command named as the Source line of the same old requirement names it)

Lengths (old, new): old 757; new "An existing target directory is refused with exit 2 before any registry access" 435; "A target inside an existing CUE module is refused with exit 2 before any registry access" 452; "The package is written completely or not at all" 475.

### Ledger: instance-init, Init reports what it wrote and does not validate it

Form: c2 (REMOVED, ADDED under new names). New requirements: "Init does not evaluate the generated package against the module config", "On success init reports the resolution, the values source, the files and a closing vet line", "The pinned version appears in the report", "Init exit codes are those of module init".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Success report | "On success init reports the resolution, the values source, the files and a closing vet line", "Success report" | No (byte equal, checked) |
| Registry unreachable | "Init exit codes are those of module init", "Registry unreachable" | No (byte equal, checked) |
| Registry unreachable while resolving dependencies | "Init exit codes are those of module init", "Registry unreachable while resolving dependencies" | No (byte equal, checked) |
| Values that violate the contract still initialize | "Init does not evaluate the generated package against the module config", "Values that violate the contract still initialize" | No (byte equal, checked) |
| (new) | "The pinned version appears in the report", "The pinned version is reported" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| Init SHALL NOT evaluate the generated package against the module's `#c... | body of "Init does not evaluate the generated package against the module config", verbatim |
| On success it SHALL report: the resolution (module path, selected majo... | body of "On success init reports the resolution, the values source, the files and a closing vet line", the part of this sentence it holds, verbatim or as the words changed below say |
| The pinned version SHALL appear in the report; pins never float. | body of "The pinned version appears in the report", verbatim |
| Exit codes SHALL be those of `opm module init`: 0 written, 2 refused, ... | body of "Init exit codes are those of module init", verbatim |
| Source: enhancement 0016 ("`opm instance init` mirrors `opm module ini... | body of "Init does not evaluate the generated package against the module config", verbatim; body of "On success init reports the resolution, the values source, the files and a closing vet line", the part of this sentence it holds, verbatim or as the words changed below say; body of "The pinned version appears in the report", verbatim; body of "Init exit codes are those of module init", the part of this sentence it holds, verbatim or as the words changed below say |

Words changed or added:
- "On success init reports the resolution, the values source, the files and a closing vet line": "On success it SHALL report: the resolution (module path, selected major and version, and h..." reads "On success `opm instance init` SHALL report: the resolution (module path, selected major a..." ("it" reads "`opm instance init`")
- "On success init reports the resolution, the values source, the files and a closing vet line": "Source: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-fr..." reads "Source: enhancement 0016 ("Init does not validate the generated package; it tells the user..." (whole titles dropped: only the titles that support this rule stay)
- "Init exit codes are those of module init": "Source: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-fr..." reads "Source: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-fr..." (whole titles dropped: only the titles that support this rule stay)

Lengths (old, new): old 1008; new "Init does not evaluate the generated package against the module config" 468; "On success init reports the resolution, the values source, the files and a closing vet line" 393; "The pinned version appears in the report" 455; "Init exit codes are those of module init" 380.

### Ledger: instance-init, Only a registry that gave no response counts as unreachable

Form: c2 (REMOVED, ADDED under new names). New requirements: "A registry failure with no HTTP response counts as unreachable", "A registry answer that a dependency is not held is not unreachable", "Reading the module's own module file stays exit 3 for every failure", "The CLI takes the unreachable decision from the library's typed fetch classification".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A dependency the registry does not hold | "A registry answer that a dependency is not held is not unreachable", "A dependency the registry does not hold" | No (byte equal, checked) |
| (new) | "A registry failure with no HTTP response counts as unreachable", "No response is unreachable" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Reading the module's own module file stays exit 3 for every failure", "Reading the module file" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The CLI takes the unreachable decision from the library's typed fetch classification", "No message matching" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| While `opm instance init` acquires the module archive and while it res... | body of "A registry failure with no HTTP response counts as unreachable", verbatim |
| A registry answer that a dependency of the staged package is not held ... | body of "A registry answer that a dependency is not held is not unreachable", verbatim |
| Reading the module's own module file, between resolution and acquire, ... | body of "Reading the module's own module file stays exit 3 for every failure", the part of this sentence it holds, verbatim or as the words changed below say |
| Source: enhancement 0016 (a published module floats to the highest cor... | body of "A registry failure with no HTTP response counts as unreachable", verbatim; body of "A registry answer that a dependency is not held is not unreachable", verbatim; body of "Reading the module's own module file stays exit 3 for every failure", verbatim |
| The CLI SHALL take the unreachable decision from the library's typed f... | body of "The CLI takes the unreachable decision from the library's typed fetch classification", verbatim |
| Source: enhancement 0021 ("A beta line reaches GA only when its exit c... | body of "The CLI takes the unreachable decision from the library's typed fetch classification", verbatim |

Words changed or added:
- "Reading the module's own module file stays exit 3 for every failure": "Reading the module's own module file, between resolution and acquire, stays exit 3 for eve..." reads "Reading the module's own module file, between resolution and acquire, SHALL stay exit 3 fo..." ("stays" reads "SHALL stay")

Lengths (old, new): old 831; new "A registry failure with no HTTP response counts as unreachable" 356; "A registry answer that a dependency is not held is not unreachable" 225; "Reading the module's own module file stays exit 3 for every failure" 197; "The CLI takes the unreachable decision from the library's typed fetch classification" 237.

### Ledger: authoring-commands, version set writes in place, idempotently, offline

Form: c2 (REMOVED, ADDED under new names). New requirements: "version set rewrites the identity Version in place", "Setting the already-declared version writes nothing", "version set performs no registry I/O and refuses an identity file of the wrong shape", "The writer tolerates every authored form of the Version field and preserves the form it finds", "Every identity file the CLI emits uses the plain-literal form".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Idempotent no-op | "Setting the already-declared version writes nothing", "Idempotent no-op" | No (byte equal, checked) |
| Plain literal rewritten in place | "version set rewrites the identity Version in place", "Plain literal rewritten in place" | No (byte equal, checked) |
| Assertion survives the write | "version set rewrites the identity Version in place", "Assertion survives the write" | No (byte equal, checked) |
| Defaulted declaration stays defaulted | "The writer tolerates every authored form of the Version field and preserves the form it finds", "Defaulted declaration stays defaulted" | No (byte equal, checked) |
| Structural refusal | "version set performs no registry I/O and refuses an identity file of the wrong shape", "Structural refusal" | No (byte equal, checked) |
| Catalog refusal points at a command that exists | "version set performs no registry I/O and refuses an identity file of the wrong shape", "Catalog refusal points at a command that exists" | No (byte equal, checked) |
| (new) | "version set performs no registry I/O and refuses an identity file of the wrong shape", "No registry access" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Every identity file the CLI emits uses the plain-literal form", "Emitted identity files" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| `opm module version set <semver> [path]` and `opm catalog version set ... | body of "version set rewrites the identity Version in place", verbatim; body of "version set performs no registry I/O and refuses an identity file of the wrong shape", the part of this sentence it holds, verbatim or as the words changed below say |
| Setting the already-declared version SHALL write nothing — no file mod... | body of "Setting the already-declared version writes nothing", verbatim |
| The command SHALL perform no registry I/O; an identity file that does ... | body of "version set performs no registry I/O and refuses an identity file of the wrong shape", the part of this sentence it holds, verbatim or as the words changed below say |
| The writer SHALL tolerate every authored form (plain literal, `#Versio... | body of "The writer tolerates every authored form of the Version field and preserves the form it finds", verbatim |
| Every identity file the CLI itself emits (scaffold, repair, official t... | body of "Every identity file the CLI emits uses the plain-literal form", verbatim |

Words changed or added:
- "version set performs no registry I/O and refuses an identity file of the wrong shape": "The command SHALL perform no registry I/O; an identity file that does not structurally mat..." reads "`opm module version set <semver> [path]` and `opm catalog version set <semver> [path]` SHA..." ("The command" reads "`opm module version set <semver> [path]` and `opm catalog version set <semver> [path]`", copied from the first sentence)

Lengths (old, new): old 1161; new "version set rewrites the identity Version in place" 313; "Setting the already-declared version writes nothing" 131; "version set performs no registry I/O and refuses an identity file of the wrong shape" 442; "The writer tolerates every authored form of the Version field and preserves the form it finds" 158; "Every identity file the CLI emits uses the plain-literal form" 188.

## Ledger entries of cli-B4, unit 4a (instance-inventory, mod-status)

One subsection per rewritten requirement, 12 in all: `instance-inventory` 6 and `mod-status` 6. The other three specs of cli-B4 (`release-cascade-wiring`, `cmdutil`, `platform-check`) follow in the second half. Every new body is built from whole sentences of the old body, copied unchanged, or from parts of one sentence cut at a list boundary with the subject repeated. A clause row names where a clause of the old body appears; the list "Words changed or added" under a requirement names each substitution, as the rewrite rules allow: a pronoun ("It", "it") read as the subject copied from the same old requirement, a closing period where a list marker or a semicolon stood, the exact header of a sibling requirement of the same spec where the old content cannot fit (the three workload kind blocks of "Status evaluates health per resource category"), an em-dash read as a colon or a comma, and an RFC 2119 keyword where the old sentence had none (also listed in the strength table). All 12 are form c2, because the old scenarios of each requirement belong to different new behaviors. Old scenario names are all kept; the byte check ran once per rewritten requirement and printed no `DIFF` and no `LOST`. One cite changes: the body "The inventory digest in the status is the library inventory digest" names the requirement "The CLI computes the inventory digest with the library and no digest of its own" where the old body named "Deterministic inventory digest"; `grep -rn --exclude-dir=archive` over the repo found no other reference to an old header of this unit.

### Ledger: instance-inventory, CLI writes a strict status subset via the status subresource

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI writes only its own fields through the status subresource", "The last applied render digest is the library shared render digest", "The inventory digest in the status is the library inventory digest", "The last applied source digest is the operator module source digest", "The last applied config digest follows the operator canonical-JSON semantics".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Status subset after successful apply | "The CLI writes only its own fields through the status subresource", "Status subset after successful apply" | No (byte equal, checked) |
| Revision increments across applies | "The CLI writes only its own fields through the status subresource", "Revision increments across applies" | No (byte equal, checked) |
| Source digest matches the operator's computation | "The last applied source digest is the operator module source digest", "Source digest matches the operator's computation" | No (byte equal, checked) |
| Render digest matches the operator's algorithm | "The last applied render digest is the library shared render digest", "Render digest matches the operator's algorithm" | No (byte equal, checked) |
| Stored digests change once on upgrade | "The inventory digest in the status is the library inventory digest", "Stored digests change once on upgrade" | No (byte equal, checked) |
| (new) | "The last applied config digest follows the operator canonical-JSON semantics", "Config digest of an empty config" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| After resources are applied and pruned, the CLI SHALL write, via the s... | body of "The CLI writes only its own fields through the status subresource", verbatim |
| The CLI MUST NOT write `status.conditions`, `status.observedGeneration... | body of "The CLI writes only its own fields through the status subresource", verbatim |
| The digest fields SHALL be operator-parity: `lastAppliedRenderDigest` ... | body of "The last applied render digest is the library shared render digest", verbatim, with the closing semicolon read as a period |
| `status.inventory.digest` SHALL be the library's `inventory.Digest` of... | body of "The inventory digest in the status is the library inventory digest", with the opening clause of the sentence copied in and the cite of the split requirement read as its new header |
| `lastAppliedSourceDigest` SHALL be computed as the operator's `ModuleS... | body of "The last applied source digest is the operator module source digest", with the opening clause of the sentence copied in and the em-dash read as a colon |
| `lastAppliedConfigDigest` SHALL match the operator's `ConfigDigest` ca... | body of "The last applied config digest follows the operator canonical-JSON semantics", with the opening clause of the sentence copied in |
| The digest fields SHALL be operator-parity: | opening clause, copied into the bodies of the render digest, inventory digest, source digest and config digest blocks |

Words changed or added:
- "The last applied render digest is the library shared render digest": "The digest fields SHALL be operator-parity: `lastAppliedRenderDigest` SHALL be the library..." reads "The digest fields SHALL be operator-parity: `lastAppliedRenderDigest` SHALL be the library..." (the closing semicolon reads a period)
- "The inventory digest in the status is the library inventory digest": "`status.inventory.digest` SHALL be the library's `inventory.Digest` of the written entries..." reads "The digest fields SHALL be operator-parity: `status.inventory.digest` SHALL be the library..." ("The digest fields SHALL be operator-parity:" copied from the first sentence of the same paragraph; the cite of the split requirement reads its new header "The CLI computes the inventory digest with the library and no digest of its own"; the closing semicolon reads a period)
- "The last applied source digest is the operator module source digest": "`lastAppliedSourceDigest` SHALL be computed as the operator's `ModuleSourceDigest` (SHA-25..." reads "The digest fields SHALL be operator-parity: `lastAppliedSourceDigest` SHALL be computed as..." ("The digest fields SHALL be operator-parity:" copied from the first sentence of the same paragraph; the em-dash reads a colon; the closing comma and "and" read a period)
- "The last applied config digest follows the operator canonical-JSON semantics": "`lastAppliedConfigDigest` SHALL match the operator's `ConfigDigest` canonical-JSON semanti..." reads "The digest fields SHALL be operator-parity: `lastAppliedConfigDigest` SHALL match the oper..." ("The digest fields SHALL be operator-parity:" copied from the first sentence of the same paragraph)

Lengths (old, new): old 1384; new "The CLI writes only its own fields through the status subresource" 492; "The last applied render digest is the library shared render digest" 434; "The inventory digest in the status is the library inventory digest" 223; "The last applied source digest is the operator module source digest" 223; "The last applied config digest follows the operator canonical-JSON semantics" 184.

### Ledger: instance-inventory, Spec write contents

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI sets the owner of the ModuleInstance spec to cli", "The CLI writes the module identity verbatim from core-v2 metadata", "The spec values are the single unified values blob", "The module identity pair applies to local resolution and the CR holds no filesystem path", "The declared module identity pair is verified where it meets a registry".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Local-path apply writes the declared reference | "The module identity pair applies to local resolution and the CR holds no filesystem path", "Local-path apply writes the declared reference" | No (byte equal, checked) |
| No address arithmetic | "The CLI writes the module identity verbatim from core-v2 metadata", "No address arithmetic" | No (byte equal, checked) |
| Values are the unified blob | "The spec values are the single unified values blob", "Values are the unified blob" | No (byte equal, checked) |
| (new) | "The CLI sets the owner of the ModuleInstance spec to cli", "CLI-executor apply sets the owner" | New restatement scenario: only words of the old clause, no new case |
| (new) | "The declared module identity pair is verified where it meets a registry", "Declared pair verified at the registry" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| On apply in CLI-executor mode, the CLI SHALL server-side-apply the CR ... | opening clause, copied into the bodies of "The CLI sets the owner of the ModuleInstance spec to cli", "The CLI writes the module identity verbatim from core-v2 metadata" and "The spec values are the single unified values blob" |
| `spec.owner: cli` | body of "The CLI sets the owner of the ModuleInstance spec to cli", with the opening clause and a closing period |
| `spec.module.path` and `spec.module.version` set to the module's decla... | body of "The CLI writes the module identity verbatim from core-v2 metadata", with the opening clause, the em-dash read as a colon and a closing period |
| `spec.values` set to the single unified values blob that the render co... | body of "The spec values are the single unified values blob", with the opening clause; the "and" that joined the item to the list is dropped |
| The pair applies for local-directory and locally-replaced module resol... | body of "The module identity pair applies to local resolution and the CR holds no filesystem path", with the pair named in the subject |
| The declared pair is verified against fetched coordinates wherever it ... | body of "The declared module identity pair is verified where it meets a registry", with the pair named in the subject and "is verified" read as "SHALL be verified" |

Words changed or added:
- "The CLI sets the owner of the ModuleInstance spec to cli": "`spec.owner: cli`" reads "On apply in CLI-executor mode, the CLI SHALL server-side-apply the CR spec with field mana..." (the opening clause "On apply in CLI-executor mode, ... containing:" copied from the same sentence; a closing period)
- "The CLI writes the module identity verbatim from core-v2 metadata": "`spec.module.path` and `spec.module.version` set to the module's declared identity **read ..." reads "On apply in CLI-executor mode, the CLI SHALL server-side-apply the CR spec with field mana..." (the opening clause "On apply in CLI-executor mode, ... containing:" copied from the same sentence; the em-dash reads a colon; a closing period)
- "The spec values are the single unified values blob": "`spec.values` set to the single unified values blob that the render consumed." reads "On apply in CLI-executor mode, the CLI SHALL server-side-apply the CR spec with field mana..." (the opening clause "On apply in CLI-executor mode, ... containing:" copied from the same sentence)
- "The module identity pair applies to local resolution and the CR holds no filesystem path": "The pair applies for local-directory and locally-replaced module resolution as well; the C..." reads "The pair (`spec.module.path` and `spec.module.version`) applies for local-directory and lo..." ("The pair" reads "The pair (`spec.module.path` and `spec.module.version`)", copied from the first sentence)
- "The declared module identity pair is verified where it meets a registry": "The declared pair is verified against fetched coordinates wherever it later meets a regist..." reads "The declared pair (`spec.module.path` and `spec.module.version`) SHALL be verified against..." ("The declared pair" reads "The declared pair (`spec.module.path` and `spec.module.version`)", copied from the first sentence; "is verified" reads "SHALL be verified")

Lengths (old, new): old 878; new "The CLI sets the owner of the ModuleInstance spec to cli" 135; "The CLI writes the module identity verbatim from core-v2 metadata" 484; "The spec values are the single unified values blob" 194; "The module identity pair applies to local resolution and the CR holds no filesystem path" 174; "The declared module identity pair is verified where it meets a registry" 215.

### Ledger: instance-inventory, Render provenance annotation

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI stamps the source local annotation when render bytes did not come from pure registry resolution", "The source local annotation is a fail-closed provenance signal that no gate reads as an authority".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Local render stamps the annotation | "The CLI stamps the source local annotation when render bytes did not come from pure registry resolution", "Local render stamps the annotation" | No (byte equal, checked) |
| Replacement in effect stamps the annotation | "The CLI stamps the source local annotation when render bytes did not come from pure registry resolution", "Replacement in effect stamps the annotation" | No (byte equal, checked) |
| Registry apply clears the annotation | "The CLI stamps the source local annotation when render bytes did not come from pure registry resolution", "Registry apply clears the annotation" | No (byte equal, checked) |
| (new) | "The source local annotation is a fail-closed provenance signal that no gate reads as an authority", "No gate reads the annotation as an authority" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| When the applied render's module bytes did not come from pure registry... | body of "The CLI stamps the source local annotation when render bytes did not come from pure registry resolution", the part of this sentence it holds, verbatim or as the words changed below say |
| When a later apply resolves fully from registries, the CLI SHALL omit ... | body of "The CLI stamps the source local annotation when render bytes did not come from pure registry resolution", verbatim |
| The annotation is a fail-closed provenance signal: the thin-editor app... | body of "The source local annotation is a fail-closed provenance signal that no gate reads as an authority", the part of this sentence it holds, verbatim or as the words changed below say |

Words changed or added:
- "The CLI stamps the source local annotation when render bytes did not come from pure registry resolution": "When the applied render's module bytes did not come from pure registry resolution [em-dash] the ma..." reads "When the applied render's module bytes did not come from pure registry resolution: the mai..." (the first em-dash reads a colon, the second a comma)
- "The source local annotation is a fail-closed provenance signal that no gate reads as an authority": "The annotation is a fail-closed provenance signal: the thin-editor apply path refuses a mo..." reads "The annotation `module-instance.opmodel.dev/source: local` is a fail-closed provenance sig..." ("The annotation" reads "The annotation `module-instance.opmodel.dev/source: local`", copied from the first sentence)

Lengths (old, new): old 695; new "The CLI stamps the source local annotation when render bytes did not come from pure registry resolution" 425; "The source local annotation is a fail-closed provenance signal that no gate reads as an authority" 311.

### Ledger: instance-inventory, Deterministic inventory digest

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI computes the inventory digest with the library and no digest of its own", "The inventory digest depends only on the field values of the entries", "The release that first records the library digest carries a migration note".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Same entries in different order produce same digest | "The inventory digest depends only on the field values of the entries", "Same entries in different order produce same digest" | No (byte equal, checked) |
| Added or removed resource changes digest | "The inventory digest depends only on the field values of the entries", "Added or removed resource changes digest" | No (byte equal, checked) |
| Component rename changes digest | "The inventory digest depends only on the field values of the entries", "Component rename changes digest" | No (byte equal, checked) |
| The stored digest is the library's | "The CLI computes the inventory digest with the library and no digest of its own", "The stored digest is the library's" | No (byte equal, checked) |
| (new) | "The release that first records the library digest carries a migration note", "Migration note for the first release" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The CLI SHALL compute the inventory digest with the library's `opm/k8s... | body of "The CLI computes the inventory digest with the library and no digest of its own", verbatim |
| The digest SHALL depend only on the entries' field values, never on th... | body of "The inventory digest depends only on the field values of the entries", verbatim |
| The CLI SHALL NOT compute an inventory digest of its own. | body of "The CLI computes the inventory digest with the library and no digest of its own", verbatim |
| Source: enhancement 0012 ("The stale set is component-blind, and the i... | body of "The CLI computes the inventory digest with the library and no digest of its own", verbatim; body of "The inventory digest depends only on the field values of the entries", verbatim; body of "The release that first records the library digest carries a migration note", verbatim |
| The release that first records this digest SHALL carry a migration not... | body of "The release that first records the library digest carries a migration note", the part of this sentence it holds, verbatim or as the words changed below say |
| Source: enhancement 0012 ("The stale set is component-blind, and the i... | body of "The CLI computes the inventory digest with the library and no digest of its own", verbatim; body of "The inventory digest depends only on the field values of the entries", verbatim; body of "The release that first records the library digest carries a migration note", verbatim |

Words changed or added:
- "The release that first records the library digest carries a migration note": "The release that first records this digest SHALL carry a migration note naming the one-tim..." reads "The release that first records this inventory digest computed with the library's `opm/k8s/..." ("this digest" reads "this inventory digest computed with the library's `opm/k8s/inventory.Digest`" (words copied from the first sentence))

Lengths (old, new): old 814; new "The CLI computes the inventory digest with the library and no digest of its own" 335; "The inventory digest depends only on the field values of the entries" 358; "The release that first records the library digest carries a migration note" 307.

### Ledger: instance-inventory, Skipped contracts annotation

Form: c2 (REMOVED, ADDED under new names). New requirements: "A skipping apply records the skipped demands in the skipped-contracts annotation", "A later apply that skips nothing omits the skipped-contracts annotation", "The skipped-contracts annotation is information and no gate reads it as an authority".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A skipping apply records its skips | "A skipping apply records the skipped demands in the skipped-contracts annotation", "A skipping apply records its skips" | No (byte equal, checked) |
| A complete apply clears the record | "A later apply that skips nothing omits the skipped-contracts annotation", "A complete apply clears the record" | No (byte equal, checked) |
| A flag that skipped nothing writes nothing | "A skipping apply records the skipped demands in the skipped-contracts annotation", "A flag that skipped nothing writes nothing" | No (byte equal, checked) |
| (new) | "The skipped-contracts annotation is information and no gate reads it as an authority", "No gate reads the skipped-contracts annotation as an authority" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| When an apply rendered with `--skip-unprovided` and the kernel skipped... | body of "A skipping apply records the skipped demands in the skipped-contracts annotation", verbatim |
| Its value SHALL be the skipped demands as `<component>=<contract fqn>`... | body of "A skipping apply records the skipped demands in the skipped-contracts annotation", verbatim |
| When a later apply skips nothing, with or without the flag, the CLI SH... | body of "A later apply that skips nothing omits the skipped-contracts annotation", the part of this sentence it holds, verbatim or as the words changed below say |
| The annotation is information for whoever inspects the instance: no CL... | body of "The skipped-contracts annotation is information and no gate reads it as an authority", the part of this sentence it holds, verbatim or as the words changed below say |

Words changed or added:
- "A later apply that skips nothing omits the skipped-contracts annotation": "When a later apply skips nothing, with or without the flag, the CLI SHALL omit the annotat..." reads "When a later apply skips nothing, with or without the flag `--skip-unprovided`, the CLI SH..." ("the flag" reads "the flag `--skip-unprovided`" and "the annotation" reads "the annotation `module-instance.opmodel.dev/skipped-contracts`", both copied from the first sentence)
- "The skipped-contracts annotation is information and no gate reads it as an authority": "The annotation is information for whoever inspects the instance: no CLI gate SHALL read it..." reads "The annotation `module-instance.opmodel.dev/skipped-contracts` is information for whoever ..." ("The annotation" reads "The annotation `module-instance.opmodel.dev/skipped-contracts`", copied from the first sentence)

Lengths (old, new): old 553; new "A skipping apply records the skipped demands in the skipped-contracts annotation" 318; "A later apply that skips nothing omits the skipped-contracts annotation" 194; "The skipped-contracts annotation is information and no gate reads it as an authority" 155.

### Ledger: instance-inventory, Inventory entries are the library's component-blind entries

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI represents owned resources as the library's inventory entries", "Two entries are the same object when group, kind, namespace and name agree".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Same object at a different API version | "Two entries are the same object when group, kind, namespace and name agree", "Same object at a different API version" | No (byte equal, checked) |
| Same object under a different component | "Two entries are the same object when group, kind, namespace and name agree", "Same object under a different component" | No (byte equal, checked) |
| Different objects | "Two entries are the same object when group, kind, namespace and name agree", "Different objects" | No (byte equal, checked) |
| (new) | "The CLI represents owned resources as the library's inventory entries", "The CLI declares no entry type of its own" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The CLI SHALL represent each currently owned Kubernetes resource as th... | body of "The CLI represents owned resources as the library's inventory entries", verbatim |
| Two entries SHALL be the same object when their group, kind, namespace... | body of "Two entries are the same object when group, kind, namespace and name agree", verbatim |
| Source: enhancement 0012 ("The stale set is component-blind, and the i... | body of "The CLI represents owned resources as the library's inventory entries", verbatim; body of "Two entries are the same object when group, kind, namespace and name agree", verbatim |

Lengths (old, new): old 616; new "The CLI represents owned resources as the library's inventory entries" 334; "Two entries are the same object when group, kind, namespace and name agree" 404.

### Ledger: mod-status, Status discovers resources via ownership inventory

Form: c2 (REMOVED, ADDED under new names). New requirements: "Status reads the persisted instance inventory record to discover the tracked resources", "A UUID argument resolves by listing the CRs and a name argument by a direct GET", "Status performs one targeted GET per tracked entry once the record is found", "Status requires no module source or re-rendering and uses no cluster-wide label-scan".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Discover by instance UUID | "A UUID argument resolves by listing the CRs and a name argument by a direct GET", "Discover by instance UUID" | No (byte equal, checked) |
| Discover by instance name | "A UUID argument resolves by listing the CRs and a name argument by a direct GET", "Discover by instance name" | No (byte equal, checked) |
| (new) | "Status reads the persisted instance inventory record to discover the tracked resources", "Status reads the inventory record" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Status performs one targeted GET per tracked entry once the record is found", "One targeted GET per tracked entry" | New restatement scenario: only words of the old clause, no new case |
| (new) | "Status requires no module source or re-rendering and uses no cluster-wide label-scan", "No module source and no label-scan" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The `opm instance status` command SHALL read the persisted instance in... | body of "Status reads the persisted instance inventory record to discover the tracked resources", verbatim |
| A UUID argument SHALL resolve via `inventory.FindRecordByInstanceUUID`... | body of "A UUID argument resolves by listing the CRs and a name argument by a direct GET", with the argument named in the subject |
| Once the record is found, it SHALL perform one targeted GET per tracke... | body of "Status performs one targeted GET per tracked entry once the record is found", with "it" read as the command |
| It MUST NOT require module source or re-rendering. | body of "Status requires no module source or re-rendering and uses no cluster-wide label-scan", with "It" read as the command |
| It MUST NOT use a cluster-wide label-scan to discover workload resourc... | body of "Status requires no module source or re-rendering and uses no cluster-wide label-scan", verbatim |
| <!-- Was: release inventory record, --release-id/--instance-name (enha... | body of "Status reads the persisted instance inventory record to discover the tracked resources", verbatim |

Words changed or added:
- "A UUID argument resolves by listing the CRs and a name argument by a direct GET": "A UUID argument SHALL resolve via `inventory.FindRecordByInstanceUUID` (list the CRs in th..." reads "A UUID argument of the `opm instance status` command SHALL resolve via `inventory.FindReco..." ("A UUID argument" reads "A UUID argument of the `opm instance status` command", copied from the first sentence)
- "Status performs one targeted GET per tracked entry once the record is found": "Once the record is found, it SHALL perform one targeted GET per tracked entry via `invento..." reads "Once the persisted instance inventory record is found, the `opm instance status` command S..." ("the record" reads "the persisted instance inventory record" and "it" reads "the `opm instance status` command", copied from the first sentence)
- "Status requires no module source or re-rendering and uses no cluster-wide label-scan": "It MUST NOT require module source or re-rendering." reads "The `opm instance status` command MUST NOT require module source or re-rendering." ("It" reads "The `opm instance status` command", copied from the first sentence)

Lengths (old, new): old 834; new "Status reads the persisted instance inventory record to discover the tracked resources" 372; "A UUID argument resolves by listing the CRs and a name argument by a direct GET" 247; "Status performs one targeted GET per tracked entry once the record is found" 185; "Status requires no module source or re-rendering and uses no cluster-wide label-scan" 155.

### Ledger: mod-status, Status evaluates health per resource category

Form: c2 (REMOVED, ADDED under new names). New requirements: "A workload is healthy only when its rollout has finished", "A Deployment is healthy when its replica counters match and no old replicas remain", "A StatefulSet is healthy when its replica counters and revisions match", "A DaemonSet is healthy when its scheduled and available counters match", "A Job is healthy when its Complete condition is True", "A CronJob is reported as Applied", "Passive kinds are reported as Applied and custom resources are healthy on a True Ready condition".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Deployment not yet ready | "A Deployment is healthy when its replica counters match and no old replicas remain", "Deployment not yet ready" | No (byte equal, checked) |
| Deployment upgrade stuck behind an available old ReplicaSet | "A Deployment is healthy when its replica counters match and no old replicas remain", "Deployment upgrade stuck behind an available old ReplicaSet" | No (byte equal, checked) |
| Job completed | "A Job is healthy when its Complete condition is True", "Job completed" | No (byte equal, checked) |
| ConfigMap always healthy | "Passive kinds are reported as Applied and custom resources are healthy on a True Ready condition", "ConfigMap always healthy" | No (byte equal, checked) |
| Custom resource with Ready condition | "Passive kinds are reported as Applied and custom resources are healthy on a True Ready condition", "Custom resource with Ready condition" | No (byte equal, checked) |
| Custom resource without Ready condition | "Passive kinds are reported as Applied and custom resources are healthy on a True Ready condition", "Custom resource without Ready condition" | No (byte equal, checked) |
| (new) | "A workload is healthy only when its rollout has finished", "Workload generation not observed" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A StatefulSet is healthy when its replica counters and revisions match", "StatefulSet with OnDelete" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A DaemonSet is healthy when its scheduled and available counters match", "DaemonSet counters" | New restatement scenario: only words of the old clause, no new case |
| (new) | "A CronJob is reported as Applied", "CronJob reported as Applied" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The command SHALL evaluate resource health using category-specific cri... | opening clause, in the bodies of all seven blocks of this requirement, with "The command" read as "The `opm instance status` command" |
| **Workloads** (Deployment, StatefulSet, DaemonSet): healthy only when ... | body of "A workload is healthy only when its rollout has finished", verbatim; the Deployment, StatefulSet and DaemonSet blocks name that requirement by its exact header instead |
| The controller SHALL have observed the current generation (`status.obs... | body of "A workload is healthy only when its rollout has finished", with the closing colon read as a period |
| Deployment: `updatedReplicas` and `availableReplicas` equal `spec.repl... | body of "A Deployment is healthy when its replica counters match and no old replicas remain", with a closing period |
| StatefulSet: `readyReplicas` and `updatedReplicas` equal `spec.replica... | body of "A StatefulSet is healthy when its replica counters and revisions match", with a closing period |
| DaemonSet: `updatedNumberScheduled` and `numberAvailable` equal `desir... | body of "A DaemonSet is healthy when its scheduled and available counters match", with a closing period |
| **Jobs** (Job): healthy when `Complete` condition is True | body of "A Job is healthy when its Complete condition is True", with a closing period |
| **CronJobs**: reported as `Applied` (scheduled, no readiness phase) | body of "A CronJob is reported as Applied", with a closing period |
| **Passive** (ConfigMap, Secret, Service, ServiceAccount, RBAC kinds, a... | body of "Passive kinds are reported as Applied and custom resources are healthy on a True Ready condition", with a closing period |
| **Custom** (CRD instances): healthy if `Ready` condition is present an... | body of "Passive kinds are reported as Applied and custom resources are healthy on a True Ready condition", with a closing period |

Words changed or added:
- "A workload is healthy only when its rollout has finished": "The command SHALL evaluate resource health using category-specific criteria: **Workloads**..." reads "The `opm instance status` command SHALL evaluate resource health using category-specific c..." ("The command" reads "The `opm instance status` command" (the command named in the Purpose of the spec; the old requirement names it nowhere else))
- "A workload is healthy only when its rollout has finished": "The controller SHALL have observed the current generation (`status.observedGeneration >= m..." reads "The controller SHALL have observed the current generation (`status.observedGeneration >= m..." (the closing colon reads a period)
- "A Deployment is healthy when its replica counters match and no old replicas remain": "The command SHALL evaluate resource health using category-specific criteria:" reads "The `opm instance status` command SHALL evaluate resource health using category-specific c..." ("The command" reads "The `opm instance status` command" (the command named in the Purpose of the spec; the old requirement names it nowhere else))
- "A Deployment is healthy when its replica counters match and no old replicas remain": "Deployment: `updatedReplicas` and `availableReplicas` equal `spec.replicas`, no old replic..." reads "Deployment: `updatedReplicas` and `availableReplicas` equal `spec.replicas`, no old replic..." (a closing period where a list marker stood; then a sentence naming the exact header of the same spec's requirement that holds the other Workloads criteria (extension for content that cannot fit))
- "A StatefulSet is healthy when its replica counters and revisions match": "The command SHALL evaluate resource health using category-specific criteria:" reads "The `opm instance status` command SHALL evaluate resource health using category-specific c..." ("The command" reads "The `opm instance status` command" (the command named in the Purpose of the spec; the old requirement names it nowhere else))
- "A StatefulSet is healthy when its replica counters and revisions match": "StatefulSet: `readyReplicas` and `updatedReplicas` equal `spec.replicas` and `updateRevisi..." reads "StatefulSet: `readyReplicas` and `updatedReplicas` equal `spec.replicas` and `updateRevisi..." (a closing period where a list marker stood; then a sentence naming the exact header of the same spec's requirement that holds the other Workloads criteria (extension for content that cannot fit))
- "A DaemonSet is healthy when its scheduled and available counters match": "The command SHALL evaluate resource health using category-specific criteria:" reads "The `opm instance status` command SHALL evaluate resource health using category-specific c..." ("The command" reads "The `opm instance status` command" (the command named in the Purpose of the spec; the old requirement names it nowhere else))
- "A DaemonSet is healthy when its scheduled and available counters match": "DaemonSet: `updatedNumberScheduled` and `numberAvailable` equal `desiredNumberScheduled`" reads "DaemonSet: `updatedNumberScheduled` and `numberAvailable` equal `desiredNumberScheduled`. ..." (a closing period where a list marker stood; then a sentence naming the exact header of the same spec's requirement that holds the other Workloads criteria (extension for content that cannot fit))
- "A Job is healthy when its Complete condition is True": "The command SHALL evaluate resource health using category-specific criteria:" reads "The `opm instance status` command SHALL evaluate resource health using category-specific c..." ("The command" reads "The `opm instance status` command" (the command named in the Purpose of the spec; the old requirement names it nowhere else))
- "A Job is healthy when its Complete condition is True": "**Jobs** (Job): healthy when `Complete` condition is True" reads "**Jobs** (Job): healthy when `Complete` condition is True." (a closing period where a list marker stood)
- "A CronJob is reported as Applied": "The command SHALL evaluate resource health using category-specific criteria:" reads "The `opm instance status` command SHALL evaluate resource health using category-specific c..." ("The command" reads "The `opm instance status` command" (the command named in the Purpose of the spec; the old requirement names it nowhere else))
- "A CronJob is reported as Applied": "**CronJobs**: reported as `Applied` (scheduled, no readiness phase)" reads "**CronJobs**: reported as `Applied` (scheduled, no readiness phase)." (a closing period where a list marker stood)
- "Passive kinds are reported as Applied and custom resources are healthy on a True Ready condition": "The command SHALL evaluate resource health using category-specific criteria:" reads "The `opm instance status` command SHALL evaluate resource health using category-specific c..." ("The command" reads "The `opm instance status` command" (the command named in the Purpose of the spec; the old requirement names it nowhere else))
- "Passive kinds are reported as Applied and custom resources are healthy on a True Ready condition": "**Passive** (ConfigMap, Secret, Service, ServiceAccount, RBAC kinds, and similar kinds wit..." reads "**Passive** (ConfigMap, Secret, Service, ServiceAccount, RBAC kinds, and similar kinds wit..." (a closing period where a list marker stood)
- "Passive kinds are reported as Applied and custom resources are healthy on a True Ready condition": "**Custom** (CRD instances): healthy if `Ready` condition is present and True, otherwise tr..." reads "**Custom** (CRD instances): healthy if `Ready` condition is present and True, otherwise tr..." (a closing period where a list marker stood)

Lengths (old, new): old 1324; new "A workload is healthy only when its rollout has finished" 399; "A Deployment is healthy when its replica counters match and no old replicas remain" 355; "A StatefulSet is healthy when its replica counters and revisions match" 409; "A DaemonSet is healthy when its scheduled and available counters match" 268; "A Job is healthy when its Complete condition is True" 157; "A CronJob is reported as Applied" 167; "Passive kinds are reported as Applied and custom resources are healthy on a True Ready condition" 410.

### Ledger: mod-status, Status supports table output format

Form: c2 (REMOVED, ADDED under new names). New requirements: "The default status output is a table with plain space-padded columns", "The output flag accepts wide and wide renders additional columns".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default table output includes component column | "The default status output is a table with plain space-padded columns", "Default table output includes component column" | No (byte equal, checked) |
| Resource without component | "The default status output is a table with plain space-padded columns", "Resource without component" | No (byte equal, checked) |
| Wide format accepted as output value | "The output flag accepts wide and wide renders additional columns", "Wide format accepted as output value" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The default output format SHALL be a table showing: resource kind, nam... | body of "The default status output is a table with plain space-padded columns", the part of this sentence it holds, verbatim or as the words changed below say |
| The COMPONENT column SHALL be populated from the inventory entry for e... | body of "The default status output is a table with plain space-padded columns", verbatim |
| Resources without a component SHALL display `-` in the COMPONENT colum... | body of "The default status output is a table with plain space-padded columns", verbatim |
| Columns SHALL be aligned and human-readable. | body of "The default status output is a table with plain space-padded columns", verbatim |
| The table SHALL render as plain, space-padded columns with no border c... | body of "The default status output is a table with plain space-padded columns", verbatim |
| The `--output`/`-o` flag SHALL accept `wide` as a valid value in addit... | body of "The output flag accepts wide and wide renders additional columns", the part of this sentence it holds, verbatim or as the words changed below say |
| When `-o wide` is specified, the command SHALL render a table format w... | body of "The output flag accepts wide and wide renders additional columns", the part of this sentence it holds, verbatim or as the words changed below say |

Words changed or added:
- "The default status output is a table with plain space-padded columns": "The default output format SHALL be a table showing: resource kind, name, namespace, compon..." reads "The default output format of `opm instance status` SHALL be a table showing: resource kind..." ("The default output format" reads "The default output format of `opm instance status`" (the command named in the scenarios of the same requirement))
- "The output flag accepts wide and wide renders additional columns": "The `--output`/`-o` flag SHALL accept `wide` as a valid value in addition to `table`, `yam..." reads "The `--output`/`-o` flag of `opm instance status` SHALL accept `wide` as a valid value in ..." ("The `--output`/`-o` flag" reads "The `--output`/`-o` flag of `opm instance status`" (the command named in the scenarios of the same requirement))
- "The output flag accepts wide and wide renders additional columns": "When `-o wide` is specified, the command SHALL render a table format with additional colum..." reads "When `-o wide` is specified, the `opm instance status` command SHALL render a table format..." ("the command" reads "the `opm instance status` command" (named in the scenarios of the same requirement))

Lengths (old, new): old 645; new "The default status output is a table with plain space-padded columns" 470; "The output flag accepts wide and wide renders additional columns" 246.

### Ledger: mod-status, Status header does not depend on inventory change history

Form: c2 (REMOVED, ADDED under new names). New requirements: "The status header shows the instance name, namespace, health and summary from the persisted inventory record", "The status header requires no inventory change history, module source or re-rendering".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Header sourced from persisted inventory | "The status header shows the instance name, namespace, health and summary from the persisted inventory record", "Header sourced from persisted inventory" | No (byte equal, checked) |
| (new) | "The status header requires no inventory change history, module source or re-rendering", "Header needs no change history" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The status output SHALL display a metadata header above the resource t... | body of "The status header shows the instance name, namespace, health and summary from the persisted inventory record", verbatim |
| Module version and ownership metadata SHALL come from the persisted in... | body of "The status header shows the instance name, namespace, health and summary from the persisted inventory record", verbatim |
| <!-- Was: release name, release inventory record, releaseMetadata (enh... | body of "The status header shows the instance name, namespace, health and summary from the persisted inventory record", verbatim |
| The command SHALL NOT require inventory change-history metadata such a... | body of "The status header requires no inventory change history, module source or re-rendering", with "The command" read as "The `opm instance status` command" |

Words changed or added:
- "The status header requires no inventory change history, module source or re-rendering": "The command SHALL NOT require inventory change-history metadata such as source version, ra..." reads "The `opm instance status` command SHALL NOT require inventory change-history metadata such..." ("The command" reads "The `opm instance status` command" (named in the scenario of the same requirement))

Lengths (old, new): old 659; new "The status header shows the instance name, namespace, health and summary from the persisted inventory record" 478; "The status header requires no inventory change history, module source or re-rendering" 202.

### Ledger: mod-status, Status output uses color

Form: c2 (REMOVED, ADDED under new names). New requirements: "The status table and header are color-coded on a TTY and plain otherwise", "The color mapping of the status output".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Color output on TTY | "The status table and header are color-coded on a TTY and plain otherwise", "Color output on TTY" | No (byte equal, checked) |
| Color disabled on pipe | "The status table and header are color-coded on a TTY and plain otherwise", "Color disabled on pipe" | No (byte equal, checked) |
| Color disabled by NO_COLOR | "The status table and header are color-coded on a TTY and plain otherwise", "Color disabled by NO_COLOR" | No (byte equal, checked) |
| (new) | "The color mapping of the status output", "Unknown renders yellow" | New restatement scenario: only words of the old clause, no new case |

| Old clause | New place |
| --- | --- |
| The status table and header SHALL use color-coded output when stdout i... | body of "The status table and header are color-coded on a TTY and plain otherwise", verbatim |
| Color SHALL be disabled when stdout is not a TTY or when the `NO_COLOR... | body of "The status table and header are color-coded on a TTY and plain otherwise", verbatim |
| The color mapping SHALL be: Health status `Ready` and `Complete`: gree... | body of "The color mapping of the status output", verbatim as a list, with the subject extended by the table and header |

Words changed or added:
- "The color mapping of the status output": "The color mapping SHALL be:
- Health status `Ready` and `Complete`: green
- Health status ..." reads "The color mapping of the status table and header SHALL be:
- Health status `Ready` and `Co..." ("The color mapping" reads "The color mapping of the status table and header", copied from the first sentence of the same requirement)

Lengths (old, new): old 508; new "The status table and header are color-coded on a TTY and plain otherwise" 179; "The color mapping of the status output" 359.

### Ledger: mod-status, Status lists unreadable tracked resources as Unknown

Form: c2 (REMOVED, ADDED under new names). New requirements: "An unreadable tracked resource is warned about and listed as Unknown", "An Unknown resource makes the instance NotReady and the command exit 2".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Forbidden resource makes the instance not ready | "An unreadable tracked resource is warned about and listed as Unknown", "Forbidden resource makes the instance not ready" | No (byte equal, checked) |
| Every read fails | "An Unknown resource makes the instance NotReady and the command exit 2", "Every read fails" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| When a tracked resource cannot be read (its GET fails with an error ot... | body of "An unreadable tracked resource is warned about and listed as Unknown", verbatim |
| An `Unknown` resource is not healthy: the instance's aggregate status ... | body of "An Unknown resource makes the instance NotReady and the command exit 2", the part of this sentence it holds, verbatim or as the words changed below say |
| An instance whose tracked resources all failed to read SHALL be report... | body of "An Unknown resource makes the instance NotReady and the command exit 2", verbatim |

Words changed or added:
- "An Unknown resource makes the instance NotReady and the command exit 2": "An `Unknown` resource is not healthy: the instance's aggregate status SHALL be `NotReady`,..." reads "An `Unknown` resource is not healthy: the instance's aggregate status SHALL be `NotReady`,..." ("the command" reads "the `opm instance status` command", copied from the first sentence of the same requirement)

Lengths (old, new): old 514; new "An unreadable tracked resource is warned about and listed as Unknown" 251; "An Unknown resource makes the instance NotReady and the command exit 2" 284.

## Ledger entries of cli-B4, unit 4b (release-cascade-wiring, cmdutil, platform-check)

One subsection per rewritten requirement, 16 in all: `release-cascade-wiring` 6, `cmdutil` 5 and `platform-check` 5. With unit 4a this completes cli-B4 (28 requirements, running N 112). Every new body is built from whole sentences of the old body, copied unchanged, or from parts of one sentence cut at a list boundary with the subject repeated. The subject words changed are listed under each requirement: a pronoun or "the command" read as the command or file named in the same old requirement or in the Purpose of the spec, an em-dash read as a colon, a closing period where a list marker or a comma stood. Each split block carries the whole old `Source:` line as a `**Source**:` metadata line. Six bodies end with a named reference to a requirement of the same spec, where the old sentence cannot stand without the referent. Old scenario names are all kept; the byte check against `origin/main` ran once per rewritten requirement and printed no `DIFF` and no `LOST`. `grep -rn --exclude-dir=archive` over the repo found no reference to an old header of this unit.

### Ledger: release-cascade-wiring, The cascade receiver runs on a dispatch, a daily sweep and a manual run

Form: b (MODIFIED parent, ADDED split-offs). The parent stays with a shorter body and both its scenarios. New requirements: "The cascade job calls the shared receiver workflow at the pinned commit with fixed permissions", "The cascade job passes the dry-run, gates-only, gate mode and setup-go inputs".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| An upstream release starts the receiver | "The cascade receiver runs on a dispatch, a daily sweep and a manual run", "An upstream release starts the receiver" | No (byte equal, checked) |
| The daily sweep runs without a dispatch | "The cascade receiver runs on a dispatch, a daily sweep and a manual run", "The daily sweep runs without a dispatch" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The cascade job calls the pinned shared receiver" in "The cascade job calls the shared receiver workflow at the pinned commit with fixed permissions"
- "The gate modes default to warn" in "The cascade job passes the dry-run, gates-only, gate mode and setup-go inputs"

| Old clause | New place |
| --- | --- |
| `.github/workflows/deps-cascade.yml` SHALL run on `repository_dispatch... | body of the parent, verbatim |
| It SHALL declare `permissions: {}` at the workflow level and SHALL hav... | body of the parent, verbatim |
| The receiver computes the cascade diff with this repository's `task -x... | body of the parent, verbatim (the parent carries SHALL in its other sentences) |
| Its job `cascade` SHALL call the shared `open-platform-model/.github`... | body of "The cascade job calls the shared receiver workflow at the pinned commit with fixed permissions", cut at the list boundary before ", and SHALL pass" |
| and SHALL pass `dry-run`, `gates-only` from the `gates_only` input, `g... | body of "The cascade job passes the dry-run, gates-only, gate mode and setup-go inputs", the second part of the same sentence |
| Source: workspace RELEASING.md, sections "The receiver", "Two-job spli... | `**Source**:` line of the parent and of both split-offs, the whole old line |

Words changed or added:
- Both split-offs: "Its job `cascade`" reads "The job `cascade` of `.github/workflows/deps-cascade.yml`" (the subject is the workflow file named in the first sentence of the same paragraph)
- "The cascade job calls ...": the closing period stands where ", and SHALL pass" began
- "The cascade job passes ...": the text starts at "SHALL pass"; the "and" before it is dropped

Lengths (old, new): old 1124; new "The cascade job calls the shared receiver workflow at the pinned commit with fixed permissions" 274; "The cascade job passes the dry-run, gates-only, gate mode and setup-go inputs" 317; "The cascade receiver runs on a dispatch, a daily sweep and a manual run" 488.

### Ledger: release-cascade-wiring, The receiver publishes only through its own cascade Environment job

Form: c2 (REMOVED, ADDED under new names). New requirements: "The publish job needs cascade, runs in the cascade Environment and grants only two read permissions", "The publish job has exactly one step, the shared cascade-publish action with fixed inputs", "The publish job runs only on the receiver caller's if: byte for byte", "The receiver checks that the cascade labels exist and never creates one".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A live run publishes from the caller's own job | "The publish job needs cascade, runs in the cascade Environment and grants only two read permissions", "A live run publishes from the caller's own job" | No (byte equal, checked) |
| The receiver never creates a label in the cli | "The receiver checks that the cascade labels exist and never creates one", "The receiver never creates a label in the cli" | No (byte equal, checked) |
| A gates-only run never publishes | "The publish job runs only on the receiver caller's if: byte for byte", "A gates-only run never publishes" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The publish job runs the shared action as its one step" in "The publish job has exactly one step, the shared cascade-publish action with fixed inputs"

| Old clause | New place |
| --- | --- |
| `deps-cascade.yml` SHALL have a second job, `publish`, that needs `cas... | body of "The publish job needs cascade, runs in the cascade Environment and grants only two read permissions", cut at the list boundary before ", and has exactly one step" |
| and has exactly one step: the shared `cascade-publish` action at the r... | body of "The publish job has exactly one step, the shared cascade-publish action with fixed inputs", the second part of the same sentence with the subject repeated |
| Its `if:` SHALL be the receiver caller's `if:` in `.github` `reference... | body of "The publish job runs only on the receiver caller's if: byte for byte" |
| Because `.github/labels.yml` owns the repository's labels, publish SHA... | body of "The receiver checks that the cascade labels exist and never creates one", verbatim |
| Source: workspace RELEASING.md, sections "Two-job split" and "Labels";... | `**Source**:` line of all four, the whole old line |

Words changed or added:
- "The publish job needs cascade ...": the list ends at "`pull-requests: read`" with a closing period where ", and has exactly one step" began
- "The publish job has exactly one step ...": "that needs `cascade`, runs ..., and has exactly one step" reads "that has exactly one step" (the earlier list items sit in the first split)
- "The publish job runs only on ...": "Its `if:`" reads "The `if:` of the `publish` job"

Lengths (old, new): old 1267; new "The publish job needs cascade, runs in the cascade Environment and grants only two read permissions" 215; "The publish job has exactly one step, the shared cascade-publish action with fixed inputs" 414; "The publish job runs only on the receiver caller's if: byte for byte" 351; "The receiver checks that the cascade labels exist and never creates one" 136.

### Ledger: release-cascade-wiring, The receiver is a dry run unless CASCADE_DRY_RUN is exactly false

Form: c2 (REMOVED, ADDED under new names). New requirements: "The caller passes dry-run so that the receiver is live only when CASCADE_DRY_RUN is exactly false", "The dry-run switch is enforced in three places and a dry run pushes nothing".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Variable unset | "The caller passes dry-run so that the receiver is live only when CASCADE_DRY_RUN is exactly false", "Variable unset" | No (byte equal, checked) |
| Variable set to true | "The caller passes dry-run so that the receiver is live only when CASCADE_DRY_RUN is exactly false", "Variable set to true" | No (byte equal, checked) |
| Manual dry run while live | "The caller passes dry-run so that the receiver is live only when CASCADE_DRY_RUN is exactly false", "Manual dry run while live" | No (byte equal, checked) |
| Repository code cannot unlock publishing in a dry run | "The dry-run switch is enforced in three places and a dry run pushes nothing", "Repository code cannot unlock publishing in a dry run" | No (byte equal, checked) |
| Live only at exactly false | "The caller passes dry-run so that the receiver is live only when CASCADE_DRY_RUN is exactly false", "Live only at exactly false" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The caller SHALL pass `dry-run` as `${{ inputs.dry_run == true || vars... | body of "The caller passes dry-run so that the receiver is live only when CASCADE_DRY_RUN is exactly false", verbatim |
| A run from any ref other than `main` is always a dry run. | body of "The caller passes dry-run ...", verbatim (the body carries SHALL in its first sentence) |
| The switch SHALL be enforced in three places that read `inputs` and `v... | body of "The dry-run switch is enforced in three places and a dry run pushes nothing" |
| In a dry run the receiver computes the diff, title, body and labels | body of "The dry-run switch is enforced in three places and a dry run pushes nothing", verbatim |
| The release-PR gate statuses are still posted. | body of "The dry-run switch is enforced in three places and a dry run pushes nothing", verbatim |
| Source: workspace RELEASING.md, sections "The receiver" and "Stop swit... | `**Source**:` line of both, the whole old line |

Words changed or added:
- "The dry-run switch ...": "The switch" reads "The `dry-run` switch" (the token is the one the first sentence of the old body names)

Lengths (old, new): old 1007; new "The caller passes dry-run so that the receiver is live only when CASCADE_DRY_RUN is exactly false" 392; "The dry-run switch is enforced in three places and a dry run pushes nothing" 496.

### Ledger: release-cascade-wiring, Pull requests into main carry the cascade gate statuses

Form: c2 (REMOVED, ADDED under new names). New requirements: "The cascade gates workflow runs on pull_request_target with no workflow permissions", "The cascade gates job calls the shared gates workflow and checks out no pull request code", "Gate statuses are posted on every pull request into main except in two cases", "An ordinary or fork pull request gets n/a gate statuses", "A same-repository release pull request gets its gate statuses from a gates-only receiver run".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Ordinary pull request | "An ordinary or fork pull request gets n/a gate statuses", "Ordinary pull request" | No (byte equal, checked) |
| Release pull request with a shipped pin behind | "A same-repository release pull request gets its gate statuses from a gates-only receiver run", "Release pull request with a shipped pin behind" | No (byte equal, checked) |
| The gate job checks out no pull request code | "The cascade gates job calls the shared gates workflow and checks out no pull request code", "The gate job checks out no pull request code" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The gates workflow runs on pull request events" in "The cascade gates workflow runs on pull_request_target with no workflow permissions"
- "A Dependabot pull request whose token cannot post statuses" in "Gate statuses are posted on every pull request into main except in two cases"

| Old clause | New place |
| --- | --- |
| `.github/workflows/cascade-gates.yml` SHALL run on `pull_request_targe... | body of "The cascade gates workflow runs on pull_request_target with no workflow permissions", verbatim |
| It SHALL declare `permissions: {}` at the workflow level. | body of "The cascade gates workflow runs on pull_request_target with no workflow permissions", verbatim |
| It SHALL have one job that calls the shared `open-platform-model/.gith... | body of "The cascade gates job calls the shared gates workflow and checks out no pull request code" |
| The commit statuses `cascade/freshness` (G2) and `cascade/settled` (G3... | opening clause and the first list item, body of "An ordinary or fork pull request gets n/a gate statuses" |
| ... on a same-repository release pull request, a gates-only receiver r... | opening clause and the second list item, body of "A same-repository release pull request gets its gate statuses from a gates-only receiver run" |
| ... opened or synchronized after this change: (opening clause) | opening clause, also the first sentence of the body of "Gate statuses are posted on every pull request into main except in two cases", which then holds the exceptions |
| Two cases are excepted: a release pull request whose gates-only evalua... | body of "Gate statuses are posted on every pull request into main except in two cases", verbatim |
| In `warn` mode a problem is posted as a success whose description star... | body of "A same-repository release pull request gets its gate statuses from a gates-only receiver run", verbatim |
| This change does not make either status a required check. | body of "Gate statuses are posted on every pull request into main except in two cases", verbatim |
| Source: workspace RELEASING.md, section "Gates"; ... | `**Source**:` line of all five, the whole old line |

Words changed or added:
- "The cascade gates job ...": "It SHALL have one job" reads "`.github/workflows/cascade-gates.yml` SHALL have one job" (the subject of the first sentence)
- Three bodies that hold the opening clause of the statuses sentence: the sentence is cut at the colon (a period stands in the exceptions body) or at the semicolon between its two list items (a period closes the first item; the second item follows the colon of the opening clause, as in the old text)
- "An ordinary or fork pull request ..." and "A same-repository release pull request ...": each body ends with "(Requirement: Gate statuses are posted on every pull request into main except in two cases)", the named reference that gives the two exceptions of the old sentence to the rule they qualify

Lengths (old, new): old 1373; new "The cascade gates workflow runs on pull_request_target with no workflow permissions" 174; "The cascade gates job calls the shared gates workflow and checks out no pull request code" 390; "Gate statuses are posted on every pull request into main except in two cases" 491; "An ordinary or fork pull request gets n/a gate statuses" 311; "A same-repository release pull request gets its gate statuses from a gates-only receiver run" 427.

### Ledger: release-cascade-wiring, Only the caller-owned cascade jobs read the App key

Form: c2 (REMOVED, ADDED under new names). New requirements: "Only the caller-owned cascade jobs read the App key with a fixed job shape", "No other job declares the cascade Environment, release.yml sets no workflow-level env key and the Environment deploys from main only".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A branch run cannot reach the key | "No other job declares the cascade Environment, release.yml sets no workflow-level env key and the Environment deploys from main only", "A branch run cannot reach the key" | No (byte equal, checked) |
| A pull request adds a step to a key-holding job | "Only the caller-owned cascade jobs read the App key with a fixed job shape", "A pull request adds a step to a key-holding job" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "Only the key-holding jobs declare the cascade Environment" in "No other job declares the cascade Environment, release.yml sets no workflow-level env key and the Environment deploys from main only"

| Old clause | New place |
| --- | --- |
| The key SHALL be read only in the caller-owned `notify-downstream` job... | body of "Only the caller-owned cascade jobs read the App key with a fixed job shape", verbatim |
| No other job in the repository SHALL declare `environment: cascade`, a... | body of "No other job declares the cascade Environment, ...", verbatim |
| The `cascade` Environment deploys from `main` only. | body of "No other job declares the cascade Environment, ...", verbatim (the body carries SHALL in its first sentence) |
| Source: workspace RELEASING.md, sections "Notify after publish" and "T... | `**Source**:` line of both, the whole old line |

Words changed or added:
- "No other job declares the cascade Environment, ...": the body ends with "(Requirement: Only the caller-owned cascade jobs read the App key with a fixed job shape)", the named reference that tells which jobs "No other job" leaves out

Lengths (old, new): old 899; new "Only the caller-owned cascade jobs read the App key with a fixed job shape" 495; "No other job declares the cascade Environment, release.yml sets no workflow-level env key and the Environment deploys from main only" 329.

### Ledger: release-cascade-wiring, Every cascade reference is pinned to one .github main commit and checked in CI

Form: c2 (REMOVED, ADDED under new names). New requirements: "The release and receiver workflows name the same full SHA of a .github main commit", "The gates workflow and the resolver checkout name the same full SHA of a .github main commit", "A .github change reaches the cli only through a pin pull request", "The wiring-check script is a byte-identical copy of the pinned file and runs offline", "The wiring-check values file holds the cli's values", "The wiring check asserts the pin, the key rules, the job shapes and the caller shapes", "The required lint job runs the wiring check on every pull request", "Nothing reaches the wiring step from around it", "Every checkout of .github is pinned to a full SHA with fixed keys".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| All five references carry one SHA | "The release and receiver workflows name the same full SHA of a .github main commit", "All five references carry one SHA" | No (byte equal, checked) |
| One reference moves alone | "The gates workflow and the resolver checkout name the same full SHA of a .github main commit", "One reference moves alone" | No (byte equal, checked) |
| A workflow env key in release.yml | "The wiring check asserts the pin, the key rules, the job shapes and the caller shapes", "A workflow env key in release.yml" | No (byte equal, checked) |
| A pin that exists only in a fork | "The required lint job runs the wiring check on every pull request", "A pin that exists only in a fork" | No (byte equal, checked) |
| The copy is the canonical script | "The wiring-check script is a byte-identical copy of the pinned file and runs offline", "The copy is the canonical script" | No (byte equal, checked) |
| A publishing workflow restores a cache | "The wiring check asserts the pin, the key rules, the job shapes and the caller shapes", "A publishing workflow restores a cache" | No (byte equal, checked) |
| A copy that drifted from the pinned file | "The required lint job runs the wiring check on every pull request", "A copy that drifted from the pinned file" | No (byte equal, checked) |
| A run step before the wiring step | "Nothing reaches the wiring step from around it", "A run step before the wiring step" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "A .github change arrives as one pin pull request" in "A .github change reaches the cli only through a pin pull request"
- "The task runs the check offline" in "The wiring-check script is a byte-identical copy of the pinned file and runs offline"
- "The values file holds the cli's values" in "The wiring-check values file holds the cli's values"
- "The resolver checkout carries only fixed keys" in "Every checkout of .github is pinned to a full SHA with fixed keys"

| Old clause | New place |
| --- | --- |
| Every reference this repository makes to the cascade code SHALL name t... | sentence over 500 on its own, cut at a list boundary: opening clause with the first three list items in "The release and receiver workflows name ...", opening clause with the last two in "The gates workflow and the resolver checkout name ..." |
| A `.github` change SHALL reach this repository only through a pull req... | body of "A .github change reaches the cli only through a pin pull request", verbatim |
| `.tasks/cascade/wiring-check.sh` SHALL be a byte-identical copy of `.g... | body of "The wiring-check script is a byte-identical copy of the pinned file and runs offline", first part of a sentence over 500, closing period where ", and" stood |
| and `.tasks/cascade/wiring-check.yaml` SHALL hold the cli's values fro... | body of "The wiring-check values file holds the cli's values", second part of the same sentence |
| The check SHALL assert the pin, the cascade key rule, the release key... | body of "The wiring check asserts the pin, the key rules, the job shapes and the caller shapes" |
| `pr.yml`'s required `lint` job SHALL run it on every pull request | body of "The required lint job runs the wiring check on every pull request" |
| Nothing SHALL reach that step from around it: ... | body of "Nothing reaches the wiring step from around it" |
| Every checkout of `.github` (the `cascade-task.yml` resolver checkout)... | body of "Every checkout of .github is pinned to a full SHA with fixed keys", verbatim |
| `task cascade:wiring:check` SHALL run the same check offline, and the... | body of "The wiring-check script is a byte-identical copy of the pinned file and runs offline", verbatim |
| Source: owner decision 24 and the supervisor's extension ...; ... | `**Source**:` line of all nine, the whole old line |

Words changed or added:
- "The release and receiver workflows ..." and "The gates workflow ...": the opening clause is repeated, the list is cut at the boundary between "`deps-cascade.yml`," and "the `cascade-gates.yml` workflow"; the first closes with a period where ", " stood, the second keeps the "and" before its last item
- "The wiring-check script ...": the sentence ends at "moved only with the pin" with a closing period; the sentence on `task cascade:wiring:check` follows unchanged
- "The wiring-check values file ...": the "and" before `.tasks/cascade/wiring-check.yaml` is dropped
- "The wiring check asserts ...": "The check" reads "The check of `.tasks/cascade/wiring-check.sh`"
- "The required lint job ...": "it" reads "`.tasks/cascade/wiring-check.sh`"
- "Nothing reaches the wiring step ...": "that step" reads "the step \"Verify the cascade wiring\"" (the step named in the same old paragraph)

Lengths (old, new): old 3255; new "The release and receiver workflows name the same full SHA of a .github main commit" 357; "The gates workflow and the resolver checkout name the same full SHA of a .github main commit" 370; "A .github change reaches the cli only through a pin pull request" 273; "The wiring-check script is a byte-identical copy of the pinned file and runs offline" 263; "The wiring-check values file holds the cli's values" 411; "The wiring check asserts the pin, the key rules, the job shapes and the caller shapes" 457; "The required lint job runs the wiring check on every pull request" 413; "Nothing reaches the wiring step from around it" 384; "Every checkout of .github is pinned to a full SHA with fixed keys" 248.

### Ledger: cmdutil, InstanceSelectorFlags carries and validates the resolved instance selector

Form: b (MODIFIED parent, ADDED split-off). The parent stays with a shorter body and its three scenarios. New requirements: "Commands build the instance selector from their positional argument".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Both selectors set is rejected | "InstanceSelectorFlags carries and validates the resolved instance selector", "Both selectors set is rejected" | No (byte equal, checked) |
| Neither selector set is rejected | "InstanceSelectorFlags carries and validates the resolved instance selector", "Neither selector set is rejected" | No (byte equal, checked) |
| Exactly one selector set is accepted | "InstanceSelectorFlags carries and validates the resolved instance selector", "Exactly one selector set is accepted" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "A command builds its selector from the positional argument" in "Commands build the instance selector from their positional argument"

| Old clause | New place |
| --- | --- |
| The `InstanceSelectorFlags` struct SHALL carry the resolved instance s... | body of the parent, verbatim |
| Commands SHALL build it from their positional `<file|name|uuid>` argum... | body of "Commands build the instance selector from their positional argument" |
| It SHALL provide a `Validate()` method that enforces mutual exclusivit... | body of the parent, verbatim ("It" is the struct of the first sentence, kept in the same body) |
| <!-- Was: ReleaseSelectorFlags, --release-name/--release-id (enhanceme... | end of the body of "Commands build the instance selector from their positional argument", verbatim |

Words changed or added:
- "Commands build ...": "it" reads "the resolved instance selector" (the phrase of the first sentence of the old body)

Lengths (old, new): old 655; new "Commands build the instance selector from their positional argument" 400; "InstanceSelectorFlags carries and validates the resolved instance selector" 282.

### Ledger: cmdutil, ShowRenderOutput checks for errors, shows transformer matches, and logs warnings

Form: c2 (REMOVED, ADDED under new names). New requirements: "ShowRenderOutput returns an ExitError with ExitValidationError when the render has errors", "ShowRenderOutput shows compact transformer matches by default and detailed matches in verbose mode", "ShowRenderOutput logs the render warnings via the module-scoped logger".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| ShowRenderOutput returns error when render has errors | "ShowRenderOutput returns an ExitError with ExitValidationError when the render has errors", "ShowRenderOutput returns error when render has errors" | No (byte equal, checked) |
| ShowRenderOutput shows compact matches in default mode | "ShowRenderOutput shows compact transformer matches by default and detailed matches in verbose mode", "ShowRenderOutput shows compact matches in default mode" | No (byte equal, checked) |
| ShowRenderOutput shows detailed matches in verbose mode | "ShowRenderOutput shows compact transformer matches by default and detailed matches in verbose mode", "ShowRenderOutput shows detailed matches in verbose mode" | No (byte equal, checked) |
| ShowRenderOutput logs warnings | "ShowRenderOutput logs the render warnings via the module-scoped logger", "ShowRenderOutput logs warnings" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The render output function (`render.ShowOutput`) SHALL accept a render... | first line of all three bodies, verbatim |
| 1. Check for render errors — if present, format and print render error... | body of "ShowRenderOutput returns an ExitError with ExitValidationError when the render has errors" |
| 2. Show transformer match output — verbose mode shows module metadata... | body of "ShowRenderOutput shows compact transformer matches by default and detailed matches in verbose mode" |
| 3. Log any warnings from the render result via the module-scoped logge... | body of "ShowRenderOutput logs the render warnings via the module-scoped logger", verbatim |

Words changed or added:
- Items 1 and 2: the em-dash reads a colon
- Each body keeps the old item number after a blank line, so that the item renders as a separate paragraph

Lengths (old, new): old 706; new "ShowRenderOutput returns an ExitError with ExitValidationError when the render has errors" 277; "ShowRenderOutput shows compact transformer matches by default and detailed matches in verbose mode" 250; "ShowRenderOutput logs the render warnings via the module-scoped logger" 421.

### Ledger: cmdutil, PrintValidationError groups CUE positions on any error

Form: b (MODIFIED parent, ADDED split-off). The parent stays with a shorter body and its two scenarios. New requirements: "PrintValidationError accepts a message string and an error".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Kernel validation error with CUE positions | "PrintValidationError groups CUE positions on any error", "Kernel validation error with CUE positions" | No (byte equal, checked) |
| Generic error without CUE details | "PrintValidationError groups CUE positions on any error", "Generic error without CUE details" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "PrintValidationError is called with a message and an error" in "PrintValidationError accepts a message string and an error"

| Old clause | New place |
| --- | --- |
| The `PrintValidationError` function SHALL accept a message string and... | body of "PrintValidationError accepts a message string and an error", verbatim |
| When the error chain carries CUE errors with at least one valid source... | body of the parent |
| Typed kernel refusals (unresolved demands) and `ValidationError` value... | body of the parent, verbatim (the parent carries SHALL in its other sentences) |
| For other errors, it SHALL use the standard key-value log format. | body of the parent |

Words changed or added:
- The parent: both "it" read "`PrintValidationError`" (the function named in the sentence that moved out)

Lengths (old, new): old 510; new "PrintValidationError accepts a message string and an error" 79; "PrintValidationError groups CUE positions on any error" 470.

### Ledger: cmdutil, PrintRenderErrors formats render errors with diagnostic detail

Form: c2 (REMOVED, ADDED under new names). New requirements: "The render refusal printer accepts a render error and prints the component name, transformer FQN and cause of a transform error", "The render refusal printer prints the component names of an unmatched-components refusal and the candidate transformers in verbose mode", "The render refusal printer prints every contract key and the competing catalogs of an over-subscription refusal", "Unresolved demands are rendered from the demand rows and name the defining catalog".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Unmatched components error with diagnostics | "The render refusal printer prints the component names of an unmatched-components refusal and the candidate transformers in verbose mode", "Unmatched components error with diagnostics" | No (byte equal, checked) |
| Over-subscribed contracts print once | "The render refusal printer prints every contract key and the competing catalogs of an over-subscription refusal", "Over-subscribed contracts print once" | No (byte equal, checked) |
| Transform error | "The render refusal printer accepts a render error and prints the component name, transformer FQN and cause of a transform error", "Transform error" | No (byte equal, checked) |
| Unresolved demands format from rows | "Unresolved demands are rendered from the demand rows and name the defining catalog", "Unresolved demands format from rows" | No (byte equal, checked) |
| A defined but unimplemented demand names its catalog | "Unresolved demands are rendered from the demand rows and name the defining catalog", "A defined but unimplemented demand names its catalog" | No (byte equal, checked) |
| A demand no catalog defines keeps the plain line | "Unresolved demands are rendered from the demand rows and name the defining catalog", "A demand no catalog defines keeps the plain line" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The render refusal printer (`printValidationError`, funnelling into `c... | body of "The render refusal printer accepts a render error and prints the component name, transformer FQN and cause of a transform error", verbatim |
| For an unmatched-components refusal it SHALL print the component names... | body of "The render refusal printer prints the component names of an unmatched-components refusal ..." |
| For an over-subscription refusal it SHALL print every contract key | body of "The render refusal printer prints every contract key and the competing catalogs of an over-subscription refusal" |
| For transform errors, it SHALL print the component name, transformer F... | body of "The render refusal printer accepts a render error and prints ...", verbatim ("it" is the printer of the first sentence in the same body) |
| Unresolved demands SHALL be rendered from the demand rows themselves, | body of "Unresolved demands are rendered from the demand rows and name the defining catalog", verbatim |

Words changed or added:
- "... unmatched-components refusal ..." and "... over-subscription refusal": the first "it" reads "the render refusal printer" (the subject of the first sentence)

Lengths (old, new): old 985; new "The render refusal printer accepts a render error and prints the component name, transformer FQN and cause of a transform error" 266; "The render refusal printer prints the component names of an unmatched-components refusal and the candidate transformers in verbose mode" 357; "The render refusal printer prints every contract key and the competing catalogs of an over-subscription refusal" 175; "Unresolved demands are rendered from the demand rows and name the defining catalog" 232.

### Ledger: cmdutil, Shared inventory resolution helper in the query workflow

Form: c2 (REMOVED, ADDED under new names). New requirements: "ResolveInventory resolves the record by instance ID or by instance name", "ResolveInventory returns an InstanceNotFoundError with exit code 5 when no record is found", "ResolveInventory returns the live resources and the missing entries on success".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Resolve by instance name | "ResolveInventory resolves the record by instance ID or by instance name", "Resolve by instance name" | No (byte equal, checked) |
| Resolve by instance UUID | "ResolveInventory resolves the record by instance ID or by instance name", "Resolve by instance UUID" | No (byte equal, checked) |
| Instance not found | "ResolveInventory returns an InstanceNotFoundError with exit code 5 when no record is found", "Instance not found" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "A successful resolution returns the live resources and the missing entries" in "ResolveInventory returns the live resources and the missing entries on success"

| Old clause | New place |
| --- | --- |
| `query.ResolveInventory` (in `internal/workflow/query`) SHALL resolve... | body of "ResolveInventory resolves the record by instance ID or by instance name", verbatim |
| If `InstanceID` is non-empty, it SHALL resolve via `inventory.FindReco... | body of "ResolveInventory resolves the record by instance ID or by instance name", verbatim |
| When no record is found, it SHALL return an `InstanceNotFoundError` wi... | body of "ResolveInventory returns an InstanceNotFoundError with exit code 5 when no record is found" |
| On success it SHALL also return the live resources and the missing ent... | body of "ResolveInventory returns the live resources and the missing entries on success" |
| <!-- Was: cmdutil.ResolveInventory over *ReleaseSelectorFlags, Release... | end of the body of "ResolveInventory returns an InstanceNotFoundError with exit code 5 when no record is found", verbatim |

Words changed or added:
- The last two bodies: "it" reads "`query.ResolveInventory`" (the subject of the first sentence)

Lengths (old, new): old 793; new "ResolveInventory resolves the record by instance ID or by instance name" 359; "ResolveInventory returns an InstanceNotFoundError with exit code 5 when no record is found" 331; "ResolveInventory returns the live resources and the missing entries on success" 145.

### Ledger: platform-check, opm platform check reports a platform's contract inventory

Form: c2 (REMOVED, ADDED under new names). New requirements: "opm platform check is registered under a platform command group and never applies, renders or contacts a cluster", "opm platform check reports the defined contracts, their implementations and the provider-fulfilled contracts nothing requires", "opm platform check reports every contract key that more than one enabled registry entry defines", "opm platform check reports the provider-fulfilled contracts required by transformers of more than one enabled registry entry", "opm platform check reports every comparable pair of enabled transformers", "opm platform check reports a verdict line for each inventory boolean and words verdicts as vacuous only when nothing is defined, provided or colliding".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A healthy platform reports clean | "opm platform check reports the defined contracts, their implementations and the provider-fulfilled contracts nothing requires", "A healthy platform reports clean" | No (byte equal, checked) |
| A platform whose catalogs list nothing reports an empty inventory | "opm platform check reports a verdict line for each inventory boolean and words verdicts as vacuous only when nothing is defined, provided or colliding", "A platform whose catalogs list nothing reports an empty inventory" | No (byte equal, checked) |
| The report names the defining catalog | "opm platform check reports the defined contracts, their implementations and the provider-fulfilled contracts nothing requires", "The report names the defining catalog" | No (byte equal, checked) |
| The report names both transformers of a comparable pair | "opm platform check reports every comparable pair of enabled transformers", "The report names both transformers of a comparable pair" | No (byte equal, checked) |
| Providers of a contract no enabled catalog defines are not vacuous | "opm platform check reports the provider-fulfilled contracts required by transformers of more than one enabled registry entry", "Providers of a contract no enabled catalog defines are not vacuous" | No (byte equal, checked) |
| A platform whose only contracts collide is not vacuous | "opm platform check reports every contract key that more than one enabled registry entry defines", "A platform whose only contracts collide is not vacuous" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The check applies, renders and contacts no cluster" in "opm platform check is registered under a platform command group and never applies, renders or contacts a cluster"

| Old clause | New place |
| --- | --- |
| The CLI SHALL provide `opm platform check`, registered under a `platfo... | body of "opm platform check is registered under a platform command group and never applies, renders or contacts a cluster", verbatim |
| The command SHALL NOT apply, render, or contact a cluster. | body of the same requirement, verbatim |
| It SHALL build the resolved platform through the kernel and read its c... | the sentence is over 500, cut at list boundaries: lead clause repeated in each of the five bodies that list its items; first three items in "opm platform check reports the defined contracts ..." |
| every contract key more than one enabled registry entry defines, each... | lead clause and this item, body of "opm platform check reports every contract key that more than one enabled registry entry defines" |
| the provider-fulfilled contracts required by transformers of more than... | lead clause and this item, body of "opm platform check reports the provider-fulfilled contracts required by transformers of more than one enabled registry entry" |
| every pair of enabled transformers whose match predicates are comparab... | lead clause and this item, body of "opm platform check reports every comparable pair of enabled transformers" |
| and a verdict line for each of the three booleans the inventory carrie... | lead clause and this item, body of "opm platform check reports a verdict line for each inventory boolean and words verdicts as vacuous only when nothing is defined, provided or colliding" |
| The report SHALL word the verdicts as vacuous only when no enabled cat... | same body as the verdict item, verbatim |

Words changed or added:
- The five list bodies: "It SHALL build" reads "`opm platform check` SHALL build" (the command of the first sentence); the list items are joined by semicolons as in the old text; the last item of a body closes with a period; the "and" before the verdict item is dropped

Lengths (old, new): old 1331; new "opm platform check is registered under a platform command group and never applies, renders or contacts a cluster" 147; "opm platform check reports the defined contracts, their implementations and the provider-fulfilled contracts nothing requires" 294; "opm platform check reports every contract key that more than one enabled registry entry defines" 251; "opm platform check reports the provider-fulfilled contracts required by transformers of more than one enabled registry entry" 345; "opm platform check reports every comparable pair of enabled transformers" 317; "opm platform check reports a verdict line for each inventory boolean and words verdicts as vacuous only when nothing is defined, provided or colliding" 466.

### Ledger: platform-check, An over-subscribed contract fails the command

Form: c2 (REMOVED, ADDED under new names). New requirements: "An over-subscribed contract is reported with its providing registry keys and fails the command", "Providers are counted per enabled registry entry and the command computes no count of its own".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Over-subscription exits non-zero | "An over-subscribed contract is reported with its providing registry keys and fails the command", "Over-subscription exits non-zero" | No (byte equal, checked) |
| Catalog-fulfilled plurality is not over-subscription | "An over-subscribed contract is reported with its providing registry keys and fails the command", "Catalog-fulfilled plurality is not over-subscription" | No (byte equal, checked) |
| Two majors of one provider catalog are two providers | "Providers are counted per enabled registry entry and the command computes no count of its own", "Two majors of one provider catalog are two providers" | No (byte equal, checked) |
| A disabled defining catalog does not hide over-subscription | "Providers are counted per enabled registry entry and the command computes no count of its own", "A disabled defining catalog does not hide over-subscription" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| A provider-fulfilled contract required by transformers of more than on... | body of "An over-subscribed contract is reported with its providing registry keys and fails the command" |
| This mirrors where the refusal lives in the system: `Routable: false`... | body of the same requirement, verbatim (the body carries SHALL in its first sentence; "This" follows the exit rule it explains) |
| Two majors of one catalog are two registry entries and so two provider... | body of "Providers are counted per enabled registry entry and the command computes no count of its own", verbatim (the body carries SHALL NOT in its last sentence) |
| The command reads the count the platform's core derives and the render... | body of "Providers are counted per enabled registry entry ..." |

Words changed or added:
- "the command SHALL exit" reads "`opm platform check` SHALL exit"; "The command reads" reads "`opm platform check` reads" (the command named in the Purpose of the spec and in the header of the spec's first requirement)

Lengths (old, new): old 795; new "An over-subscribed contract is reported with its providing registry keys and fails the command" 487; "Providers are counted per enabled registry entry and the command computes no count of its own" 325.

### Ledger: platform-check, An undiscriminated platform fails the command

Form: c2 (REMOVED, ADDED under new names). New requirements: "A comparable pair of enabled transformers is reported with both transformers and the shared contracts and fails the command", "The undiscriminated exit mirrors where platform-package generation refuses", "Transformers with incomparable predicates are reported as implementations and do not change the exit status".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A comparable pair exits non-zero | "A comparable pair of enabled transformers is reported with both transformers and the shared contracts and fails the command", "A comparable pair exits non-zero" | No (byte equal, checked) |
| Discriminated plurality exits zero | "Transformers with incomparable predicates are reported as implementations and do not change the exit status", "Discriminated plurality exits zero" | No (byte equal, checked) |
| Both refusals are reported together | "A comparable pair of enabled transformers is reported with both transformers and the shared contracts and fails the command", "Both refusals are reported together" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "An undiscriminated platform is refused where platform-package generation refuses it" in "The undiscriminated exit mirrors where platform-package generation refuses"

| Old clause | New place |
| --- | --- |
| A pair of enabled transformers whose match predicates are comparable o... | body of "A comparable pair of enabled transformers is reported with both transformers and the shared contracts and fails the command" |
| No arbitration between the two transformers SHALL be applied or sugges... | body of "A comparable pair of enabled transformers is reported with both transformers and the shared contracts and fails the command", verbatim |
| This mirrors where the refusal lives in the system: `Discriminated: fa... | body of "The undiscriminated exit mirrors where platform-package generation refuses", with SHALL added (see the strength table) and the named reference of the exit rule it explains |
| Transformers with incomparable predicates over a shared contract (a di... | body of "Transformers with incomparable predicates are reported as implementations and do not change the exit status", verbatim |

Words changed or added:
- "A comparable pair of enabled transformers is reported with both transformers and the shared contracts and fails the command": "the command SHALL exit" reads "`opm platform check` SHALL exit"; the citation in brackets stays in the sentence
- "The undiscriminated exit mirrors ...": the body ends with "(Requirement: A comparable pair of enabled transformers is reported with both transformers and the shared contracts and fails the command)", the named reference of a requirement of the same spec (the sentence "This mirrors" explains the exit rule that block holds)

Lengths (old, new): old 879; new "A comparable pair of enabled transformers is reported with both transformers and the shared contracts and fails the command" 433; "The undiscriminated exit mirrors where platform-package generation refuses" 386; "Transformers with incomparable predicates are reported as implementations and do not change the exit status" 211.

### Ledger: platform-check, A platform that cannot build fails with its build diagnostic

Form: c2 (REMOVED, ADDED under new names). New requirements: "A platform that does not build fails with the grouped CUE diagnostics and the validation error code", "A registry that refuses the credentials fails with the permission error code 4 and the registry login hint", "A platform whose inventory cannot be read fails naming the missing field and the core release", "A platform module that pins an older core also names the platform directory and the cue mod get command".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A broken platform module reports the build error | "A platform that does not build fails with the grouped CUE diagnostics and the validation error code", "A broken platform module reports the build error" | No (byte equal, checked) |
| A platform built against a core without the inventory | "A platform whose inventory cannot be read fails naming the missing field and the core release", "A platform built against a core without the inventory" | No (byte equal, checked) |
| A platform built against a core without the comparable-predicate report | "A platform whose inventory cannot be read fails naming the missing field and the core release", "A platform built against a core without the comparable-predicate report" | No (byte equal, checked) |
| A platform built against a core without the provider count | "A platform module that pins an older core also names the platform directory and the cue mod get command", "A platform built against a core without the provider count" | No (byte equal, checked) |
| A registry that refuses the credentials | "A registry that refuses the credentials fails with the permission error code 4 and the registry login hint", "A registry that refuses the credentials" | No (byte equal, checked) |
| A refused credential with dependencies on several registry hosts | "A registry that refuses the credentials fails with the permission error code 4 and the registry login hint", "A refused credential with dependencies on several registry hosts" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| If the resolved platform does not build, the command SHALL report the... | body of "A platform that does not build fails with the grouped CUE diagnostics and the validation error code" |
| When the platform does not build because the registry refused the cred... | body of "A registry that refuses the credentials fails with the permission error code 4 and the registry login hint" |
| The hint SHALL name the registry host when every dependency the platfo... | body of the same requirement, verbatim |
| If the platform builds but its inventory cannot be read, the command S... | body of "A platform whose inventory cannot be read fails naming the missing field and the core release" |
| When the inventory cannot be read because the platform module pins a c... | body of "A platform module that pins an older core also names the platform directory and the cue mod get command" |

Words changed or added:
- Four bodies: "the command" reads "`opm platform check`"
- "A platform module that pins an older core ...": the body ends with "(Requirement: A platform whose inventory cannot be read fails naming the missing field and the core release)", the named reference that gives "also" its referent

Lengths (old, new): old 1092; new "A platform that does not build fails with the grouped CUE diagnostics and the validation error code" 214; "A registry that refuses the credentials fails with the permission error code 4 and the registry login hint" 438; "A platform whose inventory cannot be read fails naming the missing field and the core release" 197; "A platform module that pins an older core also names the platform directory and the cue mod get command" 385.

### Ledger: platform-check, A colliding contract key fails the command

Form: c2 (REMOVED, ADDED under new names). New requirements: "A colliding contract key is reported under its own heading and fails the command", "The colliding heading states what is left out and tells the reader to keep one defining entry enabled", "The command computes no collision of its own and the fulfilled and discriminated verdicts may read yes on a colliding platform".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Two majors sharing keys exit non-zero and name the entries | "A colliding contract key is reported under its own heading and fails the command", "Two majors sharing keys exit non-zero and name the entries" | No (byte equal, checked) |
| A collision and an over-subscription are reported together | "A colliding contract key is reported under its own heading and fails the command", "A collision and an over-subscription are reported together" | No (byte equal, checked) |
| A disabled second major is not a collision | "The colliding heading states what is left out and tells the reader to keep one defining entry enabled", "A disabled second major is not a collision" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The fulfilled and discriminated verdicts may read yes on a colliding platform" in "The command computes no collision of its own and the fulfilled and discriminated verdicts may read yes on a colliding platform"

| Old clause | New place |
| --- | --- |
| A contract key that more than one enabled registry entry's catalog lis... | body of "A colliding contract key is reported under its own heading and fails the command" |
| The routable verdict SHALL name the colliding count beside the over-su... | body of "A colliding contract key is reported under its own heading and fails the command" |
| The heading SHALL state that a colliding key is left out of the define... | body of "The colliding heading states what is left out and tells the reader to keep one defining entry enabled" |
| A disabled entry never defines a key. | body of the same requirement, verbatim (the body carries SHALL in its first sentence) |
| The command reads the collisions the platform's core derives and the r... | body of "The command computes no collision of its own and the fulfilled and discriminated verdicts may read yes on a colliding platform" |
| Because a colliding key is absent from the defined contracts, the `ful... | body of the same requirement, verbatim |
| Source: enhancement 0026 (interim safety net until side-by-side majors... | `**Source**:` line of all three, the whole old line |

Words changed or added:
- "the command SHALL exit", "the command's error" and "The command reads" read "`opm platform check` SHALL exit", "`opm platform check`'s error" and "`opm platform check` reads"
- "The colliding heading states ...": the body ends with "(Requirement: A colliding contract key is reported under its own heading and fails the command)", the named reference that gives "The heading" its referent

Lengths (old, new): old 1181; new "A colliding contract key is reported under its own heading and fails the command" 478; "The colliding heading states what is left out and tells the reader to keep one defining entry enabled" 418; "The command computes no collision of its own and the fulfilled and discriminated verdicts may read yes on a colliding platform" 333.

## Ledger entries of cli-B5, unit 5a (test-fixture-lineage, cmd-structure, e2e-cluster-preconditions)

Unit 5a rewrites 13 long requirements (test-fixture-lineage 5, cmd-structure 4, e2e-cluster-preconditions 4), all by form c2. Unit 5b rewrites the other 15 of section 5 (inst-commands, kind-cluster-tasks, mod-vet, catalog-compatibility). The running N after unit 5a is 125.

Notes for the whole unit:

- The old `Operator applier grant is verified before operator-owned tests` and `A cluster-required run fails instead of skipping` of `e2e-cluster-preconditions` cite each other by header. Both are removed, so each new body that cited the other cites the new header of the requirement that now holds the cited rule (see the words changed below). `grep -rn --exclude-dir=archive` on 2026-10-19 finds no other reference to a removed header outside the rewritten requirements.
- `cmd-structure` has the scenario names `Default to current directory`, `Explicit module directory` and `File argument rejected` under both `module build` and `module apply`, as `main` has. Each stays with its own command (checked against the requirement block of `origin/main`, not by name alone).
- No old `Source:` line exists in these 13 bodies, so no `**Source**:` line is written.
- A new scenario restates only a case the old text names.

### Ledger: test-fixture-lineage, Maintained fixtures track the current schema line

Form: c2 (REMOVED, ADDED under new names). New requirements: "Maintained fixtures and examples import only the current published schema line", "A maintained module fixture's identity package declares Version as a plain string literal", "A fixture does not depend on a catalog mechanism the catalog has withdrawn".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| No retired-schema imports outside legacy | "Maintained fixtures and examples import only the current published schema line", "No retired-schema imports outside legacy" | No (byte equal, checked) |
| Vet fixtures exercise current schema | "Maintained fixtures and examples import only the current published schema line", "Vet fixtures exercise current schema" | No (byte equal, checked) |
| No fixture exercises a withdrawn catalog mechanism | "A fixture does not depend on a catalog mechanism the catalog has withdrawn", "No fixture exercises a withdrawn catalog mechanism" | No (byte equal, checked) |
| Fixture identity is a literal | "A maintained module fixture's identity package declares Version as a plain string literal", "Fixture identity is a literal" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| Fixtures and examples consumed by tests or presented as current docume... | body of "Maintained fixtures and examples import only the current published schema line" |
| Artifacts kept for a retired line SHALL live under an explicitly marke... | body of "Maintained fixtures and examples import only the current published schema line" |
| A maintained module fixture's identity package SHALL declare `Version`... | body of "A maintained module fixture's identity package declares Version as a plain string literal" |
| A fixture SHALL load through the kernel's module loader with no interp... | body of "A maintained module fixture's identity package declares Version as a plain string literal" |
| A fixture SHALL NOT depend on a catalog mechanism the catalog has with... | body of "A fixture does not depend on a catalog mechanism the catalog has withdrawn" |
| When a catalog member or contract type a fixture exercises is removed ... | body of "A fixture does not depend on a catalog mechanism the catalog has withdrawn" |

Words changed or added: none.

Lengths (old, new): old 1009; new "Maintained fixtures and examples import only the current published schema line" 381; "A maintained module fixture's identity package declares Version as a plain string literal" 324; "A fixture does not depend on a catalog mechanism the catalog has withdrawn" 302.

### Ledger: test-fixture-lineage, Fixtures live on the testing domain and are published

Form: c2 (REMOVED, ADDED under new names). New requirements: "A test fixture module declares its path under the testing domain", "Each fixture tree carries an identity package and is published to GHCR by repository CI", "A consumer that names a fixture version pins every shared dependency as CUE resolves it".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| No fixture occupies the production namespace | "A test fixture module declares its path under the testing domain", "No fixture occupies the production namespace" | No (byte equal, checked) |
| A fixture passes the publish gates | "Each fixture tree carries an identity package and is published to GHCR by repository CI", "A fixture passes the publish gates" | No (byte equal, checked) |
| Consumers resolve fixtures without a local registry | "Each fixture tree carries an identity package and is published to GHCR by repository CI", "Consumers resolve fixtures without a local registry" | No (byte equal, checked) |
| A version bump is an identity edit | "A consumer that names a fixture version pins every shared dependency as CUE resolves it", "A version bump is an identity edit" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| A test fixture module SHALL declare a module path under `testing.opmod... | body of "A test fixture module declares its path under the testing domain" |
| Each fixture tree SHALL carry an `identity/` package as the single sou... | body of "Each fixture tree carries an identity package and is published to GHCR by repository CI" |
| Fixture trees SHALL be published to GHCR by repository CI through `opm... | body of "Each fixture tree carries an identity package and is published to GHCR by repository CI" |
| A consumer that names a fixture version SHALL pin every dependency it ... | body of "A consumer that names a fixture version pins every shared dependency as CUE resolves it" |
| CUE keeps a dependency the consumer already lists at its listed versio... | body of "A consumer that names a fixture version pins every shared dependency as CUE resolves it" |
| The rule is mechanical, not stylistic: CUE resolves modules by longest... | body of "A test fixture module declares its path under the testing domain" |

Words changed or added: none. Order: "The rule is mechanical, not stylistic: ..." now follows the first sentence it explains.

Lengths (old, new): old 1246; new "A test fixture module declares its path under the testing domain" 388; "Each fixture tree carries an identity package and is published to GHCR by repository CI" 421; "A consumer that names a fixture version pins every shared dependency as CUE resolves it" 435.

### Ledger: test-fixture-lineage, Consumer pins are checked against CUE's resolver

Form: c2 (REMOVED, ADDED under new names). New requirements: "`hack/fixtures.sh consumers` checks each named consumer in a scratch copy against CUE's resolution", "`hack/fixtures.sh consumers` checks every named consumer, prints a FAIL line for each failure and exits non-zero", "A tracked cue.mod/module.cue that pins a fixture and is not named fails the check", "With FIX=1 `hack/fixtures.sh consumers` writes the resolved pins and does not fail", "`hack/fixtures.sh` stays byte-identical with opm-operator's copy".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Consumers that follow their fixture pass | "`hack/fixtures.sh consumers` checks each named consumer in a scratch copy against CUE's resolution", "Consumers that follow their fixture pass" | No (byte equal, checked) |
| Stale shared pin is reported with the diff | "`hack/fixtures.sh consumers` checks each named consumer in a scratch copy against CUE's resolution", "Stale shared pin is reported with the diff" | No (byte equal, checked) |
| Unresolvable fixture version is a named failure | "`hack/fixtures.sh consumers` checks every named consumer, prints a FAIL line for each failure and exits non-zero", "Unresolvable fixture version is a named failure" | No (byte equal, checked) |
| Unlisted consumer | "A tracked cue.mod/module.cue that pins a fixture and is not named fails the check", "Unlisted consumer" | No (byte equal, checked) |
| Fix mode writes the resolved pins | "With FIX=1 `hack/fixtures.sh consumers` writes the resolved pins and does not fail", "Fix mode writes the resolved pins" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The script is byte-identical with opm-operator's copy" in "`hack/fixtures.sh` stays byte-identical with opm-operator's copy"

| Old clause | New place |
| --- | --- |
| `hack/fixtures.sh consumers <dir>...` SHALL check each named consumer ... | body of "`hack/fixtures.sh consumers` checks each named consumer in a scratch copy against CUE's resolution" |
| It SHALL resolve through `CUE_REGISTRY`, defaulting to GHCR. | body of "`hack/fixtures.sh consumers` checks each named consumer in a scratch copy against CUE's resolution" |
| It SHALL check every named consumer before it exits, SHALL print a `FA... | body of "`hack/fixtures.sh consumers` checks every named consumer, prints a FAIL line for each failure and exits non-zero" |
| A git-tracked `cue.mod/module.cue` outside the fixtures directory that... | body of "A tracked cue.mod/module.cue that pins a fixture and is not named fails the check" |
| The scratch copies SHALL be removed on every exit path. | body of "`hack/fixtures.sh consumers` checks every named consumer, prints a FAIL line for each failure and exits non-zero" |
| With `FIX=1` a differing consumer's `module.cue` SHALL be overwritten ... | body of "With FIX=1 `hack/fixtures.sh consumers` writes the resolved pins and does not fail" |
| The script SHALL stay byte-identical with opm-operator's copy; each re... | body of "`hack/fixtures.sh` stays byte-identical with opm-operator's copy" |

Words changed or added:
- "It SHALL resolve" and "It SHALL check" read "`hack/fixtures.sh consumers <dir>...` SHALL resolve" and "`hack/fixtures.sh consumers <dir>...` SHALL check", the subject of the first sentence
- "The script SHALL stay" reads "`hack/fixtures.sh` SHALL stay"
- Three bodies end with "(Requirement: <header>)", the named reference that gives "also fail" and "With `FIX=1` a differing consumer" their referent: the unlisted-consumer body names the check-and-report requirement, the `FIX=1` body names the first requirement, and the check-and-report body names the `FIX=1` requirement, whose exemption ("SHALL NOT fail") limits its "exits non-zero"
- Order: "The scratch copies SHALL be removed on every exit path." now follows the sentence on the exit, because the body that reports and exits holds it

Lengths (old, new): old 998; new "`hack/fixtures.sh consumers` checks each named consumer in a scratch copy against CUE's resolution" 357; "`hack/fixtures.sh consumers` checks every named consumer, prints a FAIL line for each failure and exits non-zero" 453; "A tracked cue.mod/module.cue that pins a fixture and is not named fails the check" 272; "With FIX=1 `hack/fixtures.sh consumers` writes the resolved pins and does not fail" 225; "`hack/fixtures.sh` stays byte-identical with opm-operator's copy" 105.

### Ledger: test-fixture-lineage, Old test pins are current or frozen with a reason

Form: c2 (REMOVED, ADDED under new names). New requirements: "A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason", "The targets of a catalog pin and a core pin follow the release cascade", "A core pin already newer than the cascade target stays", "A `cue.mod` pin moves by `cue mod get` then `cue mod tidy` and is never bumped by a hand edit", "A `.cascade-frozen` entry names the file or directory, the frozen module paths and a one-sentence reason", "A pin is frozen only when the test depends on that exact old version", "A pin that is old only because a tree was copied or an upstream release moved on is not frozen".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Every old core or catalog pin in a test module is accounted for | "A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason", "Every old core or catalog pin in a test module is accounted for" | No (byte equal, checked) |
| A deliberately old pin carries its reason | "A `.cascade-frozen` entry names the file or directory, the frozen module paths and a one-sentence reason", "A deliberately old pin carries its reason" | No (byte equal, checked) |
| Bumped test trees still pass | "A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason", "Bumped test trees still pass" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The target of a catalog pin is the newest published release in its major" in "The targets of a catalog pin and a core pin follow the release cascade"
- "A core pin newer than its target stays" in "A core pin already newer than the cascade target stays"
- "A cue.mod pin moves by cue mod get then cue mod tidy" in "A `cue.mod` pin moves by `cue mod get` then `cue mod tidy` and is never bumped by a hand edit"
- "A golden may be listed with a reason" in "A `.cascade-frozen` entry names the file or directory, the frozen module paths and a one-sentence reason"
- "A frozen pin is one the test needs at that old version" in "A pin is frozen only when the test depends on that exact old version"
- "A pin that is old only because a tree was copied is not frozen" in "A pin that is old only because a tree was copied or an upstream release moved on is not frozen"

| Old clause | New place |
| --- | --- |
| Each `opmodel.dev/core@v2` or `opmodel.dev/catalogs/*` pin in a test i... | body of "A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason" |
| - **The target.** A catalog pin's target is the newest published relea... | body of "The targets of a catalog pin and a core pin follow the release cascade" |
| A core pin's target is the core version that the input's catalog pins,... | body of "The targets of a catalog pin and a core pin follow the release cascade" |
| The source is workspace RELEASING.md, section "The cascade", "Consiste... | body of "The targets of a catalog pin and a core pin follow the release cascade" |
| - **No backwards moves.** A core pin already newer than that target st... | body of "A core pin already newer than the cascade target stays" |
| - **How a pin moves.** A `cue.mod` file moves by `cue mod get` then `c... | body of "A `cue.mod` pin moves by `cue mod get` then `cue mod tidy` and is never bumped by a hand edit" |
| A Go literal moves by editing the literal. | body of "A `cue.mod` pin moves by `cue mod get` then `cue mod tidy` and is never bumped by a hand edit" |
| A `cue.mod` file SHALL NOT be bumped by a hand edit. | body of "A `cue.mod` pin moves by `cue mod get` then `cue mod tidy` and is never bumped by a hand edit" |
| - **Test inputs** include a test module's `cue.mod/module.cue` and a m... | body of "A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason" |
| - **`.cascade-frozen` entries.** Each entry names the repo-relative fi... | body of "A `.cascade-frozen` entry names the file or directory, the frozen module paths and a one-sentence reason" |
| The file uses the format in workspace RELEASING.md, section "Cascade f... | body of "A `.cascade-frozen` entry names the file or directory, the frozen module paths and a one-sentence reason" |
| A pin is frozen only when the test depends on that exact old version. | body of "A pin is frozen only when the test depends on that exact old version" |
| Examples: | body of "A pin is frozen only when the test depends on that exact old version" |
| - a platform that must be refused as too old; | body of "A pin is frozen only when the test depends on that exact old version" |
| - a platform that must show catalog version skew; | body of "A pin is frozen only when the test depends on that exact old version" |
| - a floor that the test re-pins up to. | body of "A pin is frozen only when the test depends on that exact old version" |
| A pin that is old only because a tree was copied from an older tree, o... | body of "A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason" |
| Between an upstream release and the release cascade that follows it, a... | body of "A pin that is old only because a tree was copied or an upstream release moved on is not frozen" |
| A literal that a test only compares as expected output and never resol... | body of "A `.cascade-frozen` entry names the file or directory, the frozen module paths and a one-sentence reason" |

Words changed or added:
- Keyword added where the sentence had none (strength table): "A catalog pin's target is" reads "SHALL be"; "stays" reads "SHALL stay"; "Each entry names" reads "Each entry SHALL name"; "A pin is frozen only when" reads "A pin SHALL be frozen only when"; "is not frozen" reads "SHALL NOT be frozen"
- The first body ends with "(Requirement: <header of the never-backwards requirement>)(Requirement: <header of the copied-tree requirement>)", the named references to the two limits of its rule; every other body but the `.cascade-frozen` entries body ends with "(Requirement: <header>)", the named reference that gives "that target", "a pin" and "frozen" their referent
- The bullet markers and the bold labels of the old body stay on the lines they head; the "Test inputs" bullet sits with the rule it defines

Lengths (old, new): old 1846; new "A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason" 500; "The targets of a catalog pin and a core pin follow the release cascade" 446; "A core pin already newer than the cascade target stays" 165; "A `cue.mod` pin moves by `cue mod get` then `cue mod tidy` and is never bumped by a hand edit" 350; "A `.cascade-frozen` entry names the file or directory, the frozen module paths and a one-sentence reason" 449; "A pin is frozen only when the test depends on that exact old version" 335; "A pin that is old only because a tree was copied or an upstream release moved on is not frozen" 349.

### Ledger: test-fixture-lineage, The shared fixture flow publishes catalog fixtures

Form: c2 (REMOVED, ADDED under new names). New requirements: "`hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture", "A catalog fixture goes through `opm catalog publish` wherever a module fixture goes through `opm module publish`", "`hack/fixtures.sh consumers` does not report a `cue.mod` under the catalog root as an unlisted consumer", "`tests/fixtures/fixtures.go` reads a catalog fixture's coordinate from its identity package".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| No catalog root leaves the module flow unchanged | "`hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture", "No catalog root leaves the module flow unchanged" | No (byte equal, checked) |
| A catalog fixture publishes through the catalog pipeline | "A catalog fixture goes through `opm catalog publish` wherever a module fixture goes through `opm module publish`", "A catalog fixture publishes through the catalog pipeline" | No (byte equal, checked) |
| A changed catalog fixture must be bumped | "A catalog fixture goes through `opm catalog publish` wherever a module fixture goes through `opm module publish`", "A changed catalog fixture must be bumped" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "A cue.mod under the catalog root is not an unlisted consumer" in "`hack/fixtures.sh consumers` does not report a `cue.mod` under the catalog root as an unlisted consumer"
- "A catalog fixture's coordinate comes from its identity package" in "`tests/fixtures/fixtures.go` reads a catalog fixture's coordinate from its identity package"
- "The fixture files are byte-identical with opm-operator's copies" in "`tests/fixtures/fixtures.go` reads a catalog fixture's coordinate from its identity package"

| Old clause | New place |
| --- | --- |
| `hack/fixtures.sh` SHALL treat each directory under a catalog fixture ... | body of "`hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture" |
| root is `CATALOGS_DIR`, defaulting to the `catalogs/` sibling of the m... | body of "`hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture" |
| directory exists, and absent otherwise. | body of "`hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture" |
| A catalog fixture SHALL go through `opm catalog publish` | body of "A catalog fixture goes through `opm catalog publish` wherever a module fixture goes through `opm module publish`" |
| (and `opm catalog version set` for a pre-release tag) wherever a modul... | body of "A catalog fixture goes through `opm catalog publish` wherever a module fixture goes through `opm module publish`" |
| `opm module publish`, so `check` runs every catalog publish gate and e... | body of "A catalog fixture goes through `opm catalog publish` wherever a module fixture goes through `opm module publish`" |
| `seed` and `publish` push it, and `pins` lists it, catalogs before mod... | body of "A catalog fixture goes through `opm catalog publish` wherever a module fixture goes through `opm module publish`" |
| `consumers` SHALL NOT | body of "`hack/fixtures.sh consumers` does not report a `cue.mod` under the catalog root as an unlisted consumer" |
| report a `cue.mod` under the catalog root as an unlisted consumer. | body of "`hack/fixtures.sh consumers` does not report a `cue.mod` under the catalog root as an unlisted consumer" |
| With no catalog root the script | body of "`hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture" |
| SHALL behave as it does for module fixtures alone. | body of "`hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture" |
| `tests/fixtures/fixtures.go` SHALL read a catalog | body of "`tests/fixtures/fixtures.go` reads a catalog fixture's coordinate from its identity package" |
| fixture's coordinate from its identity package through `LoadCatalog` a... | body of "`tests/fixtures/fixtures.go` reads a catalog fixture's coordinate from its identity package" |
| Both files | body of "`tests/fixtures/fixtures.go` reads a catalog fixture's coordinate from its identity package" |
| SHALL stay byte-identical with opm-operator's copies. | body of "`tests/fixtures/fixtures.go` reads a catalog fixture's coordinate from its identity package" |

Words changed or added: four bodies end with "(Requirement: <header>)", the named reference that gives `check`, `consumers` and "Both files" their referent (the first requirement names `hack/fixtures.sh`).

Lengths (old, new): old 942; new "`hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture" 322; "A catalog fixture goes through `opm catalog publish` wherever a module fixture goes through `opm module publish`" 433; "`hack/fixtures.sh consumers` does not report a `cue.mod` under the catalog root as an unlisted consumer" 194; "`tests/fixtures/fixtures.go` reads a catalog fixture's coordinate from its identity package" 308.

### Ledger: cmd-structure, `opm module build` output format, split files and ordering

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm module build` accepts only `yaml` or `json` as its output format", "`opm module build --split` writes one file per resource named by kind and name", "`opm module build` emits resources in a deterministic order".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Unsupported output format | "`opm module build` accepts only `yaml` or `json` as its output format", "Unsupported output format" | No (byte equal, checked) |
| Split output names files by kind and name | "`opm module build --split` writes one file per resource named by kind and name", "Split output names files by kind and name" | No (byte equal, checked) |
| Split output disambiguates colliding names | "`opm module build --split` writes one file per resource named by kind and name", "Split output disambiguates colliding names" | No (byte equal, checked) |
| Output order is deterministic | "`opm module build` emits resources in a deterministic order", "Output order is deterministic" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The `opm module build` subcommand SHALL accept `--output`/`-o` with ex... | body of "`opm module build` accepts only `yaml` or `json` as its output format" |
| With `--split`, it SHALL write one file per resource into `--out-dir` ... | body of "`opm module build --split` writes one file per resource named by kind and name" |
| Resources SHALL be emitted in a deterministic order, by apply weight (... | body of "`opm module build` emits resources in a deterministic order" |

Words changed or added: "it SHALL write" reads "the `opm module build` subcommand SHALL write"; the ordering body ends with "(Requirement: <header of the output-format requirement>)", the named reference that gives "Resources" their command.

Lengths (old, new): old 666; new "`opm module build` accepts only `yaml` or `json` as its output format" 225; "`opm module build --split` writes one file per resource named by kind and name" 283; "`opm module build` emits resources in a deterministic order" 272.

### Ledger: cmd-structure, `opm instance build` accepts only instance packages

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm instance build` takes one positional argument naming an instance", "`opm instance build` acquires the instance package through the library kernel and does not synthesize an instance from a module", "`opm instance build` exits 2 when the package is a module".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Argument is an instance file | "`opm instance build` acquires the instance package through the library kernel and does not synthesize an instance from a module", "Argument is an instance file" | No (byte equal, checked) |
| Argument is an instance package directory | "`opm instance build` acquires the instance package through the library kernel and does not synthesize an instance from a module", "Argument is an instance package directory" | No (byte equal, checked) |
| Argument is a module directory | "`opm instance build` exits 2 when the package is a module", "Argument is a module directory" | No (byte equal, checked) |
| Argument does not exist | "`opm instance build` takes one positional argument naming an instance", "Argument does not exist" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The `opm instance build` subcommand SHALL take one positional argument... | body of "`opm instance build` takes one positional argument naming an instance" |
| The directory may be a standalone CUE module or a package inside anoth... | body of "`opm instance build` takes one positional argument naming an instance" |
| The subcommand SHALL acquire that package through the library kernel a... | body of "`opm instance build` acquires the instance package through the library kernel and does not synthesize an instance from a module" |
| What the package is decides the outcome, not whether the argument is a... | body of "`opm instance build` exits 2 when the package is a module" |

Words changed or added: "The subcommand SHALL acquire" and "the subcommand SHALL exit 2" read "The `opm instance build` subcommand" and "the `opm instance build` subcommand"; both bodies end with "(Requirement: <header of the first requirement>)", the named reference that gives "that package" and "the package" their referent.

Lengths (old, new): old 822; new "`opm instance build` takes one positional argument naming an instance" 359; "`opm instance build` acquires the instance package through the library kernel and does not synthesize an instance from a module" 340; "`opm instance build` exits 2 when the package is a module" 333.

### Ledger: cmd-structure, `opm module build` (alias `opm mod build`) accepts a module directory or a published module

Form: c2 (REMOVED, ADDED under new names). New requirements: "The `module` command group registers `opm module build` (alias `opm mod build`) with an optional positional argument", "`opm module build` classifies its argument as a local directory or a published module path before any registry access", "`opm module build` refuses a published module path that also names an existing local directory", "`opm module build --version` selects the published version and is refused with a local directory", "`opm module build` acquires a published module from the registry and renders it as it renders a local module".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default to current directory | "The `module` command group registers `opm module build` (alias `opm mod build`) with an optional positional argument", "Default to current directory" | No (byte equal, checked) |
| Explicit module directory | "The `module` command group registers `opm module build` (alias `opm mod build`) with an optional positional argument", "Explicit module directory" | No (byte equal, checked) |
| File argument rejected | "The `module` command group registers `opm module build` (alias `opm mod build`) with an optional positional argument", "File argument rejected" | No (byte equal, checked) |
| Published module without a version | "`opm module build` acquires a published module from the registry and renders it as it renders a local module", "Published module without a version" | No (byte equal, checked) |
| Published module with a pinned version | "`opm module build --version` selects the published version and is refused with a local directory", "Published module with a pinned version" | No (byte equal, checked) |
| Version flag with a local directory | "`opm module build --version` selects the published version and is refused with a local directory", "Version flag with a local directory" | No (byte equal, checked) |
| Dotted argument that is also a local directory | "`opm module build` refuses a published module path that also names an existing local directory", "Dotted argument that is also a local directory" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "A leading dot or an absolute path is a local directory" in "`opm module build` classifies its argument as a local directory or a published module path before any registry access"
- "A dotted first element is a published module path" in "`opm module build` classifies its argument as a local directory or a published module path before any registry access"
- "Any other argument is a local directory" in "`opm module build` classifies its argument as a local directory or a published module path before any registry access"

| Old clause | New place |
| --- | --- |
| The `module` command group SHALL register a `build` subcommand with an... | body of "The `module` command group registers `opm module build` (alias `opm mod build`) with an optional positional argument" |
| The argument SHALL be classified before any registry access: | body of "`opm module build` classifies its argument as a local directory or a published module path before any registry access" |
| 1. | body of "`opm module build` classifies its argument as a local directory or a published module path before any registry access" |
| `.`, a path starting with `./` or `../`, or an absolute path is a loca... | body of "`opm module build` classifies its argument as a local directory or a published module path before any registry access" |
| 2. | body of "`opm module build` classifies its argument as a local directory or a published module path before any registry access" |
| Otherwise, an argument whose first element contains a dot is a publish... | body of "`opm module build` classifies its argument as a local directory or a published module path before any registry access" |
| 3. | body of "`opm module build` classifies its argument as a local directory or a published module path before any registry access" |
| Anything else is a local directory. | body of "`opm module build` classifies its argument as a local directory or a published module path before any registry access" |
| An argument classified as a published module path that also names an e... | body of "`opm module build` refuses a published module path that also names an existing local directory" |
| A new `--version` flag (string, default empty) selects the published v... | body of "`opm module build --version` selects the published version and is refused with a local directory" |
| A published module SHALL be acquired from the registry and synthesized... | body of "`opm module build` acquires a published module from the registry and renders it as it renders a local module" |

Words changed or added: four bodies end with "(Requirement: <header>)", the named reference that gives "The argument", "An argument", "A new `--version` flag" and "A published module" their command (the first requirement names `build`; the other three name the first or the classification requirement). The alias `opm mod build` stays in the header of the first requirement.

Lengths (old, new): old 1131; new "The `module` command group registers `opm module build` (alias `opm mod build`) with an optional positional argument" 246; "`opm module build` classifies its argument as a local directory or a published module path before any registry access" 406; "`opm module build` refuses a published module path that also names an existing local directory" 314; "`opm module build --version` selects the published version and is refused with a local directory" 317; "`opm module build` acquires a published module from the registry and renders it as it renders a local module" 373.

### Ledger: cmd-structure, `opm module apply` (alias `opm mod apply`) accepts a module directory or a published module

Form: c2 (REMOVED, ADDED under new names). New requirements: "The `module` command group registers `opm module apply` (alias `opm mod apply`) with an optional positional argument and the `--version` flag", "`opm module apply` synthesizes a `#ModuleInstance` from the module and applies the rendered resources with the semantics of `opm instance apply`", "`opm module apply` completes version resolution before any cluster contact", "`opm module apply` records the module path and version of a published module and carries no local-render provenance annotation", "`opm module apply` warns once on standard error when it applies the values of `debugValues`", "`opm module build` prints no `debugValues` warning".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default to current directory | "`opm module apply` synthesizes a `#ModuleInstance` from the module and applies the rendered resources with the semantics of `opm instance apply`", "Default to current directory" | No (byte equal, checked) |
| Apply warns when it deploys debugValues | "`opm module apply` warns once on standard error when it applies the values of `debugValues`", "Apply warns when it deploys debugValues" | No (byte equal, checked) |
| No warning with values files | "`opm module apply` warns once on standard error when it applies the values of `debugValues`", "No warning with values files" | No (byte equal, checked) |
| Explicit module directory | "`opm module apply` synthesizes a `#ModuleInstance` from the module and applies the rendered resources with the semantics of `opm instance apply`", "Explicit module directory" | No (byte equal, checked) |
| File argument rejected | "The `module` command group registers `opm module apply` (alias `opm mod apply`) with an optional positional argument and the `--version` flag", "File argument rejected" | No (byte equal, checked) |
| Alias `mod apply` resolves to `module apply` | "The `module` command group registers `opm module apply` (alias `opm mod apply`) with an optional positional argument and the `--version` flag", "Alias `mod apply` resolves to `module apply`" | No (byte equal, checked) |
| Apply a published module | "`opm module apply` records the module path and version of a published module and carries no local-render provenance annotation", "Apply a published module" | No (byte equal, checked) |
| Resolution failure exits before cluster contact | "`opm module apply` completes version resolution before any cluster contact", "Resolution failure exits before cluster contact" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The module build prints no debugValues warning" in "`opm module build` prints no `debugValues` warning"

| Old clause | New place |
| --- | --- |
| The `module` command group SHALL register an `apply` subcommand with a... | body of "The `module` command group registers `opm module apply` (alias `opm mod apply`) with an optional positional argument and the `--version` flag" |
| The subcommand SHALL synthesize a `#ModuleInstance` from the module (r... | body of "`opm module apply` synthesizes a `#ModuleInstance` from the module and applies the rendered resources with the semantics of `opm instance apply`" |
| Version resolution SHALL complete before any cluster contact. | body of "`opm module apply` completes version resolution before any cluster contact" |
| An apply from a published module SHALL record the module's declared pa... | body of "`opm module apply` records the module path and version of a published module and carries no local-render provenance annotation" |
| When the values come from the module's `debugValues` (no `-f`/`--value... | body of "`opm module apply` warns once on standard error when it applies the values of `debugValues`" |
| `opm module build` SHALL NOT print this warning, because it changes no... | body of "`opm module build` prints no `debugValues` warning" |

Words changed or added:
- "The subcommand SHALL synthesize" and "the subcommand SHALL print" read "The `opm module apply` subcommand" and "the `opm module apply` subcommand"
- Four bodies end with "(Requirement: <header of the first requirement>)", the named reference that gives them the command
- "`opm module build` SHALL NOT print this warning, because it changes nothing." is its own requirement, which ends with "(Requirement: <header of the warning requirement>)", the referent of "this warning"

Lengths (old, new): old 1256; new "The `module` command group registers `opm module apply` (alias `opm mod apply`) with an optional positional argument and the `--version` flag" 218; "`opm module apply` synthesizes a `#ModuleInstance` from the module and applies the rendered resources with the semantics of `opm instance apply`" 485; "`opm module apply` completes version resolution before any cluster contact" 218; "`opm module apply` records the module path and version of a published module and carries no local-render provenance annotation" 392; "`opm module apply` warns once on standard error when it applies the values of `debugValues`" 371; "`opm module build` prints no `debugValues` warning" 183.

### Ledger: e2e-cluster-preconditions, Operator applier grant is verified before operator-owned tests

Form: c2 (REMOVED, ADDED under new names). New requirements: "Operator-owned e2e tests verify the operator's applier grant", "The effective applier identity is the ServiceAccount named by --default-service-account when that argument is present", "A denied applier check fails the test before any cluster mutation", "An applier check that could not be performed skips the test".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Operator installed without the dev grant | "A denied applier check fails the test before any cluster mutation", "Operator installed without the dev grant" | No (byte equal, checked) |
| Check cannot be performed | "An applier check that could not be performed skips the test", "Check cannot be performed" | No (byte equal, checked) |
| Cluster prepared by task cluster:operator | "Operator-owned e2e tests verify the operator's applier grant", "Cluster prepared by task cluster:operator" | No (byte equal, checked) |
| Lifecycle test is exempt | "Operator-owned e2e tests verify the operator's applier grant", "Lifecycle test is exempt" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The default ServiceAccount argument names the applier identity" in "The effective applier identity is the ServiceAccount named by --default-service-account when that argument is present"
- "Without the argument the controller ServiceAccount is the applier identity" in "The effective applier identity is the ServiceAccount named by --default-service-account when that argument is present"

| Old clause | New place |
| --- | --- |
| Every e2e test that relies on the operator applying workload resources... | body of "Operator-owned e2e tests verify the operator's applier grant" |
| The effective applier identity is the ServiceAccount named by the cont... | body of "The effective applier identity is the ServiceAccount named by --default-service-account when that argument is present" |
| If the check reports denial, the test SHALL fail before performing any... | body of "A denied applier check fails the test before any cluster mutation" |
| A denial SHALL be distinguished from a check that could not be perform... | body of "A denied applier check fails the test before any cluster mutation" |
| If the check itself cannot be performed (no cluster, kubectl failure, ... | body of "An applier check that could not be performed skips the test" |

Words changed or added:
- Keyword added where the sentence had none (strength table): "The effective applier identity is" reads "SHALL be"; "the test skips as it does today" reads "the test SHALL skip as it does today"
- The skip body names the requirement it cited, which was removed here: "(`A cluster-required run fails instead of skipping`)" reads "(`A cluster-required run fails every precondition that would skip`)", the header of the first requirement of that split
- Three bodies end with "(Requirement: <header of the first requirement>)", the named reference that gives "the effective applier identity", "the check" and "the identity probed" their referent

Lengths (old, new): old 1312; new "Operator-owned e2e tests verify the operator's applier grant" 306; "The effective applier identity is the ServiceAccount named by --default-service-account when that argument is present" 319; "A denied applier check fails the test before any cluster mutation" 444; "An applier check that could not be performed skips the test" 494.

### Ledger: e2e-cluster-preconditions, The e2e suite resolves modules through the shipped default registry

Form: c2 (REMOVED, ADDED under new names). New requirements: "The cluster-backed e2e suite resolves published modules through the registry mapping the CLI ships as its default", "The e2e suite's stub home configuration takes its registry from the shipped default by reference", "A test that needs to write to a registry provides one it owns for the duration of that test".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A test that chooses no registry uses the shipped default | "The cluster-backed e2e suite resolves published modules through the registry mapping the CLI ships as its default", "A test that chooses no registry uses the shipped default" | No (byte equal, checked) |
| The operator lifecycle test seeds a Platform | "The cluster-backed e2e suite resolves published modules through the registry mapping the CLI ships as its default", "The operator lifecycle test seeds a Platform" | No (byte equal, checked) |
| A publishing test owns its registry | "A test that needs to write to a registry provides one it owns for the duration of that test", "A publishing test owns its registry" | No (byte equal, checked) |
| No unreachable registry address is named | "The e2e suite's stub home configuration takes its registry from the shipped default by reference", "No unreachable registry address is named" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The cluster-backed e2e suite SHALL resolve published modules through t... | body of "The cluster-backed e2e suite resolves published modules through the registry mapping the CLI ships as its default" |
| ships as its default, and SHALL NOT depend on a registry process that ... | body of "The cluster-backed e2e suite resolves published modules through the registry mapping the CLI ships as its default" |
| The suite's stub home configuration SHALL take its registry from that ... | body of "The e2e suite's stub home configuration takes its registry from the shipped default by reference" |
| rather than restating an address, so that an invocation which does not... | body of "The e2e suite's stub home configuration takes its registry from the shipped default by reference" |
| resolves exactly as an end user's would and the suite cannot drift fro... | body of "The e2e suite's stub home configuration takes its registry from the shipped default by reference" |
| A test that needs to write to a registry , publishing a module, or sca... | body of "A test that needs to write to a registry provides one it owns for the duration of that test" |
| publishes first , SHALL provide one it owns for the duration of that t... | body of "A test that needs to write to a registry provides one it owns for the duration of that test" |
| it writes to at that registry and leaving `opmodel.dev` resolving from... | body of "A test that needs to write to a registry provides one it owns for the duration of that test" |
| Such a | body of "A test that needs to write to a registry provides one it owns for the duration of that test" |
| registry SHALL require no external process, port reservation or contai... | body of "A test that needs to write to a registry provides one it owns for the duration of that test" |
| when the test ends. | body of "A test that needs to write to a registry provides one it owns for the duration of that test" |
| A registry that a test does not itself provide SHALL NOT be a precondi... | body of "The cluster-backed e2e suite resolves published modules through the registry mapping the CLI ships as its default" |
| require a developer to start one by hand. | body of "The cluster-backed e2e suite resolves published modules through the registry mapping the CLI ships as its default" |

Words changed or added: the stub-configuration body ends with "(Requirement: <header of the first requirement>)", the named reference that gives "that shipped default" its referent; the em-dashes of the second paragraph read as commas ("A test that needs to write to a registry, publishing a module, or scaffolding from a template it publishes first, SHALL provide"). The sentence order is the old order except that the last paragraph follows the first.

Lengths (old, new): old 1044; new "The cluster-backed e2e suite resolves published modules through the registry mapping the CLI ships as its default" 339; "The e2e suite's stub home configuration takes its registry from the shipped default by reference" 419; "A test that needs to write to a registry provides one it owns for the duration of that test" 411.

### Ledger: e2e-cluster-preconditions, A destructive test that cannot restore the cluster fails its own run

Form: c2 (REMOVED, ADDED under new names). New requirements: "A destructive test restores the dev cluster's operator before the run ends and fails the run when it cannot", "The failure of a destructive test names the restore step, the underlying error and the repair command".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Restore fails | "A destructive test restores the dev cluster's operator before the run ends and fails the run when it cannot", "Restore fails" | No (byte equal, checked) |
| Restore succeeds | "A destructive test restores the dev cluster's operator before the run ends and fails the run when it cannot", "Restore succeeds" | No (byte equal, checked) |
| A poisoned cluster is never reported as success | "A destructive test restores the dev cluster's operator before the run ends and fails the run when it cannot", "A poisoned cluster is never reported as success" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The failure message names the restore step, the error and the repair command" in "The failure of a destructive test names the restore step, the underlying error and the repair command"

| Old clause | New place |
| --- | --- |
| An e2e test that tears down the shared dev cluster's operator SHALL re... | body of "A destructive test restores the dev cluster's operator before the run ends and fails the run when it cannot" |
| and SHALL fail the run when it cannot. | body of "A destructive test restores the dev cluster's operator before the run ends and fails the run when it cannot" |
| Reporting the failure without failing SHALL NOT satisfy this | body of "A destructive test restores the dev cluster's operator before the run ends and fails the run when it cannot" |
| requirement: a run that leaves the cluster unable to serve the operato... | body of "A destructive test restores the dev cluster's operator before the run ends and fails the run when it cannot" |
| successfully, because the next run's failure would otherwise appear as... | body of "A destructive test restores the dev cluster's operator before the run ends and fails the run when it cannot" |
| far from its cause. | body of "A destructive test restores the dev cluster's operator before the run ends and fails the run when it cannot" |
| The failure message SHALL name the restore step that did not complete,... | body of "The failure of a destructive test names the restore step, the underlying error and the repair command" |
| the command a developer runs to repair the cluster by hand. | body of "The failure of a destructive test names the restore step, the underlying error and the repair command" |

Words changed or added: the failure-message body ends with "(Requirement: <header of the first requirement>)", the named reference that gives "The failure message" its referent.

Lengths (old, new): old 572; new "A destructive test restores the dev cluster's operator before the run ends and fails the run when it cannot" 415; "The failure of a destructive test names the restore step, the underlying error and the repair command" 279.

### Ledger: e2e-cluster-preconditions, A cluster-required run fails instead of skipping

Form: c2 (REMOVED, ADDED under new names). New requirements: "A cluster-required run fails every precondition that would skip", "The failure of a cluster-required run starts with OPM_E2E_REQUIRE_CLUSTER=1: and names the cause", "Precondition messages read correctly as either a skip or a failure", "OPM_E2E_REQUIRE_CLUSTER changes no test that does not use the cluster and takes precedence over the skip clause of the grant check".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Required cluster has no kubeconfig | "The failure of a cluster-required run starts with OPM_E2E_REQUIRE_CLUSTER=1: and names the cause", "Required cluster has no kubeconfig" | No (byte equal, checked) |
| Required cluster unreachable | "The failure of a cluster-required run starts with OPM_E2E_REQUIRE_CLUSTER=1: and names the cause", "Required cluster unreachable" | No (byte equal, checked) |
| Developer machine without a cluster | "A cluster-required run fails every precondition that would skip", "Developer machine without a cluster" | No (byte equal, checked) |
| Grant check cannot be performed in a required run | "OPM_E2E_REQUIRE_CLUSTER changes no test that does not use the cluster and takes precedence over the skip clause of the grant check", "Grant check cannot be performed in a required run" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "No precondition message says skipping" in "Precondition messages read correctly as either a skip or a failure"
- "A precondition that already fails keeps its outcome" in "OPM_E2E_REQUIRE_CLUSTER changes no test that does not use the cluster and takes precedence over the skip clause of the grant check"
- "A test that does not use the cluster is unchanged" in "OPM_E2E_REQUIRE_CLUSTER changes no test that does not use the cluster and takes precedence over the skip clause of the grant check"

| Old clause | New place |
| --- | --- |
| When the environment variable `OPM_E2E_REQUIRE_CLUSTER` is set to `1`,... | body of "A cluster-required run fails every precondition that would skip" |
| This covers a missing kubeconfig, an unreachable `kind-opm-dev` contex... | body of "A cluster-required run fails every precondition that would skip" |
| The failure message SHALL start with `OPM_E2E_REQUIRE_CLUSTER=1:` and ... | body of "The failure of a cluster-required run starts with OPM_E2E_REQUIRE_CLUSTER=1: and names the cause" |
| Precondition messages SHALL read correctly as either a skip or a failu... | body of "Precondition messages read correctly as either a skip or a failure" |
| When the variable is unset or has any other value, those preconditions... | body of "A cluster-required run fails every precondition that would skip" |
| The variable SHALL NOT change any test that does not use the cluster, ... | body of "OPM_E2E_REQUIRE_CLUSTER changes no test that does not use the cluster and takes precedence over the skip clause of the grant check" |
| When the variable is `1`, this requirement takes precedence over the s... | UNMAPPED |

Words changed or added:
- "The variable SHALL NOT" reads "The environment variable `OPM_E2E_REQUIRE_CLUSTER` SHALL NOT", the subject of the first sentence
- "this requirement takes precedence over the skip clause of `Operator applier grant is verified before operator-owned tests`" reads "the requirement `A cluster-required run fails every precondition that would skip` takes precedence over the skip clause of `An applier check that could not be performed skips the test`": both names are the new headers of the two requirements those words pointed at
- Three bodies end with "(Requirement: <header of the first requirement>)", the named reference that gives "The failure message" and "Precondition messages" their referent
- The first body keeps "as they do without this requirement" unchanged; there "this requirement" is the first requirement of the split

Lengths (old, new): old 1221; new "A cluster-required run fails every precondition that would skip" 446; "The failure of a cluster-required run starts with OPM_E2E_REQUIRE_CLUSTER=1: and names the cause" 388; "Precondition messages read correctly as either a skip or a failure" 185; "OPM_E2E_REQUIRE_CLUSTER changes no test that does not use the cluster and takes precedence over the skip clause of the grant check" 457.

### Ledger: inst-commands, instance cluster-query commands accept an instance identifier as positional argument

Form: c2 (REMOVED, ADDED under new names). New requirements: "Instance cluster-query commands take one instance identifier, detected as a path, a UUID or a name from its shape", "A path identifier is acquired through the library kernel and the `--namespace` flag overrides the namespace in the file", "A path identifier whose package is not a valid instance is refused with exit code 1".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| status by instance name | "Instance cluster-query commands take one instance identifier, detected as a path, a UUID or a name from its shape", "status by instance name" | No (byte equal, checked) |
| status by UUID | "Instance cluster-query commands take one instance identifier, detected as a path, a UUID or a name from its shape", "status by UUID" | No (byte equal, checked) |
| status by instance path | "A path identifier is acquired through the library kernel and the `--namespace` flag overrides the namespace in the file", "status by instance path" | No (byte equal, checked) |
| namespace flag overrides the file | "A path identifier is acquired through the library kernel and the `--namespace` flag overrides the namespace in the file", "namespace flag overrides the file" | No (byte equal, checked) |
| path that is not a valid instance is refused | "A path identifier whose package is not a valid instance is refused with exit code 1", "path that is not a valid instance is refused" | No (byte equal, checked) |
| delete by instance name | "Instance cluster-query commands take one instance identifier, detected as a path, a UUID or a name from its shape", "delete by instance name" | No (byte equal, checked) |
| identifier auto-detection | "Instance cluster-query commands take one instance identifier, detected as a path, a UUID or a name from its shape", "identifier auto-detection" | No (byte equal, checked) |
| a bare word is a name even when a directory of that name exists | "Instance cluster-query commands take one instance identifier, detected as a path, a UUID or a name from its shape", "a bare word is a name even when a directory of that name exists" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The cluster-query commands (`status`, `tree`, `events`, `delete`) unde... | body of "Instance cluster-query commands take one instance identifier, detected as a path, a UUID or a name from its shape" |
| The CLI SHALL auto-detect whether the identifier is a path, a UUID or ... | body of "Instance cluster-query commands take one instance identifier, detected as a path, a UUID or a name from its shape" |
| A path (an `instance.cue` file, or a directory holding one) SHALL be a... | body of "A path identifier is acquired through the library kernel and the `--namespace` flag overrides the namespace in the file" |
| A path whose package is not a valid instance (wrong kind, no concrete ... | body of "A path identifier whose package is not a valid instance is refused with exit code 1" |

Words changed or added: none in the sentences; the second and third bodies end with "(Requirement: <header of the first requirement>)", the named reference that gives "A path" its cluster-query commands.

Lengths (old, new): old 840; new "Instance cluster-query commands take one instance identifier, detected as a path, a UUID or a name from its shape" 315; "A path identifier is acquired through the library kernel and the `--namespace` flag overrides the namespace in the file" 463; "A path identifier whose package is not a valid instance is refused with exit code 1" 318.

### Ledger: inst-commands, instance render commands accept --platform flag

Form: c2 (REMOVED, ADDED under new names). New requirements: "Instance render commands accept a `--platform <dir>` flag that names the highest-precedence platform source", "The platform is not specified in the instance file and there is no `--provider` flag".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| platform flag overrides the resolved platform source | "Instance render commands accept a `--platform <dir>` flag that names the highest-precedence platform source", "platform flag overrides the resolved platform source" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "There is no `--provider` flag on the render commands" in "The platform is not specified in the instance file and there is no `--provider` flag"

| Old clause | New place |
| --- | --- |
| Render commands (`vet`, `build`, `apply`, `diff`) SHALL accept a `--pl... | body of "Instance render commands accept a `--platform <dir>` flag that names the highest-precedence platform source" |
| The flag help SHALL say it overrides the cluster Platform and the inst... | body of "Instance render commands accept a `--platform <dir>` flag that names the highest-precedence platform source" |
| The platform SHALL NOT be specified in the instance file, and there SH... | body of "The platform is not specified in the instance file and there is no `--provider` flag" |

Words changed or added: none in the bodies; the second body ends with "(Requirement: <header of the first requirement>)", the named reference that gives "The platform" its render commands.

Lengths (old, new): old 522; new "Instance render commands accept a `--platform <dir>` flag that names the highest-precedence platform source" 372; "The platform is not specified in the instance file and there is no `--provider` flag" 272.

### Ledger: inst-commands, instance render commands refuse a namespace override that disagrees with the instance file

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`", "The namespace refusal names the source of the override, the override value, the instance's namespace and the instance argument", "An override equal to the instance's `metadata.namespace` is accepted and a configured or default namespace is not an override", "The `-n` help text of `opm instance apply`, `build`, `diff` and `vet` states the constraint on the value", "The module commands are unaffected by the namespace refusal of the instance render commands".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Flag equal to the file's namespace | "An override equal to the instance's `metadata.namespace` is accepted and a configured or default namespace is not an override", "Flag equal to the file's namespace" | No (byte equal, checked) |
| Flag differs from the file's namespace | "`opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`", "Flag differs from the file's namespace" | No (byte equal, checked) |
| Environment variable differs from the file's namespace | "The namespace refusal names the source of the override, the override value, the instance's namespace and the instance argument", "Environment variable differs from the file's namespace" | No (byte equal, checked) |
| Flag matches while OPM_NAMESPACE differs | "An override equal to the instance's `metadata.namespace` is accepted and a configured or default namespace is not an override", "Flag matches while OPM_NAMESPACE differs" | No (byte equal, checked) |
| No override | "An override equal to the instance's `metadata.namespace` is accepted and a configured or default namespace is not an override", "No override" | No (byte equal, checked) |
| Module path still honours the override | "The module commands are unaffected by the namespace refusal of the instance render commands", "Module path still honours the override" | No (byte equal, checked) |
| Help text states the constraint | "The `-n` help text of `opm instance apply`, `build`, `diff` and `vet` states the constraint on the value", "Help text states the constraint" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The instance file owns its namespace: `metadata.namespace` is part of ... | body of "`opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`" |
| When `opm instance apply`, `build`, `diff` or `vet` resolves its names... | body of "`opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`" |
| The refusal SHALL name the source of the override (`--namespace` or `O... | body of "The namespace refusal names the source of the override, the override value, the instance's namespace and the instance argument" |
| An override equal to `metadata.namespace` SHALL be accepted. | body of "An override equal to the instance's `metadata.namespace` is accepted and a configured or default namespace is not an override" |
| A namespace that comes from `kubernetes.namespace` in `~/.opm/config.c... | body of "An override equal to the instance's `metadata.namespace` is accepted and a configured or default namespace is not an override" |
| The `-n` help text of these four commands SHALL state that the value m... | body of "The `-n` help text of `opm instance apply`, `build`, `diff` and `vet` states the constraint on the value" |
| The module commands (`opm module build`, `apply`, `vet`) SHALL be unaf... | body of "The module commands are unaffected by the namespace refusal of the instance render commands" |

Words changed or added: four bodies end with "(Requirement: <header of the first requirement>)", the named reference that gives "The refusal", "An override", "these four commands" and "the override" their referent; "The module commands ... are unaffected" reads "SHALL be unaffected" (strength table).

Lengths (old, new): old 1273; new "`opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`" 435; "The namespace refusal names the source of the override, the override value, the instance's namespace and the instance argument" 427; "An override equal to the instance's `metadata.namespace` is accepted and a configured or default namespace is not an override" 400; "The `-n` help text of `opm instance apply`, `build`, `diff` and `vet` states the constraint on the value" 266; "The module commands are unaffected by the namespace refusal of the instance render commands" 326.

### Ledger: inst-commands, Instance diff fails when a rendered resource could not be read

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm instance diff` treats a rendered resource it could not read or compare as a failure", "`opm instance diff` exits non-zero and prints no `No differences found` when a failure exists", "`opm instance diff` keeps NotFound as a new resource, behaves as before with no failure and warns on an unreadable tracked resource".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Every rendered resource is unreadable | "`opm instance diff` exits non-zero and prints no `No differences found` when a failure exists", "Every rendered resource is unreadable" | No (byte equal, checked) |
| One unreadable resource beside a modified one | "`opm instance diff` treats a rendered resource it could not read or compare as a failure", "One unreadable resource beside a modified one" | No (byte equal, checked) |
| Failures of two classes | "`opm instance diff` exits non-zero and prints no `No differences found` when a failure exists", "Failures of two classes" | No (byte equal, checked) |
| Unreadable tracked resource that is not rendered | "`opm instance diff` keeps NotFound as a new resource, behaves as before with no failure and warns on an unreadable tracked resource", "Unreadable tracked resource that is not rendered" | No (byte equal, checked) |
| Nothing failed and nothing differs | "`opm instance diff` keeps NotFound as a new resource, behaves as before with no failure and warns on an unreadable tracked resource", "Nothing failed and nothing differs" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm instance diff` SHALL treat a rendered resource whose live read fa... | body of "`opm instance diff` treats a rendered resource it could not read or compare as a failure" |
| For each failure the command SHALL print an error that names the resou... | body of "`opm instance diff` treats a rendered resource it could not read or compare as a failure" |
| The command SHALL still compare the resources it can read and SHALL pr... | body of "`opm instance diff` treats a rendered resource it could not read or compare as a failure" |
| When at least one failure exists, the `opm instance diff` command SHAL... | body of "`opm instance diff` exits non-zero and prints no `No differences found` when a failure exists" |
| A NotFound answer for a rendered resource SHALL still mean a new resou... | body of "`opm instance diff` keeps NotFound as a new resource, behaves as before with no failure and warns on an unreadable tracked resource" |
| With no failure the `opm instance diff` command SHALL behave as before... | body of "`opm instance diff` keeps NotFound as a new resource, behaves as before with no failure and warns on an unreadable tracked resource" |
| A tracked resource that orphan detection could not read stays a warnin... | body of "`opm instance diff` keeps NotFound as a new resource, behaves as before with no failure and warns on an unreadable tracked resource" |

Words changed or added: "the command SHALL NOT print" reads "the `opm instance diff` command SHALL NOT print" and "the command SHALL behave as before" reads "the `opm instance diff` command SHALL behave as before" (subject of the old first sentence); the second body ends with "(Requirement: <header of the first requirement>)", the referent of "failure".

Lengths (old, new): old 974; new "`opm instance diff` treats a rendered resource it could not read or compare as a failure" 388; "`opm instance diff` exits non-zero and prints no `No differences found` when a failure exists" 474; "`opm instance diff` keeps NotFound as a new resource, behaves as before with no failure and warns on an unreadable tracked resource" 254.

### Ledger: kind-cluster-tasks, Cluster creation task

Form: c2 (REMOVED, ADDED under new names). New requirements: "`cluster:create` provisions a single-node kind cluster named by `CLUSTER_NAME` from `hack/kind-config.yaml`", "`cluster:create` selects the node image from the `K8S_VERSION` Taskfile variable".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Create cluster with defaults | "`cluster:create` provisions a single-node kind cluster named by `CLUSTER_NAME` from `hack/kind-config.yaml`", "Create cluster with defaults" | No (byte equal, checked) |
| Create cluster with custom Kubernetes version | "`cluster:create` selects the node image from the `K8S_VERSION` Taskfile variable", "Create cluster with custom Kubernetes version" | No (byte equal, checked) |
| Create cluster with custom name | "`cluster:create` provisions a single-node kind cluster named by `CLUSTER_NAME` from `hack/kind-config.yaml`", "Create cluster with custom name" | No (byte equal, checked) |
| Kind not installed | "`cluster:create` provisions a single-node kind cluster named by `CLUSTER_NAME` from `hack/kind-config.yaml`", "Kind not installed" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The Taskfile SHALL provide a `cluster:create` task that provisions a s... | body of "`cluster:create` provisions a single-node kind cluster named by `CLUSTER_NAME` from `hack/kind-config.yaml`" |
| The task SHALL use the `CLUSTER_NAME` Taskfile variable (default: `opm... | body of "`cluster:create` provisions a single-node kind cluster named by `CLUSTER_NAME` from `hack/kind-config.yaml`" |
| The `cluster:create` task SHALL use the `K8S_VERSION` Taskfile variabl... | body of "`cluster:create` selects the node image from the `K8S_VERSION` Taskfile variable" |
| Developers MAY override this to test against a different Kubernetes ve... | body of "`cluster:create` selects the node image from the `K8S_VERSION` Taskfile variable" |

Words changed or added: "The task SHALL use the `K8S_VERSION`" reads "The `cluster:create` task SHALL use the `K8S_VERSION`" (subject of the old first sentence).

Lengths (old, new): old 535; new "`cluster:create` provisions a single-node kind cluster named by `CLUSTER_NAME` from `hack/kind-config.yaml`" 313; "`cluster:create` selects the node image from the `K8S_VERSION` Taskfile variable" 238.

### Ledger: kind-cluster-tasks, Operator install task needs no local registry

Form: c2 (REMOVED, ADDED under new names). New requirements: "The `cluster:operator` task installs the pinned operator without requiring a local registry container", "The `cluster:operator` task remains idempotent and applies the cluster `Platform` singleton and the dev-only RBAC grant", "The `cluster:operator` task confirms the operator can serve the cluster by waiting for `Ready` of the Platform's current generation", "A `Ready=True` observed for an earlier generation does not satisfy the wait of `cluster:operator`", "The `cluster:operator` task fails when `Ready` is not `True` for the current generation and prints the Platform's state".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Install with no registry container running | "The `cluster:operator` task installs the pinned operator without requiring a local registry container", "Install with no registry container running" | No (byte equal, checked) |
| Re-run is a no-op | "The `cluster:operator` task remains idempotent and applies the cluster `Platform` singleton and the dev-only RBAC grant", "Re-run is a no-op" | No (byte equal, checked) |
| Operator that cannot build the Platform | "The `cluster:operator` task fails when `Ready` is not `True` for the current generation and prints the Platform's state", "Operator that cannot build the Platform" | No (byte equal, checked) |
| Ready left over from an earlier generation | "A `Ready=True` observed for an earlier generation does not satisfy the wait of `cluster:operator`", "Ready left over from an earlier generation" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "Ready for the current generation confirms the operator can serve the cluster" in "The `cluster:operator` task confirms the operator can serve the cluster by waiting for `Ready` of the Platform's current generation"

| Old clause | New place |
| --- | --- |
| The `cluster:operator` task SHALL install the pinned operator into the... | body of "The `cluster:operator` task installs the pinned operator without requiring a local registry container" |
| Everything the flow resolves: core, the catalogs, and this repository'... | body of "The `cluster:operator` task installs the pinned operator without requiring a local registry container" |
| The `cluster:operator` task SHALL remain idempotent, SHALL apply the c... | body of "The `cluster:operator` task remains idempotent and applies the cluster `Platform` singleton and the dev-only RBAC grant" |
| The `cluster:operator` task SHALL confirm the operator can actually se... | body of "The `cluster:operator` task confirms the operator can serve the cluster by waiting for `Ready` of the Platform's current generation" |
| A `Ready=True` observed for an earlier generation (the Platform the in... | body of "A `Ready=True` observed for an earlier generation does not satisfy the wait of `cluster:operator`" |
| If `Ready` is not `True` for the current generation within the task's ... | body of "The `cluster:operator` task fails when `Ready` is not `True` for the current generation and prints the Platform's state" |

Words changed or added: the em-dashes of "Everything the flow resolves" read ":" and ","; the old second paragraph is cut at the list boundary after "the dev-only RBAC grant" into two bodies, each with the subject "The `cluster:operator` task" (copied from the first sentence), and the joining "and" falls at the cut; the two bodies about the wait end with "(Requirement: <header of the confirm requirement>)", the referent of "the wait".

Lengths (old, new): old 1312; new "The `cluster:operator` task installs the pinned operator without requiring a local registry container" 372; "The `cluster:operator` task remains idempotent and applies the cluster `Platform` singleton and the dev-only RBAC grant" 126; "The `cluster:operator` task confirms the operator can serve the cluster by waiting for `Ready` of the Platform's current generation" 399; "A `Ready=True` observed for an earlier generation does not satisfy the wait of `cluster:operator`" 299; "The `cluster:operator` task fails when `Ready` is not `True` for the current generation and prints the Platform's state" 447.

### Ledger: kind-cluster-tasks, Local registry is an explicit opt-in for the kind flow

Form: c2 (REMOVED, ADDED under new names). New requirements: "A local registry for the kind flow requires setting `KIND_CUE_REGISTRY` explicitly", "When `KIND_CUE_REGISTRY` is set the `cluster:operator` task verifies, joins and passes the registry mapping as a values file", "The operator's registry is a command-line flag rendered from the registry value, not an environment variable".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Opt-in with the registry running | "When `KIND_CUE_REGISTRY` is set the `cluster:operator` task verifies, joins and passes the registry mapping as a values file", "Opt-in with the registry running" | No (byte equal, checked) |
| Opt-in without the registry running | "When `KIND_CUE_REGISTRY` is set the `cluster:operator` task verifies, joins and passes the registry mapping as a values file", "Opt-in without the registry running" | No (byte equal, checked) |
| Default path reports its registry source | "A local registry for the kind flow requires setting `KIND_CUE_REGISTRY` explicitly", "Default path reports its registry source" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "A pod environment variable does not set the operator's registry" in "The operator's registry is a command-line flag rendered from the registry value, not an environment variable"

| Old clause | New place |
| --- | --- |
| Pointing the kind flow at a local registry SHALL require setting `KIND... | body of "A local registry for the kind flow requires setting `KIND_CUE_REGISTRY` explicitly" |
| That variable SHALL default to empty. | body of "A local registry for the kind flow requires setting `KIND_CUE_REGISTRY` explicitly" |
| When it is unset the `cluster:operator` task SHALL pass no registry va... | body of "A local registry for the kind flow requires setting `KIND_CUE_REGISTRY` explicitly" |
| The `cluster:operator` task SHALL NOT patch the operator Deployment. | body of "A local registry for the kind flow requires setting `KIND_CUE_REGISTRY` explicitly" |
| When `KIND_CUE_REGISTRY` is set, and only then, the `cluster:operator`... | body of "When `KIND_CUE_REGISTRY` is set the `cluster:operator` task verifies, joins and passes the registry mapping as a values file" |
| The operator's registry is a command-line flag the module renders from... | body of "The operator's registry is a command-line flag rendered from the registry value, not an environment variable" |
| Setting either as a pod environment variable SHALL have no effect. | body of "The operator's registry is a command-line flag rendered from the registry value, not an environment variable" |

Words changed or added: "the task" reads "the `cluster:operator` task" (the task the scenarios of this requirement name) and "it" reads `KIND_CUE_REGISTRY` in the second body; "Setting either as a pod environment variable has no effect" reads "SHALL have no effect" (strength table); the second body ends with "(Requirement: <header of the first requirement>)" and the third with "(Requirement: <header of the second requirement>)", the named references that give "the operator's registry value" and "that value" their referent.

Lengths (old, new): old 979; new "A local registry for the kind flow requires setting `KIND_CUE_REGISTRY` explicitly" 373; "When `KIND_CUE_REGISTRY` is set the `cluster:operator` task verifies, joins and passes the registry mapping as a values file" 448; "The operator's registry is a command-line flag rendered from the registry value, not an environment variable" 475.

### Ledger: kind-cluster-tasks, cluster:operator builds the CLI it installs with

Form: c2 (REMOVED, ADDED under new names). New requirements: "The `cluster:operator` task depends on the build task so the CLI binary it invokes exists and is current", "The `cluster:operator` task succeeds from a clean checkout and is callable unattended".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Clean tree with no binary | "The `cluster:operator` task depends on the build task so the CLI binary it invokes exists and is current", "Clean tree with no binary" | No (byte equal, checked) |
| Binary already current | "The `cluster:operator` task depends on the build task so the CLI binary it invokes exists and is current", "Binary already current" | No (byte equal, checked) |
| Called unattended by a test cleanup | "The `cluster:operator` task succeeds from a clean checkout and is callable unattended", "Called unattended by a test cleanup" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The `cluster:operator` task SHALL ensure the CLI binary it invokes exi... | body of "The `cluster:operator` task depends on the build task so the CLI binary it invokes exists and is current" |
| The dependency SHALL reuse the build task's existing up-to-date check,... | body of "The `cluster:operator` task depends on the build task so the CLI binary it invokes exists and is current" |
| The `cluster:operator` task SHALL therefore succeed from a clean check... | body of "The `cluster:operator` task succeeds from a clean checkout and is callable unattended" |

Words changed or added: the em-dash of "callable unattended" reads ":"; the second body ends with "(Requirement: <header of the first requirement>)", the referent of "therefore"; the line wraps of the old text are joined.

Lengths (old, new): old 613; new "The `cluster:operator` task depends on the build task so the CLI binary it invokes exists and is current" 342; "The `cluster:operator` task succeeds from a clean checkout and is callable unattended" 408.

### Ledger: mod-vet, mod vet command validates module without generating manifests

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm mod vet` validates values against `#config` and renders the synthesized instance without outputting manifests", "`opm mod vet` accepts a module path argument and values flags for external values files", "`opm mod vet` takes its values from `debugValues` unless `-f` is provided".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Valid module passes validation using debugValues | "`opm mod vet` validates values against `#config` and renders the synthesized instance without outputting manifests", "Valid module passes validation using debugValues" | No (byte equal, checked) |
| -f flag overrides debugValues | "`opm mod vet` takes its values from `debugValues` unless `-f` is provided", "-f flag overrides debugValues" | No (byte equal, checked) |
| No debugValues and no -f flag | "`opm mod vet` takes its values from `debugValues` unless `-f` is provided", "No debugValues and no -f flag" | No (byte equal, checked) |
| Open debugValues is refused at synthesis | "`opm mod vet` takes its values from `debugValues` unless `-f` is provided", "Open debugValues is refused at synthesis" | No (byte equal, checked) |
| Values files against a module without #config | "`opm mod vet` takes its values from `debugValues` unless `-f` is provided", "Values files against a module without #config" | No (byte equal, checked) |
| Module with CUE validation errors fails with details | "`opm mod vet` validates values against `#config` and renders the synthesized instance without outputting manifests", "Module with CUE validation errors fails with details" | No (byte equal, checked) |
| Multiple values files with disallowed fields show per-file attribution | "`opm mod vet` accepts a module path argument and values flags for external values files", "Multiple values files with disallowed fields show per-file attribution" | No (byte equal, checked) |
| Nested validation errors show full path with positions | "`opm mod vet` validates values against `#config` and renders the synthesized instance without outputting manifests", "Nested validation errors show full path with positions" | No (byte equal, checked) |
| Multiple values files report schema errors and merge conflicts together | "`opm mod vet` accepts a module path argument and values flags for external values files", "Multiple values files report schema errors and merge conflicts together" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The `opm mod vet` command SHALL load the module directly, validate val... | body of "`opm mod vet` validates values against `#config` and renders the synthesized instance without outputting manifests" |
| It SHALL NOT output manifests (YAML/JSON). | body of "`opm mod vet` validates values against `#config` and renders the synthesized instance without outputting manifests" |
| Its purpose is a pass/fail verdict with clear diagnostics for module a... | body of "`opm mod vet` validates values against `#config` and renders the synthesized instance without outputting manifests" |
| The `opm mod vet` command SHALL accept a module path argument (default... | body of "`opm mod vet` accepts a module path argument and values flags for external values files" |
| When no `-f`/`--values` flag is provided, `opm mod vet` SHALL use the ... | body of "`opm mod vet` takes its values from `debugValues` unless `-f` is provided" |
| If `-f` is provided, it SHALL override `debugValues` and use the speci... | body of "`opm mod vet` takes its values from `debugValues` unless `-f` is provided" |
| If neither `debugValues` nor `-f` exists, the command SHALL return an ... | body of "`opm mod vet` takes its values from `debugValues` unless `-f` is provided" |

Words changed or added: "The command SHALL accept" reads "The `opm mod vet` command SHALL accept" (subject of the old first sentence).

Lengths (old, new): old 987; new "`opm mod vet` validates values against `#config` and renders the synthesized instance without outputting manifests" 465; "`opm mod vet` accepts a module path argument and values flags for external values files" 156; "`opm mod vet` takes its values from `debugValues` unless `-f` is provided" 378.

### Ledger: mod-vet, mod vet command flags and syntax

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm mod vet` accepts an optional module path argument and exposes `-f`/`--values`, `-n`/`--namespace`, `--name` and `--platform`", "`--instance-name` stays accepted as a deprecated alias of `--name` on `opm mod vet`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default flags match expected behavior | "`opm mod vet` accepts an optional module path argument and exposes `-f`/`--values`, `-n`/`--namespace`, `--name` and `--platform`", "Default flags match expected behavior" | No (byte equal, checked) |
| Name flag sets the synthesized instance name | "`opm mod vet` accepts an optional module path argument and exposes `-f`/`--values`, `-n`/`--namespace`, `--name` and `--platform`", "Name flag sets the synthesized instance name" | No (byte equal, checked) |
| Deprecated alias keeps working | "`--instance-name` stays accepted as a deprecated alias of `--name` on `opm mod vet`", "Deprecated alias keeps working" | No (byte equal, checked) |
| Both spellings together are refused | "`--instance-name` stays accepted as a deprecated alias of `--name` on `opm mod vet`", "Both spellings together are refused" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "Command syntax" in "`opm mod vet` accepts an optional module path argument and exposes `-f`/`--values`, `-n`/`--namespace`, `--name` and `--platform`"

| Old clause | New place |
| --- | --- |
| The `opm mod vet` command SHALL accept an optional module path argumen... | body of "`opm mod vet` accepts an optional module path argument and exposes `-f`/`--values`, `-n`/`--namespace`, `--name` and `--platform`" |
| All four affect the verdict: `-f` selects the values, `-n` and `--name... | body of "`opm mod vet` accepts an optional module path argument and exposes `-f`/`--values`, `-n`/`--namespace`, `--name` and `--platform`" |
| The syntax is shown in the scenario `Command syntax`. | body of "`opm mod vet` accepts an optional module path argument and exposes `-f`/`--values`, `-n`/`--namespace`, `--name` and `--platform`" |
| `--instance-name` SHALL stay accepted as a deprecated alias of `--name... | body of "`--instance-name` stays accepted as a deprecated alias of `--name` on `opm mod vet`" |
| It SHALL NOT appear in the command's help, and its use SHALL print one... | body of "`--instance-name` stays accepted as a deprecated alias of `--name` on `opm mod vet`" |
| Passing both `--name` and `--instance-name` SHALL be a usage error, an... | body of "`--instance-name` stays accepted as a deprecated alias of `--name` on `opm mod vet`" |

Words changed or added: the first body ends with "The syntax is shown in the scenario `Command syntax`.", the pointer to the new scenario `Command syntax` that holds the fenced syntax block of the old body byte for byte, indented in a THEN line (a fenced layout moves into a THEN line when the body names the scenario); the second body (the deprecated alias) ends with "(Requirement: <header of the first requirement>)", the named reference that gives `--instance-name` its command.

Lengths (old, new): old 755; new "`opm mod vet` accepts an optional module path argument and exposes `-f`/`--values`, `-n`/`--namespace`, `--name` and `--platform`" 488; "`--instance-name` stays accepted as a deprecated alias of `--name` on `opm mod vet`" 465.

### Ledger: mod-vet, mod vet exit codes

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm mod vet` signals its verdict through the process exit code", "`opm mod vet` maps a failed core-schema fetch as `opm module publish` does and prints the login hint on a refusal".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Exit code 0 on success | "`opm mod vet` signals its verdict through the process exit code", "Exit code 0 on success" | No (byte equal, checked) |
| Exit code 2 on validation failure | "`opm mod vet` signals its verdict through the process exit code", "Exit code 2 on validation failure" | No (byte equal, checked) |
| Exit code 2 on render refusal | "`opm mod vet` signals its verdict through the process exit code", "Exit code 2 on render refusal" | No (byte equal, checked) |
| Exit code 2 when no registry is configured | "`opm mod vet` signals its verdict through the process exit code", "Exit code 2 when no registry is configured" | No (byte equal, checked) |
| Exit code 4 when the registry refuses the credentials | "`opm mod vet` maps a failed core-schema fetch as `opm module publish` does and prints the login hint on a refusal", "Exit code 4 when the registry refuses the credentials" | No (byte equal, checked) |
| Exit code 3 when the registry is unreachable or fails | "`opm mod vet` signals its verdict through the process exit code", "Exit code 3 when the registry is unreachable or fails" | No (byte equal, checked) |
| A 403 on the tag lookup reads as a failed registry operation | "`opm mod vet` maps a failed core-schema fetch as `opm module publish` does and prints the login hint on a refusal", "A 403 on the tag lookup reads as a failed registry operation" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "Exit code table" in "`opm mod vet` signals its verdict through the process exit code"

| Old clause | New place |
| --- | --- |
| The `opm mod vet` command SHALL signal its verdict through the process... | body of "`opm mod vet` signals its verdict through the process exit code" |
| The mapping of a failed core-schema fetch SHALL be the one `opm module... | body of "`opm mod vet` maps a failed core-schema fetch as `opm module publish` does and prints the login hint on a refusal" |

Words changed or added: "as the table below details" reads "as the table in the scenario `Exit code table` details" (a rule table moves into the THEN of the scenario `Exit code table`, byte for byte as old rows); the second body ends with "(Requirement: <header of the first requirement>)", the referent of the core-schema fetch.

Lengths (old, new): old 1340; new "`opm mod vet` signals its verdict through the process exit code" 488; "`opm mod vet` maps a failed core-schema fetch as `opm module publish` does and prints the login hint on a refusal" 228.

### Ledger: mod-vet, Identity and coordinate checks before values validation

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm module vet` verifies the identity package, the derived coordinates and the `cue.mod` module line before values resolution", "Failures of the identity and coordinate checks report as validation errors with exit code 2 even without `debugValues`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Coordinate drift caught at vet | "Failures of the identity and coordinate checks report as validation errors with exit code 2 even without `debugValues`", "Coordinate drift caught at vet" | No (byte equal, checked) |
| Replaced derivation caught at vet | "`opm module vet` verifies the identity package, the derived coordinates and the `cue.mod` module line before values resolution", "Replaced derivation caught at vet" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm module vet` SHALL, after loading the module and before values res... | body of "`opm module vet` verifies the identity package, the derived coordinates and the `cue.mod` module line before values resolution" |
| Failures SHALL report through the standard validation-error rendering ... | body of "Failures of the identity and coordinate checks report as validation errors with exit code 2 even without `debugValues`" |

Words changed or added: the second body ends with "(Requirement: <header of the first requirement>)", the referent of "Failures".

Lengths (old, new): old 569; new "`opm module vet` verifies the identity package, the derived coordinates and the `cue.mod` module line before values resolution" 348; "Failures of the identity and coordinate checks report as validation errors with exit code 2 even without `debugValues`" 362.

### Ledger: catalog-compatibility, Compatibility Comparison

Form: c2 (REMOVED, ADDED under new names). New requirements: "`internal/compat.Check` reports every violation of the additive-only rule", "`Check` is a field-wise walk", "Each violation carries the dotted path from the compared root and a stable kind discriminator", "A field that was optional and is required in the new definition reports `field made required`", "A defaulted field that loses its default and the reverse transition do not report `field made required`", "A default is compared only where an author wrote one, and the implicit default of a plain open list is not compared", "An open list's value domain is still judged at the leaf", "The default rule is applied to each pair of an open list's fixed elements", "The walk applies the provenance denylist at every depth", "Lists whose two sides have equal length are walked element-wise and lists of unequal length are judged as a leaf", "A leaf whose emitted syntax is identical on both sides reports nothing".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Field removal reported | "`internal/compat.Check` reports every violation of the additive-only rule", "Field removal reported" | No (byte equal, checked) |
| Strict field addition reported | "`internal/compat.Check` reports every violation of the additive-only rule", "Strict field addition reported" | No (byte equal, checked) |
| Field made required reported | "A field that was optional and is required in the new definition reports `field made required`", "Field made required reported" | No (byte equal, checked) |
| Implicit open-list default is not compared | "A default is compared only where an author wrote one, and the implicit default of a plain open list is not compared", "Implicit open-list default is not compared" | No (byte equal, checked) |
| Open list narrowed reports only the narrowing | "An open list's value domain is still judged at the leaf", "Open list narrowed reports only the narrowing" | No (byte equal, checked) |
| Authored list default is still compared | "A default is compared only where an author wrote one, and the implicit default of a plain open list is not compared", "Authored list default is still compared" | No (byte equal, checked) |
| Default inside an open list's fixed element is compared | "The default rule is applied to each pair of an open list's fixed elements", "Default inside an open list's fixed element is compared" | No (byte equal, checked) |
| Default change reported with both values | "Each violation carries the dotted path from the compared root and a stable kind discriminator", "Default change reported with both values" | No (byte equal, checked) |
| Domain narrowing reported at a leaf | "`Check` is a field-wise walk", "Domain narrowing reported at a leaf" | No (byte equal, checked) |
| Additive widening passes | "`internal/compat.Check` reports every violation of the additive-only rule", "Additive widening passes" | No (byte equal, checked) |
| Provenance inside a member reference is ignored | "The walk applies the provenance denylist at every depth", "Provenance inside a member reference is ignored" | No (byte equal, checked) |
| Unchanged guarded leaf is silent | "A leaf whose emitted syntax is identical on both sides reports nothing", "Unchanged guarded leaf is silent" | No (byte equal, checked) |
| Equal-length lists are walked | "Lists whose two sides have equal length are walked element-wise and lists of unequal length are judged as a leaf", "Equal-length lists are walked" | No (byte equal, checked) |
| Unequal-length lists are a leaf | "Lists whose two sides have equal length are walked element-wise and lists of unequal length are judged as a leaf", "Unequal-length lists are a leaf" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "A defaulted field that loses its default reports `default removed`, not `field made required`" in "A defaulted field that loses its default and the reverse transition do not report `field made required`"

| Old clause | New place |
| --- | --- |
| The CLI SHALL provide a pure comparison (`internal/compat.Check`) that... | body of "`internal/compat.Check` reports every violation of the additive-only rule" |
| The comparison (`internal/compat.Check`) SHALL be a field-wise walk: s... | body of "`Check` is a field-wise walk" |
| Each violation SHALL carry the dotted path from the compared root and ... | body of "Each violation carries the dotted path from the compared root and a stable kind discriminator" |
| A field is "required" for this rule when it carries the required const... | body of "A field that was optional and is required in the new definition reports `field made required`" |
| A field that carried the optional constraint (`?`) in the prior defini... | body of "A field that was optional and is required in the new definition reports `field made required`" |
| A defaulted field that loses its default is the default rule's finding... | body of "A defaulted field that loses its default and the reverse transition do not report `field made required`" |
| The reverse transition (required to optional, or an optional field gai... | body of "A defaulted field that loses its default and the reverse transition do not report `field made required`" |
| A default is compared only where an author wrote one: a value whose di... | body of "A default is compared only where an author wrote one, and the implicit default of a plain open list is not compared" |
| The implicit default CUE gives a plain open list (the list closed at i... | body of "A default is compared only where an author wrote one, and the implicit default of a plain open list is not compared" |
| An open list's value domain SHALL still be judged at the leaf, so narr... | body of "An open list's value domain is still judged at the leaf" |
| Defaults written inside an open list's fixed elements (`[*"a" \| string... | body of "The default rule is applied to each pair of an open list's fixed elements" |
| The walk SHALL apply the provenance denylist of enhancement 0010 at ev... | body of "The walk applies the provenance denylist at every depth" |
| Lists whose two sides have equal length SHALL be walked element-wise s... | body of "Lists whose two sides have equal length are walked element-wise and lists of unequal length are judged as a leaf" |
| A leaf whose emitted syntax is identical on both sides SHALL report no... | body of "A leaf whose emitted syntax is identical on both sides reports nothing" |

Words changed or added: "The comparison SHALL be" reads "The comparison (`internal/compat.Check`) SHALL be" (subject of the old first sentence); the em-dashes of that sentence read ":" and ","; "An open list's value domain is still judged" reads "SHALL still be judged" (strength table); the default body ends with "(Requirement: <header of the walk requirement>)", the named reference that gives "the comparison" its subject; the bodies for the violation, the made-required kind, the loss of a default, the open list, the walk and the leaf end with "(Requirement: <header>)", the named reference that gives "Each violation", "this rule", "this one's", "The walk", "that rule" and "the default rule" their referent.

Lengths (old, new): old 2972; new "`internal/compat.Check` reports every violation of the additive-only rule" 434; "`Check` is a field-wise walk" 423; "Each violation carries the dotted path from the compared root and a stable kind discriminator" 258; "A field that was optional and is required in the new definition reports `field made required`" 424; "A defaulted field that loses its default and the reverse transition do not report `field made required`" 330; "A default is compared only where an author wrote one, and the implicit default of a plain open list is not compared" 498; "An open list's value domain is still judged at the leaf" 244; "The default rule is applied to each pair of an open list's fixed elements" 442; "The walk applies the provenance denylist at every depth" 435; "Lists whose two sides have equal length are walked element-wise and lists of unequal length are judged as a leaf" 225; "A leaf whose emitted syntax is identical on both sides reports nothing" 120.

### Ledger: catalog-compatibility, Level Classification

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI classifies a primitive's `apiVersion` on the `vNalphaM | vNbetaM | vN` ladder, exposes whether a level is enforced and pins the grammar against core", "The level-aware entry point `CheckAtLevel` returns no violations for an alpha `apiVersion` and an error for an `apiVersion` the grammar rejects", "The CLI provides a total, transitive ordering over valid `apiVersion` strings".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Alpha is not gated | "The level-aware entry point `CheckAtLevel` returns no violations for an alpha `apiVersion` and an error for an `apiVersion` the grammar rejects", "Alpha is not gated" | No (byte equal, checked) |
| Beta and GA are gated | "The CLI classifies a primitive's `apiVersion` on the `vNalphaM | vNbetaM | vN` ladder, exposes whether a level is enforced and pins the grammar against core", "Beta and GA are gated" | No (byte equal, checked) |
| Unparseable apiVersion is an error | "The level-aware entry point `CheckAtLevel` returns no violations for an alpha `apiVersion` and an error for an `apiVersion` the grammar rejects", "Unparseable apiVersion is an error" | No (byte equal, checked) |
| Ordering is transitive | "The CLI provides a total, transitive ordering over valid `apiVersion` strings", "Ordering is transitive" | No (byte equal, checked) |
| Grammar parity with core is pinned | "The CLI classifies a primitive's `apiVersion` on the `vNalphaM | vNbetaM | vN` ladder, exposes whether a level is enforced and pins the grammar against core", "Grammar parity with core is pinned" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The CLI SHALL classify a primitive's `apiVersion` on the `vNalphaM \| v... | body of "The CLI classifies a primitive's `apiVersion` on the `vNalphaM | vNbetaM | vN` ladder, exposes whether a level is enforced and pins the grammar against core" |
| The grammar SHALL be pinned against core's own declaration by a parity... | body of "The CLI classifies a primitive's `apiVersion` on the `vNalphaM | vNbetaM | vN` ladder, exposes whether a level is enforced and pins the grammar against core" |
| The level-aware entry point (`CheckAtLevel`) SHALL return no violation... | body of "The level-aware entry point `CheckAtLevel` returns no violations for an alpha `apiVersion` and an error for an `apiVersion` the grammar rejects" |
| The CLI SHALL provide a total, transitive ordering over valid `apiVers... | body of "The CLI provides a total, transitive ordering over valid `apiVersion` strings" |

Words changed or added: the em-dashes of "an error, not a violation" read ","; "The grammar SHALL be pinned" is moved next to the sentence that defines "the grammar" (the first sentence).

Lengths (old, new): old 775; new "The CLI classifies a primitive's `apiVersion` on the `vNalphaM | vNbetaM | vN` ladder, exposes whether a level is enforced and pins the grammar against core" 437; "The level-aware entry point `CheckAtLevel` returns no violations for an alpha `apiVersion` and an error for an `apiVersion` the grammar rejects" 228; "The CLI provides a total, transitive ordering over valid `apiVersion` strings" 106.

### Ledger: catalog-compatibility, Template Version Selection

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI provides a stable-preferring version selector over a published-version list that is pure", "The stable-preferring version selector is not exported and is not the compatibility gate's predecessor selection", "The publish gate's predecessor is found by the literal backward scan of the published history", "The literal predecessor rule is implemented in `internal/publish` over the registry's version enumeration", "The documentation of the selector states how it differs from the predecessor selection of the publish gate".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Stable preferred over higher prerelease | "The CLI provides a stable-preferring version selector over a published-version list that is pure", "Stable preferred over higher prerelease" | No (byte equal, checked) |
| Prerelease-only fallback | "The CLI provides a stable-preferring version selector over a published-version list that is pure", "Prerelease-only fallback" | No (byte equal, checked) |
| Unparseable entry is skipped | "The CLI provides a stable-preferring version selector over a published-version list that is pure", "Unparseable entry is skipped" | No (byte equal, checked) |
| The gate does not use it | "The publish gate's predecessor is found by the literal backward scan of the published history", "The gate does not use it" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The selector is not exported and is not in the compatibility package" in "The stable-preferring version selector is not exported and is not the compatibility gate's predecessor selection"
- "The literal predecessor rule is implemented in `internal/publish`" in "The literal predecessor rule is implemented in `internal/publish` over the registry's version enumeration"
- "The selector documentation states the distinction" in "The documentation of the selector states how it differs from the predecessor selection of the publish gate"

| Old clause | New place |
| --- | --- |
| The CLI SHALL provide a stable-preferring version selector over a publ... | body of "The CLI provides a stable-preferring version selector over a published-version list that is pure" |
| Selection SHALL be pure: enumeration and fetching are the caller's. | body of "The CLI provides a stable-preferring version selector over a published-version list that is pure" |
| The selector SHALL NOT be exported: template resolution is its only ca... | body of "The stable-preferring version selector is not exported and is not the compatibility gate's predecessor selection" |
| This selector is NOT the compatibility gate's predecessor selection, a... | body of "The stable-preferring version selector is not exported and is not the compatibility gate's predecessor selection" |
| Enhancement 0011 ("Predecessor selection is D9's literal rule: a backw... | body of "The publish gate's predecessor is found by the literal backward scan of the published history" |
| Enhancement 0011 ("Predecessor selection is D9's literal rule: a backw... | body of "The literal predecessor rule is implemented in `internal/publish` over the registry's version enumeration" |
| A stable-preferring selector coincides with that rule only on a prerel... | body of "The documentation of the selector states how it differs from the predecessor selection of the publish gate" |
| The documentation of the selector SHALL state this distinction, becaus... | body of "The documentation of the selector states how it differs from the predecessor selection of the publish gate" |

Words changed or added: the em-dash of "Selection SHALL be pure" reads ":"; the long sentence about enhancement 0011 is cut at the list boundary of its dash list: the first body ends after the scan list (its em-dash reads ":"), the second body repeats the subject up to "the literal rule" and takes "implemented in `internal/publish` ..." (its em-dash reads ","), and the third body takes the part after the semicolon as a sentence ("a stable-preferring selector" reads "A stable-preferring selector"); "is found" reads "SHALL be found" in both bodies that repeat the subject (strength table); the selector body ends with "(Requirement: <header of the first selector requirement>)", the referent of "The selector"; the last body ends with "(Requirement: <header of the first 0011 requirement>)", the referent of "that rule".

Lengths (old, new): old 1504; new "The CLI provides a stable-preferring version selector over a published-version list that is pure" 435; "The stable-preferring version selector is not exported and is not the compatibility gate's predecessor selection" 305; "The publish gate's predecessor is found by the literal backward scan of the published history" 473; "The literal predecessor rule is implemented in `internal/publish` over the registry's version enumeration" 350; "The documentation of the selector states how it differs from the predecessor selection of the publish gate" 437.

## Ledger entries of cli-B6, unit 6a (published-module-resolution, template-modules, dependency-tidy, e2e-cluster-workflow)

### Ledger: published-module-resolution, A published module is named by a major-free module path

Form: c2 (REMOVED, ADDED under new names). New requirements: "A published module path is a CUE module path without a major suffix and the version selector chooses the major", "A published module path with an `@` suffix or a URL scheme is refused with exit code 2".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Major suffix refused | "A published module path with an `@` suffix or a URL scheme is refused with exit code 2", "Major suffix refused" | No (byte equal, checked) |
| OCI URL refused | "A published module path with an `@` suffix or a URL scheme is refused with exit code 2", "OCI URL refused" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "A major-free module path" in "A published module path is a CUE module path without a major suffix and the version selector chooses the major"

| Old clause | New place |
| --- | --- |
| A published module path SHALL be a CUE module path without a major suf... | body of "A published module path is a CUE module path without a major suffix and the version selector chooses the major" |
| Source line (whole) | body of "A published module path is a CUE module path without a major suffix and the version selector chooses the major (as `**Source**`)" |
| A path carrying an `@` suffix SHALL be refused with exit code 2 before... | body of "A published module path with an `@` suffix or a URL scheme is refused with exit code 2" |
| A path with a URL scheme (`oci://`, `https://`) SHALL be refused with ... | body of "A published module path with an `@` suffix or a URL scheme is refused with exit code 2" |
| Source line (whole) | body of "A published module path with an `@` suffix or a URL scheme is refused with exit code 2 (as `**Source**`)" |

Words changed or added: none.

Lengths (old, new): old 886; new "A published module path is a CUE module path without a major suffix and the version selector chooses the major" 319; "A published module path with an `@` suffix or a URL scheme is refused with exit code 2" 384.

### Ledger: published-module-resolution, The version selector floats within a major or pins a tag

Form: c2 (REMOVED, ADDED under new names). New requirements: "`--version` accepts exactly the forms `vN` and `X.Y.Z` or `X.Y.Z-<prerelease>` and any other value is refused with exit code 2", "The newest release within a major is the newest stable version, else the newest prerelease that is not a development build", "An exact pin names a published tag, development builds included, and a pin the registry does not hold is refused with exit code 2".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Major float prefers stable | "The newest release within a major is the newest stable version, else the newest prerelease that is not a development build", "Major float prefers stable" | No (byte equal, checked) |
| Prerelease-only major | "The newest release within a major is the newest stable version, else the newest prerelease that is not a development build", "Prerelease-only major" | No (byte equal, checked) |
| Exact pin | "An exact pin names a published tag, development builds included, and a pin the registry does not hold is refused with exit code 2", "Exact pin" | No (byte equal, checked) |
| Malformed selector | "`--version` accepts exactly the forms `vN` and `X.Y.Z` or `X.Y.Z-<prerelease>` and any other value is refused with exit code 2", "Malformed selector" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `--version` SHALL accept exactly two forms: `vN` (for example `v1`) se... | body of "`--version` accepts exactly the forms `vN` and `X.Y.Z` or `X.Y.Z-<prerelease>` and any other value is refused with exit code 2" |
| Any other value, including `v1.0.4`, SHALL be refused with exit code 2... | body of "`--version` accepts exactly the forms `vN` and `X.Y.Z` or `X.Y.Z-<prerelease>` and any other value is refused with exit code 2" |
| Source line (whole) | body of "`--version` accepts exactly the forms `vN` and `X.Y.Z` or `X.Y.Z-<prerelease>` and any other value is refused with exit code 2 (as `**Source**`)" |
| The newest release within a major is the newest stable version; when t... | body of "The newest release within a major is the newest stable version, else the newest prerelease that is not a development build" |
| Source line (whole) | body of "The newest release within a major is the newest stable version, else the newest prerelease that is not a development build (as `**Source**`)" |
| An exact pin SHALL name a published tag, development builds included; ... | body of "An exact pin names a published tag, development builds included, and a pin the registry does not hold is refused with exit code 2" |
| Source line (whole) | body of "An exact pin names a published tag, development builds included, and a pin the registry does not hold is refused with exit code 2 (as `**Source**`)" |

Words changed or added: the body ends with "(Requirement: `--version` accepts exactly the forms `vN` and `X.Y.Z` or `X.Y.Z-<prerelease>` and any other value is refused with exit code 2)", the named reference that gives it its referent.

Lengths (old, new): old 1093; new "`--version` accepts exactly the forms `vN` and `X.Y.Z` or `X.Y.Z-<prerelease>` and any other value is refused with exit code 2" 355; "The newest release within a major is the newest stable version, else the newest prerelease that is not a development build" 352; "An exact pin names a published tag, development builds included, and a pin the registry does not hold is refused with exit code 2" 350.

### Ledger: published-module-resolution, No selector selects the highest core-compatible major

Form: c2 (REMOVED, ADDED under new names). New requirements: "Without `--version` the command walks the majors from highest to lowest and selects the first major on the core major the CLI renders against", "A major with no selectable release, no core dependency or a different core major is skipped, and the command exits 2 when every major is skipped".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Newer major on an incompatible core is skipped | "A major with no selectable release, no core dependency or a different core major is skipped, and the command exits 2 when every major is skipped", "Newer major on an incompatible core is skipped" | No (byte equal, checked) |
| Major without a core dependency is skipped | "A major with no selectable release, no core dependency or a different core major is skipped, and the command exits 2 when every major is skipped", "Major without a core dependency is skipped" | No (byte equal, checked) |
| Nothing compatible | "A major with no selectable release, no core dependency or a different core major is skipped, and the command exits 2 when every major is skipped", "Nothing compatible" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The first core-compatible major is selected" in "Without `--version` the command walks the majors from highest to lowest and selects the first major on the core major the CLI renders against"

| Old clause | New place |
| --- | --- |
| Without `--version`, the command SHALL list the module's published ver... | body of "Without `--version` the command walks the majors from highest to lowest and selects the first major on the core major the CLI renders against" |
| Source line (whole) | body of "Without `--version` the command walks the majors from highest to lowest and selects the first major on the core major the CLI renders against (as `**Source**`)" |
| When every major is skipped, the command SHALL exit 2 naming each majo... | body of "A major with no selectable release, no core dependency or a different core major is skipped, and the command exits 2 when every major is skipped" |
| A major is skipped when it holds no selectable release, declares no `o... | body of "A major with no selectable release, no core dependency or a different core major is skipped, and the command exits 2 when every major is skipped" |
| Source line (whole) | body of "A major with no selectable release, no core dependency or a different core major is skipped, and the command exits 2 when every major is skipped (as `**Source**`)" |

Words changed or added: the body ends with "(Requirement: Without `--version` the command walks the majors from highest to lowest and selects the first major on the core major the CLI renders against)", the named reference that gives it its referent.

Lengths (old, new): old 804; new "Without `--version` the command walks the majors from highest to lowest and selects the first major on the core major the CLI renders against" 391; "A major with no selectable release, no core dependency or a different core major is skipped, and the command exits 2 when every major is skipped" 393.

### Ledger: published-module-resolution, Resolution reports the choice and never corrupts command output

Form: c2 (REMOVED, ADDED under new names). New requirements: "After resolving, the command reports the choice and the skipped higher majors on standard error", "Resolution routes through the CLI's registry configuration and the command exits 3 naming the module path and the registry when the registry cannot be reached".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Report names the selection | "After resolving, the command reports the choice and the skipped higher majors on standard error", "Report names the selection" | No (byte equal, checked) |
| Unreachable registry | "Resolution routes through the CLI's registry configuration and the command exits 3 naming the module path and the registry when the registry cannot be reached", "Unreachable registry" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| After resolving, the command SHALL report one line naming the module p... | body of "After resolving, the command reports the choice and the skipped higher majors on standard error" |
| Source line (whole) | body of "After resolving, the command reports the choice and the skipped higher majors on standard error (as `**Source**`)" |
| Resolution SHALL route through the CLI's registry configuration (`--re... | body of "Resolution routes through the CLI's registry configuration and the command exits 3 naming the module path and the registry when the registry cannot be reached" |
| Source line (whole) | body of "Resolution routes through the CLI's registry configuration and the command exits 3 naming the module path and the registry when the registry cannot be reached (as `**Source**`)" |

Words changed or added: none.

Lengths (old, new): old 812; new "After resolving, the command reports the choice and the skipped higher majors on standard error" 391; "Resolution routes through the CLI's registry configuration and the command exits 3 naming the module path and the registry when the registry cannot be reached" 238.

### Ledger: template-modules, Templates are published modules in a reserved segment

Form: c2 (REMOVED, ADDED under new names). New requirements: "The official templates are published modules in the reserved segment", "A template passes the vet and a failing template fails the release", "The `templates` segment is module-kind and cli-CI-published only and no gate reserves a name within it", "Publish never skips an already-published tag and the release job invokes it only for the versions its gates found unpublished", "A template's tree is exactly the module that publishes", "A published template version holds its tree and a change bumps `Version`", "A template's `Version` is a stable SemVer with the module path of its major", "The cli's PR check and release job refuse a template that breaks a publish gate, the vet, the tree, bump, `Version` or module path rule", "A template's identity package declares `Version` as a concrete string literal and the template tree loads through the kernel's module loader unmodified".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Templates are gated artifacts | "A template passes the vet and a failing template fails the release", "Templates are gated artifacts" | No (byte equal, checked) |
| Release job is idempotent by filtering | "Publish never skips an already-published tag and the release job invokes it only for the versions its gates found unpublished", "Release job is idempotent by filtering" | No (byte equal, checked) |
| Template identity is a literal the kernel loads | "A template's identity package declares `Version` as a concrete string literal and the template tree loads through the kernel's module loader unmodified", "Template identity is a literal the kernel loads" | No (byte equal, checked) |
| Template renders its own debugValues | "A template passes the vet and a failing template fails the release", "Template renders its own debugValues" | No (byte equal, checked) |
| Template tree is exactly its published module | "The cli's PR check and release job refuse a template that breaks a publish gate, the vet, the tree, bump, `Version` or module path rule", "Template tree is exactly its published module" | No (byte equal, checked) |
| Template version is the one init resolves | "The cli's PR check and release job refuse a template that breaks a publish gate, the vet, the tree, bump, `Version` or module path rule", "Template version is the one init resolves" | No (byte equal, checked) |
| No name in the segment is reserved | "The `templates` segment is module-kind and cli-CI-published only and no gate reserves a name within it", "No name in the segment is reserved" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "Official templates are published through the release pipeline" in "The official templates are published modules in the reserved segment"
- "A template tree holds only what its module zip keeps" in "A template's tree is exactly the module that publishes"
- "A change to a template bumps its identity Version" in "A published template version holds its tree and a change bumps `Version`"
- "Template Version and module path" in "A template's `Version` is a stable SemVer with the module path of its major"

| Old clause | New place |
| --- | --- |
| The official templates, `minimal`, `standard`, `advanced`, SHALL be re... | body of "The official templates are published modules in the reserved segment" |
| Source line (whole) | body of "The official templates are published modules in the reserved segment (as `**Source**`)" |
| A template SHALL also pass `opm module vet` with its own `debugValues`... | body of "A template passes the vet and a failing template fails the release" |
| Source line (whole) | body of "A template passes the vet and a failing template fails the release (as `**Source**`)" |
| The segment is module-kind and cli-CI-published only. | body of "The `templates` segment is module-kind and cli-CI-published only and no gate reserves a name within it" |
| No gate reserves a name within it: the publish gates admit any single-... | body of "The `templates` segment is module-kind and cli-CI-published only and no gate reserves a name within it" |
| Source line (whole) | body of "The `templates` segment is module-kind and cli-CI-published only and no gate reserves a name within it (as `**Source**`)" |
| Publish itself never skips an already-published tag; the release job S... | body of "Publish never skips an already-published tag and the release job invokes it only for the versions its gates found unpublished" |
| Source line (whole) | body of "Publish never skips an already-published tag and the release job invokes it only for the versions its gates found unpublished (as `**Source**`)" |
| A template's tree SHALL be exactly the module that publishes: it SHALL... | body of "A template's tree is exactly the module that publishes" |
| Source line (whole) | body of "A template's tree is exactly the module that publishes (as `**Source**`)" |
| Because published versions are immutable, the artifact published at a ... | body of "A published template version holds its tree and a change bumps `Version`" |
| Source line (whole) | body of "A published template version holds its tree and a change bumps `Version` (as `**Source**`)" |
| A template's `Version` SHALL be a stable SemVer, and SHALL be the high... | body of "A template's `Version` is a stable SemVer with the module path of its major" |
| Source line (whole) | body of "A template's `Version` is a stable SemVer with the module path of its major (as `**Source**`)" |
| The cli's PR check and release job refuse a template that breaks any o... | body of "The cli's PR check and release job refuse a template that breaks a publish gate, the vet, the tree, bump, `Version` or module path rule" |
| Source line (whole) | body of "The cli's PR check and release job refuse a template that breaks a publish gate, the vet, the tree, bump, `Version` or module path rule (as `**Source**`)" |
| A template's identity package SHALL declare `Version` as a concrete st... | body of "A template's identity package declares `Version` as a concrete string literal and the template tree loads through the kernel's module loader unmodified" |
| Source line (whole) | body of "A template's identity package declares `Version` as a concrete string literal and the template tree loads through the kernel's module loader unmodified (as `**Source**`)" |

Words changed or added: the body ends with "(Requirement: The official templates are published modules in the reserved segment)", the named reference that gives it its referent; the body ends with "(Requirements: "The official templates are published modules in the reserved segment", "A template passes the vet and a failing template fails the release", "A template's tree is exactly the module that publishes", "A published template version holds its tree and a change bumps `Version`", "A template's `Version` is a stable SemVer with the module path of its major")", the named reference that gives it its referent; the em-dashes of the old text read ",".

Lengths (old, new): old 2038; new "The official templates are published modules in the reserved segment" 264; "A template passes the vet and a failing template fails the release" 274; "The `templates` segment is module-kind and cli-CI-published only and no gate reserves a name within it" 315; "Publish never skips an already-published tag and the release job invokes it only for the versions its gates found unpublished" 135; "A template's tree is exactly the module that publishes" 162; "A published template version holds its tree and a change bumps `Version`" 260; "A template's `Version` is a stable SemVer with the module path of its major" 255; "The cli's PR check and release job refuse a template that breaks a publish gate, the vet, the tree, bump, `Version` or module path rule" 460; "A template's identity package declares `Version` as a concrete string literal and the template tree loads through the kernel's module loader unmodified" 269.

### Ledger: template-modules, init scaffolds by fetch and re-identification

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm mod init <new-module-path> [template]` fetches the template module, copies its source tree and re-identifies it to the new path, leaving metadata untouched", "The scaffold carries nothing of the template's identity and a scaffolded module passes `opm module vet`, loads through the kernel's module loader and passes the `publish --dry-run` gates", "Init with no template on a directory that already holds a module tree repairs that tree and adopts the version the tree states", "Init requires a reachable registry for an uncached template, refuses offline naming the expansion and the registry tried, and embeds no fallback template".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Scaffold is publishable and leak-free | "The scaffold carries nothing of the template's identity and a scaffolded module passes `opm module vet`, loads through the kernel's module loader and passes the `publish --dry-run` gates", "Scaffold is publishable and leak-free" | No (byte equal, checked) |
| Scaffold identity is a literal | "`opm mod init <new-module-path> [template]` fetches the template module, copies its source tree and re-identifies it to the new path, leaving metadata untouched", "Scaffold identity is a literal" | No (byte equal, checked) |
| Offline refuses honestly | "Init requires a reachable registry for an uncached template, refuses offline naming the expansion and the registry tried, and embeds no fallback template", "Offline refuses honestly" | No (byte equal, checked) |
| Repair adopts the version the tree states | "Init with no template on a directory that already holds a module tree repairs that tree and adopts the version the tree states", "Repair adopts the version the tree states" | No (byte equal, checked) |
| Repair refuses a tree that states no version | "Init with no template on a directory that already holds a module tree repairs that tree and adopts the version the tree states", "Repair refuses a tree that states no version" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm mod init <new-module-path> [template]` SHALL fetch the template m... | body of "`opm mod init <new-module-path> [template]` fetches the template module, copies its source tree and re-identifies it to the new path, leaving metadata untouched" |
| Metadata SHALL be untouched, it derives from the identity package. | body of "`opm mod init <new-module-path> [template]` fetches the template module, copies its source tree and re-identifies it to the new path, leaving metadata untouched" |
| The template's own identity SHALL NOT appear anywhere in the scaffold. | body of "The scaffold carries nothing of the template's identity and a scaffolded module passes `opm module vet`, loads through the kernel's module loader and passes the `publish --dry-run` gates" |
| A scaffolded module SHALL pass `opm module vet`, load through the kern... | body of "The scaffold carries nothing of the template's identity and a scaffolded module passes `opm module vet`, loads through the kernel's module loader and passes the `publish --dry-run` gates" |
| When repair has to create a missing identity package, `Version` SHALL ... | body of "Init with no template on a directory that already holds a module tree repairs that tree and adopts the version the tree states" |
| Init with no template, on a directory that already holds a module tree... | body of "Init with no template on a directory that already holds a module tree repairs that tree and adopts the version the tree states" |
| Repair leaves the `Version` of an identity file that exists untouched. | body of "Init with no template on a directory that already holds a module tree repairs that tree and adopts the version the tree states" |
| Init requires a reachable registry for an uncached template and SHALL ... | body of "Init requires a reachable registry for an uncached template, refuses offline naming the expansion and the registry tried, and embeds no fallback template" |

Words changed or added: the body ends with "(Requirement: `opm mod init <new-module-path> [template]` fetches the template module, copies its source tree and re-identifies it to the new path, leaving metadata untouched)", the named reference that gives it its referent; the em-dashes of the old text read ",".

Lengths (old, new): old 1223; new "`opm mod init <new-module-path> [template]` fetches the template module, copies its source tree and re-identifies it to the new path, leaving metadata untouched" 423; "The scaffold carries nothing of the template's identity and a scaffolded module passes `opm module vet`, loads through the kernel's module loader and passes the `publish --dry-run` gates" 414; "Init with no template on a directory that already holds a module tree repairs that tree and adopts the version the tree states" 409; "Init requires a reachable registry for an uncached template, refuses offline naming the expansion and the registry tried, and embeds no fallback template" 159.

### Ledger: template-modules, Shortcut expansion is syntactic and safe

Form: c2 (REMOVED, ADDED under new names). New requirements: "A bare word expands into the templates segment and a reference with `/` or `.` is a literal path", "An `@vN` suffix floats within a major and a full semver pins a tag", "No suffix selects a default major for an official template only", "`--from` accepts the same forms explicitly and any published module as a clone source, and `-t/--template` is an alias", "A template-only invocation prompts for the new module path interactively and refuses non-interactively".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Bare word expands, path never does | "A bare word expands into the templates segment and a reference with `/` or `.` is a literal path", "Bare word expands, path never does" | No (byte equal, checked) |
| Typo fails inside the segment | "A bare word expands into the templates segment and a reference with `/` or `.` is a literal path", "Typo fails inside the segment" | No (byte equal, checked) |
| Interactive form asks for the path | "A template-only invocation prompts for the new module path interactively and refuses non-interactively", "Interactive form asks for the path" | No (byte equal, checked) |
| Only official templates get a default major | "No suffix selects a default major for an official template only", "Only official templates get a default major" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "A suffix selects the release" in "An `@vN` suffix floats within a major and a full semver pins a tag"
- "--from takes the same forms and -t is an alias" in "`--from` accepts the same forms explicitly and any published module as a clone source, and `-t/--template` is an alias"

| Old clause | New place |
| --- | --- |
| A template reference that is a bare word (letters, digits, underscores... | body of "A bare word expands into the templates segment and a reference with `/` or `.` is a literal path" |
| An `@vN` suffix selects the newest release within that major (stable p... | body of "An `@vN` suffix floats within a major and a full semver pins a tag" |
| no suffix selects the default major of the official template the bare ... | body of "No suffix selects a default major for an official template only" |
| `--from` accepts the same forms explicitly and any published module as... | body of "`--from` accepts the same forms explicitly and any published module as a clone source, and `-t/--template` is an alias" |
| A template-only invocation SHALL prompt for the new module path intera... | body of "A template-only invocation prompts for the new module path interactively and refuses non-interactively" |

Words changed or added: the body ends with "(Requirement: A bare word expands into the templates segment and a reference with `/` or `.` is a literal path)", the named reference that gives it its referent; the body ends with "(Requirements: "A bare word expands into the templates segment and a reference with `/` or `.` is a literal path", "An `@vN` suffix floats within a major and a full semver pins a tag", "No suffix selects a default major for an official template only")", the named reference that gives it its referent.

Lengths (old, new): old 836; new "A bare word expands into the templates segment and a reference with `/` or `.` is a literal path" 245; "An `@vN` suffix floats within a major and a full semver pins a tag" 255; "No suffix selects a default major for an official template only" 344; "`--from` accepts the same forms explicitly and any published module as a clone source, and `-t/--template` is an alias" 378; "A template-only invocation prompts for the new module path interactively and refuses non-interactively" 107.

### Ledger: template-modules, Module tree layout

Form: c2 (REMOVED, ADDED under new names). New requirements: "A module tree contains `cue.mod/module.cue` and `identity/identity.cue`, the module body is a single CUE package at the tree root and every root `.cue` file carries the same `package` clause", "A module tree has no `values.cue` and its default values for validation and debugging live in the `debugValues` field".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Scaffold carries the mandatory files | "A module tree contains `cue.mod/module.cue` and `identity/identity.cue`, the module body is a single CUE package at the tree root and every root `.cue` file carries the same `package` clause", "Scaffold carries the mandatory files" | No (byte equal, checked) |
| No values.cue is required | "A module tree has no `values.cue` and its default values for validation and debugging live in the `debugValues` field", "No values.cue is required" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| A module tree SHALL contain `cue.mod/module.cue` (the CUE module file)... | body of "A module tree contains `cue.mod/module.cue` and `identity/identity.cue`, the module body is a single CUE package at the tree root and every root `.cue` file carries the same `package` clause" |
| The official templates use `module.cue` and `components.cue` by conven... | body of "A module tree contains `cue.mod/module.cue` and `identity/identity.cue`, the module body is a single CUE package at the tree root and every root `.cue` file carries the same `package` clause" |
| There SHALL be no `values.cue`: default values for validation and debu... | body of "A module tree has no `values.cue` and its default values for validation and debugging live in the `debugValues` field" |

Words changed or added: the body ends with "(Requirement: A module tree contains `cue.mod/module.cue` and `identity/identity.cue`, the module body is a single CUE package at the tree root and every root `.cue` file carries the same `package` clause)", the named reference that gives it its referent.

Lengths (old, new): old 537; new "A module tree contains `cue.mod/module.cue` and `identity/identity.cue`, the module body is a single CUE package at the tree root and every root `.cue` file carries the same `package` clause" 396; "A module tree has no `values.cue` and its default values for validation and debugging live in the `debugValues` field" 356.

### Ledger: dependency-tidy, Tidy produces the files cue mod tidy would

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm module tidy` and `opm catalog tidy` rewrite the dependency block of `cue.mod/module.cue` to the exact content `cue mod tidy` produces", "The exact content `cue mod tidy` produces holds every needed dependency at the version minimum version selection picks and no other", "`opm module tidy` and `opm catalog tidy` require no `cue` executable and start no child process", "`opm module tidy` and `opm catalog tidy` behave identically and only the wording of messages follows the kind", "Tidy does not read or validate the artifact's OPM identity, schema conformance or declared version".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Missing dependency is added | "The exact content `cue mod tidy` produces holds every needed dependency at the version minimum version selection picks and no other", "Missing dependency is added" | No (byte equal, checked) |
| Unused dependency is removed | "The exact content `cue mod tidy` produces holds every needed dependency at the version minimum version selection picks and no other", "Unused dependency is removed" | No (byte equal, checked) |
| Parity with cue mod tidy | "`opm module tidy` and `opm catalog tidy` rewrite the dependency block of `cue.mod/module.cue` to the exact content `cue mod tidy` produces", "Parity with cue mod tidy" | No (byte equal, checked) |
| No cue binary needed | "`opm module tidy` and `opm catalog tidy` require no `cue` executable and start no child process", "No cue binary needed" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The two commands behave identically" in "`opm module tidy` and `opm catalog tidy` behave identically and only the wording of messages follows the kind"
- "Tidy ignores the OPM identity" in "Tidy does not read or validate the artifact's OPM identity, schema conformance or declared version"

| Old clause | New place |
| --- | --- |
| `opm module tidy [path]` (alias `opm mod tidy`) and `opm catalog tidy ... | body of "`opm module tidy` and `opm catalog tidy` rewrite the dependency block of `cue.mod/module.cue` to the exact content `cue mod tidy` produces" |
| the exact content `cue mod tidy` of the CUE version the CLI embeds pro... | body of "The exact content `cue mod tidy` produces holds every needed dependency at the version minimum version selection picks and no other" |
| That means every dependency an import in the module needs, at the vers... | body of "The exact content `cue mod tidy` produces holds every needed dependency at the version minimum version selection picks and no other" |
| The command SHALL NOT require a `cue` executable on the machine and SH... | body of "`opm module tidy` and `opm catalog tidy` require no `cue` executable and start no child process" |
| The two commands SHALL behave identically; the kind only changes the w... | body of "`opm module tidy` and `opm catalog tidy` behave identically and only the wording of messages follows the kind" |
| Tidy is a CUE module operation: it SHALL NOT read or validate the arti... | body of "Tidy does not read or validate the artifact's OPM identity, schema conformance or declared version" |

Words changed or added: capital letter at the start of "The exact content `cue mod tidy` of the CUE version the CLI ..."; the body starts with "`opm module tidy [path]` (alias `opm mod tidy`) and `opm catalog tidy [path]`: " (words of the same spec, copied from the old text) so that it states its own subject.

Lengths (old, new): old 909; new "`opm module tidy` and `opm catalog tidy` rewrite the dependency block of `cue.mod/module.cue` to the exact content `cue mod tidy` produces" 317; "The exact content `cue mod tidy` produces holds every needed dependency at the version minimum version selection picks and no other" 374; "`opm module tidy` and `opm catalog tidy` require no `cue` executable and start no child process" 182; "`opm module tidy` and `opm catalog tidy` behave identically and only the wording of messages follows the kind" 169; "Tidy does not read or validate the artifact's OPM identity, schema conformance or declared version" 130.

### Ledger: dependency-tidy, Tidy targets an explicit module root

Form: c2 (REMOVED, ADDED under new names). New requirements: "The optional `path` argument of tidy names the module root and tidy searches no parent directory", "Tidy refuses a path that is not a module root with exit 2 before any registry access".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default path | "The optional `path` argument of tidy names the module root and tidy searches no parent directory", "Default path" | No (byte equal, checked) |
| Not a module root | "Tidy refuses a path that is not a module root with exit 2 before any registry access", "Not a module root" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The optional `path` argument SHALL name the module root directory and ... | body of "The optional `path` argument of tidy names the module root and tidy searches no parent directory" |
| The command SHALL tidy that directory only; it SHALL NOT search parent... | body of "The optional `path` argument of tidy names the module root and tidy searches no parent directory" |
| A path that does not exist, is not a directory, or has no `cue.mod/mod... | body of "Tidy refuses a path that is not a module root with exit 2 before any registry access" |
| The refusal names the absolute path it checked; for `opm module tidy` ... | body of "Tidy refuses a path that is not a module root with exit 2 before any registry access" |

Words changed or added: the body starts with "`opm module tidy` and `opm catalog tidy`: " (words of the same spec, copied from the old text) so that it states its own subject.

Lengths (old, new): old 570; new "The optional `path` argument of tidy names the module root and tidy searches no parent directory" 252; "Tidy refuses a path that is not a module root with exit 2 before any registry access" 360.

### Ledger: dependency-tidy, Tidy reports the outcome, not the steps

Form: c2 (REMOVED, ADDED under new names). New requirements: "On success tidy prints one line saying whether it changed anything and rewrites no file of an already-tidy module", "Output from the embedded CUE tooling reaches the terminal only through the command's own error message", "Tidy exit codes are 0 tidied, already tidy or check passed, 1 resolution or registry failure and 2 not tidy or not a module root".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Update reported | "On success tidy prints one line saying whether it changed anything and rewrites no file of an already-tidy module", "Update reported" | No (byte equal, checked) |
| Already tidy is a no-op | "On success tidy prints one line saying whether it changed anything and rewrites no file of an already-tidy module", "Already tidy is a no-op" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "Embedded tooling output stays off the terminal" in "Output from the embedded CUE tooling reaches the terminal only through the command's own error message"
- "Exit codes by outcome" in "Tidy exit codes are 0 tidied, already tidy or check passed, 1 resolution or registry failure and 2 not tidy or not a module root"

| Old clause | New place |
| --- | --- |
| On success the command SHALL print one line saying whether it changed ... | body of "On success tidy prints one line saying whether it changed anything and rewrites no file of an already-tidy module" |
| An already-tidy module SHALL NOT have any file rewritten or its mtime ... | body of "On success tidy prints one line saying whether it changed anything and rewrites no file of an already-tidy module" |
| Output from the embedded CUE tooling SHALL NOT reach the terminal exce... | body of "Output from the embedded CUE tooling reaches the terminal only through the command's own error message" |
| Exit codes: 0 tidied, already tidy, or check passed; 1 dependency reso... | body of "Tidy exit codes are 0 tidied, already tidy or check passed, 1 resolution or registry failure and 2 not tidy or not a module root" |

Words changed or added: the body starts with "`opm module tidy` and `opm catalog tidy`: " (words of the same spec, copied from the old text) so that it states its own subject.

Lengths (old, new): old 579; new "On success tidy prints one line saying whether it changed anything and rewrites no file of an already-tidy module" 346; "Output from the embedded CUE tooling reaches the terminal only through the command's own error message" 156; "Tidy exit codes are 0 tidied, already tidy or check passed, 1 resolution or registry failure and 2 not tidy or not a module root" 213.

### Ledger: e2e-cluster-workflow, The job applies to operator-facing pull requests, cascade pull requests and release pull requests

Form: c2 (REMOVED, ADDED under new names). New requirements: "The workflow runs one job named `E2E (kind, embedded operator)` on pull requests targeting `main` and on `workflow_dispatch`", "The job applies on `workflow_dispatch`, on `release-please--` and `deps/cascade` head branches and on the `deps-cascade` label", "The job applies when a pull request changes a file under `internal/operator/`, `internal/cmd/operator/`, `templates/` or `hack/platform/`", "The job applies when a pull request changes the workflow file, its decision script, `Taskfile.yml` or `hack/fixtures.sh`", "The job applies when a pull request changes `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operator-rbac.yaml` or `hack/opm-config.cue`", "The job applies when a pull request changes a Go file directly under `tests/e2e/` or anything under `tests/e2e/testdata/operator-owned/`", "The decision to apply is computed from the pull request's state, never from the triggering event alone", "A pull request that is no longer open does not apply".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Operator pin bump applies | "The job applies when a pull request changes a file under `internal/operator/`, `internal/cmd/operator/`, `templates/` or `hack/platform/`", "Operator pin bump applies" | No (byte equal, checked) |
| Release pull request applies | "The job applies on `workflow_dispatch`, on `release-please--` and `deps/cascade` head branches and on the `deps-cascade` label", "Release pull request applies" | No (byte equal, checked) |
| Label added after opening | "The job applies on `workflow_dispatch`, on `release-please--` and `deps/cascade` head branches and on the `deps-cascade` label", "Label added after opening" | No (byte equal, checked) |
| Unrelated label does not change the outcome | "The decision to apply is computed from the pull request's state, never from the triggering event alone", "Unrelated label does not change the outcome" | No (byte equal, checked) |
| Cascade branch applies without its label | "The job applies on `workflow_dispatch`, on `release-please--` and `deps/cascade` head branches and on the `deps-cascade` label", "Cascade branch applies without its label" | No (byte equal, checked) |
| Change to the preparation task applies | "The job applies when a pull request changes the workflow file, its decision script, `Taskfile.yml` or `hack/fixtures.sh`", "Change to the preparation task applies" | No (byte equal, checked) |
| Operator command change applies | "The job applies when a pull request changes a file under `internal/operator/`, `internal/cmd/operator/`, `templates/` or `hack/platform/`", "Operator command change applies" | No (byte equal, checked) |
| Merged release pull request does not apply | "A pull request that is no longer open does not apply", "Merged release pull request does not apply" | No (byte equal, checked) |
| Unrelated pull request | "The job applies when a pull request changes a file under `internal/operator/`, `internal/cmd/operator/`, `templates/` or `hack/platform/`", "Unrelated pull request" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The job is named and triggered as the ruleset requires" in "The workflow runs one job named `E2E (kind, embedded operator)` on pull requests targeting `main` and on `workflow_dispatch`"
- "Change to a kind file applies" in "The job applies when a pull request changes `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operator-rbac.yaml` or `hack/opm-config.cue`"
- "Change to a Go file under tests/e2e applies" in "The job applies when a pull request changes a Go file directly under `tests/e2e/` or anything under `tests/e2e/testdata/operator-owned/`"

| Old clause | New place |
| --- | --- |
| The workflow SHALL run one job, named `E2E (kind, embedded operator)` ... | body of "The workflow runs one job named `E2E (kind, embedded operator)` on pull requests targeting `main` and on `workflow_dispatch`" |
| The job SHALL do the cluster-backed work (it *applies*) when at least ... | body of "The job applies on `workflow_dispatch`, on `release-please--` and `deps/cascade` head branches and on the `deps-cascade` label" |
| - the run was started by `workflow_dispatch`; - the pull request's hea... | body of "The job applies on `workflow_dispatch`, on `release-please--` and `deps/cascade` head branches and on the `deps-cascade` label" |
| The job SHALL do the cluster-backed work (it *applies*) when at least ... | body of "The job applies when a pull request changes a file under `internal/operator/`, `internal/cmd/operator/`, `templates/` or `hack/platform/`" |
| - the pull request changes a file under `internal/operator/`, `interna... | body of "The job applies when a pull request changes a file under `internal/operator/`, `internal/cmd/operator/`, `templates/` or `hack/platform/`" |
| The job SHALL do the cluster-backed work (it *applies*) when at least ... | body of "The job applies when a pull request changes the workflow file, its decision script, `Taskfile.yml` or `hack/fixtures.sh`" |
| - | body of "The job applies when a pull request changes the workflow file, its decision script, `Taskfile.yml` or `hack/fixtures.sh`" |
| the pull request changes one of the job's own inputs: the workflow fil... | body of "The job applies when a pull request changes the workflow file, its decision script, `Taskfile.yml` or `hack/fixtures.sh`" |
| The job SHALL do the cluster-backed work (it *applies*) when at least ... | body of "The job applies when a pull request changes `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operator-rbac.yaml` or `hack/opm-config.cue`" |
| - | body of "The job applies when a pull request changes `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operator-rbac.yaml` or `hack/opm-config.cue`" |
| the pull request changes one of the job's own inputs: | body of "The job applies when a pull request changes `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operator-rbac.yaml` or `hack/opm-config.cue`" |
| `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operato... | body of "The job applies when a pull request changes `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operator-rbac.yaml` or `hack/opm-config.cue`" |
| The job SHALL do the cluster-backed work (it *applies*) when at least ... | body of "The job applies when a pull request changes a Go file directly under `tests/e2e/` or anything under `tests/e2e/testdata/operator-owned/`" |
| - | body of "The job applies when a pull request changes a Go file directly under `tests/e2e/` or anything under `tests/e2e/testdata/operator-owned/`" |
| the pull request changes one of the job's own inputs: | body of "The job applies when a pull request changes a Go file directly under `tests/e2e/` or anything under `tests/e2e/testdata/operator-owned/`" |
| any Go file directly under `tests/e2e/` (the cluster-backed tests and ... | body of "The job applies when a pull request changes a Go file directly under `tests/e2e/` or anything under `tests/e2e/testdata/operator-owned/`" |
| The decision SHALL be computed from the pull request's state when the ... | body of "The decision to apply is computed from the pull request's state, never from the triggering event alone" |
| A pull request that is no longer open (read live: closed or merged) SH... | body of "A pull request that is no longer open does not apply" |

Words changed or added: the body gains "The job *applies*:".

Lengths (old, new): old 1833; new "The workflow runs one job named `E2E (kind, embedded operator)` on pull requests targeting `main` and on `workflow_dispatch`" 325; "The job applies on `workflow_dispatch`, on `release-please--` and `deps/cascade` head branches and on the `deps-cascade` label" 334; "The job applies when a pull request changes a file under `internal/operator/`, `internal/cmd/operator/`, `templates/` or `hack/platform/`" 211; "The job applies when a pull request changes the workflow file, its decision script, `Taskfile.yml` or `hack/fixtures.sh`" 352; "The job applies when a pull request changes `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operator-rbac.yaml` or `hack/opm-config.cue`" 252; "The job applies when a pull request changes a Go file directly under `tests/e2e/` or anything under `tests/e2e/testdata/operator-owned/`" 339; "The decision to apply is computed from the pull request's state, never from the triggering event alone" 299; "A pull request that is no longer open does not apply" 205.

### Ledger: e2e-cluster-workflow, The suite runs with the cluster required and leaves evidence on failure

Form: c2 (REMOVED, ADDED under new names). New requirements: "The suite runs `go test ./tests/e2e/... -v` with the cluster required and the tools the cluster tests probe for present", "An operator-owned test that fails logs its ModuleInstance, the cluster Platform and the operator's logs before its cleanup deletes the evidence", "When an operator-owned test fails, a test that times out waiting for the operator to reconcile names the ModuleInstance's `Ready` condition in its failure message", "After any failed step the job prints the operator's logs, the cluster Platform, every ModuleInstance and recent cluster events".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A cluster test cannot skip | "The suite runs `go test ./tests/e2e/... -v` with the cluster required and the tools the cluster tests probe for present", "A cluster test cannot skip" | No (byte equal, checked) |
| Lifecycle test takes the full-rollout path | "The suite runs `go test ./tests/e2e/... -v` with the cluster required and the tools the cluster tests probe for present", "Lifecycle test takes the full-rollout path" | No (byte equal, checked) |
| Failure leaves diagnostics | "An operator-owned test that fails logs its ModuleInstance, the cluster Platform and the operator's logs before its cleanup deletes the evidence", "Failure leaves diagnostics" | No (byte equal, checked) |
| Preparation failure leaves diagnostics | "After any failed step the job prints the operator's logs, the cluster Platform, every ModuleInstance and recent cluster events", "Preparation failure leaves diagnostics" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "A reconcile timeout names the Ready condition" in "When an operator-owned test fails, a test that times out waiting for the operator to reconcile names the ModuleInstance's `Ready` condition in its failure message"

| Old clause | New place |
| --- | --- |
| When the job applies, it SHALL run `go test ./tests/e2e/... -v` with t... | body of "The suite runs `go test ./tests/e2e/... -v` with the cluster required and the tools the cluster tests probe for present" |
| When an operator-owned test fails, the test itself SHALL log, before i... | body of "An operator-owned test that fails logs its ModuleInstance, the cluster Platform and the operator's logs before its cleanup deletes the evidence" |
| Later tests (the operator lifecycle test tears the operator, CRDs and ... | body of "An operator-owned test that fails logs its ModuleInstance, the cluster Platform and the operator's logs before its cleanup deletes the evidence" |
| When an operator-owned test fails, | body of "When an operator-owned test fails, a test that times out waiting for the operator to reconcile names the ModuleInstance's `Ready` condition in its failure message" |
| a test that times out waiting for the operator to reconcile SHALL name... | body of "When an operator-owned test fails, a test that times out waiting for the operator to reconcile names the ModuleInstance's `Ready` condition in its failure message" |
| After any failed step, including a preparation step before the suite s... | body of "After any failed step the job prints the operator's logs, the cluster Platform, every ModuleInstance and recent cluster events" |

Words changed or added: none.

Lengths (old, new): old 1075; new "The suite runs `go test ./tests/e2e/... -v` with the cluster required and the tools the cluster tests probe for present" 346; "An operator-owned test that fails logs its ModuleInstance, the cluster Platform and the operator's logs before its cleanup deletes the evidence" 375; "When an operator-owned test fails, a test that times out waiting for the operator to reconcile names the ModuleInstance's `Ready` condition in its failure message" 168; "After any failed step the job prints the operator's logs, the cluster Platform, every ModuleInstance and recent cluster events" 214.

### Ledger: e2e-cluster-workflow, The cluster is prepared with the local loop's own tasks and the pinned operator module

Form: c2 (REMOVED, ADDED under new names). New requirements: "The job installs kind, creates the `opm-dev` cluster with `task cluster:create` and prepares it with `task cluster:operator`", "The job does not restate the kind installation, cluster creation and operator preparation steps in the workflow", "The job runs a registry container seeded with `hack/fixtures.sh seed` from which the CLI and the operator resolve `testing.opmodel.dev`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The pinned operator module is what runs | "The job installs kind, creates the `opm-dev` cluster with `task cluster:create` and prepares it with `task cluster:operator`", "The pinned operator module is what runs" | No (byte equal, checked) |
| Bumped fixture before it is published | "The job runs a registry container seeded with `hack/fixtures.sh seed` from which the CLI and the operator resolve `testing.opmodel.dev`", "Bumped fixture before it is published" | No (byte equal, checked) |
| Stalled Platform fails the preparation | "The job installs kind, creates the `opm-dev` cluster with `task cluster:create` and prepares it with `task cluster:operator`", "Stalled Platform fails the preparation" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The preparation steps are not restated" in "The job does not restate the kind installation, cluster creation and operator preparation steps in the workflow"

| Old clause | New place |
| --- | --- |
| When the job applies, it SHALL install kind at the version and checksu... | body of "The job installs kind, creates the `opm-dev` cluster with `task cluster:create` and prepares it with `task cluster:operator`" |
| The job SHALL NOT restate those steps in the workflow. | body of "The job does not restate the kind installation, cluster creation and operator preparation steps in the workflow" |
| The job SHALL run a registry container seeded from the tree with `hack... | body of "The job runs a registry container seeded with `hack/fixtures.sh seed` from which the CLI and the operator resolve `testing.opmodel.dev`" |

Words changed or added: the body ends with "(Requirement: The job installs kind, creates the `opm-dev` cluster with `task cluster:create` and prepares it with `task cluster:operator`)", the named reference that gives it its referent.

Lengths (old, new): old 975; new "The job installs kind, creates the `opm-dev` cluster with `task cluster:create` and prepares it with `task cluster:operator`" 469; "The job does not restate the kind installation, cluster creation and operator preparation steps in the workflow" 194; "The job runs a registry container seeded with `hack/fixtures.sh seed` from which the CLI and the operator resolve `testing.opmodel.dev`" 450.

## Ledger entries of cli-B6, unit 6b (errors-domain, flag-conventions, inventory-ownership, log-output-style, apply-preflight-gates)

### Ledger: errors-domain, Error types package location

Form: c2 (REMOVED, ADDED under new names). New requirements: "All error types are defined in `pkg/errors/` and the package exports the named types and sentinels", "Types that no code constructs are removed and every exported type keeps its `.Error()` string and `Unwrap()` behavior across moves".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Error types importable from pkg/errors | "All error types are defined in `pkg/errors/` and the package exports the named types and sentinels", "Error types importable from pkg/errors" | No (byte equal, checked) |
| Import alias convention | "All error types are defined in `pkg/errors/` and the package exports the named types and sentinels", "Import alias convention" | No (byte equal, checked) |
| ValidationError wrapping is unchanged | "Types that no code constructs are removed and every exported type keeps its `.Error()` string and `Unwrap()` behavior across moves", "ValidationError wrapping is unchanged" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| All error types SHALL be defined in `pkg/errors/`. | body of "All error types are defined in `pkg/errors/` and the package exports the named types and sentinels" |
| The package SHALL export `DetailError` ... | body of "All error types are defined in `pkg/errors/` and the package exports the named types and sentinels" |
| Types that no code in the tree or its consumers constructs ... are removed rather than preserved. | body of "Types that no code constructs are removed and every exported type keeps its `.Error()` string and `Unwrap()` behavior across moves" |
| Every exported type SHALL keep its `.Error()` string and `Unwrap()` behavior ... | body of "Types that no code constructs are removed and every exported type keeps its `.Error()` string and `Unwrap()` behavior across moves" |

Lengths (old, new): old 570; new "All error types are defined in `pkg/errors/` and the package exports the named types and sentinels" 311; "Types that no code constructs are removed and every exported type keeps its `.Error()` string and `Unwrap()` behavior across moves" 258.

### Ledger: errors-domain, Fetch and resolution answers are decided by error type

Form: c2 (REMOVED, ADDED under new names). New requirements: "The cli decides what a registry fetch or dependency resolution failure means from the typed error and reads a CUE evaluation error by its path", "Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate", "The error-text test reads the source and not its types".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A predecessor package with an unprovided import reads as absent | "The cli decides what a registry fetch or dependency resolution failure means from the typed error and reads a CUE evaluation error by its path", "A predecessor package with an unprovided import reads as absent" | No (byte equal, checked) |
| A registry failure never reads as absent | "The cli decides what a registry fetch or dependency resolution failure means from the typed error and reads a CUE evaluation error by its path", "A registry failure never reads as absent" | No (byte equal, checked) |
| A new text match is refused | "Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate", "A new text match is refused" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "A listed text read is allowed" in "Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate"
- "A message made by formatting the error is not seen" in "The error-text test reads the source and not its types"

| Old clause | New place |
| --- | --- |
| The cli SHALL decide what a registry fetch failure or a dependency resolution failure means ... from the typed error ... and SHALL NOT decide it from the text | body of "The cli decides what a registry fetch or dependency resolution failure means from the typed error and reads a CUE evaluation error by its path" |
| Source line (whole) | body of "The cli decides what a registry fetch or dependency resolution failure means from the typed error and reads a CUE evaluation error by its path" (as `**Source**`); body of "Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate" (as `**Source**`); body of "The error-text test reads the source and not its types" (as `**Source**`) |
| Where the cause is a CUE evaluation error, the cli SHALL read the error's path, never its text. | body of "The cli decides what a registry fetch or dependency resolution failure means from the typed error and reads a CUE evaluation error by its path" |
| Non-test cli code MAY read error text only in these cases ... (four bullets) | body of "Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate" |
| A test SHALL fail when non-test cli code gains an error-text predicate outside that list. | body of "Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate" |
| The test reads the source, not its types ... (two sentences) | body of "The error-text test reads the source and not its types" |

Lengths (old, new): old 1225; new "The cli decides what a registry fetch or dependency resolution failure means from the typed error and reads a CUE evaluation error by its path" 332; "Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate" 497; "The error-text test reads the source and not its types" 440.

### Ledger: errors-domain, Platform module build failure hints

Form: c2 (REMOVED, ADDED under new names). New requirements: "Platform build hint for the package shape and the `#registry` key", "Platform build hint for a refused credential", "Platform build hint for another fetch failure or an unresolved dependency", "Platform build hint for anything else", "A platform module build failure hint is read from the type of the error and only a refused credential changes the exit code".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| An unpublished catalog pin | "Platform build hint for another fetch failure or an unresolved dependency", "An unpublished catalog pin" | No (byte equal, checked) |
| A refused registry | "Platform build hint for another fetch failure or an unresolved dependency", "A refused registry" | No (byte equal, checked) |
| A refused credential | "Platform build hint for a refused credential", "A refused credential" | No (byte equal, checked) |
| A refused credential under a prefix mapping | "Platform build hint for a refused credential", "A refused credential under a prefix mapping" | No (byte equal, checked) |
| A refused credential with dependencies on several registry hosts | "Platform build hint for a refused credential", "A refused credential with dependencies on several registry hosts" | No (byte equal, checked) |
| A refusal the cli receives as not found | "Platform build hint for another fetch failure or an unresolved dependency", "A refusal the cli receives as not found" | No (byte equal, checked) |
| A token endpoint that refuses the credentials | "Platform build hint for a refused credential", "A token endpoint that refuses the credentials" | No (byte equal, checked) |
| A directly imported dependency whose module file does not parse | "Platform build hint for another fetch failure or an unresolved dependency", "A directly imported dependency whose module file does not parse" | No (byte equal, checked) |
| An import whose package name does not match | "Platform build hint for anything else", "An import whose package name does not match" | No (byte equal, checked) |
| A registry key that differs from the imported catalog | "Platform build hint for the package shape and the `#registry` key", "A registry key that differs from the imported catalog" | No (byte equal, checked) |
| A registry entry the shape check refuses | "Platform build hint for anything else", "A registry entry the shape check refuses" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The package is not a single `#Platform` package" in "Platform build hint for the package shape and the `#registry` key"
- "A refused credential exits 4 and every other cause exits 2" in "A platform module build failure hint is read from the type of the error and only a refused credential changes the exit code"

| Old clause | New place |
| --- | --- |
| When a platform module fails to build, the cli SHALL add one hint chosen from the cause: (intro) | body of "Platform build hint for the package shape and the `#registry` key"; body of "Platform build hint for a refused credential"; body of "Platform build hint for another fetch failure or an unresolved dependency"; body of "Platform build hint for anything else" |
| bullet: the package is not a single `#Platform` package | body of "Platform build hint for the package shape and the `#registry` key" |
| bullet: a CUE evaluation error at a path under `#registry` | body of "Platform build hint for the package shape and the `#registry` key" |
| bullet: the registry refused the credentials ... | body of "Platform build hint for a refused credential" |
| bullet: any other registry fetch failed, or a dependency did not resolve ... | body of "Platform build hint for another fetch failure or an unresolved dependency" |
| bullet: anything else ... | body of "Platform build hint for anything else" |
| The cause SHALL be read from the type of the error, never from its text. | body of "A platform module build failure hint is read from the type of the error and only a refused credential changes the exit code" |
| A refused credential SHALL exit with the permission exit code 4. | body of "A platform module build failure hint is read from the type of the error and only a refused credential changes the exit code" |
| Every other cause SHALL exit with the validation exit code 2, and its hint SHALL NOT change the exit code. | body of "A platform module build failure hint is read from the type of the error and only a refused credential changes the exit code" |

Lengths (old, new): old 1331; new "Platform build hint for the package shape and the `#registry` key" 350; "Platform build hint for a refused credential" 480; "Platform build hint for another fetch failure or an unresolved dependency" 393; "Platform build hint for anything else" 417; "A platform module build failure hint is read from the type of the error and only a refused credential changes the exit code" 283.

### Ledger: flag-conventions, Force overrides a protective refusal and never answers a prompt

Form: c2 (REMOVED, ADDED under new names). New requirements: "Force carries out what a command otherwise refuses in order to protect existing state", "No command offers a non-deprecated `--force` whose effect is to skip a confirmation prompt".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Force is not the prompt flag of instance delete | "No command offers a non-deprecated `--force` whose effect is to skip a confirmation prompt", "Force is not the prompt flag of instance delete" | No (byte equal, checked) |
| Force keeps its meaning on the commands that refuse | "Force carries out what a command otherwise refuses in order to protect existing state", "Force keeps its meaning on the commands that refuse" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| On every command that offers `--force` without deprecation, the flag SHALL mean one thing ... | body of "Force carries out what a command otherwise refuses in order to protect existing state" |
| `opm instance apply` and `opm module apply` SHALL prune ... `opm platform pull` ... `opm config init` ... | body of "Force carries out what a command otherwise refuses in order to protect existing state" |
| `--force` SHALL default to false. | body of "Force carries out what a command otherwise refuses in order to protect existing state" |
| No command SHALL offer a non-deprecated `--force` whose effect is to skip a confirmation prompt. | body of "No command offers a non-deprecated `--force` whose effect is to skip a confirmation prompt" |

Lengths (old, new): old 595; new "Force carries out what a command otherwise refuses in order to protect existing state" 498; "No command offers a non-deprecated `--force` whose effect is to skip a confirmation prompt" 96.

### Ledger: flag-conventions, Version and template list print machine-readable output

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm version` accepts `-o`/`--output` with exactly the values `text`, `json` and `yaml`", "`opm module template list` accepts `-o`/`--output` with exactly the values `table`, `json` and `yaml`", "Any other `--output` value exits 1 with an invalid output format message and prints nothing on standard output".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Version as JSON | "`opm version` accepts `-o`/`--output` with exactly the values `text`, `json` and `yaml`", "Version as JSON" | No (byte equal, checked) |
| Version default is unchanged | "`opm version` accepts `-o`/`--output` with exactly the values `text`, `json` and `yaml`", "Version default is unchanged" | No (byte equal, checked) |
| Template list as JSON | "`opm module template list` accepts `-o`/`--output` with exactly the values `table`, `json` and `yaml`", "Template list as JSON" | No (byte equal, checked) |
| Invalid format | "Any other `--output` value exits 1 with an invalid output format message and prints nothing on standard output", "Invalid format" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm version` SHALL accept `-o`/`--output` ... `text` ... `json` and `yaml` ... fields | body of "`opm version` accepts `-o`/`--output` with exactly the values `text`, `json` and `yaml`" |
| `opm module template list` SHALL accept ... `json` and `yaml` SHALL print a list ... | body of "`opm module template list` accepts `-o`/`--output` with exactly the values `table`, `json` and `yaml`" |
| On both commands any other value SHALL exit 1 with the message ... | body of "Any other `--output` value exits 1 with an invalid output format message and prints nothing on standard output" |

Lengths (old, new): old 759; new "`opm version` accepts `-o`/`--output` with exactly the values `text`, `json` and `yaml`" 318; "`opm module template list` accepts `-o`/`--output` with exactly the values `table`, `json` and `yaml`" 273; "Any other `--output` value exits 1 with an invalid output format message and prints nothing on standard output" 386.

### Ledger: flag-conventions, Delete-data is the only flag that allows deletion of PersistentVolumeClaims

Form: c2 (REMOVED, ADDED under new names). New requirements: "`--delete-data` is the only flag that deletes tracked PersistentVolumeClaims and it defaults to false", "The help of each command that accepts `--delete-data` says that PersistentVolumeClaims are kept by default".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The flag is offered with a safe default | "`--delete-data` is the only flag that deletes tracked PersistentVolumeClaims and it defaults to false", "The flag is offered with a safe default" | No (byte equal, checked) |
| Yes alone keeps claims | "`--delete-data` is the only flag that deletes tracked PersistentVolumeClaims and it defaults to false", "Yes alone keeps claims" | No (byte equal, checked) |
| Force alone keeps claims | "`--delete-data` is the only flag that deletes tracked PersistentVolumeClaims and it defaults to false", "Force alone keeps claims" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "The help names the flag that deletes claims" in "The help of each command that accepts `--delete-data` says that PersistentVolumeClaims are kept by default"

| Old clause | New place |
| --- | --- |
| `opm instance delete`, `opm instance apply` and `opm module apply` SHALL each accept `--delete-data` ... | body of "`--delete-data` is the only flag that deletes tracked PersistentVolumeClaims and it defaults to false" |
| No other flag SHALL have that effect: `--yes` ... `--force` ... | body of "`--delete-data` is the only flag that deletes tracked PersistentVolumeClaims and it defaults to false" |
| The help of each of the three commands SHALL say ... and SHALL name `--delete-data`. | body of "The help of each command that accepts `--delete-data` says that PersistentVolumeClaims are kept by default" |

Lengths (old, new): old 503; new "`--delete-data` is the only flag that deletes tracked PersistentVolumeClaims and it defaults to false" 374; "The help of each command that accepts `--delete-data` says that PersistentVolumeClaims are kept by default" 245.

### Ledger: inventory-ownership, Thin-editor apply on operator-owned instances

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm instance apply` against an operator-owned instance acts as a spec editor only", "A module resolving from local bytes is refused in thin-editor mode", "A thin-editor apply carries the live `spec.owner` value and never changes it".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The operator's ownership survives a values edit | "A thin-editor apply carries the live `spec.owner` value and never changes it", "The operator's ownership survives a values edit" | No (byte equal, checked) |
| Values update on an operator-owned instance | "`opm instance apply` against an operator-owned instance acts as a spec editor only", "Values update on an operator-owned instance" | No (byte equal, checked) |
| Local module refused in thin-editor mode | "A module resolving from local bytes is refused in thin-editor mode", "Local module refused in thin-editor mode" | No (byte equal, checked) |
| No status writes in thin-editor mode | "`opm instance apply` against an operator-owned instance acts as a spec editor only", "No status writes in thin-editor mode" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm instance apply` against an operator-owned instance SHALL act as a spec editor only ... | body of "`opm instance apply` against an operator-owned instance acts as a spec editor only" |
| In this mode the CLI MUST NOT render-and-apply resources, prune, write `status`, or run the status-RBAC pre-flight. | body of "`opm instance apply` against an operator-owned instance acts as a spec editor only" |
| The version-skew ceiling gate still applies. | body of "`opm instance apply` against an operator-owned instance acts as a spec editor only" |
| A module resolving from local bytes ... SHALL be refused in this mode | body of "A module resolving from local bytes is refused in thin-editor mode" |
| The applied document SHALL carry the instance's current `spec.owner` value ... | body of "A thin-editor apply carries the live `spec.owner` value and never changes it" |
| Omitting it does not leave ownership untouched ... | body of "A thin-editor apply carries the live `spec.owner` value and never changes it" |
| The CLI SHALL NOT change the owner's value in this mode. | body of "A thin-editor apply carries the live `spec.owner` value and never changes it" |

Lengths (old, new): old 970; new "`opm instance apply` against an operator-owned instance acts as a spec editor only" 464; "A module resolving from local bytes is refused in thin-editor mode" 267; "A thin-editor apply carries the live `spec.owner` value and never changes it" 432.

### Ledger: inventory-ownership, Ownership mode resolution is a single branch point

Form: c2 (REMOVED, ADDED under new names). New requirements: "Ownership mode resolution is implemented in exactly one function that all mutating commands consume", "The CLI never changes an instance's `spec.owner`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Apply and delete share the resolver | "Ownership mode resolution is implemented in exactly one function that all mutating commands consume", "Apply and delete share the resolver" | No (byte equal, checked) |
| Modes route to distinct paths | "Ownership mode resolution is implemented in exactly one function that all mutating commands consume", "Modes route to distinct paths" | No (byte equal, checked) |
| No CLI command changes the owner | "The CLI never changes an instance's `spec.owner`", "No CLI command changes the owner" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The mapping from `spec.owner` to a CLI execution mode SHALL be implemented in exactly one function ... | body of "Ownership mode resolution is implemented in exactly one function that all mutating commands consume" |
| CLI-executor mode (absent CR or `spec.owner: cli`) drives ... | body of "Ownership mode resolution is implemented in exactly one function that all mutating commands consume" |
| The CLI SHALL NOT change an instance's `spec.owner`: `apply` writes `cli` ... | body of "The CLI never changes an instance's `spec.owner`" |

Lengths (old, new): old 557; new "Ownership mode resolution is implemented in exactly one function that all mutating commands consume" 343; "The CLI never changes an instance's `spec.owner`" 213.

### Ledger: inventory-ownership, Operator-owned delete hands cleanup to the operator's finalizer

Form: c2 (REMOVED, ADDED under new names). New requirements: "Operator-owned instance delete hands cleanup to the finalizer", "`opm instance delete` verifies that the operator is ready before it deletes an operator-owned instance and refuses when it is not", "`opm instance delete` of an operator-owned instance reports the outcome that actually occurred and never claims a prune it has not established".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Delete without spec.prune reports the orphaning | "`opm instance delete` of an operator-owned instance reports the outcome that actually occurred and never claims a prune it has not established", "Delete without spec.prune reports the orphaning" | No (byte equal, checked) |
| Delete of an operator-owned instance | "Operator-owned instance delete hands cleanup to the finalizer", "Delete of an operator-owned instance" | No (byte equal, checked) |
| Delete refused while the operator is down | "`opm instance delete` verifies that the operator is ready before it deletes an operator-owned instance and refuses when it is not", "Delete refused while the operator is down" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm instance delete` against an operator-owned instance SHALL delete the `ModuleInstance` CR ... | body of "Operator-owned instance delete hands cleanup to the finalizer" |
| Before deleting, the CLI SHALL verify the operator is ready and refuse when it is not ... | body of "`opm instance delete` verifies that the operator is ready before it deletes an operator-owned instance and refuses when it is not" |
| The readiness check SHALL locate the operator by its fixed names ... | body of "`opm instance delete` verifies that the operator is ready before it deletes an operator-owned instance and refuses when it is not" |
| Whether the operator removes the workloads is governed by `spec.prune` ... | body of "`opm instance delete` of an operator-owned instance reports the outcome that actually occurred and never claims a prune it has not established" |
| The CR's disappearance therefore proves the finalizer completed ... | body of "`opm instance delete` of an operator-owned instance reports the outcome that actually occurred and never claims a prune it has not established" |
| The CLI SHALL report the outcome that actually occurred and MUST NOT claim a prune it has not established. | body of "`opm instance delete` of an operator-owned instance reports the outcome that actually occurred and never claims a prune it has not established" |

Lengths (old, new): old 948; new "Operator-owned instance delete hands cleanup to the finalizer" 216; "`opm instance delete` verifies that the operator is ready before it deletes an operator-owned instance and refuses when it is not" 395; "`opm instance delete` of an operator-owned instance reports the outcome that actually occurred and never claims a prune it has not established" 488.

### Ledger: log-output-style, Centralized color palette for human log output

Form: c2 (REMOVED, ADDED under new names). New requirements: "The `internal/output` package defines named color constants and all human-readable styled output references them", "The palette includes six named colors for nouns, statuses and completion checkmarks".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Color constants are defined and accessible | "The `internal/output` package defines named color constants and all human-readable styled output references them", "Color constants are defined and accessible" | No (byte equal, checked) |
| All status colors follow the visibility hierarchy | "The palette includes six named colors for nouns, statuses and completion checkmarks", "All status colors follow the visibility hierarchy" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The `internal/output` package SHALL define a centralized set of named color constants ... | body of "The `internal/output` package defines named color constants and all human-readable styled output references them" |
| All human-readable styled output SHALL reference these constants ... | body of "The `internal/output` package defines named color constants and all human-readable styled output references them" |
| The palette SHALL include: (six bullets) | body of "The palette includes six named colors for nouns, statuses and completion checkmarks" |

Lengths (old, new): old 549; new "The `internal/output` package defines named color constants and all human-readable styled output references them" 213; "The palette includes six named colors for nouns, statuses and completion checkmarks" 463.

### Ledger: log-output-style, Semantic style constructors

Form: c2 (REMOVED, ADDED under new names). New requirements: "The output package provides semantic style functions that map domain concepts to visual styles", "`StatusStyle` returns the appropriate style for a given resource status string".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| StatusStyle returns correct style for each status | "`StatusStyle` returns the appropriate style for a given resource status string", "StatusStyle returns correct style for each status" | No (byte equal, checked) |
| Unknown status falls back to default style | "`StatusStyle` returns the appropriate style for a given resource status string", "Unknown status falls back to default style" | No (byte equal, checked) |

New scenarios (each restates only a case the old text names):

- "Noun style is a cyan foreground" in "The output package provides semantic style functions that map domain concepts to visual styles"

| Old clause | New place |
| --- | --- |
| The output package SHALL provide semantic style functions ... | body of "The output package provides semantic style functions that map domain concepts to visual styles" |
| The following semantic styles SHALL be defined: (four bullets) | body of "The output package provides semantic style functions that map domain concepts to visual styles" |
| A `StatusStyle(status string) lipgloss.Style` function SHALL return the appropriate style ... | body of "`StatusStyle` returns the appropriate style for a given resource status string" |

Lengths (old, new): old 606; new "The output package provides semantic style functions that map domain concepts to visual styles" 480; "`StatusStyle` returns the appropriate style for a given resource status string" 125.

### Ledger: log-output-style, Validation check line formatting

Form: c2 (REMOVED, ADDED under new names). New requirements: "`FormatVetCheck` renders a validation check result with a green checkmark, a label and an optional right-aligned detail string", "The detail text of a validation check line aligns at column 34", "A validation check line with an empty detail renders only the checkmark and label".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Check line with detail renders aligned output | "`FormatVetCheck` renders a validation check result with a green checkmark, a label and an optional right-aligned detail string", "Check line with detail renders aligned output" | No (byte equal, checked) |
| Check line without detail renders cleanly | "A validation check line with an empty detail renders only the checkmark and label", "Check line without detail renders cleanly" | No (byte equal, checked) |
| Multiple check lines align consistently | "The detail text of a validation check line aligns at column 34", "Multiple check lines align consistently" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The output package SHALL provide a `FormatVetCheck(label, detail string)` function ... | body of "`FormatVetCheck` renders a validation check result with a green checkmark, a label and an optional right-aligned detail string" |
| The format SHALL be: `✔ <label>` followed by right-aligned dim detail text when provided. | body of "`FormatVetCheck` renders a validation check result with a green checkmark, a label and an optional right-aligned detail string" |
| The checkmark SHALL use the existing `colorGreenCheck` (`10`) constant. | body of "`FormatVetCheck` renders a validation check result with a green checkmark, a label and an optional right-aligned detail string" |
| The detail text SHALL be rendered in dim/faint style ... | body of "`FormatVetCheck` renders a validation check result with a green checkmark, a label and an optional right-aligned detail string" |
| The alignment column for detail text SHALL be 34 characters ... | body of "The detail text of a validation check line aligns at column 34" |
| When `detail` is empty, the function SHALL render only the checkmark and label ... | body of "A validation check line with an empty detail renders only the checkmark and label" |

Lengths (old, new): old 730; new "`FormatVetCheck` renders a validation check result with a green checkmark, a label and an optional right-aligned detail string" 474; "The detail text of a validation check line aligns at column 34" 289; "A validation check line with an empty detail renders only the checkmark and label" 249.

### Ledger: apply-preflight-gates, Operator-version ceiling

Form: c2 (REMOVED, ADDED under new names). New requirements: "Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only", "Apply refuses with an upgrade-the-CLI error when the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR", "The operator-version ceiling check is skipped when the Platform or the field is absent or when reading the Platform fails due to RBAC", "The operator-version ceiling check is skipped with a warning when the CLI version or the operator version is not valid semver", "The instance apply that `opm operator install` performs for the operator's own instance skips the operator-version ceiling check", "Every other apply keeps the operator-version ceiling check and the operator's instance still runs the CRD gates".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Solo cluster skips the ceiling | "The operator-version ceiling check is skipped when the Platform or the field is absent or when reading the Platform fails due to RBAC", "Solo cluster skips the ceiling" | No (byte equal, checked) |
| Older CLI refused | "Apply refuses with an upgrade-the-CLI error when the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR", "Older CLI refused" | No (byte equal, checked) |
| Dev build skips with warning | "The operator-version ceiling check is skipped with a warning when the CLI version or the operator version is not valid semver", "Dev build skips with warning" | No (byte equal, checked) |
| Platform read denied degrades to warning | "The operator-version ceiling check is skipped when the Platform or the field is absent or when reading the Platform fails due to RBAC", "Platform read denied degrades to warning" | No (byte equal, checked) |
| Newer prerelease counter on the same line passes | "Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only", "Newer prerelease counter on the same line passes" | No (byte equal, checked) |
| Beta operator against an alpha CLI of the same line passes | "Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only", "Beta operator against an alpha CLI of the same line passes" | No (byte equal, checked) |
| Newer operator patch passes | "Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only", "Newer operator patch passes" | No (byte equal, checked) |
| Newer operator minor refused | "Apply refuses with an upgrade-the-CLI error when the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR", "Newer operator minor refused" | No (byte equal, checked) |
| Older CLI repairs a newer operator | "The instance apply that `opm operator install` performs for the operator's own instance skips the operator-version ceiling check", "Older CLI repairs a newer operator" | No (byte equal, checked) |
| Ordinary apply against the same cluster still refused | "Every other apply keeps the operator-version ceiling check and the operator's instance still runs the CRD gates", "Ordinary apply against the same cluster still refused" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| Before applying, the CLI SHALL read the cluster-scoped singleton `Platform` and compare ... MAJOR.MINOR only ... SHALL be ignored on both sides ... | body of "Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only" |
| If the Platform or the field is absent, the check SHALL be skipped (solo cluster semantics). | body of "The operator-version ceiling check is skipped when the Platform or the field is absent or when reading the Platform fails due to RBAC" |
| If the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR, apply SHALL refuse ... name both full versions as read. | body of "Apply refuses with an upgrade-the-CLI error when the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR" |
| If the CLI's own version is not valid semver (dev build), the check SHALL be skipped with a warning. | body of "The operator-version ceiling check is skipped with a warning when the CLI version or the operator version is not valid semver" |
| If the operator version is not valid semver, the check SHALL be skipped with a warning. | body of "The operator-version ceiling check is skipped with a warning when the CLI version or the operator version is not valid semver" |
| If reading the Platform fails due to RBAC, the check SHALL be skipped with a warning ... | body of "The operator-version ceiling check is skipped when the Platform or the field is absent or when reading the Platform fails due to RBAC" |
| The instance apply that `opm operator install` performs ... SHALL skip this check ... | body of "The instance apply that `opm operator install` performs for the operator's own instance skips the operator-version ceiling check" |
| Every other apply keeps the check, and the CRD-presence and CRD field-floor gates still run ... | body of "Every other apply keeps the operator-version ceiling check and the operator's instance still runs the CRD gates" |
| Source line (whole) | body of "Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only" (as `**Source**`); body of "Apply refuses with an upgrade-the-CLI error when the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR" (as `**Source**`); body of "The operator-version ceiling check is skipped when the Platform or the field is absent or when reading the Platform fails due to RBAC" (as `**Source**`); body of "The operator-version ceiling check is skipped with a warning when the CLI version or the operator version is not valid semver" (as `**Source**`); body of "The instance apply that `opm operator install` performs for the operator's own instance skips the operator-version ceiling check" (as `**Source**`); body of "Every other apply keeps the operator-version ceiling check and the operator's instance still runs the CRD gates" (as `**Source**`) |

Lengths (old, new): old 1522; new "Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only" 357; "Apply refuses with an upgrade-the-CLI error when the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR" 314; "The operator-version ceiling check is skipped when the Platform or the field is absent or when reading the Platform fails due to RBAC" 357; "The operator-version ceiling check is skipped with a warning when the CLI version or the operator version is not valid semver" 315; "The instance apply that `opm operator install` performs for the operator's own instance skips the operator-version ceiling check" 465; "Every other apply keeps the operator-version ceiling check and the operator's instance still runs the CRD gates" 275.

### Ledger: apply-preflight-gates, Gate ordering and dry-run exemption

Form: c2 (REMOVED, ADDED under new names). New requirements: "The gates run in a fixed order, all before the first resource write", "Dry-run applies skip four gates and still resolve ownership and run the ownership guard".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Gates precede all writes | "The gates run in a fixed order, all before the first resource write", "Gates precede all writes" | No (byte equal, checked) |
| Dry-run skips gates | "Dry-run applies skip four gates and still resolve ownership and run the ownership guard", "Dry-run skips gates" | No (byte equal, checked) |
| Dry-run runs the ownership guard | "Dry-run applies skip four gates and still resolve ownership and run the ownership guard", "Dry-run runs the ownership guard" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The gates SHALL run in the order: ... all before the first resource write. | body of "The gates run in a fixed order, all before the first resource write" |
| Dry-run applies SHALL skip the CRD presence, CRD field floor, operator-version ceiling and status-RBAC gates ... | body of "Dry-run applies skip four gates and still resolve ownership and run the ownership guard" |
| A dry run SHALL still resolve ownership and SHALL still run the ownership guard ... | body of "Dry-run applies skip four gates and still resolve ownership and run the ownership guard" |

Lengths (old, new): old 639; new "The gates run in a fixed order, all before the first resource write" 329; "Dry-run applies skip four gates and still resolve ownership and run the ownership guard" 309.


### Words changed or added in unit 6b

- Named references: a body ends with "(Requirement: <exact header>)", the named reference that gives it its referent ("that list", "On both commands", "this mode", "the palette", "the function", "this check", "other", "anything else", "the operator-owned instance"). The bodies are: `errors-domain` "The error-text test reads the source and not its types" (to the allowed-cases requirement), "Platform build hint for another fetch failure or an unresolved dependency" (to the refused-credential hint) and "Platform build hint for anything else" (to the other three hints); `flag-conventions` "Any other `--output` value exits 1 with an invalid output format message and prints nothing on standard output" (to both `--output` requirements) and "The help of each command that accepts `--delete-data` says that PersistentVolumeClaims are kept by default"; `inventory-ownership` "A module resolving from local bytes is refused in thin-editor mode", "A thin-editor apply carries the live `spec.owner` value and never changes it", "`opm instance delete` verifies that the operator is ready before it deletes an operator-owned instance and refuses when it is not" and "`opm instance delete` of an operator-owned instance reports the outcome that actually occurred and never claims a prune it has not established" (the last two to "Operator-owned instance delete hands cleanup to the finalizer"); `log-output-style` "The palette includes six named colors for nouns, statuses and completion checkmarks" (on its own line after a blank line), "The detail text of a validation check line aligns at column 34" and "A validation check line with an empty detail renders only the checkmark and label"; `apply-preflight-gates` five bodies (the comparison requirement is the referent of four; "Every other apply keeps the operator-version ceiling check and the operator's instance still runs the CRD gates" refers to the install-exemption requirement).
- `errors-domain`, "A platform module build failure hint is read from the type of the error and only a refused credential changes the exit code": the body starts with "When a platform module fails to build," copied from the intro of the old list, and the first sentence reads "the cause SHALL be read" with a lower-case "the".
- `errors-domain`, "The error-text test reads the source and not its types": "The test reads the source" became "The test SHALL read the source" (keyword, see the strength table).
- `errors-domain`, "Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate": the old sentence "A test SHALL fail when non-test cli code gains an error-text predicate outside that list." moved into this body so that "that list" has its referent in the same body.
- `inventory-ownership`, bodies "A module resolving from local bytes is refused in thin-editor mode" and "`opm instance delete` verifies that the operator is ready before it deletes an operator-owned instance and refuses when it is not": the em-dash became a colon.
- `apply-preflight-gates`, "Every other apply keeps the operator-version ceiling check and the operator's instance still runs the CRD gates": "keeps" became "SHALL keep" and "still run" became "SHALL still run" (keyword, see the strength table).
- The `**Source**` metadata lines carry the whole old `Source:` line (enhancement 0021, both quotes), on every split of "Fetch and resolution answers are decided by error type" and of "Operator-version ceiling".
- Every other body is built from whole old sentences, unchanged.

## Ledger entries of cli-B7, unit 7a (pr-workflow, catalog-registry-check, ci-workflow, config, config-commands, config-types, docs-bundle)

One subsection per rewritten requirement, 15 in all: `pr-workflow` 3, `catalog-registry-check` 2, `ci-workflow` 2, `config` 2, `config-commands` 2, `config-types` 2 and `docs-bundle` 2. The other seven specs of cli-B7 (`health-evaluation`, `instance-building`, `k8s-helpers`, `mod-apply`, `module-synthetic-instance`, `release-gates`, `command-reference`) follow in the second half. Every new body is built from whole sentences of the old body, copied unchanged, or from parts of one sentence cut at a boundary with the subject repeated. A clause row names where a clause of the old body appears; the list "Words changed or added in unit 7a" names each substitution. 14 are form c2, because the old scenarios of each belong to different new behaviors. "Template publish gates run dry-run" is form b, because `release-workflow` cites its header by name, so the parent keeps the header and all 11 scenarios, and the parent states the gate order once and names the four gate blocks, which are plain parts of the list; the split-off bodies get restatement scenarios whose WHEN and THEN lines are copied from an old scenario. Old scenario names are all kept; the byte check ran once per rewritten requirement and printed no `DIFF` and no `LOST`. `grep -rn --exclude-dir=archive` over the repo found no reference to an old header of this unit other than the two `release-workflow` cites of "Template publish gates run dry-run", which stay true. The `**Source**` lines carry the whole old `Source:` lines.

### Ledger: pr-workflow, Registry mapping is set at workflow level

Form: c2 (REMOVED, ADDED under new names). New requirements: "The PR workflow sets `OPM_REGISTRY` and `CUE_REGISTRY` at workflow level", "Only the `unit`, `fixtures` and `e2e` jobs run a registry service and every other dependency resolves from GHCR".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Tests resolve fixtures and core from GHCR | "The PR workflow sets `OPM_REGISTRY` and `CUE_REGISTRY` at workflow level", "Tests resolve fixtures and core from GHCR" | No (byte equal, checked) |
| Seeded jobs resolve the tree's fixture version | "Only the `unit`, `fixtures` and `e2e` jobs run a registry service and every other dependency resolves from GHCR", "Seeded jobs resolve the tree's fixture version" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The PR workflow SHALL set `OPM_REGISTRY` and `CUE_REGISTRY` as workflo... | body of "The PR workflow sets `OPM_REGISTRY` and `CUE_REGISTRY` at workflow level", verbatim |
| Only the `unit`, `fixtures` and `e2e` jobs SHALL run a registry servic... | body of "Only the `unit`, `fixtures` and `e2e` jobs run a registry service and every other dependency resolves from GHCR", verbatim |
| Every other dependency, core and the catalogs included, SHALL resolve... | body of "Only the `unit`, `fixtures` and `e2e` jobs run a registry service and every other dependency resolves from GHCR", verbatim |

Lengths (old, new): old 635; new "The PR workflow sets `OPM_REGISTRY` and `CUE_REGISTRY` at workflow level" 205; "Only the `unit`, `fixtures` and `e2e` jobs run a registry service and every other dependency resolves from GHCR" 429.

### Ledger: pr-workflow, Template publish gates run dry-run

Form: b (MODIFIED parent with its header and all 11 old scenarios, ADDED splits). The header stays because the `release-workflow` requirements "The script runs the identity, layout, tidy, vet and zip gates on every template tree" and "The script runs the dry-run publish gates, the version order and the content check on every template tree" cite it by name. The parent keeps the first sentence of the old body and the list intro "For every template the gates SHALL be, in order:", followed by a named reference that lists the four gate blocks by exact header in the old order, so the order across the parts is stated once, in the parent, and the two release-workflow cites reach the whole gate list through it. The sentence on reporting every failing template has its own body. New requirements: "The identity and layout gates", "The tidy, vet and zip gates", "The publish dry-run gates", "The version order gate", "An already-published template version passes only when the artifact GHCR holds contains exactly the tree's files", "A template whose tree differs from its published artifact fails the job", "A failure to fetch, list, verify, build or compare fails the job and never counts the template as unpublished", "The job reports every failing template and not only the first". Each split body names the parent, or the requirement that gives it its referent, with a trailing "(Requirement: <exact header>)". The gate blocks are plain parts of the list and do not repeat the intro.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Template gate violation fails the PR | "Template publish gates run dry-run", "Template gate violation fails the PR" | No (byte equal, checked) |
| Already-published template passes | "Template publish gates run dry-run", "Already-published template passes" | No (byte equal, checked) |
| Template that does not vet fails the PR | "Template publish gates run dry-run", "Template that does not vet fails the PR" | No (byte equal, checked) |
| Changed template at an already-published version fails the PR | "Template publish gates run dry-run", "Changed template at an already-published version fails the PR" | No (byte equal, checked) |
| A change pushed to main without a bump fails later pull requests | "Template publish gates run dry-run", "A change pushed to main without a bump fails later pull requests" | No (byte equal, checked) |
| A change and its revert are not a change | "Template publish gates run dry-run", "A change and its revert are not a change" | No (byte equal, checked) |
| A fetch error never passes as unpublished | "Template publish gates run dry-run", "A fetch error never passes as unpublished" | No (byte equal, checked) |
| A file the module zip omits fails the PR | "Template publish gates run dry-run", "A file the module zip omits fails the PR" | No (byte equal, checked) |
| A version init would not resolve fails the PR | "Template publish gates run dry-run", "A version init would not resolve fails the PR" | No (byte equal, checked) |
| A module path that does not match its directory fails the PR | "Template publish gates run dry-run", "A module path that does not match its directory fails the PR" | No (byte equal, checked) |
| A broken identity does not hide other failures | "Template publish gates run dry-run", "A broken identity does not hide other failures" | No (byte equal, checked) |
| (new) | "The identity and layout gates", "The identity gate fails the PR on a module path that does not match its directory" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "A module path that does not match its directory fails the PR", byte for byte; only the name differs |
| (new) | "The tidy, vet and zip gates", "The vet gate fails a template that does not vet" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "Template that does not vet fails the PR", byte for byte; only the name differs |
| (new) | "The tidy, vet and zip gates", "The module zip gate fails a tree with a file the zip omits" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "A file the module zip omits fails the PR", byte for byte; only the name differs |
| (new) | "The publish dry-run gates", "A publish gate refusal other than already-published fails the PR" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "Template gate violation fails the PR", byte for byte; only the name differs |
| (new) | "The version order gate", "The version order gate fails a version init would not resolve" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "A version init would not resolve fails the PR", byte for byte; only the name differs |
| (new) | "An already-published template version passes only when the artifact GHCR holds contains exactly the tree's files", "A published artifact that holds exactly the tree's files passes" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "Already-published template passes", byte for byte; only the name differs |
| (new) | "An already-published template version passes only when the artifact GHCR holds contains exactly the tree's files", "A reverted change passes the content check" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "A change and its revert are not a change", byte for byte; only the name differs |
| (new) | "A template whose tree differs from its published artifact fails the job", "A changed tree at an already-published version fails the PR" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "Changed template at an already-published version fails the PR", byte for byte; only the name differs |
| (new) | "A template whose tree differs from its published artifact fails the job", "A change that reached main without a bump fails every later pull request" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "A change pushed to main without a bump fails later pull requests", byte for byte; only the name differs |
| (new) | "A failure to fetch, list, verify, build or compare fails the job and never counts the template as unpublished", "A fetch, list or verify failure never passes as unpublished" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "A fetch error never passes as unpublished", byte for byte; only the name differs |
| (new) | "The job reports every failing template and not only the first", "A broken identity does not hide the failure of another template" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "A broken identity does not hide other failures", byte for byte; only the name differs |

| Old clause | New place |
| --- | --- |
| The `template-gates` job SHALL build `opm` from the pull request, inst... | body of "Template publish gates run dry-run", verbatim |
| For every template the gates SHALL be, in order: | body of "Template publish gates run dry-run", verbatim |
| - the identity: its `Version` SHALL be a stable SemVer `X.Y.Z`, and it... | body of "The identity and layout gates", verbatim |
| - the tree layout: it SHALL hold no symbolic link, no special file and... | body of "The identity and layout gates": the closing ";" reads "." |
| - `opm module tidy --check`; | body of "The tidy, vet and zip gates", verbatim |
| - `opm module vet`, which renders the template's `debugValues` against... | body of "The tidy, vet and zip gates", verbatim |
| - the module zip that `cue mod publish --out` builds from the tree, th... | body of "The tidy, vet and zip gates": the closing ";" reads "." |
| - the `opm module publish --dry-run` gates, which SHALL either pass or... | body of "The publish dry-run gates": the closing ";" reads "." |
| - the version order: an unpublished version SHALL be above the highest... | body of "The version order gate", verbatim |
| An already-published version SHALL pass only when the artifact GHCR ho... | body of "An already-published template version passes only when the artifact GHCR holds contains exactly the tree's files", verbatim |
| The script SHALL fetch the published module zip from GHCR with an anon... | body of "An already-published template version passes only when the artifact GHCR holds contains exactly the tree's files", verbatim |
| A template whose tree differs from its published artifact SHALL fail t... | body of "A template whose tree differs from its published artifact fails the job", verbatim |
| The comparison SHALL NOT depend on git history, so a change that reach... | body of "A template whose tree differs from its published artifact fails the job", verbatim |
| A failure to fetch, list, verify, build or compare SHALL fail the job... | body of "A failure to fetch, list, verify, build or compare fails the job and never counts the template as unpublished", verbatim |
| Any other refusal or a failed vet SHALL fail the job. | body of "The publish dry-run gates", verbatim |
| The job SHALL report every failing template, not only the first, inclu... | body of "The job reports every failing template and not only the first", verbatim |

Lengths (old, new): old 2243; new "Template publish gates run dry-run" 438; "The identity and layout gates" 292; "The tidy, vet and zip gates" 422; "The publish dry-run gates" 239; "The version order gate" 275; "An already-published template version passes only when the artifact GHCR holds contains exactly the tree's files" 395; "A template whose tree differs from its published artifact fails the job" 438; "A failure to fetch, list, verify, build or compare fails the job and never counts the template as unpublished" 168; "The job reports every failing template and not only the first" 172.

### Ledger: pr-workflow, Fixture gates, seed and render parity

Form: c2 (REMOVED, ADDED under new names). New requirements: "The `fixtures` job runs `hack/fixtures.sh check`", "The `fixtures` job seeds a job-local registry", "The `fixtures` job runs the consumers and the render parity program against the mixed mapping", "`task test:fixtures` runs the steps of the `fixtures` job against a local registry".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Fixture with a gate violation | "The `fixtures` job runs `hack/fixtures.sh check`", "Fixture with a gate violation" | No (byte equal, checked) |
| Changed fixture at an already-published version | "The `fixtures` job runs `hack/fixtures.sh check`", "Changed fixture at an already-published version" | No (byte equal, checked) |
| Unchanged fixture at a published version | "The `fixtures` job runs `hack/fixtures.sh check`", "Unchanged fixture at a published version" | No (byte equal, checked) |
| Render parity runs against the seeded registry | "The `fixtures` job runs the consumers and the render parity program against the mixed mapping", "Render parity runs against the seeded registry" | No (byte equal, checked) |
| Consumer left on a stale core pin | "The `fixtures` job runs the consumers and the render parity program against the mixed mapping", "Consumer left on a stale core pin" | No (byte equal, checked) |
| (new) | "The `fixtures` job seeds a job-local registry", "The seed step publishes the tree's fixtures into a job-local registry" | New restatement scenario: only words of the old clause, no new case. |
| (new) | "`task test:fixtures` runs the steps of the `fixtures` job against a local registry", "The fixtures steps run against a local registry" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| The `fixtures` job SHALL check out with `fetch-depth: 0`, build `opm`,... | body of "The `fixtures` job runs `hack/fixtures.sh check`", verbatim |
| It SHALL then run `hack/fixtures.sh seed` to publish the tree's fixtur... | body of "The `fixtures` job seeds a job-local registry": "It" reads "The `fixtures` job" (subject copied from the first sentence of the same old requirement) |
| Against the mixed mapping (`testing.opmodel.dev` local, everything els... | body of "The `fixtures` job runs the consumers and the render parity program against the mixed mapping": "it" reads "the `fixtures` job" (same subject) |
| `task test:fixtures` SHALL run the same steps against a local registry... | body of "`task test:fixtures` runs the steps of the `fixtures` job against a local registry", verbatim |

Lengths (old, new): old 885; new "The `fixtures` job runs `hack/fixtures.sh check`" 365; "The `fixtures` job seeds a job-local registry" 240; "The `fixtures` job runs the consumers and the render parity program against the mixed mapping" 363; "`task test:fixtures` runs the steps of the `fixtures` job against a local registry" 278.

### Ledger: catalog-registry-check, Out-of-band verification of a published catalog

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm catalog registry check` pulls the named published build and verifies out of band what a platform verifies when it imports the catalog", "`opm catalog registry check` reports the member inventory per kind and apiVersion", "`opm catalog registry check --compat` runs the predecessor comparison for the fetched build exactly as publish would have", "The help text of `opm catalog registry check` states that the check is an aid and not a guarantee".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Identity mismatch found out of band | "`opm catalog registry check` pulls the named published build and verifies out of band what a platform verifies when it imports the catalog", "Identity mismatch found out of band" | No (byte equal, checked) |
| Compat aid over a published build | "`opm catalog registry check --compat` runs the predecessor comparison for the fetched build exactly as publish would have", "Compat aid over a published build" | No (byte equal, checked) |
| Aid, not guarantee | "The help text of `opm catalog registry check` states that the check is an aid and not a guarantee", "Aid, not guarantee" | No (byte equal, checked) |
| (new) | "`opm catalog registry check` reports the member inventory per kind and apiVersion", "The member inventory is reported per kind and apiVersion" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| `opm catalog registry check <path@version>` SHALL pull the named publi... | body of "`opm catalog registry check` pulls the named published build and verifies out of band what a platform verifies when it imports the catalog", verbatim |
| It SHALL report the catalog's member inventory per kind and apiVersion... | body of "`opm catalog registry check` reports the member inventory per kind and apiVersion": "It" reads "`opm catalog registry check <path@version>`" (subject copied from the first sentence of the same old requirement) |
| With `--compat`, it SHALL additionally run the predecessor comparison... | body of "`opm catalog registry check --compat` runs the predecessor comparison for the fetched build exactly as publish would have": "it" reads "`opm catalog registry check <path@version>`" (same subject) |
| The command's help text SHALL state that the check is an aid and not a... | body of "The help text of `opm catalog registry check` states that the check is an aid and not a guarantee": "The command's" reads "The `opm catalog registry check <path@version>` command's" (subject copied in); the em-dash reads a colon |
| Exit codes: 0 clean, 2 findings, 3 registry unreachable, 5 not publish... | body of "`opm catalog registry check` pulls the named published build and verifies out of band what a platform verifies when it imports the catalog", verbatim |

Lengths (old, new): old 796; new "`opm catalog registry check` pulls the named published build and verifies out of band what a platform verifies when it imports the catalog" 482; "`opm catalog registry check` reports the member inventory per kind and apiVersion" 112; "`opm catalog registry check --compat` runs the predecessor comparison for the fetched build exactly as publish would have" 163; "The help text of `opm catalog registry check` states that the check is an aid and not a guarantee" 161.

### Ledger: catalog-registry-check, An unpublished coordinate exits 5 and every other registry answer exits 3

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry", "`opm catalog registry check` takes the not-held decision from the library's typed fetch classification and never from the message text".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Version not published | "`opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry", "Version not published" | No (byte equal, checked) |
| Forbidden tag lookup reads as not published | "`opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry", "Forbidden tag lookup reads as not published" | No (byte equal, checked) |
| (new) | "`opm catalog registry check` takes the not-held decision from the library's typed fetch classification and never from the message text", "The not-held decision comes from the typed fetch classification" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| `opm catalog registry check <path@version>` SHALL exit 5 when the regi... | body of "`opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry", verbatim; body of "`opm catalog registry check` takes the not-held decision from the library's typed fetch classification and never from the message text", verbatim |
| A registry answer other than not-held, or no response, while the named... | body of "`opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry", verbatim |
| A failure that asks no registry (a registry mapping that does not pars... | body of "`opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry", verbatim |
| Source: enhancement 0011 ("A published catalog can be verified out of... | `**Source**` line (whole old line) of "`opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry"; "`opm catalog registry check` takes the not-held decision from the library's typed fetch classification and never from the message text" |
| The CLI SHALL take the not-held decision from the library's typed fetc... | body of "`opm catalog registry check` takes the not-held decision from the library's typed fetch classification and never from the message text", verbatim |
| Source: enhancement 0021 ("A beta line reaches GA only when its exit c... | `**Source**` line (whole old line) of "`opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry"; "`opm catalog registry check` takes the not-held decision from the library's typed fetch classification and never from the message text" |

Lengths (old, new): old 754; new "`opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry" 441; "`opm catalog registry check` takes the not-held decision from the library's typed fetch classification and never from the message text" 382.

### Ledger: ci-workflow, Fixture modules are published to GHCR by CI

Form: c2 (REMOVED, ADDED under new names). New requirements: "A workflow publishes the repository's fixture modules to GHCR through `opm module publish`", "The fixture publish workflow triggers on pushes to `main` that touch the fixtures and offers `workflow_dispatch`", "The fixture publish workflow requests `packages: write` at the job level only and skips cleanly on forks", "Idempotency of the fixture publish lives in the caller `hack/fixtures.sh publish`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Unchanged fixture republished | "Idempotency of the fixture publish lives in the caller `hack/fixtures.sh publish`", "Unchanged fixture republished" | No (byte equal, checked) |
| Bumped fixture publishes | "A workflow publishes the repository's fixture modules to GHCR through `opm module publish`", "Bumped fixture publishes" | No (byte equal, checked) |
| Fork run | "The fixture publish workflow requests `packages: write` at the job level only and skips cleanly on forks", "Fork run" | No (byte equal, checked) |
| (new) | "The fixture publish workflow triggers on pushes to `main` that touch the fixtures and offers `workflow_dispatch`", "A push that touches the fixtures triggers the workflow" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| A workflow (`.github/workflows/publish-fixtures.yml`) SHALL publish th... | body of "A workflow publishes the repository's fixture modules to GHCR through `opm module publish`", verbatim |
| It SHALL trigger on pushes to `main` that touch `tests/fixtures/module... | body of "The fixture publish workflow triggers on pushes to `main` that touch the fixtures and offers `workflow_dispatch`": "It" reads "A workflow (`.github/workflows/publish-fixtures.yml`)" (the subject of the first sentence of the same old requirement) |
| The workflow SHALL request `packages: write` at the job level only, an... | body of "The fixture publish workflow requests `packages: write` at the job level only and skips cleanly on forks": "The workflow" reads "The workflow (`.github/workflows/publish-fixtures.yml`)" (path copied from the first sentence) |
| Idempotency SHALL live in the caller: `hack/fixtures.sh publish` SHALL... | body of "Idempotency of the fixture publish lives in the caller `hack/fixtures.sh publish`", verbatim |

Lengths (old, new): old 912; new "A workflow publishes the repository's fixture modules to GHCR through `opm module publish`" 242; "The fixture publish workflow triggers on pushes to `main` that touch the fixtures and offers `workflow_dispatch`" 315; "The fixture publish workflow requests `packages: write` at the job level only and skips cleanly on forks" 180; "Idempotency of the fixture publish lives in the caller `hack/fixtures.sh publish`" 266.

### Ledger: ci-workflow, One file names the linter version

Form: c2 (REMOVED, ADDED under new names). New requirements: "One file names the linter version CI installs", "Every use of the golangci-lint action reads that file and pins one action commit", "The committed golangci-lint schema matches the linter version's minor line and its checksum", "The configuration check fails when the linter version rules do not hold or the installed golangci-lint is of another minor line".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A workflow names its own version | "Every use of the golangci-lint action reads that file and pins one action commit", "A workflow names its own version" | No (byte equal, checked) |
| The version moves to a minor line without a committed schema | "The committed golangci-lint schema matches the linter version's minor line and its checksum", "The version moves to a minor line without a committed schema" | No (byte equal, checked) |
| The committed schema was edited | "The committed golangci-lint schema matches the linter version's minor line and its checksum", "The committed schema was edited" | No (byte equal, checked) |
| The two workflows pin different action commits | "Every use of the golangci-lint action reads that file and pins one action commit", "The two workflows pin different action commits" | No (byte equal, checked) |
| The installed linter is of another minor line | "The configuration check fails when the linter version rules do not hold or the installed golangci-lint is of another minor line", "The installed linter is of another minor line" | No (byte equal, checked) |
| go.mod names a linter version | "One file names the linter version CI installs", "go.mod names a linter version" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The file `.golangci-lint-version` SHALL be the only place that names t... | body of "One file names the linter version CI installs", verbatim |
| Every use of the golangci-lint action, in any letter case and quoted o... | body of "Every use of the golangci-lint action reads that file and pins one action commit", verbatim |
| The committed schema SHALL be the one for that version's minor line, S... | body of "The committed golangci-lint schema matches the linter version's minor line and its checksum", verbatim |
| The configuration check SHALL fail when any of these does not hold, an... | body of "The configuration check fails when the linter version rules do not hold or the installed golangci-lint is of another minor line", verbatim |

Lengths (old, new): old 732; new "One file names the linter version CI installs" 167; "Every use of the golangci-lint action reads that file and pins one action commit" 310; "The committed golangci-lint schema matches the linter version's minor line and its checksum" 224; "The configuration check fails when the linter version rules do not hold or the installed golangci-lint is of another minor line" 387.

### Ledger: config-commands, Config init writes only the configuration file

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm config init` creates the default configuration `config.cue` in `~/.opm/`", "`opm config init` writes no platform module or platform file", "`opm config init` remains offline".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Initialize configuration for first time | "`opm config init` creates the default configuration `config.cue` in `~/.opm/`", "Initialize configuration for first time" | No (byte equal, checked) |
| An existing platform directory is left alone | "`opm config init` writes no platform module or platform file", "An existing platform directory is left alone" | No (byte equal, checked) |
| Legacy data-only platform file is removed | "`opm config init` writes no platform module or platform file", "Legacy data-only platform file is removed" | No (byte equal, checked) |
| Refuse to overwrite existing configuration | "`opm config init` creates the default configuration `config.cue` in `~/.opm/`", "Refuse to overwrite existing configuration" | No (byte equal, checked) |
| Force overwrite existing configuration | "`opm config init` creates the default configuration `config.cue` in `~/.opm/`", "Force overwrite existing configuration" | No (byte equal, checked) |
| (new) | "`opm config init` remains offline", "Init stays offline" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| The `opm config init` command SHALL create the default configuration i... | body of "`opm config init` creates the default configuration `config.cue` in `~/.opm/`", verbatim |
| The command creates: | body of "`opm config init` creates the default configuration `config.cue` in `~/.opm/`", verbatim |
| - `~/.opm/config.cue` [em-dash] scalar-only configuration file (registry, kube... | body of "`opm config init` creates the default configuration `config.cue` in `~/.opm/`": the em-dash reads a colon |
| The command SHALL NOT write a platform module or any platform file: no... | body of "`opm config init` writes no platform module or platform file": "The command" reads "The `opm config init` command" (subject copied from the first sentence) |
| An existing `~/.opm/platform/` directory SHALL be left untouched, with... | body of "`opm config init` writes no platform module or platform file", verbatim |
| When a legacy data-only `~/.opm/platform.cue` exists, `--force` SHALL... | body of "`opm config init` writes no platform module or platform file", verbatim |
| The command SHALL remain normatively offline: it SHALL NOT run `cue mo... | body of "`opm config init` remains offline": "The command" reads "The `opm config init` command" (same subject) |

Lengths (old, new): old 645; new "`opm config init` creates the default configuration `config.cue` in `~/.opm/`" 205; "`opm config init` writes no platform module or platform file" 351; "`opm config init` remains offline" 122.

### Ledger: config-commands, Config vet checks only the configuration file

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm config vet` validates the `~/.opm` configuration with two checks", "`opm config vet` builds no platform module and warns about a `platform/` directory", "`opm config vet` fails on a leftover legacy `~/.opm/platform.cue`", "`opm config vet` prints a styled line for each passing check".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Valid configuration passes validation | "`opm config vet` prints a styled line for each passing check", "Valid configuration passes validation" | No (byte equal, checked) |
| Missing config file fails with actionable error | "`opm config vet` validates the `~/.opm` configuration with two checks", "Missing config file fails with actionable error" | No (byte equal, checked) |
| A leftover platform directory warns | "`opm config vet` builds no platform module and warns about a `platform/` directory", "A leftover platform directory warns" | No (byte equal, checked) |
| Legacy platform file fails with migration hint | "`opm config vet` fails on a leftover legacy `~/.opm/platform.cue`", "Legacy platform file fails with migration hint" | No (byte equal, checked) |
| Stale providers block fails with migration hint | "`opm config vet` validates the `~/.opm` configuration with two checks", "Stale providers block fails with migration hint" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The `opm config vet` command SHALL validate the `~/.opm` configuration... | body of "`opm config vet` validates the `~/.opm` configuration with two checks", verbatim |
| Checks performed: | body of "`opm config vet` validates the `~/.opm` configuration with two checks", verbatim |
| 1. Config file exists at resolved path | body of "`opm config vet` validates the `~/.opm` configuration with two checks", verbatim |
| 2. Config file is syntactically valid CUE and satisfies the embedded c... | body of "`opm config vet` validates the `~/.opm` configuration with two checks", verbatim |
| Vet SHALL NOT build, load or require any platform module. | body of "`opm config vet` builds no platform module and warns about a `platform/` directory": "Vet" reads "The `opm config vet` command" (subject copied from the first sentence) |
| When a `platform/` directory exists beside the config file, vet SHALL... | body of "`opm config vet` builds no platform module and warns about a `platform/` directory", verbatim |
| A leftover legacy data-only `~/.opm/platform.cue` SHALL fail vet namin... | body of "`opm config vet` fails on a leftover legacy `~/.opm/platform.cue`", verbatim |
| Each check SHALL print a styled line to stdout using `FormatVetCheck`... | body of "`opm config vet` prints a styled line for each passing check", verbatim |

Lengths (old, new): old 794; new "`opm config vet` validates the `~/.opm` configuration with two checks" 243; "`opm config vet` builds no platform module and warns about a `platform/` directory" 298; "`opm config vet` fails on a leftover legacy `~/.opm/platform.cue`" 132; "`opm config vet` prints a styled line for each passing check" 228.

### Ledger: config, Configuration precedence chain

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", "There is no provider configuration value", "Subcommands do not call `config.Load()` independently", "A subcommand-local flag overrides the globally-resolved value for that invocation", "Registry and config path resolve in `config.Load` and Kubernetes config at the point of use", "Config loading is single-pass and has no registry-bootstrap phase".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Flag overrides environment variable | "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", "Flag overrides environment variable" | No (byte equal, checked) |
| Environment overrides config file | "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", "Environment overrides config file" | No (byte equal, checked) |
| Config file overrides default for kubeconfig | "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", "Config file overrides default for kubeconfig" | No (byte equal, checked) |
| Config file overrides default for namespace | "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", "Config file overrides default for namespace" | No (byte equal, checked) |
| Default used when no override present | "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", "Default used when no override present" | No (byte equal, checked) |
| Subcommand local flag overrides global resolution | "A subcommand-local flag overrides the globally-resolved value for that invocation", "Subcommand local flag overrides global resolution" | No (byte equal, checked) |
| Timestamps flag overrides config | "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", "Timestamps flag overrides config" | No (byte equal, checked) |
| Timestamps config overrides default | "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", "Timestamps config overrides default" | No (byte equal, checked) |
| Configuration loaded once per CLI invocation | "Subcommands do not call `config.Load()` independently", "Configuration loaded once per CLI invocation" | No (byte equal, checked) |
| No separate base resolution step | "Registry and config path resolve in `config.Load` and Kubernetes config at the point of use", "No separate base resolution step" | No (byte equal, checked) |
| Single-pass load | "Config loading is single-pass and has no registry-bootstrap phase", "Single-pass load" | No (byte equal, checked) |
| Providers field rejected | "There is no provider configuration value", "Providers field rejected" | No (byte equal, checked) |
| Imports rejected | "Config loading is single-pass and has no registry-bootstrap phase", "Imports rejected" | No (byte equal, checked) |
| (new) | "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", "The flag, environment variable, config field and default of each value" | New restatement scenario: only words of the old clause, no new case. The rule table moved into the THEN line; the body keeps the SHALL and names the scenario. |

| Old clause | New place |
| --- | --- |
| The CLI SHALL resolve ALL configuration values using precedence: Flag... | body of "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", verbatim |
| Resolution SHALL occur once during `PersistentPreRunE` and the results... | body of "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", verbatim |
| There SHALL be no provider configuration value: the `providers` config... | body of "There is no provider configuration value", verbatim |
| Platform selection is a per-command concern (`--platform`, see the `pl... | body of "There is no provider configuration value", verbatim |
| Each resolved value SHALL record the final value, the source it came f... | body of "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", verbatim |
| Subcommands SHALL NOT call `config.Load()` independently. | body of "Subcommands do not call `config.Load()` independently", verbatim |
| They SHALL access the pre-loaded configuration via the `*config.Global... | body of "Subcommands do not call `config.Load()` independently", verbatim |
| The one exception is `opm config vet`, which loads the file named on i... | body of "Subcommands do not call `config.Load()` independently", verbatim |
| Subcommand-local flags (e.g., `opm mod apply --namespace production`)... | body of "A subcommand-local flag overrides the globally-resolved value for that invocation", verbatim |
| Registry and config path resolution SHALL be performed by `config.Load... | body of "Registry and config path resolve in `config.Load` and Kubernetes config at the point of use", verbatim |
| Kubernetes config resolution SHALL be performed by `config.ResolveKube... | body of "Registry and config path resolve in `config.Load` and Kubernetes config at the point of use", verbatim |
| There SHALL NOT be a separate `ResolveBase` or `ResolveAll` resolution... | body of "Registry and config path resolve in `config.Load` and Kubernetes config at the point of use", verbatim |
| Config loading SHALL be single-pass: because `~/.opm/config.cue` conta... | body of "Config loading is single-pass and has no registry-bootstrap phase", verbatim |
| Registry SHALL resolve by ordinary field precedence after the file is... | body of "Config loading is single-pass and has no registry-bootstrap phase", verbatim |
| The table of seven values (header row, separator row, 7 rows: Registry, Config Path, Kubeconfig, Context, Namespace, Timestamps, Skew policy) | scenario "The flag, environment variable, config field and default of each value" of "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each", in the THEN line, byte for byte; the body names the scenario |

Lengths (old, new): old 2357; new "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each" 434; "There is no provider configuration value" 246; "Subcommands do not call `config.Load()` independently" 334; "A subcommand-local flag overrides the globally-resolved value for that invocation" 133; "Registry and config path resolve in `config.Load` and Kubernetes config at the point of use" 294; "Config loading is single-pass and has no registry-bootstrap phase" 307.

### Ledger: config, Resolution tracking for debugging

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI tracks the source of each resolved configuration value for verbose logging", "The CLI emits a single `"initializing CLI"` debug log line when `--verbose` is specified", "The debug log lines `"resolved config path"` and `"resolved registry"` are retained".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Verbose mode shows initialization-time values | "The CLI emits a single `"initializing CLI"` debug log line when `--verbose` is specified", "Verbose mode shows initialization-time values" | No (byte equal, checked) |
| Verbose mode shows resolution source | "The CLI tracks the source of each resolved configuration value for verbose logging", "Verbose mode shows resolution source" | No (byte equal, checked) |
| No bootstrap log lines exist | "The debug log lines `"resolved config path"` and `"resolved registry"` are retained", "No bootstrap log lines exist" | No (byte equal, checked) |
| (new) | "The debug log lines `"resolved config path"` and `"resolved registry"` are retained", "The retained debug log lines are present" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| The CLI SHALL track the source of each resolved configuration value fo... | body of "The CLI tracks the source of each resolved configuration value for verbose logging", verbatim |
| Each resolved value records: | body of "The CLI tracks the source of each resolved configuration value for verbose logging", verbatim |
| - The final value used | body of "The CLI tracks the source of each resolved configuration value for verbose logging", verbatim |
| - The source (flag, env, config, or default) | body of "The CLI tracks the source of each resolved configuration value for verbose logging", verbatim |
| - Any shadowed values from lower-precedence sources | body of "The CLI tracks the source of each resolved configuration value for verbose logging", verbatim |
| The CLI SHALL emit a single "initializing CLI" debug log line when `--... | body of "The CLI emits a single `"initializing CLI"` debug log line when `--verbose` is specified", verbatim |
| Kubernetes values resolve at point of use per command; the platform so... | body of "The CLI emits a single `"initializing CLI"` debug log line when `--verbose` is specified", verbatim |
| Provider does not exist as a resolved value. | body of "The CLI emits a single `"initializing CLI"` debug log line when `--verbose` is specified", verbatim |
| The following debug log lines SHALL be retained: | body of "The debug log lines `"resolved config path"` and `"resolved registry"` are retained", verbatim |
| - `"resolved config path"` [em-dash] shows config file location and source | body of "The debug log lines `"resolved config path"` and `"resolved registry"` are retained": the em-dash reads a colon |
| - `"resolved registry"` [em-dash] shows registry URL and source | body of "The debug log lines `"resolved config path"` and `"resolved registry"` are retained": the em-dash reads a colon |

Lengths (old, new): old 829; new "The CLI tracks the source of each resolved configuration value for verbose logging" 237; "The CLI emits a single `"initializing CLI"` debug log line when `--verbose` is specified" 419; "The debug log lines `"resolved config path"` and `"resolved registry"` are retained" 169.

### Ledger: config-types, Single GlobalConfig type in config package

Form: c2 (REMOVED, ADDED under new names). New requirements: "The `internal/config` package exports a single `GlobalConfig` struct and no `Config` or `OPMConfig` type", "`GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`", "`GlobalConfig` contains no `Providers` field and no `CueContext` field".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| GlobalConfig contains only surviving fields | "`GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`", "GlobalConfig contains only surviving fields" | No (byte equal, checked) |
| Config and OPMConfig types do not exist | "The `internal/config` package exports a single `GlobalConfig` struct and no `Config` or `OPMConfig` type", "Config and OPMConfig types do not exist" | No (byte equal, checked) |
| (new) | "`GlobalConfig` contains no `Providers` field and no `CueContext` field", "GlobalConfig has no Providers field and no CueContext field" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| The `internal/config` package SHALL export a single `GlobalConfig` str... | body of "The `internal/config` package exports a single `GlobalConfig` struct and no `Config` or `OPMConfig` type", verbatim |
| The types `Config` and `OPMConfig` SHALL NOT exist. | body of "The `internal/config` package exports a single `GlobalConfig` struct and no `Config` or `OPMConfig` type", verbatim |
| `GlobalConfig` SHALL contain the following fields: | body of "`GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`", verbatim |
| - `Kubernetes` (`KubernetesConfig`): Kubernetes settings from config f... | body of "`GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`", verbatim |
| - `Log` (`LogConfig`): Logging settings from config file | body of "`GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`", verbatim |
| - `Registry` (`string`): Resolved registry URL (after flag > env > con... | body of "`GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`", verbatim |
| - `ConfigPath` (`string`): Resolved config file path | body of "`GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`", verbatim |
| - `Flags` (`GlobalFlags`): Raw CLI flag values | body of "`GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`", verbatim |
| `GlobalConfig` SHALL NOT contain a `Providers` field or a `CueContext`... | body of "`GlobalConfig` contains no `Providers` field and no `CueContext` field", verbatim |
| CUE evaluation contexts are owned by the per-invocation kernel, not by... | body of "`GlobalConfig` contains no `Providers` field and no `CueContext` field", verbatim |

Lengths (old, new): old 753; new "The `internal/config` package exports a single `GlobalConfig` struct and no `Config` or `OPMConfig` type" 221; "`GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`" 367; "`GlobalConfig` contains no `Providers` field and no `CueContext` field" 163.

### Ledger: config-types, Skew policy configuration key

Form: c2 (REMOVED, ADDED under new names). New requirements: "`config.cue` accepts an optional `skewPolicy` key with the values `warn` and `refuse`, defaulting to `warn`", "The `skewPolicy` value governs how a render responds to a module that requires a newer build than the platform pins", "The `skewPolicy` key applies to the local default or a `--platform` directory and the cluster Platform CR takes precedence".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default is warn | "`config.cue` accepts an optional `skewPolicy` key with the values `warn` and `refuse`, defaulting to `warn`", "Default is warn" | No (byte equal, checked) |
| Refuse configured | "The `skewPolicy` value governs how a render responds to a module that requires a newer build than the platform pins", "Refuse configured" | No (byte equal, checked) |
| Cluster CR overrides the key | "The `skewPolicy` key applies to the local default or a `--platform` directory and the cluster Platform CR takes precedence", "Cluster CR overrides the key" | No (byte equal, checked) |
| Invalid value rejected | "`config.cue` accepts an optional `skewPolicy` key with the values `warn` and `refuse`, defaulting to `warn`", "Invalid value rejected" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `config.cue` SHALL accept an optional `skewPolicy` key with the values... | body of "`config.cue` accepts an optional `skewPolicy` key with the values `warn` and `refuse`, defaulting to `warn`", verbatim |
| The value SHALL govern how a render responds when a module's `cue.mod`... | body of "The `skewPolicy` value governs how a render responds to a module that requires a newer build than the platform pins": "The value" reads "The `skewPolicy` value" (key name copied from the first sentence) |
| The key applies when the platform is the local default or a `--platfor... | body of "The `skewPolicy` key applies to the local default or a `--platform` directory and the cluster Platform CR takes precedence": "The key" reads "The `skewPolicy` key" (same name) |
| Any other value SHALL fail config validation naming the allowed values... | body of "`config.cue` accepts an optional `skewPolicy` key with the values `warn` and `refuse`, defaulting to `warn`", verbatim |
| No flag SHALL exist for it. | body of "`config.cue` accepts an optional `skewPolicy` key with the values `warn` and `refuse`, defaulting to `warn`", verbatim |

Lengths (old, new): old 883; new "`config.cue` accepts an optional `skewPolicy` key with the values `warn` and `refuse`, defaulting to `warn`" 374; "The `skewPolicy` value governs how a render responds to a module that requires a newer build than the platform pins" 286; "The `skewPolicy` key applies to the local default or a `--platform` directory and the cluster Platform CR takes precedence" 247.

### Ledger: docs-bundle, The cli publishes one docs bundle that carries its pins

Form: c2 (REMOVED, ADDED under new names). New requirements: "The repository declares one docs-kit project, `cli`, in `docs-kit.cue`", "The docs bundle carries the authored pages and the generated command reference together and `task docs:bundle` builds it".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The bundle holds both kinds of page and the pins | "The docs bundle carries the authored pages and the generated command reference together and `task docs:bundle` builds it", "The bundle holds both kinds of page and the pins" | No (byte equal, checked) |
| A dev pin is refused | "The docs bundle carries the authored pages and the generated command reference together and `task docs:bundle` builds it", "A dev pin is refused" | No (byte equal, checked) |
| (new) | "The repository declares one docs-kit project, `cli`, in `docs-kit.cue`", "The cli project is declared in docs-kit.cue" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| The repository SHALL declare one docs-kit project, `cli`, in `docs-kit... | body of "The repository declares one docs-kit project, `cli`, in `docs-kit.cue`", verbatim |
| The bundle SHALL carry the authored pages under `docs/site/` and the g... | body of "The docs bundle carries the authored pages and the generated command reference together and `task docs:bundle` builds it", verbatim |
| `task docs:bundle` SHALL build it into `out/cli/` and `task docs:bundl... | body of "The docs bundle carries the authored pages and the generated command reference together and `task docs:bundle` builds it", verbatim |

Lengths (old, new): old 711; new "The repository declares one docs-kit project, `cli`, in `docs-kit.cue`" 404; "The docs bundle carries the authored pages and the generated command reference together and `task docs:bundle` builds it" 306.

### Ledger: docs-bundle, The dump program prints the command tree and the pins the cli compiles in

Form: c2 (REMOVED, ADDED under new names). New requirements: "`go run ./hack/docskit-dump` prints the command tree as one `docs.opmodel.dev/cobradump/v1` document", "`go run ./hack/docskit-dump pins` prints the pins the cli compiles in as one `docs.opmodel.dev/pins/v1` document", "Two runs of the dump program print the same bytes", "Any other argument to the dump program exits 1 with a usage line", "The `opm` binary does not contain the dump program or `cobradump`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The pins equal the sources the site read | "`go run ./hack/docskit-dump pins` prints the pins the cli compiles in as one `docs.opmodel.dev/pins/v1` document", "The pins equal the sources the site read" | No (byte equal, checked) |
| A library that pins only a core major is refused | "`go run ./hack/docskit-dump pins` prints the pins the cli compiles in as one `docs.opmodel.dev/pins/v1` document", "A library that pins only a core major is refused" | No (byte equal, checked) |
| A wrong argument is refused | "Any other argument to the dump program exits 1 with a usage line", "A wrong argument is refused" | No (byte equal, checked) |
| (new) | "`go run ./hack/docskit-dump` prints the command tree as one `docs.opmodel.dev/cobradump/v1` document", "The dump prints the command tree" | New restatement scenario: only words of the old clause, no new case. |
| (new) | "Two runs of the dump program print the same bytes", "Two runs print the same bytes" | New restatement scenario: only words of the old clause, no new case. |
| (new) | "The `opm` binary does not contain the dump program or `cobradump`", "The opm binary holds neither the program nor cobradump" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| `go run ./hack/docskit-dump` SHALL print one `docs.opmodel.dev/cobradu... | body of "`go run ./hack/docskit-dump` prints the command tree as one `docs.opmodel.dev/cobradump/v1` document", the part up to ", and" with a period; body of "`go run ./hack/docskit-dump pins` prints the pins the cli compiles in as one `docs.opmodel.dev/pins/v1` document", the rest, the second part gets "SHALL print" from the first part |
| Two runs SHALL print the same bytes. | body of "Two runs of the dump program print the same bytes", verbatim |
| Any other argument SHALL exit 1 with `docskit-dump: usage: docskit-dum... | body of "Any other argument to the dump program exits 1 with a usage line", verbatim |
| The `opm` binary SHALL NOT contain the program or `cobradump`. | body of "The `opm` binary does not contain the dump program or `cobradump`", verbatim |

Lengths (old, new): old 783; new "`go run ./hack/docskit-dump` prints the command tree as one `docs.opmodel.dev/cobradump/v1` document" 131; "`go run ./hack/docskit-dump pins` prints the pins the cli compiles in as one `docs.opmodel.dev/pins/v1` document" 468; "Two runs of the dump program print the same bytes" 267; "Any other argument to the dump program exits 1 with a usage line" 321; "The `opm` binary does not contain the dump program or `cobradump`" 293.

### Words changed or added in unit 7a

- Named references: a body ends with "(Requirement: <exact header>)" or "(Requirements: <exact header>; <exact header>)", the named reference that gives it its referent ("every template the gates", "the script", "the comparison", "it" for the linter file, "that version", "any of these", "the same steps", "each check", "Any other argument", "the program", "Two runs"). The bodies are: `pr-workflow` the eight bodies split off "Template publish gates run dry-run" (to the parent; "A template whose tree differs from its published artifact fails the job" also to the content-check requirement), "The `fixtures` job seeds a job-local registry" (to the check requirement), "The `fixtures` job runs the consumers and the render parity program against the mixed mapping" (to the seed requirement), because both keep "then", and "`task test:fixtures` runs the steps of the `fixtures` job against a local registry" (to the three `fixtures` job requirements); `ci-workflow` "Every use of the golangci-lint action reads that file and pins one action commit" and "The committed golangci-lint schema matches the linter version's minor line and its checksum" (to "One file names the linter version CI installs") and "The configuration check fails when the linter version rules do not hold or the installed golangci-lint is of another minor line" (to the three requirements, because "any of these" names the whole set); `config-commands` "`opm config vet` prints a styled line for each passing check" (to the checks requirement); `docs-bundle` "Two runs of the dump program print the same bytes", "Any other argument to the dump program exits 1 with a usage line" and "The `opm` binary does not contain the dump program or `cobradump`" (to the two dump requirements).
- `config`, "The CLI resolves every configuration value by Flag > Environment > Config > Default, once, and records the source of each": the body names the scenario "The flag, environment variable, config field and default of each value" in "(Scenario: ...)", where the old table of seven values now stands in the THEN line, byte for byte. The body keeps both SHALL sentences and the "Each resolved value SHALL record" sentence.
- `pr-workflow`, "Template publish gates run dry-run": the parent states the old list intro "For every template the gates SHALL be, in order:" once, followed by "(Requirements: ...)" with the exact headers of the four gate blocks in the old order. The four gate blocks do not repeat the intro; the closing ";" of the last bullet of a block reads ".".
- `pr-workflow`, "The `fixtures` job seeds a job-local registry" and "The `fixtures` job runs the consumers and the render parity program against the mixed mapping": "It" and "it" read "The `fixtures` job" and "the `fixtures` job" (the subject of the first sentence of the same old requirement).
- `catalog-registry-check`, "`opm catalog registry check` reports the member inventory per kind and apiVersion" and "`opm catalog registry check --compat` runs the predecessor comparison for the fetched build exactly as publish would have": "It" and "it" read "`opm catalog registry check <path@version>`" (the subject of the first sentence of the same old requirement). The help text body reads "The `opm catalog registry check <path@version>` command's help text" where the old text said "The command's help text", and its em-dash reads a colon.
- `catalog-registry-check`, "An unpublished coordinate exits 5 and every other registry answer exits 3": the sentence on exit 5 repeats verbatim in both bodies as context ("not-held" has its meaning there); the `**Source**` metadata lines carry both old `Source:` lines (enhancement 0011 and enhancement 0021, both quotes) on both bodies.
- `ci-workflow`, "The fixture publish workflow triggers on pushes to `main` that touch the fixtures and offers `workflow_dispatch`" and "The fixture publish workflow requests `packages: write` at the job level only and skips cleanly on forks": "It" reads "A workflow (`.github/workflows/publish-fixtures.yml`)" and "The workflow" reads "The workflow (`.github/workflows/publish-fixtures.yml`)" (the subject and path of the first sentence of the same old requirement).
- `config-commands`, "`opm config init` writes no platform module or platform file" and "`opm config init` remains offline": "The command" reads "The `opm config init` command"; "`opm config vet` builds no platform module and warns about a `platform/` directory": "Vet SHALL NOT build" reads "The `opm config vet` command SHALL NOT build". The em-dash of the `config.cue` bullet reads a colon.
- `config`, "The debug log lines `\"resolved config path\"` and `\"resolved registry\"` are retained": the two em-dashes read colons.
- `config-types`, "The `skewPolicy` value governs how a render responds to a module that requires a newer build than the platform pins" and "The `skewPolicy` key applies to the local default or a `--platform` directory and the cluster Platform CR takes precedence": "The value" and "The key" read "The `skewPolicy` value" and "The `skewPolicy` key" (the key name of the first sentence of the same old requirement). "Any other value SHALL fail config validation naming the allowed values. No flag SHALL exist for it." moves up beside the sentence that names the values, so "other" and "it" keep their referent.
- `docs-bundle`, "`go run ./hack/docskit-dump` prints the command tree as one `docs.opmodel.dev/cobradump/v1` document" and "`go run ./hack/docskit-dump pins` prints the pins the cli compiles in as one `docs.opmodel.dev/pins/v1` document": the old sentence is cut at ", and" (the list boundary between the two documents), a period is added after "builds", and the second part reads "SHALL print" from the first part. "Two runs SHALL print the same bytes." has its own body because it holds for both documents.
- `config-types`, "The `skewPolicy` value governs how a render responds to a module that requires a newer build than the platform pins": a `**Source**` line carries the whole citation of the old first sentence (enhancement 0019, with its quote), because the old sentence tied that citation to the whole key, including this rule.
- Every other body is built from whole old sentences, unchanged.

## Ledger entries of cli-B7, unit 7b (health-evaluation, instance-building, k8s-helpers, mod-apply, module-synthetic-instance, release-gates, command-reference)

One subsection per rewritten requirement, 13 in all: `health-evaluation` 2, `instance-building` 2, `k8s-helpers` 2, `mod-apply` 2, `module-synthetic-instance` 2, `release-gates` 2 and `command-reference` 1. With unit 7a (15) this completes section 7 (28 requirements, all fourteen specs). All 13 are form c2, because the old scenarios of each belong to different new behaviors; no other form is used, so no parent keeps a header and no other spec cites a renamed header (`grep -rnF --exclude-dir=archive` over the repo finds the 13 old headers only in `openspec/specs/` and in this change). Every new body is built from whole sentences or whole lines of the old body, copied unchanged, or from parts of one sentence cut at a list boundary with the subject repeated. A clause row names where the old text appears; the list "Words changed or added in unit 7b" names each substitution. Each split of a requirement that had a `Source:` line carries the whole old line as a `**Source**` metadata line (`health-evaluation` both requirements, enhancement 0012; `release-gates` both requirements, the first the line with RELEASING.md and enhancement 0021, the second the line with the security pass; the last body of the second, "G1 runs the docs pins script without `--warn` and counts each printed line as a violation", follows the old `Source:` line in the old text and carries none). Old scenario names are all kept; the byte check ran once per rewritten requirement and printed no `DIFF` and no `LOST`. A new scenario restates a case the old text names, or copies the WHEN and THEN lines of an old scenario; the Changed column says which.

### Ledger: health-evaluation, Readiness is judged by the library health package

Form: c2 (REMOVED, ADDED under new names). New requirements: "The cli takes every readiness decision from the library health package", "The library health decisions cover instance status, list and tree and the apply wait", "The Established wait and the operator install dispatch by kind stay in the cli", "The tree child rows and the status colouring stay in the cli".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Status uses the library verdict | "The cli takes every readiness decision from the library health package", "Status uses the library verdict" | No (byte equal, checked) |
| A passive kind is applied, not ready | "The cli takes every readiness decision from the library health package", "A passive kind is applied, not ready" | No (byte equal, checked) |
| Missing and unreadable objects count against the aggregate | "The cli takes every readiness decision from the library health package", "Missing and unreadable objects count against the aggregate" | No (byte equal, checked) |
| Nothing to fold is unknown | "The cli takes every readiness decision from the library health package", "Nothing to fold is unknown" | No (byte equal, checked) |
| (new) | "The library health decisions cover instance status, list and tree and the apply wait", "Instance status takes its row from the library verdict" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "Status uses the library verdict", byte for byte; only the name differs |
| (new) | "The Established wait and the operator install dispatch by kind stay in the cli", "The Established wait stays in the cli" | New restatement scenario: only words of the old clause, no new case. |
| (new) | "The tree child rows and the status colouring stay in the cli", "The tree child rows never feed an aggregate" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| The cli SHALL take every decision the library's `opm/k8s/health` mak... | body of "The cli takes every readiness decision from the library health package", verbatim |
| This `opm instance status`, `opm instance list`, `opm instance tree`... | body of "The library health decisions cover instance status, list and tree and the apply wait", with the word changes of the list "Words changed or added in unit 7b" |
| Three readiness checks are not decisions of the package and stay in ... | body of "The Established wait and the operator install dispatch by kind stay in the cli", with the word changes of the list "Words changed or added in unit 7b" |
| Three readiness checks are not decisions of the package and stay in ... | body of "The tree child rows and the status colouring stay in the cli", with the word changes of the list "Words changed or added in unit 7b" |

Lengths (old, new): old 1217; new "The cli takes every readiness decision from the library health package" 422; "The library health decisions cover instance status, list and tree and the apply wait" 247; "The Established wait and the operator install dispatch by kind stay in the cli" 373; "The tree child rows and the status colouring stay in the cli" 379.

### Ledger: health-evaluation, The cli carries no readiness evaluator of its own

Form: c2 (REMOVED, ADDED under new names). New requirements: "No non-test Go file in the cli declares a copy of the library health package's decisions", "The readiness exemptions are not copies", "A unit test fails when a health decision is declared again or bound to a name of `opm/k8s/health`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A reintroduced evaluator fails the unit tests | "A unit test fails when a health decision is declared again or bound to a name of `opm/k8s/health`", "A reintroduced evaluator fails the unit tests" | No (byte equal, checked) |
| A re-export of the library package fails the unit tests | "A unit test fails when a health decision is declared again or bound to a name of `opm/k8s/health`", "A re-export of the library package fails the unit tests" | No (byte equal, checked) |
| (new) | "No non-test Go file in the cli declares a copy of the library health package's decisions", "An alias of the library health type is refused" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "A re-export of the library package fails the unit tests", byte for byte; only the name differs |
| (new) | "The readiness exemptions are not copies", "The colouring of FormatHealthStatus is not a copy" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| No non-test Go file in the cli SHALL declare a copy of the package's... | body of "No non-test Go file in the cli declares a copy of the library health package's decisions", verbatim |
| The exemptions named in the previous requirement copies. | body of "The readiness exemptions are not copies", with the word changes of the list "Words changed or added in unit 7b" |
| A unit test SHALL fail when a top-level declaration named `HealthSta... | body of "A unit test fails when a health decision is declared again or bound to a name of `opm/k8s/health`", verbatim |

Lengths (old, new): old 796; new "No non-test Go file in the cli declares a copy of the library health package's decisions" 313; "The readiness exemptions are not copies" 226; "A unit test fails when a health decision is declared again or bound to a name of `opm/k8s/health`" 358.

### Ledger: instance-building, Loader validates consumer values and produces a concrete ModuleInstance

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI produces validated, concrete instances exclusively through the library kernel", "The module-directory path acquires a directory with kernel AcquireModuleFromDir and SynthesizeInstance", "The published-module path acquires a module with kernel AcquireModuleFromRegistry and SynthesizeInstance", "The instance-package path acquires an instance package directory with kernel AcquireInstanceFromDir".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Successful load from module directory | "The module-directory path acquires a directory with kernel AcquireModuleFromDir and SynthesizeInstance", "Successful load from module directory" | No (byte equal, checked) |
| Successful load from instance file | "The instance-package path acquires an instance package directory with kernel AcquireInstanceFromDir", "Successful load from instance file" | No (byte equal, checked) |
| Successful synthesis from a module-package directory | "The module-directory path acquires a directory with kernel AcquireModuleFromDir and SynthesizeInstance", "Successful synthesis from a module-package directory" | No (byte equal, checked) |
| Module Gate catches type mismatch | "The CLI produces validated, concrete instances exclusively through the library kernel", "Module Gate catches type mismatch" | No (byte equal, checked) |
| Successful synthesis from a published module | "The published-module path acquires a module with kernel AcquireModuleFromRegistry and SynthesizeInstance", "Successful synthesis from a published module" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The CLI SHALL produce validated, concrete instances exclusively thro... | body of "The CLI produces validated, concrete instances exclusively through the library kernel", verbatim |
| The loading entry points map onto kernel acquire verbs: | body of "The module-directory path acquires a directory with kernel AcquireModuleFromDir and SynthesizeInstance", with the word changes of the list "Words changed or added in unit 7b" |
| 1. **Module-directory path**: kernel `AcquireModuleFromDir` + `Synth... | body of "The module-directory path acquires a directory with kernel AcquireModuleFromDir and SynthesizeInstance", verbatim |
| The loading entry points map onto kernel acquire verbs: | body of "The published-module path acquires a module with kernel AcquireModuleFromRegistry and SynthesizeInstance", with the word changes of the list "Words changed or added in unit 7b" |
| 2. **Published-module path**: kernel `AcquireModuleFromRegistry` + `... | body of "The published-module path acquires a module with kernel AcquireModuleFromRegistry and SynthesizeInstance", verbatim |
| The loading entry points map onto kernel acquire verbs: | body of "The instance-package path acquires an instance package directory with kernel AcquireInstanceFromDir", with the word changes of the list "Words changed or added in unit 7b" |
| 3. **Instance package**: kernel `AcquireInstanceFromDir` on an insta... | body of "The instance-package path acquires an instance package directory with kernel AcquireInstanceFromDir", verbatim |

Lengths (old, new): old 1593; new "The CLI produces validated, concrete instances exclusively through the library kernel" 483; "The module-directory path acquires a directory with kernel AcquireModuleFromDir and SynthesizeInstance" 491; "The published-module path acquires a module with kernel AcquireModuleFromRegistry and SynthesizeInstance" 457; "The instance-package path acquires an instance package directory with kernel AcquireInstanceFromDir" 407.

### Ledger: instance-building, Value selection falls back to module defaults when no files are given

Form: c2 (REMOVED, ADDED under new names). New requirements: "With no `--values` files the builder discovers values by a three-step priority", "The builder never reads values from `Module.Values` and `--values` files make `values.cue` and `debugValues` ignored", "An instance package acquired from its directory holds its `values` inline and has no `values.cue` fallback".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| No values file, `values.cue` exists in module directory | "With no `--values` files the builder discovers values by a three-step priority", "No values file, `values.cue` exists in module directory" | No (byte equal, checked) |
| Instance file is self-contained | "An instance package acquired from its directory holds its `values` inline and has no `values.cue` fallback", "Instance file is self-contained" | No (byte equal, checked) |
| No values files, no values.cue, debugValues defined | "With no `--values` files the builder discovers values by a three-step priority", "No values files, no values.cue, debugValues defined" | No (byte equal, checked) |
| No values files, no values.cue, no debugValues | "With no `--values` files the builder discovers values by a three-step priority", "No values files, no values.cue, no debugValues" | No (byte equal, checked) |
| Multiple `--values` files are unified | "The builder never reads values from `Module.Values` and `--values` files make `values.cue` and `debugValues` ignored", "Multiple `--values` files are unified" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| When no `--values` files are provided, the builder SHALL discover va... | body of "With no `--values` files the builder discovers values by a three-step priority", verbatim |
| 1. When `instance.cue` is present: auto-discover `values.cue` from t... | body of "With no `--values` files the builder discovers values by a three-step priority", verbatim |
| 2. When `instance.cue` is absent and `debugValues` is defined in the... | body of "With no `--values` files the builder discovers values by a three-step priority", verbatim |
| 3. When neither `instance.cue` nor `values.cue` nor `debugValues` is... | body of "With no `--values` files the builder discovers values by a three-step priority", verbatim |
| The builder SHALL NOT read values from `Module.Values`. If `--values... | body of "The builder never reads values from `Module.Values` and `--values` files make `values.cue` and `debugValues` ignored", verbatim |
| When an instance package is acquired from its directory (`AcquireIns... | body of "An instance package acquired from its directory holds its `values` inline and has no `values.cue` fallback", with the word changes of the list "Words changed or added in unit 7b" |

Lengths (old, new): old 823; new "With no `--values` files the builder discovers values by a three-step priority" 433; "The builder never reads values from `Module.Values` and `--values` files make `values.cue` and `debugValues` ignored" 143; "An instance package acquired from its directory holds its `values` inline and has no `values.cue` fallback" 250.

### Ledger: k8s-helpers, A kind the cluster does not serve is an error that names it

Form: c2 (REMOVED, ADDED under new names). New requirements: "A kind the cluster does not serve is an unreadable object with an error that names the kind and its API version", "`opm instance delete` and the stale-resource prune do not count an object of an unserved kind as already gone".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Delete with a recorded kind the cluster does not serve | "`opm instance delete` and the stale-resource prune do not count an object of an unserved kind as already gone", "Delete with a recorded kind the cluster does not serve" | No (byte equal, checked) |
| Prune with a stale kind the cluster does not serve | "`opm instance delete` and the stale-resource prune do not count an object of an unserved kind as already gone", "Prune with a stale kind the cluster does not serve" | No (byte equal, checked) |
| Apply of a kind the cluster does not serve | "A kind the cluster does not serve is an unreadable object with an error that names the kind and its API version", "Apply of a kind the cluster does not serve" | No (byte equal, checked) |
| An existence check before an apply | "A kind the cluster does not serve is an unreadable object with an error that names the kind and its API version", "An existence check before an apply" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| When discovery answers and the group and version, or the kind in it,... | body of "A kind the cluster does not serve is an unreadable object with an error that names the kind and its API version", verbatim |
| `opm instance delete` and the stale-resource prune SHALL NOT count s... | body of "`opm instance delete` and the stale-resource prune do not count an object of an unserved kind as already gone", verbatim |

Lengths (old, new): old 544; new "A kind the cluster does not serve is an unreadable object with an error that names the kind and its API version" 295; "`opm instance delete` and the stale-resource prune do not count an object of an unserved kind as already gone" 375.

### Ledger: k8s-helpers, A failed discovery request is an error

Form: c2 (REMOVED, ADDED under new names). New requirements: "A failed discovery request is not read as an unserved kind or a missing object and the API error stays in the error chain", "An apply stops at a failed discovery request and exits with code 4, 3 or 1", "`opm instance delete` exits with the same codes and any other command that reads recorded objects reports the failure through the same path as a failed read".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Discovery is forbidden during delete | "`opm instance delete` exits with the same codes and any other command that reads recorded objects reports the failure through the same path as a failed read", "Discovery is forbidden during delete" | No (byte equal, checked) |
| Discovery is forbidden during diff | "`opm instance delete` exits with the same codes and any other command that reads recorded objects reports the failure through the same path as a failed read", "Discovery is forbidden during diff" | No (byte equal, checked) |
| Discovery is unavailable during the first-install check | "An apply stops at a failed discovery request and exits with code 4, 3 or 1", "Discovery is unavailable during the first-install check" | No (byte equal, checked) |
| Discovery fails during the apply of objects | "An apply stops at a failed discovery request and exits with code 4, 3 or 1", "Discovery fails during the apply of objects" | No (byte equal, checked) |
| Discovery fails during prune | "An apply stops at a failed discovery request and exits with code 4, 3 or 1", "Discovery fails during prune" | No (byte equal, checked) |
| (new) | "A failed discovery request is not read as an unserved kind or a missing object and the API error stays in the error chain", "The error chain carries the ServiceUnavailable error" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "Discovery is unavailable during the first-install check", byte for byte; only the name differs |

| Old clause | New place |
| --- | --- |
| When a discovery request fails (the API server denies it, is unavail... | body of "A failed discovery request is not read as an unserved kind or a missing object and the API error stays in the error chain", verbatim |
| An apply SHALL stop at a failed discovery request, in its first-inst... | body of "An apply stops at a failed discovery request and exits with code 4, 3 or 1", verbatim |
| `opm instance delete` SHALL exit with the same codes when a discover... | body of "`opm instance delete` exits with the same codes and any other command that reads recorded objects reports the failure through the same path as a failed read", verbatim |

Lengths (old, new): old 710; new "A failed discovery request is not read as an unserved kind or a missing object and the API error stays in the error chain" 256; "An apply stops at a failed discovery request and exits with code 4, 3 or 1" 220; "`opm instance delete` exits with the same codes and any other command that reads recorded objects reports the failure through the same path as a failed read" 446.

### Ledger: module-synthetic-instance, Synthesize a `#ModuleInstance` from a module-package directory

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI synthesizes a concrete instance from a module CUE package directory via kernel `SynthesizeInstance`", "The synthesis loads the module as a whole CUE package and the kernel derives uuid, components, auto-secrets and standard labels in CUE", "The produced instance has kind `ModuleInstance` and the synthesis does not apply `#ModuleRelease`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Module directory loads as a whole CUE package | "The synthesis loads the module as a whole CUE package and the kernel derives uuid, components, auto-secrets and standard labels in CUE", "Module directory loads as a whole CUE package" | No (byte equal, checked) |
| Emitted kind is ModuleInstance | "The produced instance has kind `ModuleInstance` and the synthesis does not apply `#ModuleRelease`", "Emitted kind is ModuleInstance" | No (byte equal, checked) |
| No synthetic wrapper module | "The synthesis loads the module as a whole CUE package and the kernel derives uuid, components, auto-secrets and standard labels in CUE", "No synthetic wrapper module" | No (byte equal, checked) |
| One CUE context | "The synthesis loads the module as a whole CUE package and the kernel derives uuid, components, auto-secrets and standard labels in CUE", "One CUE context" | No (byte equal, checked) |
| (new) | "The CLI synthesizes a concrete instance from a module CUE package directory via kernel `SynthesizeInstance`", "Only the opm module commands synthesize" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| The CLI SHALL synthesize a concrete instance from a module CUE packa... | body of "The CLI synthesizes a concrete instance from a module CUE package directory via kernel `SynthesizeInstance`", verbatim |
| The synthesis SHALL load the module as a whole CUE package (matching... | body of "The synthesis loads the module as a whole CUE package and the kernel derives uuid, components, auto-secrets and standard labels in CUE", verbatim |
| The produced instance SHALL have `kind: "ModuleInstance"`; the synth... | body of "The produced instance has kind `ModuleInstance` and the synthesis does not apply `#ModuleRelease`", verbatim |

Lengths (old, new): old 742; new "The CLI synthesizes a concrete instance from a module CUE package directory via kernel `SynthesizeInstance`" 260; "The synthesis loads the module as a whole CUE package and the kernel derives uuid, components, auto-secrets and standard labels in CUE" 308; "The produced instance has kind `ModuleInstance` and the synthesis does not apply `#ModuleRelease`" 172.

### Ledger: module-synthetic-instance, Synthesize a `#ModuleInstance` from a published module

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI synthesizes a concrete instance from a published module through the kernel and writes nothing to disk but CUE's module cache", "Values selection, metadata defaults and the output banner of a published module are the same as for a module-package directory".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Published module synthesizes | "The CLI synthesizes a concrete instance from a published module through the kernel and writes nothing to disk but CUE's module cache", "Published module synthesizes" | No (byte equal, checked) |
| debugValues of a published module | "Values selection, metadata defaults and the output banner of a published module are the same as for a module-package directory", "debugValues of a published module" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The CLI SHALL synthesize a concrete instance from a published module... | body of "The CLI synthesizes a concrete instance from a published module through the kernel and writes nothing to disk but CUE's module cache", verbatim |
| Values selection, metadata defaults and the output banner SHALL be t... | body of "Values selection, metadata defaults and the output banner of a published module are the same as for a module-package directory", verbatim |

Lengths (old, new): old 544; new "The CLI synthesizes a concrete instance from a published module through the kernel and writes nothing to disk but CUE's module cache" 433; "Values selection, metadata defaults and the output banner of a published module are the same as for a module-package directory" 386.

### Ledger: command-reference, The command reference is generated into the docs bundle

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI reference is generated by docs-kit's `cobra` extractor into `reference/cli/` when the docs bundle is built", "Hidden and deprecated commands and flags and the `help` command are left out of the command reference and cobra's `completion` command is listed".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A new subcommand appears in the reference | "The CLI reference is generated by docs-kit's `cobra` extractor into `reference/cli/` when the docs bundle is built", "A new subcommand appears in the reference" | No (byte equal, checked) |
| A hidden command stays out | "Hidden and deprecated commands and flags and the `help` command are left out of the command reference and cobra's `completion` command is listed", "A hidden command stays out" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The CLI reference SHALL be generated by docs-kit's `cobra` extractor... | body of "The CLI reference is generated by docs-kit's `cobra` extractor into `reference/cli/` when the docs bundle is built", verbatim |
| Hidden and deprecated commands and flags and the `help` command SHAL... | body of "Hidden and deprecated commands and flags and the `help` command are left out of the command reference and cobra's `completion` command is listed", verbatim |

Lengths (old, new): old 563; new "The CLI reference is generated by docs-kit's `cobra` extractor into `reference/cli/` when the docs bundle is built" 430; "Hidden and deprecated commands and flags and the `help` command are left out of the command reference and cobra's `completion` command is listed" 262.

### Ledger: mod-apply, `opm module apply` deploys a module package via the synthetic instance flow

Form: c2 (REMOVED, ADDED under new names). New requirements: "The `module apply` subcommand accepts an optional module-package directory and deploys the synthesized `#ModuleInstance`", "The `module apply` subcommand accepts only directory inputs", "The `module apply` subcommand produces the same cluster-side effects as `opm instance apply` after the render stage".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default to current directory | "The `module apply` subcommand accepts an optional module-package directory and deploys the synthesized `#ModuleInstance`", "Default to current directory" | No (byte equal, checked) |
| File argument rejected | "The `module apply` subcommand accepts only directory inputs", "File argument rejected" | No (byte equal, checked) |
| (new) | "The `module apply` subcommand produces the same cluster-side effects as `opm instance apply` after the render stage", "Dry-run is one of the cluster-side effects" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "Dry-run makes no cluster changes", byte for byte; only the name differs |

| Old clause | New place |
| --- | --- |
| The CLI SHALL provide a `module apply` subcommand (with alias `mod a... | body of "The `module apply` subcommand accepts an optional module-package directory and deploys the synthesized `#ModuleInstance`", verbatim |
| The subcommand SHALL accept only directory inputs.<!-- Was: syntheti... | body of "The `module apply` subcommand accepts only directory inputs", verbatim |
| The subcommand SHALL produce the same cluster-side effects as `opm i... | body of "The `module apply` subcommand produces the same cluster-side effects as `opm instance apply` after the render stage", verbatim |

Lengths (old, new): old 785; new "The `module apply` subcommand accepts an optional module-package directory and deploys the synthesized `#ModuleInstance`" 417; "The `module apply` subcommand accepts only directory inputs" 328; "The `module apply` subcommand produces the same cluster-side effects as `opm instance apply` after the render stage" 452.

### Ledger: mod-apply, Flag surface matches `opm instance apply` plus `--name`

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm module apply` accepts the flags `-f` and `--platform`", "`opm module apply` accepts the flags `--name` and `-n`", "`opm module apply` accepts the flags `--kubeconfig`, `--context` and `--dry-run`", "`opm module apply` accepts the flags `--create-namespace`, `--no-prune` and `--force`", "`opm module apply` accepts the flag `--wait`", "`opm module apply` accepts the flag `--timeout`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Values files override debugValues | "`opm module apply` accepts the flags `-f` and `--platform`", "Values files override debugValues" | No (byte equal, checked) |
| Namespace flag participates in instance identity | "`opm module apply` accepts the flags `--name` and `-n`", "Namespace flag participates in instance identity" | No (byte equal, checked) |
| Name flag participates in instance identity | "`opm module apply` accepts the flags `--name` and `-n`", "Name flag participates in instance identity" | No (byte equal, checked) |
| Dry-run makes no cluster changes | "`opm module apply` accepts the flags `--kubeconfig`, `--context` and `--dry-run`", "Dry-run makes no cluster changes" | No (byte equal, checked) |
| Wait blocks until the applied resources are healthy | "`opm module apply` accepts the flag `--wait`", "Wait blocks until the applied resources are healthy" | No (byte equal, checked) |
| Wait is skipped on dry-run | "`opm module apply` accepts the flag `--wait`", "Wait is skipped on dry-run" | No (byte equal, checked) |
| Create-namespace auto-creates the target namespace | "`opm module apply` accepts the flags `--create-namespace`, `--no-prune` and `--force`", "Create-namespace auto-creates the target namespace" | No (byte equal, checked) |
| No-prune preserves stale resources | "`opm module apply` accepts the flags `--create-namespace`, `--no-prune` and `--force`", "No-prune preserves stale resources" | No (byte equal, checked) |
| Force allows 0-resource render to prune all | "`opm module apply` accepts the flags `--create-namespace`, `--no-prune` and `--force`", "Force allows 0-resource render to prune all" | No (byte equal, checked) |
| (new) | "`opm module apply` accepts the flag `--timeout`", "The timeout bounds the readiness wait" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "Wait blocks until the applied resources are healthy", byte for byte; only the name differs |

| Old clause | New place |
| --- | --- |
| The `module apply` subcommand SHALL accept the following flags with ... | body of "`opm module apply` accepts the flags `-f` and `--platform`", verbatim |
| \| Flag \| Type \| Default \| Behavior \| | body of "`opm module apply` accepts the flags `-f` and `--platform`", verbatim |
| \| --- \| --- \| --- \| --- \| | body of "`opm module apply` accepts the flags `-f` and `--platform`", verbatim |
| \| `-f`, `--values` \| repeatable string \| empty \| Values files ov... | body of "`opm module apply` accepts the flags `-f` and `--platform`", verbatim |
| \| `--platform` \| string \| resolved by precedence \| Platform modu... | body of "`opm module apply` accepts the flags `-f` and `--platform`", verbatim |
| The `module apply` subcommand SHALL accept the following flags with ... | body of "`opm module apply` accepts the flags `--name` and `-n`", verbatim |
| \| Flag \| Type \| Default \| Behavior \| | body of "`opm module apply` accepts the flags `--name` and `-n`", verbatim |
| \| --- \| --- \| --- \| --- \| | body of "`opm module apply` accepts the flags `--name` and `-n`", verbatim |
| \| `--name` \| string \| `<module>-debug` \| Synthetic `metadata.nam... | body of "`opm module apply` accepts the flags `--name` and `-n`", verbatim |
| \| `-n`, `--namespace` \| string \| from config \| Target namespace ... | body of "`opm module apply` accepts the flags `--name` and `-n`", verbatim |
| The `module apply` subcommand SHALL accept the following flags with ... | body of "`opm module apply` accepts the flags `--kubeconfig`, `--context` and `--dry-run`", verbatim |
| \| Flag \| Type \| Default \| Behavior \| | body of "`opm module apply` accepts the flags `--kubeconfig`, `--context` and `--dry-run`", verbatim |
| \| --- \| --- \| --- \| --- \| | body of "`opm module apply` accepts the flags `--kubeconfig`, `--context` and `--dry-run`", verbatim |
| \| `--kubeconfig` \| string \| from env/config \| Path to kubeconfig... | body of "`opm module apply` accepts the flags `--kubeconfig`, `--context` and `--dry-run`", verbatim |
| \| `--context` \| string \| current-context \| Kubernetes context to... | body of "`opm module apply` accepts the flags `--kubeconfig`, `--context` and `--dry-run`", verbatim |
| \| `--dry-run` \| bool \| false \| Server-side dry-run; no cluster c... | body of "`opm module apply` accepts the flags `--kubeconfig`, `--context` and `--dry-run`", verbatim |
| The `module apply` subcommand SHALL accept the following flags with ... | body of "`opm module apply` accepts the flags `--create-namespace`, `--no-prune` and `--force`", verbatim |
| \| Flag \| Type \| Default \| Behavior \| | body of "`opm module apply` accepts the flags `--create-namespace`, `--no-prune` and `--force`", verbatim |
| \| --- \| --- \| --- \| --- \| | body of "`opm module apply` accepts the flags `--create-namespace`, `--no-prune` and `--force`", verbatim |
| \| `--create-namespace` \| bool \| false \| Create the target namesp... | body of "`opm module apply` accepts the flags `--create-namespace`, `--no-prune` and `--force`", verbatim |
| \| `--no-prune` \| bool \| false \| Skip pruning of stale resources \| | body of "`opm module apply` accepts the flags `--create-namespace`, `--no-prune` and `--force`", verbatim |
| \| `--force` \| bool \| false \| Allow a 0-resource render to prune ... | body of "`opm module apply` accepts the flags `--create-namespace`, `--no-prune` and `--force`", verbatim |
| The `module apply` subcommand SHALL accept the following flags with ... | body of "`opm module apply` accepts the flag `--wait`", verbatim |
| \| Flag \| Type \| Default \| Behavior \| | body of "`opm module apply` accepts the flag `--wait`", verbatim |
| \| --- \| --- \| --- \| --- \| | body of "`opm module apply` accepts the flag `--wait`", verbatim |
| \| `--wait` \| bool \| false \| After a successful apply and invento... | body of "`opm module apply` accepts the flag `--wait`", verbatim |
| The `module apply` subcommand SHALL accept the following flags with ... | body of "`opm module apply` accepts the flag `--timeout`", verbatim |
| \| Flag \| Type \| Default \| Behavior \| | body of "`opm module apply` accepts the flag `--timeout`", verbatim |
| \| --- \| --- \| --- \| --- \| | body of "`opm module apply` accepts the flag `--timeout`", verbatim |
| \| `--timeout` \| duration \| 5m \| Bound on the CustomResourceDefin... | body of "`opm module apply` accepts the flag `--timeout`", verbatim |

Lengths (old, new): old 1418; new "`opm module apply` accepts the flags `-f` and `--platform`" 341; "`opm module apply` accepts the flags `--name` and `-n`" 347; "`opm module apply` accepts the flags `--kubeconfig`, `--context` and `--dry-run`" 367; "`opm module apply` accepts the flags `--create-namespace`, `--no-prune` and `--force`" 403; "`opm module apply` accepts the flag `--wait`" 297; "`opm module apply` accepts the flag `--timeout`" 418.

### Ledger: release-gates, Release PRs refuse local, unpublished or mismatched pins, the operator module pin included

Form: c2 (REMOVED, ADDED under new names). New requirements: "The release-pin check fails on a `replace` directive and on a pseudo-version or untagged OPM Go requirement", "The release-pin check fails on a template `-0.dev.` pin and on a tracked `local-module.cue`", "The release-pin check fails on an unserved or mismatched operator module pin", "The release-pin check reports all failures in one run and exits zero when none holds", "The release-pin check runs as a step of the `lint` job on release-please branches only", "`task deps:release-check` runs the release-pin check locally on any branch".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Clean release PR passes | "The release-pin check reports all failures in one run and exits zero when none holds", "Clean release PR passes" | No (byte equal, checked) |
| Pseudo-version library pin fails the release PR | "The release-pin check fails on a `replace` directive and on a pseudo-version or untagged OPM Go requirement", "Pseudo-version library pin fails the release PR" | No (byte equal, checked) |
| Several violations are reported together | "The release-pin check reports all failures in one run and exits zero when none holds", "Several violations are reported together" | No (byte equal, checked) |
| Recorded operator version out of step with the pinned module | "The release-pin check fails on an unserved or mismatched operator module pin", "Recorded operator version out of step with the pinned module" | No (byte equal, checked) |
| Pinned module version not served | "The release-pin check fails on an unserved or mismatched operator module pin", "Pinned module version not served" | No (byte equal, checked) |
| Tag lookup failure is not a pass | "The release-pin check fails on a `replace` directive and on a pseudo-version or untagged OPM Go requirement", "Tag lookup failure is not a pass" | No (byte equal, checked) |
| Ordinary PR is not gated | "The release-pin check runs as a step of the `lint` job on release-please branches only", "Ordinary PR is not gated" | No (byte equal, checked) |
| Release branch push is gated | "The release-pin check runs as a step of the `lint` job on release-please branches only", "Release branch push is gated" | No (byte equal, checked) |
| (new) | "The release-pin check fails on a template `-0.dev.` pin and on a tracked `local-module.cue`", "A template pin and a replace directive are reported together" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "Several violations are reported together", byte for byte; only the name differs |
| (new) | "`task deps:release-check` runs the release-pin check locally on any branch", "The task runs the check on any branch" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| A release-pin check (G1) SHALL fail, naming every offending pin and ... | body of "The release-pin check fails on a `replace` directive and on a pseudo-version or untagged OPM Go requirement", verbatim |
| - `go.mod` carries a `replace` directive; | body of "The release-pin check fails on a `replace` directive and on a pseudo-version or untagged OPM Go requirement", verbatim |
| - a `github.com/open-platform-model/*` requirement in `go.mod` is a ... | body of "The release-pin check fails on a `replace` directive and on a pseudo-version or untagged OPM Go requirement", verbatim |
| A release-pin check (G1) SHALL fail, naming every offending pin and ... | body of "The release-pin check fails on a template `-0.dev.` pin and on a tracked `local-module.cue`", verbatim |
| - a template `cue.mod/module.cue` under `templates/` pins a dependen... | body of "The release-pin check fails on a template `-0.dev.` pin and on a tracked `local-module.cue`", verbatim |
| - a `cue.mod/local-module.cue` file is tracked anywhere in the repos... | body of "The release-pin check fails on a template `-0.dev.` pin and on a tracked `local-module.cue`", verbatim |
| A release-pin check (G1) SHALL fail, naming every offending pin and ... | body of "The release-pin check fails on an unserved or mismatched operator module pin", verbatim |
| - the operator module pin in `internal/operator/pin.go` names a modu... | body of "The release-pin check fails on an unserved or mismatched operator module pin", verbatim |
| The check SHALL report all failures in one run, not only the first, ... | body of "The release-pin check reports all failures in one run and exits zero when none holds", verbatim |
| A release-pin check (G1)SHALL run as a step inside the existing `lin... | body of "The release-pin check runs as a step of the `lint` job on release-please branches only", verbatim |
| `task deps:release-check` SHALL run the same check locally on any br... | body of "`task deps:release-check` runs the release-pin check locally on any branch", verbatim |

Lengths (old, new): old 1375; new "The release-pin check fails on a `replace` directive and on a pseudo-version or untagged OPM Go requirement" 327; "The release-pin check fails on a template `-0.dev.` pin and on a tracked `local-module.cue`" 303; "The release-pin check fails on an unserved or mismatched operator module pin" 383; "The release-pin check reports all failures in one run and exits zero when none holds" 399; "The release-pin check runs as a step of the `lint` job on release-please branches only" 321; "`task deps:release-check` runs the release-pin check locally on any branch" 175.

### Ledger: release-gates, Pull requests that move library or the operator warn about missing docs bundles

Form: c2 (REMOVED, ADDED under new names). New requirements: "The `lint` job of `pr.yml` runs the step "Docs bundles for moved pins" on every pull request and fetches the base commit", "The docs bundle step checks nothing when the library version and `PinnedOperatorVersion` equal the base's", "The docs bundle step runs the dump program on the tree and looks up each printed pin anonymously", "The docs bundle step prints a GitHub warning annotation for each missing bundle and each lookup or read it cannot complete", "The docs bundle step bounds each GHCR request and has a timeout and `continue-on-error: true`", "The docs bundle step exits zero in every case that prints a warning and so never fails the job", "G1 runs the docs pins script without `--warn` and counts each printed line as a violation".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A cascade pull request moves library to a version without a bundle | "The docs bundle step prints a GitHub warning annotation for each missing bundle and each lookup or read it cannot complete", "A cascade pull request moves library to a version without a bundle" | No (byte equal, checked) |
| An ordinary pull request checks nothing | "The docs bundle step checks nothing when the library version and `PinnedOperatorVersion` equal the base's", "An ordinary pull request checks nothing" | No (byte equal, checked) |
| An unreadable pin is a warning, not a pass | "The docs bundle step prints a GitHub warning annotation for each missing bundle and each lookup or read it cannot complete", "An unreadable pin is a warning, not a pass" | No (byte equal, checked) |
| G1 still fails a release pull request | "G1 runs the docs pins script without `--warn` and counts each printed line as a violation", "G1 still fails a release pull request" | No (byte equal, checked) |
| (new) | "The `lint` job of `pr.yml` runs the step "Docs bundles for moved pins" on every pull request and fetches the base commit", "The step runs on every pull request" | New restatement scenario: only words of the old clause, no new case. |
| (new) | "The docs bundle step runs the dump program on the tree and looks up each printed pin anonymously", "The dump program runs when a pin moved" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "A cascade pull request moves library to a version without a bundle", byte for byte; only the name differs |
| (new) | "The docs bundle step bounds each GHCR request and has a timeout and `continue-on-error: true`", "Each GHCR request is bounded" | New restatement scenario: only words of the old clause, no new case. |
| (new) | "The docs bundle step exits zero in every case that prints a warning and so never fails the job", "A warning never fails the job" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "A cascade pull request moves library to a version without a bundle", byte for byte; only the name differs |

| Old clause | New place |
| --- | --- |
| `pr.yml`'s `lint` job SHALL run the step "Docs bundles for moved pin... | body of "The `lint` job of `pr.yml` runs the step "Docs bundles for moved pins" on every pull request and fetches the base commit", verbatim |
| When the `github.com/open-platform-model/library` version in `go.mod... | body of "The docs bundle step checks nothing when the library version and `PinnedOperatorVersion` equal the base's", with the word changes of the list "Words changed or added in unit 7b" |
| Otherwise SHALL run `go run ./hack/docskit-dump pins` on the tree an... | body of "The docs bundle step runs the dump program on the tree and looks up each printed pin anonymously", with the word changes of the list "Words changed or added in unit 7b" |
| Otherwise SHALL print a GitHub warning annotation, also written to t... | body of "The docs bundle step prints a GitHub warning annotation for each missing bundle and each lookup or read it cannot complete", with the word changes of the list "Words changed or added in unit 7b" |
| Each GHCR request SHALL be bounded (`--connect-timeout 10 --max-time... | body of "The docs bundle step bounds each GHCR request and has a timeout and `continue-on-error: true`", verbatim |
| The step SHALL exit zero in every one of those cases, so it never fa... | body of "The docs bundle step exits zero in every case that prints a warning and so never fails the job", verbatim |
| G1 (`release-pin-check.sh`) SHALL run the same script without `--war... | body of "G1 runs the docs pins script without `--warn` and counts each printed line as a violation", verbatim |

Lengths (old, new): old 1701; new "The `lint` job of `pr.yml` runs the step "Docs bundles for moved pins" on every pull request and fetches the base commit" 268; "The docs bundle step checks nothing when the library version and `PinnedOperatorVersion` equal the base's" 469; "The docs bundle step runs the dump program on the tree and looks up each printed pin anonymously" 291; "The docs bundle step prints a GitHub warning annotation for each missing bundle and each lookup or read it cannot complete" 479; "The docs bundle step bounds each GHCR request and has a timeout and `continue-on-error: true`" 330; "The docs bundle step exits zero in every case that prints a warning and so never fails the job" 275; "G1 runs the docs pins script without `--warn` and counts each printed line as a violation" 314.

### Words changed or added in unit 7b

- Named references: a body ends with "(Requirement: <exact header>)" or "(Requirements: <exact header>; <exact header>)", the named reference that gives it its referent ("the package", "such an object", "the same codes", "the failure", "those cases", "Otherwise", "the same check", "the same script", "the exemptions named in the previous requirement", "the module-directory path", "the subcommand", "all failures"). The bodies are: `health-evaluation` "The library health decisions cover instance status, list and tree and the apply wait", "The Established wait and the operator install dispatch by kind stay in the cli", "The tree child rows and the status colouring stay in the cli" and "No non-test Go file in the cli declares a copy of the library health package's decisions" (to "The cli takes every readiness decision from the library health package"), "The readiness exemptions are not copies" (to the two exemption requirements; its sentence names their exact headers in place of "the previous requirement", changed after unit 7b, see "Words changed or added in unit 8a"); `instance-building` "The published-module path acquires a module with kernel AcquireModuleFromRegistry and SynthesizeInstance" (to the module-directory path requirement); `k8s-helpers` "`opm instance delete` and the stale-resource prune do not count an object of an unserved kind as already gone" (to the unserved-kind requirement) and the last body of "A failed discovery request is an error" (to the apply requirement and the first body); `mod-apply` "The `module apply` subcommand accepts only directory inputs" and "The `module apply` subcommand produces the same cluster-side effects as `opm instance apply` after the render stage" (to the first body); `module-synthetic-instance` "Values selection, metadata defaults and the output banner of a published module are the same as for a module-package directory" (to the published-module body and to the three live requirements of the same spec that it names); `release-gates` "The release-pin check reports all failures in one run and exits zero when none holds" (to the three pin-check bodies), "`task deps:release-check` runs the release-pin check locally on any branch" (to the step requirement), the docs bundle bodies "checks nothing when ..." and "bounds each GHCR request ..." (to the first docs bundle body, which names the step "Docs bundles for moved pins"), the docs bundle bodies "runs the dump program on the tree and looks up each printed pin anonymously" and "prints a GitHub warning annotation ..." (to the check-nothing body), "exits zero in every case that prints a warning and so never fails the job" (to the warning body) and "G1 runs the docs pins script without `--warn` and counts each printed line as a violation" (to the first docs bundle body).
- `health-evaluation`, "The cli takes every readiness decision from the library health package" and "The library health decisions cover instance status, list and tree and the apply wait": the old sentence "This covers ..." has its own body and reads "This SHALL cover ..." (keyword, see the strength table).
- `health-evaluation`, "The Established wait and the operator install dispatch by kind stay in the cli" and "The tree child rows and the status colouring stay in the cli": the old sentence "Three readiness checks are not decisions of the package and stay in the cli: ..." is cut at the list boundary after "once they exist)". The first part ends with a period in place of the comma. The second part repeats the opening "Three readiness checks are not decisions of the package and ... stay in the cli:". Both read "and SHALL stay in the cli" (keyword).
- `health-evaluation`, "The readiness exemptions are not copies": "are not copies" reads "SHALL NOT be copies" (keyword).
- `instance-building`, the three path bodies: the intro sentence "The loading entry points map onto kernel acquire verbs:" is repeated in each and reads "SHALL map" (keyword). The list item keeps its number (1., 2., 3.) and its whole line.
- `instance-building`, "An instance package acquired from its directory holds its `values` inline and has no `values.cue` fallback": "There is no `values.cue` fallback" reads "There SHALL be no `values.cue` fallback" (keyword) and the em-dash reads a colon.
- `release-gates`, the three pin-check bodies: the old intro "A release-pin check (G1) SHALL fail, naming every offending pin and its file, when any of the following holds in the tree under test:" is repeated in each; the last bullet of each body ends with a period in place of the semicolon.
- `release-gates`, "The release-pin check runs as a step of the `lint` job on release-please branches only": "It SHALL run" reads "A release-pin check (G1) SHALL run" (the subject of the first sentence of the same old requirement).
- `release-gates`, "The docs bundle step checks nothing when the library version and `PinnedOperatorVersion` equal the base's": "it SHALL check nothing" reads "the step SHALL check nothing" (the subject "The step" of the same old text).
- `release-gates`, "The docs bundle step runs the dump program on the tree and looks up each printed pin anonymously" and "The docs bundle step prints a GitHub warning annotation for each missing bundle and each lookup or read it cannot complete": the old sentence "Otherwise it SHALL run ..., and SHALL print ..." is cut at the boundary ", and". The first part ends with a period in place of the comma. Both parts read "Otherwise the step SHALL" ("it" reads "the step", the subject of the same old text; "and" falls at the cut).
- `mod-apply`, "The `module apply` subcommand accepts an optional module-package directory and deploys the synthesized `#ModuleInstance`" and "The `module apply` subcommand accepts only directory inputs": the `<!-- Was: ... -->` comment closes the old line that holds both sentences, so both bodies carry it unchanged.
- `mod-apply`, the five flag bodies: the old intro line, the table header row and the separator row are repeated in each body, followed by the rows of that body only; the rows are old lines, unchanged.
- `command-reference`, "Hidden and deprecated commands and flags and the `help` command are left out of the command reference and cobra's `completion` command is listed": a named reference to the generation body gives the scope of "left out" and "listed".
- `module-synthetic-instance` and the other bodies not named above: built from whole old sentences, unchanged.

## Ledger entries of cli-B8, unit 8a (repo-automation, resource-conversion, resource-discovery, status-verbose, workflow-security)

### Ledger: repo-automation, The label sync keeps bot-managed and cascade labels

Form: c2 (REMOVED, ADDED under new names). New requirements: "`.github/labels.yml` lists every label that a tool applies to this repository's pull requests", "`.github/labels.yml` lists the release-please, Dependabot and release-cascade labels", "Bot-managed labels keep the color and description they carry in the repository today".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Sync after a labels.yml change deletes no bot label | "Bot-managed labels keep the color and description they carry in the repository today", "Sync after a labels.yml change deletes no bot label" | No (byte equal, checked) |
| Cascade labels exist before the cascade runs | "`.github/labels.yml` lists the release-please, Dependabot and release-cascade labels", "Cascade labels exist before the cascade runs" | No (byte equal, checked) |
| Pull-request dry run shows no deletion | "`.github/labels.yml` lists every label that a tool applies to this repository's pull requests", "Pull-request dry run shows no deletion" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `.github/labels.yml` SHALL list every label that a tool applies to t... | body of "`.github/labels.yml` lists every label that a tool applies to this repository's pull requests", verbatim |
| It SHALL list the release-please labels `autorelease: pending` and `... | body of "`.github/labels.yml` lists the release-please, Dependabot and release-cascade labels", with the word change of the list "Words changed or added in unit 8a" |
| Bot-managed labels SHALL keep the color and description they carry i... | body of "Bot-managed labels keep the color and description they carry in the repository today", verbatim |
| Source: workspace RELEASING.md, section "The cascade" (subsection La... | the `**Source**:` line of each of the three new requirements, whole |

Lengths (old, new): old 677; new "`.github/labels.yml` lists every label that a tool applies to this repository's pull requests" 175; "`.github/labels.yml` lists the release-please, Dependabot and release-cascade labels" 314; "Bot-managed labels keep the color and description they carry in the repository today" 129.

### Ledger: repo-automation, Dependabot leaves cuelang.org/go to library releases

Form: b (MODIFIED parent, ADDED split-offs). New requirements: "The cli's CUE version moves only through a library release", "Other third-party Go modules keep their weekly Dependabot updates". The parent "Dependabot leaves cuelang.org/go to library releases" stays, with a shorter body.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| New CUE release | "Dependabot leaves cuelang.org/go to library releases", "New CUE release" | No (byte equal, checked) |
| CUE arrives with a library release | "Dependabot leaves cuelang.org/go to library releases", "CUE arrives with a library release" | No (byte equal, checked) |
| Other third-party modules still update | "Dependabot leaves cuelang.org/go to library releases", "Other third-party modules still update" | No (byte equal, checked) |
| (new) | "The cli's CUE version moves only through a library release", "A library release moves CUE in the cli" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "CUE arrives with a library release", byte for byte; only the name differs |
| (new) | "Other third-party Go modules keep their weekly Dependabot updates", "A newer k8s.io/client-go is still proposed" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "Other third-party modules still update", byte for byte; only the name differs |

| Old clause | New place |
| --- | --- |
| The Dependabot `gomod` update SHALL ignore the dependencies `cuelang... | body of the parent "Dependabot leaves cuelang.org/go to library releases", with the word change of the list "Words changed or added in unit 8a" |
| the cli's CUE version moves only through a library release: the libr... | body of "The cli's CUE version moves only through a library release", with the word changes of the list "Words changed or added in unit 8a" |
| Other third-party Go modules SHALL keep their weekly updates. | body of "Other third-party Go modules keep their weekly Dependabot updates", verbatim, with a named reference to the parent |
| Source: library ADR-013, decision j4 (`adr/013-kernel-plan-walkthrou... | the `**Source**:` line of the parent and of both split-offs, whole |

Lengths (old, new): old 952; new "The cli's CUE version moves only through a library release" 346; "Other third-party Go modules keep their weekly Dependabot updates" 129; "Dependabot leaves cuelang.org/go to library releases" 304.

### Ledger: resource-conversion, Object order follows the library weight table

Form: c2 (REMOVED, ADDED under new names). New requirements: "Every CLI path orders objects by the library's kind-class weight", "Apply, delete, prune and the tree view sort through `object.Sort` and keep objects of equal weight in input order", "The `module build` and `instance build` output breaks equal weights by namespace, then name", "Library weight classes: CustomResourceDefinition to ClusterRole", "Library weight classes: StorageClass to RoleBinding", "Library weight classes: Secret to the webhook configurations", "The CLI pins the library weight order in a unit test of its own".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Ascending order for apply | "Apply, delete, prune and the tree view sort through `object.Sort` and keep objects of equal weight in input order", "Ascending order for apply" | No (byte equal, checked) |
| Descending order for delete | "Apply, delete, prune and the tree view sort through `object.Sort` and keep objects of equal weight in input order", "Descending order for delete" | No (byte equal, checked) |
| Equal weights keep input order | "Apply, delete, prune and the tree view sort through `object.Sort` and keep objects of equal weight in input order", "Equal weights keep input order" | No (byte equal, checked) |
| A claim and a Job apply after the workloads | "Library weight classes: Secret to the webhook configurations", "A claim and a Job apply after the workloads" | No (byte equal, checked) |
| Webhook configurations apply last and delete first | "Library weight classes: Secret to the webhook configurations", "Webhook configurations apply last and delete first" | No (byte equal, checked) |
| A class kind applies before the bindings and the namespaced objects | "Library weight classes: StorageClass to RoleBinding", "A class kind applies before the bindings and the namespaced objects" | No (byte equal, checked) |
| A moved weight fails the CLI's order test | "The CLI pins the library weight order in a unit test of its own", "A moved weight fails the CLI's order test" | No (byte equal, checked) |
| (new) | "Every CLI path orders objects by the library's kind-class weight", "A CLI path orders by the library weight" | New restatement scenario: only words of the old clause, no new case. |
| (new) | "The `module build` and `instance build` output breaks equal weights by namespace, then name", "Build output breaks equal weights by namespace, then name" | New restatement scenario: only words of the old clause, no new case. |
| (new) | "Library weight classes: CustomResourceDefinition to ClusterRole", "A CustomResourceDefinition applies first" | New restatement scenario: its WHEN and THEN lines are those of the old scenario "Ascending order for apply", byte for byte; only the name differs |

| Old clause | New place |
| --- | --- |
| Every CLI path that orders Kubernetes objects (apply, delete, prune,... | body of "Every CLI path orders objects by the library's kind-class weight", verbatim |
| Apply, delete, prune and the tree view SHALL sort through `object.So... | body of "Apply, delete, prune and the tree view sort through `object.Sort` and keep objects of equal weight in input order", verbatim |
| The `module build` and `instance build` output SHALL break equal wei... | body of "The `module build` and `instance build` output breaks equal weights by namespace, then name", verbatim |
| The CLI SHALL NOT keep its own weight table. | body of "Every CLI path orders objects by the library's kind-class weight", verbatim |
| The order follows Flux's staged apply (enhancement 0012, "Ordering i... | body of "Library weight classes: CustomResourceDefinition to ClusterRole", verbatim (a plain sentence, no keyword; see the strength table) |
| Ascending, the classes are: CustomResourceDefinition of `apiextensio... | body of "Library weight classes: CustomResourceDefinition to ClusterRole", with the word changes of the list "Words changed or added in unit 8a" |
| ... StorageClass and every kind whose name ends in `Class`, ...; Clu... | body of "Library weight classes: StorageClass to RoleBinding", with the word changes of the list "Words changed or added in unit 8a" |
| ... Secret and ConfigMap; Service; LimitRange; ... and last Validati... | body of "Library weight classes: Secret to the webhook configurations", with the word changes of the list "Words changed or added in unit 8a" |
| The CLI SHALL pin this order in a unit test of its own, holding ever... | body of "The CLI pins the library weight order in a unit test of its own", verbatim |
| The library's own weight-table guard test covers rows the library ad... | body of "The CLI pins the library weight order in a unit test of its own", verbatim (a plain sentence, no keyword; see the strength table) |

Lengths (old, new): old 1934; new "Every CLI path orders objects by the library's kind-class weight" 347; "Apply, delete, prune and the tree view sort through `object.Sort` and keep objects of equal weight in input order" 190; "The `module build` and `instance build` output breaks equal weights by namespace, then name" 161; "Library weight classes: CustomResourceDefinition to ClusterRole" 415; "Library weight classes: StorageClass to RoleBinding" 392; "Library weight classes: Secret to the webhook configurations" 438; "The CLI pins the library weight order in a unit test of its own" 412.

### Ledger: resource-conversion, One export feeds the apply objects and the shared render digest

Form: c2 (REMOVED, ADDED under new names). New requirements: "The CLI exports a render's compiled objects once", "The applied objects and the render digest come from the same exported set", "An export failure exits with the general error code and names the failing resource".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The digest and the apply objects come from one export | "The CLI exports a render's compiled objects once", "The digest and the apply objects come from one export" | No (byte equal, checked) |
| The render digest is the library's | "The applied objects and the render digest come from the same exported set", "The render digest is the library's" | No (byte equal, checked) |
| (new) | "An export failure exits with the general error code and names the failing resource", "A failed export names the resource" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| The CLI SHALL convert a render's compiled objects with the library's... | body of "The CLI exports a render's compiled objects once", verbatim |
| The objects the CLI applies SHALL be the exported objects, in render... | body of "The applied objects and the render digest come from the same exported set", with a named reference to "The CLI exports a render's compiled objects once" |
| An export failure SHALL exit with the general error code and name th... | body of "An export failure exits with the general error code and names the failing resource", with a named reference to "The CLI exports a render's compiled objects once" |

Lengths (old, new): old 691; new "The CLI exports a render's compiled objects once" 195; "The applied objects and the render digest come from the same exported set" 471; "An export failure exits with the general error code and names the failing resource" 151.

### Ledger: resource-discovery, Child resource discovery via ownerReference traversal

Form: c2 (REMOVED, ADDED under new names). New requirements: "The resource discovery package provides `DiscoverChildren`", "`DiscoverChildren` returns children as `[]*unstructured.Unstructured` and differs from the tree ownership walk in its return contract", "The traversal of `DiscoverChildren` is targeted, not generic", "`DiscoverChildren` skips non-workload parent kinds", "`DiscoverChildren` matches children by ownerReference uid".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Deployment children discovered | "The traversal of `DiscoverChildren` is targeted, not generic", "Deployment children discovered" | No (byte equal, checked) |
| StatefulSet children discovered | "`DiscoverChildren` matches children by ownerReference uid", "StatefulSet children discovered" | No (byte equal, checked) |
| Non-workload parents skipped | "`DiscoverChildren` skips non-workload parent kinds", "Non-workload parents skipped" | No (byte equal, checked) |
| No children exist | "The resource discovery package provides `DiscoverChildren`", "No children exist" | No (byte equal, checked) |
| API errors during child listing are non-fatal | "`DiscoverChildren` returns children as `[]*unstructured.Unstructured` and differs from the tree ownership walk in its return contract", "API errors during child listing are non-fatal" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The resource discovery package SHALL provide a `DiscoverChildren` fu... | body of "The resource discovery package provides `DiscoverChildren`", verbatim |
| It returns children as `[]*unstructured.Unstructured` for UID extrac... | body of "`DiscoverChildren` returns children as `[]*unstructured.Unstructured` and differs from the tree ownership walk in its return contract", with the word change of the list "Words changed or added in unit 8a" |
| Note: `internal/kubernetes/tree.go` already implements equivalent ow... | body of "`DiscoverChildren` returns children as `[]*unstructured.Unstructured` and differs from the tree ownership walk in its return contract", with the word change of the list "Words changed or added in unit 8a" |
| The traversal SHALL be targeted, not generic. It SHALL use knowledge... | body of "The traversal of `DiscoverChildren` is targeted, not generic", verbatim, with a named reference to "The resource discovery package provides `DiscoverChildren`" |
| \| Parent Kind \| Child Kind(s) \| Grandchild Kind(s) \| \|---------... | body of "The traversal of `DiscoverChildren` is targeted, not generic", verbatim |
| Non-workload parent kinds (ConfigMap, Secret, Service, etc.) SHALL b... | body of "`DiscoverChildren` skips non-workload parent kinds", with the word change of the list "Words changed or added in unit 8a" |
| Child matching SHALL be performed by comparing `ownerReferences[].ui... | body of "`DiscoverChildren` matches children by ownerReference uid", verbatim, with a named reference to "The resource discovery package provides `DiscoverChildren`" |

Lengths (old, new): old 1312; new "The resource discovery package provides `DiscoverChildren`" 223; "`DiscoverChildren` returns children as `[]*unstructured.Unstructured` and differs from the tree ownership walk in its return contract" 484; "The traversal of `DiscoverChildren` is targeted, not generic" 436; "`DiscoverChildren` skips non-workload parent kinds" 194; "`DiscoverChildren` matches children by ownerReference uid" 212.

### Ledger: resource-discovery, Inventory discovery reports tracked resources it could not read

Form: c2 (REMOVED, ADDED under new names). New requirements: "An inventory entry lands in exactly one of three groups and none is dropped", "Every command that reads an instance by its inventory tells the user about each unreadable entry", "`opm instance diff` says that orphan detection could not check an unreadable resource", "A warning about an unreadable entry does not change the exit code of `diff`, `tree` or `events`", "`opm instance tree` exits 5 with `no resources found` when it can read none of the tracked resources".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A forbidden read is reported as unreadable | "An inventory entry lands in exactly one of three groups and none is dropped", "A forbidden read is reported as unreadable" | No (byte equal, checked) |
| NotFound is still missing | "An inventory entry lands in exactly one of three groups and none is dropped", "NotFound is still missing" | No (byte equal, checked) |
| Tree warns about an unreadable resource | "Every command that reads an instance by its inventory tells the user about each unreadable entry", "Tree warns about an unreadable resource" | No (byte equal, checked) |
| Tree with every tracked resource unreadable | "`opm instance tree` exits 5 with `no resources found` when it can read none of the tracked resources", "Tree with every tracked resource unreadable" | No (byte equal, checked) |
| Diff warns that orphan detection is incomplete | "`opm instance diff` says that orphan detection could not check an unreadable resource", "Diff warns that orphan detection is incomplete" | No (byte equal, checked) |
| (new) | "A warning about an unreadable entry does not change the exit code of `diff`, `tree` or `events`", "A forbidden read of one tracked resource does not make tree exit 4" | New restatement scenario: the case "a Forbidden read of one tracked resource does not make `tree` exit 4" is in the old clause |

| Old clause | New place |
| --- | --- |
| When the CLI reads the live objects an instance's inventory tracks (... | body of "An inventory entry lands in exactly one of three groups and none is dropped", verbatim |
| No entry SHALL be dropped. | body of "An inventory entry lands in exactly one of three groups and none is dropped", verbatim |
| Every command that reads an instance this way (`opm instance delete`... | body of "Every command that reads an instance by its inventory tells the user about each unreadable entry", verbatim, with a named reference to "An inventory entry lands in exactly one of three groups and none is dropped" |
| For `diff`, the warning SHALL also say that orphan detection could n... | body of "`opm instance diff` says that orphan detection could not check an unreadable resource", verbatim, with a named reference to "Every command that reads an instance by its inventory tells the user about each unreadable entry" |
| A warning SHALL NOT change the exit code of `diff`, `tree` or `event... | body of "A warning about an unreadable entry does not change the exit code of `diff`, `tree` or `events`", verbatim, with a named reference to "Every command that reads an instance by its inventory tells the user about each unreadable entry" |
| When `tree` can read none of the tracked resources, it SHALL print t... | body of "`opm instance tree` exits 5 with `no resources found` when it can read none of the tracked resources", verbatim, with a named reference to "Every command that reads an instance by its inventory tells the user about each unreadable entry" |

Lengths (old, new): old 1276; new "An inventory entry lands in exactly one of three groups and none is dropped" 403; "Every command that reads an instance by its inventory tells the user about each unreadable entry" 469; "`opm instance diff` says that orphan detection could not check an unreadable resource" 205; "A warning about an unreadable entry does not change the exit code of `diff`, `tree` or `events`" 333; "`opm instance tree` exits 5 with `no resources found` when it can read none of the tracked resources" 289.

### Ledger: status-verbose, Verbose mode shows pod details for unhealthy workloads

Form: c2 (REMOVED, ADDED under new names). New requirements: "`--details` shows a pod detail block for a NotReady workload", "The pod detail block lists each pod of the unhealthy workload with name, phase and detail", "Pods of the unhealthy workload are discovered by its `.spec.selector.matchLabels`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Verbose shows pod details for NotReady Deployment | "`--details` shows a pod detail block for a NotReady workload", "Verbose shows pod details for NotReady Deployment" | No (byte equal, checked) |
| Verbose omits pod details for Ready workloads | "`--details` shows a pod detail block for a NotReady workload", "Verbose omits pod details for Ready workloads" | No (byte equal, checked) |
| Verbose omits pod details for non-workload resources | "`--details` shows a pod detail block for a NotReady workload", "Verbose omits pod details for non-workload resources" | No (byte equal, checked) |
| (new) | "The pod detail block lists each pod of the unhealthy workload with name, phase and detail", "A healthy pod reads (ready)" | New restatement scenario: the case is in the old clause (For healthy pods, "(ready)") |
| (new) | "Pods of the unhealthy workload are discovered by its `.spec.selector.matchLabels`", "Pods are listed by the workload's selector" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| The command SHALL accept a `--details` flag. When `--details` is set... | body of "`--details` shows a pod detail block for a NotReady workload", verbatim |
| The pod detail block SHALL list each pod belonging to the unhealthy ... | body of "The pod detail block lists each pod of the unhealthy workload with name, phase and detail", verbatim, with a named reference to "`--details` shows a pod detail block for a NotReady workload" |
| Pods SHALL be discovered by listing pods in the workload's namespace... | body of "Pods of the unhealthy workload are discovered by its `.spec.selector.matchLabels`", verbatim, with a named reference to "`--details` shows a pod detail block for a NotReady workload" |

Lengths (old, new): old 893; new "`--details` shows a pod detail block for a NotReady workload" 252; "The pod detail block lists each pod of the unhealthy workload with name, phase and detail" 483; "Pods of the unhealthy workload are discovered by its `.spec.selector.matchLabels`" 308.

### Ledger: status-verbose, Verbose pod detail block format

Form: c2 (REMOVED, ADDED under new names). New requirements: "The pod detail block has a fixed layout", "The pod detail block is colour-coded by ready ratio, phase, detail text and restart count".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Pod detail block format | "The pod detail block has a fixed layout", "Pod detail block format" | No (byte equal, checked) |
| (new) | "The pod detail block has a fixed layout", "Pod detail block layout" | New scenario: its THEN line holds the format block of the old body byte for byte, indented (a fenced layout moves into a THEN line when the body names the scenario) |
| (new) | "The pod detail block is colour-coded by ready ratio, phase, detail text and restart count", "The ready ratio is red when no pod is ready" | New restatement scenario: the case is in the old clause (red (none ready)) |

| Old clause | New place |
| --- | --- |
| The pod detail block SHALL be rendered below the resource table, sep... | body of "The pod detail block has a fixed layout", verbatim, with a named reference to "`--details` shows a pod detail block for a NotReady workload" |
| The format SHALL be: ``` <Kind>/<Name> (<ready>/<total> ready):     ... | THEN line of the new scenario "Pod detail block layout" of "The pod detail block has a fixed layout", verbatim; the body names the scenario |
| The ready ratio SHALL be color-coded: green (all ready), yellow (par... | body of "The pod detail block is colour-coded by ready ratio, phase, detail text and restart count", verbatim, with a named reference to "The pod detail block has a fixed layout" |

Lengths (old, new): old 635; new "The pod detail block has a fixed layout" 336; "The pod detail block is colour-coded by ready ratio, phase, detail text and restart count" 471.

### Ledger: workflow-security, Every workflow declares least-privilege permissions

Form: c2 (REMOVED, ADDED under new names). New requirements: "Every workflow file declares `permissions` at workflow level and each job grants exactly the scopes its steps use", "`release.yml` declares `permissions: {}` and grants each job the scopes it lists".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The default token turns read-only | "Every workflow file declares `permissions` at workflow level and each job grants exactly the scopes its steps use", "The default token turns read-only" | No (byte equal, checked) |
| A workflow without a permissions key | "Every workflow file declares `permissions` at workflow level and each job grants exactly the scopes its steps use", "A workflow without a permissions key" | No (byte equal, checked) |
| (new) | "`release.yml` declares `permissions: {}` and grants each job the scopes it lists", "A release job holds only its listed scopes" | New restatement scenario: only the words and values of the old clause on `release.yml` |

| Old clause | New place |
| --- | --- |
| Every file under `.github/workflows/` SHALL declare `permissions` at... | body of "Every workflow file declares `permissions` at workflow level and each job grants exactly the scopes its steps use", verbatim |
| `release.yml` SHALL declare `permissions: {}` at workflow level, and... | body of "`release.yml` declares `permissions: {}` and grants each job the scopes it lists", verbatim |
| Source: owner decision 30 (2026-10-04); security pass finding GOV-3. | the `**Source**:` line of both new requirements, whole |

Lengths (old, new): old 676; new "Every workflow file declares `permissions` at workflow level and each job grants exactly the scopes its steps use" 277; "`release.yml` declares `permissions: {}` and grants each job the scopes it lists" 329.

### Ledger: workflow-security, Publishing jobs run only from main and only on a tag

Form: c2 (REMOVED, ADDED under new names). New requirements: "The publish-templates job and the publish job of `publish-fixtures.yml` run only from main", "publish-templates and goreleaser check out the release tag as `refs/tags/<tag>`".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A manual release run from a branch | "The publish-templates job and the publish job of `publish-fixtures.yml` run only from main", "A manual release run from a branch" | No (byte equal, checked) |
| A branch name as the tag input | "publish-templates and goreleaser check out the release tag as `refs/tags/<tag>`", "A branch name as the tag input" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The publish-templates job of `release.yml` and the publish job of `p... | body of "The publish-templates job and the publish job of `publish-fixtures.yml` run only from main", verbatim |
| publish-templates and goreleaser SHALL check out the release tag as ... | body of "publish-templates and goreleaser check out the release tag as `refs/tags/<tag>`", verbatim |
| These guards stop a mistaken or bot-branch dispatch; a person with w... | body of "publish-templates and goreleaser check out the release tag as `refs/tags/<tag>`", verbatim, with a named reference to "The publish-templates job and the publish job of `publish-fixtures.yml` run only from main" (a plain sentence, no keyword; see the strength table) |
| Source: security pass review of cli PR 306. | the `**Source**:` line of both new requirements, whole |

Lengths (old, new): old 618; new "The publish-templates job and the publish job of `publish-fixtures.yml` run only from main" 245; "publish-templates and goreleaser check out the release tag as `refs/tags/<tag>`" 434.

### Words changed or added in unit 8a

- Named references: a body ends with "(Requirement: <exact header>)" or "(Requirements: <exact header>; <exact header>)", the named reference that gives it its referent. The bodies are: `repo-automation` "Other third-party Go modules keep their weekly Dependabot updates" (to the parent "Dependabot leaves cuelang.org/go to library releases", for "Other"); `resource-conversion` the three class bodies (to their neighbours, for the intro "Ascending, the classes"), "The CLI pins the library weight order in a unit test of its own" (to the first order body, for "this order") and "The applied objects and the render digest come from the same exported set" and "An export failure exits with the general error code and names the failing resource" (to the export body, for "the exported objects" and "the failing resource"); `resource-discovery` the traversal, skip and match bodies of `DiscoverChildren` (to the body that provides it, for "The traversal", "Non-workload parent kinds" and "Child matching") and the four later bodies of the unreadable-entry set (to the groups body and to the command body, for "this way", "the warning", "A warning" and "the tracked resources"); `status-verbose` the pod list, pod discovery, layout and colour bodies (to the `--details` body and the layout body, for "the pod detail block" and "the unhealthy workload"); `workflow-security` "publish-templates and goreleaser check out the release tag as `refs/tags/<tag>`" (to the first publishing body, for "These guards").
- `repo-automation`, "`.github/labels.yml` lists the release-please, Dependabot and release-cascade labels": "It SHALL list" reads "`.github/labels.yml` SHALL list" (the subject of the first sentence of the same old requirement).
- `repo-automation`, parent "Dependabot leaves cuelang.org/go to library releases" and "The cli's CUE version moves only through a library release": the old sentence is cut at ", so". The parent part ends with a period in place of the comma. The second part drops "so" at the cut, starts with a capital letter and reads "SHALL move" in place of "moves" (keyword).
- `repo-automation`, the three label bodies, the three cuelang.org/go bodies, the two `workflow-security` permission bodies and the two publishing bodies: the old `Source:` line is whole in a `**Source**:` line of each new block (the citation rule: a split block carries its full old `Source:` line as metadata); the inline `Source:` sentence leaves the body.
- `resource-conversion`, the three class bodies: the old sentence "Ascending, the classes are: ..." is cut at list boundaries (after "ClusterRole of `rbac.authorization.k8s.io`" and after "ServiceAccount, Role and RoleBinding"). The intro reads "Ascending, the classes SHALL be:" in all three (keyword, subject repeated). The first two parts end with a period in place of the semicolon.
- `resource-discovery`, "`DiscoverChildren` returns children as `[]*unstructured.Unstructured` and differs from the tree ownership walk in its return contract": "It returns" reads "`DiscoverChildren` SHALL return" (subject copied from the old text, keyword). The em-dash of the Note reads a colon.
- `resource-discovery`, "`DiscoverChildren` skips non-workload parent kinds": the em-dash reads a colon.
- `workflow-security`, "publish-templates and goreleaser check out the release tag as `refs/tags/<tag>`": the old sentences 2 and 3 are in one body, unchanged.
- `status-verbose`, "The pod detail block has a fixed layout": the old sentence "The format SHALL be:" and the fenced block of the old text move into the THEN line of the new scenario "Pod detail block layout"; the body ends with the pointer "The format is shown in the scenario `Pod detail block layout`." (the pointer of the mod-vet precedent).
- The `health-evaluation` body "The readiness exemptions are not copies" (first commit of unit 8a): "the previous requirement" reads the exact headers of the two requirements that name the exemptions.

## Ledger entries of cli-B8, unit 8b (instance-identity-labeling, mod-instance-optional, mod-list, pkg-types, platform-pull, refactoring-requirements, registry-login, secret-inventory-migration, status-wide-output)

### Ledger: instance-identity-labeling, Runtime identity injected via catalog mandatory field

Form: b (MODIFIED parent, ADDED split-offs). New requirements: "The catalog requires `#runtimeName` and CUE evaluation fails when the CLI omits it". The parent "Runtime identity injected via catalog mandatory field" stays, with a shorter body.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| CLI-applied resources carry runtime identity | "Runtime identity injected via catalog mandatory field", "CLI-applied resources carry runtime identity" | No (byte equal, checked) |
| Runtime identity stays in sync with Go constant | "Runtime identity injected via catalog mandatory field", "Runtime identity stays in sync with Go constant" | No (byte equal, checked) |
| (new) | "The catalog requires `#runtimeName` and CUE evaluation fails when the CLI omits it", "Evaluation fails when the CLI omits #runtimeName" | New restatement scenario: the case "the CLI omits it" and "CUE evaluation MUST fail" are in the old clause |

| Old clause | New place |
| --- | --- |
| The CLI's `mod apply` (and any other render entrypoint that produces... | body of the parent "Runtime identity injected via catalog mandatory field", verbatim |
| The catalog declares `#runtimeName` as a mandatory field; CUE evalua... | body of "The catalog requires `#runtimeName` and CUE evaluation fails when the CLI omits it", verbatim |
| The `#runtimeName` value drives the `app.kubernetes.io/managed-by` l... | body of the parent "Runtime identity injected via catalog mandatory field", verbatim, after the first sentence |

Lengths (old, new): old 504; new "The catalog requires `#runtimeName` and CUE evaluation fails when the CLI omits it" 103; "Runtime identity injected via catalog mandatory field" 400.

### Ledger: mod-instance-optional, mod build and mod apply work without an instance.cue file

Form: b (MODIFIED parent, ADDED split-offs). New requirements: "The commands synthesize a ModuleInstance in memory when no instance.cue exists". The parent "mod build and mod apply work without an instance.cue file" stays, with a shorter body.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| build succeeds on bare module with debugValues | "mod build and mod apply work without an instance.cue file", "build succeeds on bare module with debugValues" | No (byte equal, checked) |
| apply succeeds on bare module with debugValues | "mod build and mod apply work without an instance.cue file", "apply succeeds on bare module with debugValues" | No (byte equal, checked) |
| build with -f on bare module | "mod build and mod apply work without an instance.cue file", "build with -f on bare module" | No (byte equal, checked) |
| (new) | "The commands synthesize a ModuleInstance in memory when no instance.cue exists", "Bare module renders through a synthesized ModuleInstance" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| `opm mod build` and `opm mod apply` SHALL produce correct output whe... | body of the parent "mod build and mod apply work without an instance.cue file", verbatim |
| The commands SHALL synthesize a `ModuleInstance` in memory by fillin... | body of "The commands synthesize a ModuleInstance in memory when no instance.cue exists", with the word change of the list "Words changed or added in unit 8b" |

Lengths (old, new): old 548; new "The commands synthesize a ModuleInstance in memory when no instance.cue exists" 403; "mod build and mod apply work without an instance.cue file" 240.

### Ledger: mod-list, List metadata extraction does not depend on inventory change history

Form: b (MODIFIED parent, ADDED split-offs). New requirements: "Owner display metadata comes from the top-level `createdBy` field", "`opm instance list` does not require inventory change-history metadata". The parent "List metadata extraction does not depend on inventory change history" stays, with a shorter body.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Display metadata sourced from persisted inventory | "List metadata extraction does not depend on inventory change history", "Display metadata sourced from persisted inventory" | No (byte equal, checked) |
| (new) | "Owner display metadata comes from the top-level `createdBy` field", "Owner defaults to cli for legacy inventories" | New restatement scenario: the case "that field is omitted for legacy inventories" is in the old clause |
| (new) | "`opm instance list` does not require inventory change-history metadata", "List works without change-history metadata" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| The command SHALL extract display metadata from each persisted insta... | body of the parent "List metadata extraction does not depend on inventory change history", with the word change of the list "Words changed or added in unit 8b" |
| Owner display metadata SHALL come from the top-level `createdBy` fie... | body of "Owner display metadata comes from the top-level `createdBy` field", verbatim, with a named reference to the parent |
| The command SHALL NOT require inventory change-history metadata such... | body of "`opm instance list` does not require inventory change-history metadata", with the word change of the list "Words changed or added in unit 8b" and a named reference to the parent; the `<!-- Was ... -->` comment stays on the sentence it ends |

Lengths (old, new): old 787; new "Owner display metadata comes from the top-level `createdBy` field" 226; "`opm instance list` does not require inventory change-history metadata" 375; "List metadata extraction does not depend on inventory change history" 368.

### Ledger: pkg-types, Core types exported in pkg/

Form: c2 (REMOVED, ADDED under new names). New requirements: "Shared domain types the CLI owns are exported under `pkg/errors/` and `pkg/loader/`", "The Kubernetes object wrapper, the OPM label vocabulary and the kind-class weight table are the library's", "There is no `pkg/core/` and no `pkg/resourceorder/` package", "The inventory entry, its identity comparison, the stale set and both digests are the library's", "There is no `pkg/inventory/` package", "Module and instance metadata types are the library's", "The CLI reads an acquired module's identity fields from its decoded `Metadata`", "Instance values and module metadata are read through the library's accessors", "A module's `debugValues` and `#config` are read through the library's accessors", "There is no `pkg/bundle/` package".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| External tool imports pkg/core | "There is no `pkg/core/` and no `pkg/resourceorder/` package", "External tool imports pkg/core" | No (byte equal, checked) |
| External tool imports pkg/resourceorder | "There is no `pkg/core/` and no `pkg/resourceorder/` package", "External tool imports pkg/resourceorder" | No (byte equal, checked) |
| External tool imports pkg/inventory | "There is no `pkg/inventory/` package", "External tool imports pkg/inventory" | No (byte equal, checked) |
| External tool imports pkg/module | "Module and instance metadata types are the library's", "External tool imports pkg/module" | No (byte equal, checked) |
| pkg/bundle does not exist | "There is no `pkg/bundle/` package", "pkg/bundle does not exist" | No (byte equal, checked) |
| Scaffold reads module identity from decoded metadata | "The CLI reads an acquired module's identity fields from its decoded `Metadata`", "Scaffold reads module identity from decoded metadata" | No (byte equal, checked) |
| Render reads the instance's values and module metadata through accessors | "Instance values and module metadata are read through the library's accessors", "Render reads the instance's values and module metadata through accessors" | No (byte equal, checked) |
| Instance without module metadata | "Instance values and module metadata are read through the library's accessors", "Instance without module metadata" | No (byte equal, checked) |
| debugValues read through the module | "A module's `debugValues` and `#config` are read through the library's accessors", "debugValues read through the module" | No (byte equal, checked) |
| (new) | "Shared domain types the CLI owns are exported under `pkg/errors/` and `pkg/loader/`", "Package structure under pkg/" | New restatement scenario: only words of the old clause, no new case. |
| (new) | "The Kubernetes object wrapper, the OPM label vocabulary and the kind-class weight table are the library's", "The CLI uses the library packages for objects, labels and order" | New restatement scenario: only words of the old clause, no new case. |
| (new) | "The inventory entry, its identity comparison, the stale set and both digests are the library's", "The CLI uses the library inventory package" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| Shared domain types the CLI owns SHALL be exported under `pkg/` for ... | body of "Shared domain types the CLI owns are exported under `pkg/errors/` and `pkg/loader/`", with the word change of the list "Words changed or added in unit 8b" (em-dash) |
| The Kubernetes object wrapper, the OPM label vocabulary and the kind... | body of "The Kubernetes object wrapper, the OPM label vocabulary and the kind-class weight table are the library's", verbatim |
| There SHALL be no `pkg/core/` and no `pkg/resourceorder/` package (e... | body of "There is no `pkg/core/` and no `pkg/resourceorder/` package", verbatim |
| The inventory entry, its identity comparison, the stale set, the inv... | body of "The inventory entry, its identity comparison, the stale set and both digests are the library's", verbatim |
| There SHALL be no `pkg/inventory/` package (enhancement 0012, ...) | body of "There is no `pkg/inventory/` package", verbatim |
| Module and instance metadata types SHALL NOT be declared in `pkg/`: ... | body of "Module and instance metadata types are the library's", verbatim |
| Where the CLI holds an acquired `*module.Module`, it SHALL read the ... | body of "The CLI reads an acquired module's identity fields from its decoded `Metadata`", verbatim |
| The CLI SHALL read the other kernel artifact fields ... : the first ... | body of "Instance values and module metadata are read through the library's accessors", with the word change of the list "Words changed or added in unit 8b" (the list is cut after the second item) and a named reference to "The CLI reads an acquired module's identity fields from its decoded `Metadata`" |
| The CLI SHALL read the other kernel artifact fields ... : the third ... | body of "A module's `debugValues` and `#config` are read through the library's accessors", with the intro repeated and a named reference to "The CLI reads an acquired module's identity fields from its decoded `Metadata`" |
| There SHALL be no `pkg/bundle/` package — bundle support is not impl... | body of "There is no `pkg/bundle/` package", with the word change of the list "Words changed or added in unit 8b" (em-dash) |

Lengths (old, new): old 2922; new "Shared domain types the CLI owns are exported under `pkg/errors/` and `pkg/loader/`" 278; "The Kubernetes object wrapper, the OPM label vocabulary and the kind-class weight table are the library's" 346; "There is no `pkg/core/` and no `pkg/resourceorder/` package" 395; "The inventory entry, its identity comparison, the stale set and both digests are the library's" 388; "There is no `pkg/inventory/` package" 322; "Module and instance metadata types are the library's" 285; "The CLI reads an acquired module's identity fields from its decoded `Metadata`" 223; "Instance values and module metadata are read through the library's accessors" 490; "A module's `debugValues` and `#config` are read through the library's accessors" 367; "There is no `pkg/bundle/` package" 168.

### Ledger: platform-pull, opm platform pull writes the cluster's platform package to a directory

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm platform pull <dir>` is a cluster-facing command of the `platform` group", "`opm platform pull <dir>` writes the platform module that the cluster renders against to `<dir>`", "The module that `opm platform pull` writes is the generated module unchanged", "`opm platform pull` writes nothing to the cluster and publishes nothing".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A pulled package reproduces the cluster's render | "`opm platform pull <dir>` writes the platform module that the cluster renders against to `<dir>`", "A pulled package reproduces the cluster's render" | No (byte equal, checked) |
| The pulled module is the generated module | "The module that `opm platform pull` writes is the generated module unchanged", "The pulled module is the generated module" | No (byte equal, checked) |
| (new) | "`opm platform pull <dir>` is a cluster-facing command of the `platform` group", "The pull command accepts the cluster flags" | New restatement scenario: only words of the old clause, no new case. |
| (new) | "`opm platform pull` writes nothing to the cluster and publishes nothing", "The pull command writes nothing to the cluster" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| The CLI SHALL provide `opm platform pull <dir>`, registered under th... | body of "`opm platform pull <dir>` is a cluster-facing command of the `platform` group", verbatim |
| It SHALL read the cluster Platform through the same resolution the r... | body of "`opm platform pull <dir>` writes the platform module that the cluster renders against to `<dir>`", with the word change of the list "Words changed or added in unit 8b" |
| The written module SHALL be the generated module unchanged: reserved... | body of "The module that `opm platform pull` writes is the generated module unchanged", verbatim, with a named reference to "`opm platform pull <dir>` writes the platform module that the cluster renders against to `<dir>`" |
| The command SHALL NOT write to the cluster and SHALL NOT publish any... | body of "`opm platform pull` writes nothing to the cluster and publishes nothing", with the word change of the list "Words changed or added in unit 8b" |

Lengths (old, new): old 749; new "`opm platform pull <dir>` is a cluster-facing command of the `platform` group" 178; "`opm platform pull <dir>` writes the platform module that the cluster renders against to `<dir>`" 387; "The module that `opm platform pull` writes is the generated module unchanged" 242; "`opm platform pull` writes nothing to the cluster and publishes nothing" 88.

### Ledger: refactoring-requirements, Module and ModuleInstance identifiers are distinct

Form: b (MODIFIED parent, ADDED split-offs). New requirements: "The identifier rule applies to internal packages only". The parent "Module and ModuleInstance identifiers are distinct" stays, with a shorter body.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Instance-scoped function names | "Module and ModuleInstance identifiers are distinct", "Instance-scoped function names" | No (byte equal, checked) |
| Instance-scoped type names | "Module and ModuleInstance identifiers are distinct", "Instance-scoped type names" | No (byte equal, checked) |
| Inventory types unambiguous | "Module and ModuleInstance identifiers are distinct", "Inventory types unambiguous" | No (byte equal, checked) |
| Dead types removed | "Module and ModuleInstance identifiers are distinct", "Dead types removed" | No (byte equal, checked) |
| JSON backward compatibility preserved | "Module and ModuleInstance identifiers are distinct", "JSON backward compatibility preserved" | No (byte equal, checked) |
| (new) | "The identifier rule applies to internal packages only", "User-facing terminal output is excluded" | New restatement scenario: only words of the old clause |

| Old clause | New place |
| --- | --- |
| Internal Go identifiers (function names, type names, field names, va... | body of the parent "Module and ModuleInstance identifiers are distinct", verbatim; the `<!-- Was ... -->` comment stays on its sentence |
| This requirement applies to internal packages only. User-facing term... | body of "The identifier rule applies to internal packages only", with the word change of the list "Words changed or added in unit 8b" (keyword) and a named reference to the parent |

Lengths (old, new): old 614; new "The identifier rule applies to internal packages only" 301; "Module and ModuleInstance identifiers are distinct" 382.

### Ledger: registry-login, Login writes where the push reads

Form: c2 (REMOVED, ADDED under new names). New requirements: "`opm registry login` stores a credential in the standard OCI/docker credential file", "`opm registry login` without a host argument resolves the configured registry mapping to its host set", "The credential is verified against the registry before the write", "The write modifies only the targeted host entry of the credential file".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Single-host mapping logs in | "`opm registry login` without a host argument resolves the configured registry mapping to its host set", "Single-host mapping logs in" | No (byte equal, checked) |
| Multi-host mapping refuses with actions | "`opm registry login` without a host argument resolves the configured registry mapping to its host set", "Multi-host mapping refuses with actions" | No (byte equal, checked) |
| Bad credential never lands | "The credential is verified against the registry before the write", "Bad credential never lands" | No (byte equal, checked) |
| Shared file preserved | "The write modifies only the targeted host entry of the credential file", "Shared file preserved" | No (byte equal, checked) |
| Non-interactive refuses | "`opm registry login` stores a credential in the standard OCI/docker credential file", "Non-interactive refuses" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| `opm registry login [host]` SHALL store a credential in the standard... | body of "`opm registry login` stores a credential in the standard OCI/docker credential file", with the word change of the list "Words changed or added in unit 8b" (em-dash) |
| With a host argument, that host is targeted literally.  | body of "`opm registry login` without a host argument resolves the configured registry mapping to its host set", verbatim (a plain sentence, no keyword; see the strength table) |
| Without one, the command SHALL resolve the configured registry mappi... | body of "`opm registry login` without a host argument resolves the configured registry mapping to its host set", with the word change of the list "Words changed or added in unit 8b" |
| The credential SHALL be verified against the registry before writing... | body of "The credential is verified against the registry before the write", verbatim, with a named reference to "`opm registry login` stores a credential in the standard OCI/docker credential file" |
| The write SHALL modify only the targeted host's entry, preserving ev... | body of "The write modifies only the targeted host entry of the credential file", verbatim, with a named reference to "`opm registry login` stores a credential in the standard OCI/docker credential file" |

Lengths (old, new): old 873; new "`opm registry login` stores a credential in the standard OCI/docker credential file" 151; "`opm registry login` without a host argument resolves the configured registry mapping to its host set" 396; "The credential is verified against the registry before the write" 217; "The write modifies only the targeted host entry of the credential file" 319.

### Ledger: secret-inventory-migration, No command reads a legacy inventory Secret

Form: c2 (REMOVED, ADDED under new names). New requirements: "No command gets, lists, decodes, migrates or deletes a legacy inventory Secret", "An instance recorded only in a legacy Secret is not seen by the instance commands and applies as a first install".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Apply makes no request on Secrets | "No command gets, lists, decodes, migrates or deletes a legacy inventory Secret", "Apply makes no request on Secrets" | No (byte equal, checked) |
| Instance recorded only in a legacy Secret is a first install | "An instance recorded only in a legacy Secret is not seen by the instance commands and applies as a first install", "Instance recorded only in a legacy Secret is a first install" | No (byte equal, checked) |
| Status does not fall back to Secrets | "An instance recorded only in a legacy Secret is not seen by the instance commands and applies as a first install", "Status does not fall back to Secrets" | No (byte equal, checked) |

| Old clause | New place |
| --- | --- |
| The CLI MUST NOT get, list, decode, migrate or delete a legacy inven... | body of "No command gets, lists, decodes, migrates or deletes a legacy inventory Secret", verbatim |
| An instance whose only inventory is a legacy Secret has no `ModuleIn... | body of "An instance recorded only in a legacy Secret is not seen by the instance commands and applies as a first install", verbatim |

Lengths (old, new): old 571; new "No command gets, lists, decodes, migrates or deletes a legacy inventory Secret" 193; "An instance recorded only in a legacy Secret is not seen by the instance commands and applies as a first install" 377.

### Ledger: status-wide-output, Wide format shows replica counts for workloads

Form: c2 (REMOVED, ADDED under new names). New requirements: "Wide format adds a REPLICAS column to the status table", "Replica values come from the status and spec fields of each workload kind", "A missing or zero status replica field displays as 0".

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Deployment replicas in wide format | "Replica values come from the status and spec fields of each workload kind", "Deployment replicas in wide format" | No (byte equal, checked) |
| Deployment with partial readiness | "Replica values come from the status and spec fields of each workload kind", "Deployment with partial readiness" | No (byte equal, checked) |
| StatefulSet replicas in wide format | "Replica values come from the status and spec fields of each workload kind", "StatefulSet replicas in wide format" | No (byte equal, checked) |
| DaemonSet replicas in wide format | "Replica values come from the status and spec fields of each workload kind", "DaemonSet replicas in wide format" | No (byte equal, checked) |
| Non-workload resource in wide format | "Wide format adds a REPLICAS column to the status table", "Non-workload resource in wide format" | No (byte equal, checked) |
| (new) | "A missing or zero status replica field displays as 0", "A missing status field displays as 0" | New restatement scenario: only words of the old clause, no new case. |

| Old clause | New place |
| --- | --- |
| When the user specifies `-o wide`, the status table SHALL include a ... | body of "Wide format adds a REPLICAS column to the status table", verbatim |
| The replica values SHALL be extracted from the resource's status and... | body of "Replica values come from the status and spec fields of each workload kind", verbatim, with a named reference to "Wide format adds a REPLICAS column to the status table" |
| If a status field is missing or zero, it SHALL be displayed as `0`. | body of "A missing or zero status replica field displays as 0", verbatim, with a named reference to "Replica values come from the status and spec fields of each workload kind" |

Lengths (old, new): old 603; new "Wide format adds a REPLICAS column to the status table" 262; "Replica values come from the status and spec fields of each workload kind" 342; "A missing or zero status replica field displays as 0" 156.

### Words changed or added in unit 8b

- Named references: a body ends with "(Requirement: <exact header>)", the named reference that gives it its referent. The bodies are: `mod-instance-optional` "The commands synthesize a ModuleInstance in memory when no instance.cue exists" (to the parent, for the module directory and the values condition); `mod-list` "Owner display metadata comes from the top-level `createdBy` field" and "`opm instance list` does not require inventory change-history metadata" (to the parent, for the persisted inventory record); `platform-pull` "The module that `opm platform pull` writes is the generated module unchanged" (to the pull body, for "The written module"); `pkg-types` "Instance values and module metadata are read through the library's accessors" and "A module's `debugValues` and `#config` are read through the library's accessors" (to the identity-field body, for "the other kernel artifact fields"); `refactoring-requirements` "The identifier rule applies to internal packages only" (to the parent, for "This requirement"); `registry-login` "The credential is verified against the registry before the write" and "The write modifies only the targeted host entry of the credential file" (to the storing body, for "The credential" and "The write"); `status-wide-output` "Replica values come from the status and spec fields of each workload kind" and "A missing or zero status replica field displays as 0" (to the column body and the values body, for "The replica values" and "a status field").
- Subjects copied from the same old requirement: `mod-instance-optional`, "The commands SHALL synthesize" reads "`opm mod build` and `opm mod apply` SHALL synthesize" (the subject of the first paragraph). `mod-list`, "The command SHALL extract" and "The command SHALL NOT require" read "`opm instance list` SHALL extract" and "`opm instance list` SHALL NOT require" (the command of the scenario). `platform-pull`, "It SHALL read" and "The command SHALL NOT write" read "`opm platform pull <dir>` SHALL read" and "`opm platform pull <dir>` SHALL NOT write" (the command of the first sentence). `registry-login`, "the command SHALL resolve" reads "`opm registry login [host]` SHALL resolve" (the command of the first sentence).
- `refactoring-requirements`, "The identifier rule applies to internal packages only": "applies" reads "SHALL apply" (keyword).
- `pkg-types`, the list items of "Shared domain types the CLI owns are exported under `pkg/errors/` and `pkg/loader/`" and the body of "There is no `pkg/bundle/` package", and `registry-login`, "`opm registry login` stores a credential in the standard OCI/docker credential file": the em-dash reads a colon.
- `pkg-types`, "Instance values and module metadata are read through the library's accessors": the old list is cut after its second item at a list boundary. The intro is in both parts (the old sentence is over 500 characters on its own), and the second item ends with a period in place of the semicolon.

## Strength table

Plain sentences that carry an RFC 2119 keyword in the new text and had none in the old text:

| Old sentence | New body | Word added |
| --- | --- | --- |
| `opm operator install` is the exception: it refuses on such a resource | "Operator install refuses a resource adopted by another instance" | "it refuses" reads "it SHALL refuse" |
| An instance deploys the operator when any one of these holds, read from its record without rendering: it is na... | "An instance deploys the operator when one of three signals holds" | "any one of these holds" reads "any one of these SHALL hold" |
| With `--wait` the command also waits, bounded, for the deleted resources to disappear ("Instance delete waits ... | "With --wait the command also waits for the deleted resources to disappear" | "the command also waits" reads "the command SHALL also wait" |
| Two selection modes exist, and the caller chooses between them explicitly: | "Release and prerelease are the two selection modes" | "exist," reads "SHALL exist," |
| The write-if-absent seeding of a missing Platform is unchanged: it happens only when no CR exists, and it seed... | "The write-if-absent seeding of a missing Platform is unchanged" | "is unchanged:" reads "SHALL be unchanged:" |
| - The same generator serves the deps fallback of every other render (see "Renders fall back to their own deps"... | "The same generator serves every other render" | "serves" reads "SHALL serve" |
| Every source resolves to a platform module directory the kernel acquires; the CR source is generated into one ... | "Every platform source resolves to a platform module directory" | "Every source resolves to" reads "Every source SHALL resolve to" |
| - `opm instance diff`, `opm instance apply` and `opm module apply` keep failing on an unreachable cluster: the... | "Instance diff, instance apply and module apply keep failing on an unreachable cluster" | "keep failing" reads "SHALL keep failing" |
| A registry that hands out bearer tokens and whose token endpoint refuses the caller is a refused credential li... | "A refusing token endpoint is a refused credential" | "is a refused credential" reads "SHALL be a refused credential" |
| The gates: kernel loadability (modules only: ...) ... already-published tag. | "Publish evaluates the gates of the gate list (THEN line of the scenario "The ten gates of the gate list")" | "The gates:" reads "The gates SHALL be:"; the body sentence "The gates SHALL be those listed in scenario ..." is added |
| A release with no template changed publishes nothing and succeeds, and a re-run after a partial publish skips ... | "A release with no template changed publishes nothing" | "publishes" reads "SHALL publish", "succeeds" reads "succeed", "skips" reads "SHALL skip" |
| A `BREAKING CHANGE:` footer in a pull request body or a branch commit message SHALL NOT be relied on, because ... | "The migration note goes in the pull request body" | "the migration note goes" reads "The migration note of a breaking change SHALL go"; the trailing comma reads a period |
| The action mints the dispatch token and chooses the targets: catalog_opm and opm-operator, as release-tool edg... | "The cascade-notify action mints the dispatch token and chooses the targets" | "The action" reads "The `cascade-notify` action"; "mints" reads "SHALL mint", "chooses" reads "choose" |
| The kernel reports advisory facts as structured rows and attaches no message strings to a render result. | "The CLI words the advisory facts of the kernel as warnings" | "reports" reads "SHALL report", "attaches" reads "SHALL attach" |
| The digest leaves the value of the `app.kubernetes.io/managed-by` label out, so the CLI (`opm-cli`) and the op... | "The digest leaves out the value of the managed-by label" | "The digest leaves" reads "`status.lastAppliedRenderDigest` SHALL leave" |
| Evaluator-version skew reporting applies to a future cross-binary comparison, where the CLI and operator binar... | "Evaluator-version skew applies to a future cross-binary comparison" | "applies" reads "SHALL apply" |
| When the platform is generated from the render's own deps (`opm module build` and `opm module vet` without `--... | "A platform generated from the render's own deps carries the module's replacements" | "are carried" reads "SHALL be carried" |
| The provenance signal (`SourceLocal`, the `module-instance.opmodel.dev/source: local` annotation) keeps its fi... | "The provenance signal keeps its file-presence rule" | "keeps" reads "SHALL keep" |
| The switch is the only thing the flag does: which demands are skippable, and what a skip does to the rendered ... | "Render-bearing commands accept --skip-unprovided and pass it to the kernel" | "is" reads "SHALL be" |
| A platform the CLI generates (from the cluster Platform or from the render's own dependency pins) pins the ker... | "A platform the CLI generates is never refused for its core" | "pins" reads "SHALL pin"; "this way" reads the quoted refusal |
| **Catalog and core as a consistent set.** This covers the `cue.mod/module.cue` files of the three templates, `... | "The catalog and core pins cover a fixed set of cue.mod/module.cue files" | "This covers" reads "The consistent set SHALL cover" |
| **Catalog and core as a consistent set.** This covers the `cue.mod/module.cue` files of the three templates, `... | "The catalog and core pins also cover the remaining cue.mod/module.cue files of the fixed set" | "This covers" reads "The consistent set SHALL cover" |
| Exit codes: 0 when the working tree changed; 3 when it did not; any other code on error. | "Exit 0 means the working tree changed and exit 3 means it did not" | "Exit codes:" reads "Exit codes SHALL be:" |
| `cue.mod/module.cue` declares the package's own module path, a language version, and dependencies: the deploye... | "The cue.mod/module.cue file declares the package's module path, language version and dependencies" | "declares" reads "SHALL declare" |
| `instance.cue` declares `package instance`, imports core at the module's core major and the module at its majo... | "The instance.cue file declares the package, the imports and the module instance" | "declares" reads "SHALL declare" |
| `values.cue` declares `package instance` and one `values` field, filled as "Values start from the author's int... | "The values.cue file declares one values field" | "declares" reads "SHALL declare"; the cite of the renamed requirement reads its new header "values.cue is filled from the first applicable source and the report names it" |
| With one positional, a value containing `/` or `.` is the module path, and anything else is the instance name. | "With one positional a value with a slash or a dot is the module path" | "is" reads "SHALL be" twice |
| A prompted value passes the same checks as the flag or positional it replaces, before any registry access. | "A prompted value passes the same checks as the flag it replaces" | "passes" reads "SHALL pass" |
| Reading the module's own module file, between resolution and acquire, stays exit 3 for every failure. | "Reading the module's own module file stays exit 3 for every failure" | "stays" reads "SHALL stay" |
| The declared pair is verified against fetched coordinates wherever it later meets a registry (operator acquire... | "The declared module identity pair is verified where it meets a registry" | "The declared pair" reads "The declared pair (`spec.module.path` and `spec.module.version`)", copied from the first sentence; "is verified" reads "SHALL be verified" |
| This mirrors where the refusal lives in the system: `Discriminated: false` is a condition platform-package gener... | "The undiscriminated exit mirrors where platform-package generation refuses" | "This mirrors" reads "This SHALL mirror" |
| A catalog pin's target is the newest published release in its major. | "The targets of a catalog pin and a core pin follow the release cascade" | "is" reads "SHALL be" |
| A core pin already newer than that target stays. | "A core pin already newer than the cascade target stays" | "stays" reads "SHALL stay" |
| Each entry names the repo-relative file or directory, the module paths frozen there, and a one-sentence reason saying why the pin must stay old. | "A `.cascade-frozen` entry names the file or directory, the frozen module paths and a one-sentence reason" | "Each entry names" reads "Each entry SHALL name" |
| A pin is frozen only when the test depends on that exact old version. | "A pin is frozen only when the test depends on that exact old version" | "A pin is frozen" reads "A pin SHALL be frozen" |
| A pin that is old only because a tree was copied from an older tree, or because an upstream release moved on, is not frozen. | "A pin that is old only because a tree was copied or an upstream release moved on is not frozen" | "is not frozen" reads "SHALL NOT be frozen" |
| The effective applier identity is the ServiceAccount named by the controller Deployment's `--default-service-account` argument, ... | "The effective applier identity is the ServiceAccount named by --default-service-account when that argument is present" | "is" reads "SHALL be" |
| ... the existing cluster-reachability precondition applies and the test skips as it does today, ... | "An applier check that could not be performed skips the test" | "the test skips" reads "the test SHALL skip" |
| The module commands (`opm module build`, `apply`, `vet`) are unaffected: they synthesize the instance in the override namespace, so the override and `metadata.namespace` agree. | "The module commands are unaffected by the namespace refusal of the instance render commands" | "are unaffected" reads "SHALL be unaffected" |
| Setting either as a pod environment variable has no effect. | "The operator's registry is a command-line flag rendered from the registry value, not an environment variable" | "has no effect" reads "SHALL have no effect" |
| An open list's value domain is still judged at the leaf, so narrowing it keeps reporting `domain narrowed`. | "An open list's value domain is still judged at the leaf" | "is still judged" reads "SHALL still be judged" |
| Enhancement 0011 (...) settled that the publish gate's predecessor is found by the literal rule ... | "The publish gate's predecessor is found by the literal backward scan of the published history" | "is found" reads "SHALL be found" |
| Enhancement 0011 (...) settled that the publish gate's predecessor is found by the literal rule ... implemented in `internal/publish` over the registry's version enumeration | "The literal predecessor rule is implemented in `internal/publish` over the registry's version enumeration" | "is found" reads "SHALL be found" |
| The newest release within a major is the newest stable version; when the major has no stable version, the newest prerelease that is not a development build (for... | "The newest release within a major is the newest stable version, else the newest prerelease that is not a development build" | "SHALL be" |
| A major is skipped when it holds no selectable release, declares no `opmodel.dev/core` dependency, or declares a different core major. | "A major with no selectable release, no core dependency or a different core major is skipped, and the command exits 2 when every major is skipped" | "SHALL be" |
| The segment is module-kind and cli-CI-published only. | "The `templates` segment is module-kind and cli-CI-published only and no gate reserves a name within it" | "SHALL be" |
| No gate reserves a name within it: the publish gates admit any single-segment name made of lowercase letters, digits, dots, underscores and hyphens, `index` inc... | "The `templates` segment is module-kind and cli-CI-published only and no gate reserves a name within it" | "SHALL reserve" |
| The cli's PR check and release job refuse a template that breaks any of these rules. | "The cli's PR check and release job refuse a template that breaks a publish gate, the vet, the tree, bump, `Version` or module path rule" | "SHALL refuse" |
| Init with no template, on a directory that already holds a module tree, repairs that tree instead of scaffolding. | "Init with no template on a directory that already holds a module tree repairs that tree and adopts the version the tree states" | "SHALL repair" |
| Repair leaves the `Version` of an identity file that exists untouched. | "Init with no template on a directory that already holds a module tree repairs that tree and adopts the version the tree states" | "SHALL leave" |
| An `@vN` suffix selects the newest release within that major (stable preferred, prerelease fallback); a full semver selects the exact tag | "An `@vN` suffix floats within a major and a full semver pins a tag" | "SHALL select" |
| no suffix selects the default major of the official template the bare word names, and no other reference has a default major: an unknown bare word and a literal... | "No suffix selects a default major for an official template only" | "SHALL select" |
| `--from` accepts the same forms explicitly and any published module as a clone source; `-t/--template` is an alias. | "`--from` accepts the same forms explicitly and any published module as a clone source, and `-t/--template` is an alias" | "SHALL accept"; "SHALL be" |
| The official templates use `module.cue` and `components.cue` by convention, and every root `.cue` file carries the same `package` clause. | "A module tree contains `cue.mod/module.cue` and `identity/identity.cue`, the module body is a single CUE package at the tree root and every root `.cue` file carries the same `package` clause" | "SHALL carry"; "SHALL use" |
| That means every dependency an import in the module needs, at the version minimum version selection picks; a dependency imported for the first time at its lates... | "The exact content `cue mod tidy` produces holds every needed dependency at the version minimum version selection picks and no other" | "SHALL mean" |
| Exit codes: 0 tidied, already tidy, or check passed; 1 dependency resolution or registry failure; 2 not tidy under `--check`, or the target is not a module root... | "Tidy exit codes are 0 tidied, already tidy or check passed, 1 resolution or registry failure and 2 not tidy or not a module root" | "SHALL be" |
| The test reads the source, not its types: it sees a string or pattern predicate, ... | "The error-text test reads the source and not its types" | "SHALL read" |
| Every other apply keeps the check, and the CRD-presence and CRD field-floor gates still run for the operator's instance. | "Every other apply keeps the operator-version ceiling check and the operator's instance still runs the CRD gates" | "SHALL keep"; "SHALL still run" |
| This covers `opm instance status`, `opm instance list`, `opm instance tree`, and the `--wait` readiness wait of `opm instance apply` and `opm module apply`. | "The library health decisions cover instance status, list and tree and the apply wait" | "SHALL cover" |
| Three readiness checks are not decisions of the package and stay in the cli: ... | "The Established wait and the operator install dispatch by kind stay in the cli"; "The tree child rows and the status colouring stay in the cli" | "SHALL stay" |
| The exemptions named in the previous requirement are not copies. | "The readiness exemptions are not copies" | "SHALL NOT be" |
| The loading entry points map onto kernel acquire verbs: | the three path bodies of "Loader validates consumer values and produces a concrete ModuleInstance" | "SHALL map" |
| There is no `values.cue` fallback: the instance package is self-contained. | "An instance package acquired from its directory holds its `values` inline and has no `values.cue` fallback" | "SHALL be no" |
| Ascending, the classes are: ... | the three class bodies "Library weight classes: ..." | "SHALL be" |
| ... so the cli's CUE version moves only through a library release: ... | "The cli's CUE version moves only through a library release" | "SHALL move" |
| It returns children as `[]*unstructured.Unstructured` for UID extraction (used by the events command to match `event.involvedObject.uid`). | "`DiscoverChildren` returns children as `[]*unstructured.Unstructured` and differs from the tree ownership walk in its return contract" | "SHALL return" |
| This requirement applies to internal packages only. | "The identifier rule applies to internal packages only" | "applies" reads "SHALL apply" |

Plain sentences copied unchanged, with no keyword, into a body that carries a keyword in another sentence, in a list intro or in a SHALL intro:

| Old sentence | New body |
| --- | --- |
| The receiver computes the cascade diff with this repository's `task -x deps:cascade`. | "The cascade receiver runs on a dispatch, a daily sweep and a manual run" |
| A run from any ref other than `main` is always a dry run. | "The caller passes dry-run so that the receiver is live only when CASCADE_DRY_RUN is exactly false" |
| In a dry run the receiver computes the diff, title, body and labels and writes them to the job summary, and push... | "The dry-run switch is enforced in three places and a dry run pushes nothing" |
| The release-PR gate statuses are still posted. | "The dry-run switch is enforced in three places and a dry run pushes nothing" |
| Two cases are excepted: a release pull request whose gates-only evaluation fails before it writes its result in... | "Gate statuses are posted on every pull request into main except in two cases" |
| This change does not make either status a required check. | "Gate statuses are posted on every pull request into main except in two cases" |
| In `warn` mode a problem is posted as a success whose description starts `WARN:`. | "A same-repository release pull request gets its gate statuses from a gates-only receiver run" |
| The `cascade` Environment deploys from `main` only. | "No other job declares the cascade Environment, release.yml sets no workflow-level env key and the Environment deploys from main only" |
| Typed kernel refusals (unresolved demands) and `ValidationError` values with details keep their dedicated format... | "PrintValidationError groups CUE positions on any error" |
| The warnings are worded by the CLI from the kernel's advisory diagnostic rows (unhandled optional traits, and re... | "ShowRenderOutput logs the render warnings via the module-scoped logger" |
| This mirrors where the refusal lives in the system: `Routable: false` is the condition a platform-package gener... | "An over-subscribed contract is reported with its providing registry keys and fails the command" |
| Two majors of one catalog are two registry entries and so two providers; two transformers of one entry are one ... | "Providers are counted per enabled registry entry and the command computes no count of its own" |
| A disabled entry never defines a key. | "The colliding heading states what is left out and tells the reader to keep one defining entry enabled" |
| When a catalog member or contract type a fixture exercises is removed from the published catalog, the fixture is removed with it rather than kept unvetted; a replacement fixture is added when the replacement mechanism exists. | "A fixture does not depend on a catalog mechanism the catalog has withdrawn" |
| The rule is mechanical, not stylistic: CUE resolves modules by longest-prefix match on the module path, ... | "A test fixture module declares its path under the testing domain" |
| CUE keeps a dependency the consumer already lists at its listed version, so a consumer re-pinned to a new fixture version but left on an older core still passes `cue mod tidy --check` ... | "A consumer that names a fixture version pins every shared dependency as CUE resolves it" |
| A core pin's target is the core version that the input's catalog pins, ... The source is workspace RELEASING.md, section "The cascade", "Consistent set". | "The targets of a catalog pin and a core pin follow the release cascade" |
| A `cue.mod` file moves by `cue mod get` then `cue mod tidy`, ... A Go literal moves by editing the literal. | "A `cue.mod` pin moves by `cue mod get` then `cue mod tidy` and is never bumped by a hand edit" |
| Test inputs include a test module's `cue.mod/module.cue` and a module file a test writes from a Go literal. | "A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason" |
| The file uses the format in workspace RELEASING.md, section "Cascade files". | "A `.cascade-frozen` entry names the file or directory, the frozen module paths and a one-sentence reason" |
| Between an upstream release and the release cascade that follows it, a pin is behind but not in breach. | "A pin that is old only because a tree was copied or an upstream release moved on is not frozen" |
| The directory may be a standalone CUE module or a package inside another CUE module, such as an instance directory within a module tree. | "`opm instance build` takes one positional argument naming an instance" |
| 2. Otherwise, an argument whose first element contains a dot is a published module path. (and list items 1 and 3) | "`opm module build` classifies its argument as a local directory or a published module path before any registry access" |
| This covers a missing kubeconfig, an unreachable `kind-opm-dev` context, and an applier-grant check that could not be performed. | "A cluster-required run fails every precondition that would skip" |
| The instance file owns its namespace: `metadata.namespace` is part of the instance's identity. | "`opm instance apply`, `build`, `diff` and `vet` refuse a namespace override that differs from the instance's `metadata.namespace`" |
| A tracked resource that orphan detection could not read stays a warning that does not change the exit code. | "`opm instance diff` keeps NotFound as a new resource, behaves as before with no failure and warns on an unreadable tracked resource" |
| Everything the flow resolves: core, the catalogs, and this repository's fixture modules, is published on GHCR, and the pinned operator's built-in `--registry` default routes both `opmodel.dev/*` and `testing.opmodel.dev/*` there. | "The `cluster:operator` task installs the pinned operator without requiring a local registry container" |
| The operator's registry is a command-line flag the module renders from that value, not an environment variable: `--registry` wins over `OPM_REGISTRY`, which is read only when the flag is empty, and the operator overwrites `CUE_REGISTRY` in its own process environment. | "The operator's registry is a command-line flag rendered from the registry value, not an environment variable" |
| Its purpose is a pass/fail verdict with clear diagnostics for module authors: identity, values, and whether the module's components render with the catalogs it declares. | "`opm mod vet` validates values against `#config` and renders the synthesized instance without outputting manifests" |
| All four affect the verdict: `-f` selects the values, `-n` and `--name` set the synthesized instance's namespace and name for the render, and `--platform` renders against that platform module instead of the module-deps platform. | "`opm mod vet` accepts an optional module path argument and exposes `-f`/`--values`, `-n`/`--namespace`, `--name` and `--platform`" |
| A field is "required" for this rule when it carries the required constraint (`!`) or is a regular field with no default. | "A field that was optional and is required in the new definition reports `field made required`" |
| A defaulted field that loses its default is the default rule's finding (`default removed`), not this one's. | "A defaulted field that loses its default and the reverse transition do not report `field made required`" |
| A default is compared only where an author wrote one: a value whose disjunction marks a default with `*`. | "A default is compared only where an author wrote one, and the implicit default of a plain open list is not compared" |
| A stable-preferring selector coincides with that rule only on a prerelease-only history and would miss breaks that prerelease pinners (blessed by enhancement 0010: a subscription names exactly one catalog build) can see. | "The documentation of the selector states how it differs from the predecessor selection of the publish gate" |
| the exact content `cue mod tidy` of the CUE version the CLI embeds produces for the same tree and registry | "The exact content `cue mod tidy` produces holds every needed dependency at the version minimum version selection picks and no other" |
| The refusal names the absolute path it checked; for `opm module tidy` on an existing directory without `cue.mod`, it also points at `opm module init` (no catalo... | "Tidy refuses a path that is not a module root with exit 2 before any registry access" |
| - the run was started by `workflow_dispatch`; - the pull request's head branch starts with `release-please--`; - the pull request's head branch is the cascade b... | "The job applies on `workflow_dispatch`, on `release-please--` and `deps/cascade` head branches and on the `deps-cascade` label" |
| - the pull request changes a file under `internal/operator/`, `internal/cmd/operator/`, `templates/` or `hack/platform/`; | "The job applies when a pull request changes a file under `internal/operator/`, `internal/cmd/operator/`, `templates/` or `hack/platform/`" |
| the pull request changes one of the job's own inputs: the workflow file itself, its decision script `.github/scripts/e2e-cluster-applies.sh`, `Taskfile.yml` (wh... | "The job applies when a pull request changes the workflow file, its decision script, `Taskfile.yml` or `hack/fixtures.sh`" |
| the pull request changes one of the job's own inputs: | "The job applies when a pull request changes `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operator-rbac.yaml` or `hack/opm-config.cue`" |
| `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operator-rbac.yaml`, `hack/opm-config.cue` | "The job applies when a pull request changes `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operator-rbac.yaml` or `hack/opm-config.cue`" |
| the pull request changes one of the job's own inputs: | "The job applies when a pull request changes a Go file directly under `tests/e2e/` or anything under `tests/e2e/testdata/operator-owned/`" |
| any Go file directly under `tests/e2e/` (the cluster-backed tests and the suite's shared helpers such as `TestMain` and `runOPMWithEnv`), or anything under `tes... | "The job applies when a pull request changes a Go file directly under `tests/e2e/` or anything under `tests/e2e/testdata/operator-owned/`" |
| Later tests (the operator lifecycle test tears the operator, CRDs and every ModuleInstance down and reinstalls) leave nothing of that state for a step after the... | "An operator-owned test that fails logs its ModuleInstance, the cluster Platform and the operator's logs before its cleanup deletes the evidence" |
| When an operator-owned test fails, | "When an operator-owned test fails, a test that times out waiting for the operator to reconcile names the ModuleInstance's `Ready` condition in its failure message" |
| Types that no code in the tree or its consumers constructs (`ConfigError`, `TransformError`, `FieldError`) are removed rather than preserved. | "Types that no code constructs are removed and every exported type keeps its `.Error()` string and `Unwrap()` behavior across moves" |
| It does not see a message made by formatting the error, or a message handed to another function as a string. | "The error-text test reads the source and not its types" |
| The version-skew ceiling gate still applies. | "`opm instance apply` against an operator-owned instance acts as a spec editor only" |
| Omitting it does not leave ownership untouched: it releases this field manager's claim, and the API server prunes the operator's ownership marker. | "A thin-editor apply carries the live `spec.owner` value and never changes it" |
| CLI-executor mode (absent CR or `spec.owner: cli`) drives the direct render/apply/prune/status path; operator-owned mode drives the thin-editor apply and finalizer-delegating delete paths. | "Ownership mode resolution is implemented in exactly one function that all mutating commands consume" |
| Whether the operator removes the workloads is governed by `spec.prune`, which has no CRD default and which the CLI does not write: absent it, the operator removes the CR and deliberately orphans the workloads. | "`opm instance delete` of an operator-owned instance reports the outcome that actually occurred and never claims a prune it has not established" |
| The CR's disappearance therefore proves the finalizer completed, not that anything was pruned. | the same body |
| The four bullets and the six bullets of the lists (`errors-domain` allowed cases, `log-output-style` palette and semantic styles, `errors-domain` hint causes) | the body whose intro line carries the keyword |
| Exit codes: 0 clean, 2 findings, 3 registry unreachable, 5 not published. | "`opm catalog registry check` pulls the named published build and verifies out of band what a platform verifies when it imports the catalog" |
| A failure that asks no registry (a registry mapping that does not parse) exits 1. | "`opm catalog registry check` exits 5 for an unpublished coordinate, 3 for every other registry answer and 1 for a failure that asks no registry" |
| Platform selection is a per-command concern (`--platform`, see the `platform-resolution` capability), not a config-file field. | "There is no provider configuration value" |
| The one exception is `opm config vet`, which loads the file named on its command line into a throwaway value to validate it without touching the resolved configuration. | "Subcommands do not call `config.Load()` independently" |
| Kubernetes values resolve at point of use per command; the platform source is a per-command concern reported by every render-bearing command (see `platform-resolution`), not an initialization-time value. Provider does not exist as a resolved value. | "The CLI emits a single `\"initializing CLI\"` debug log line when `--verbose` is specified" |
| CUE evaluation contexts are owned by the per-invocation kernel, not by configuration. | "`GlobalConfig` contains no `Providers` field and no `CueContext` field" |
| The command creates: and its bullet | "`opm config init` creates the default configuration `config.cue` in `~/.opm/`" |
| Checks performed: and its two numbered items | "`opm config vet` validates the `~/.opm` configuration with two checks" |
| Each resolved value records: and its three bullets | "The CLI tracks the source of each resolved configuration value for verbose logging" |
| The two bullets `"resolved config path"` and `"resolved registry"` | "The debug log lines `\"resolved config path\"` and `\"resolved registry\"` are retained" |
| The five bullets of the field list (`Kubernetes`, `Log`, `Registry`, `ConfigPath`, `Flags`) | "`GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`" |
| The bullets `opm module tidy --check` and `opm module vet` of the gate list | "The tidy, vet and zip gates" |
| In every synthesis path the kernel unifies inputs against the resolved `#ModuleInstance` schema and lets CUE derive uuid, components, auto-secrets, and standard labels. All paths run the kernel's shape gate and concreteness enforcement, producing a `*module.Instance`. | "The CLI produces validated, concrete instances exclusively through the library kernel" |
| The three list items of the loading entry points | the path body whose intro line carries the keyword |
| The three list items of the value priority | "With no `--values` files the builder discovers values by a three-step priority" |
| The step runs with the job's read-only token. | "The `lint` job of `pr.yml` runs the step \"Docs bundles for moved pins\" on every pull request and fetches the base commit" |
| Only the `opm module` commands (`build`, `apply` and `vet`) synthesize; `opm instance build` never does. | "The CLI synthesizes a concrete instance from a module CUE package directory via kernel `SynthesizeInstance`" |
| The table rows of the flags | the flag body whose intro line carries the keyword |
| The order follows Flux's staged apply (enhancement 0012, "..."). | "Library weight classes: CustomResourceDefinition to ClusterRole" |
| The library's own weight-table guard test covers rows the library adds. | "The CLI pins the library weight order in a unit test of its own" |
| Note: `internal/kubernetes/tree.go` already implements equivalent ownership walking ... | "`DiscoverChildren` returns children as `[]*unstructured.Unstructured` and differs from the tree ownership walk in its return contract" |
| The table rows of the traversal | "The traversal of `DiscoverChildren` is targeted, not generic" |
| The three list items of the pod detail block | "The pod detail block lists each pod of the unhealthy workload with name, phase and detail" |
| (The global `--verbose` flag controls log verbosity only and does not affect status output.) | "Pods of the unhealthy workload are discovered by its `.spec.selector.matchLabels`" |
| These guards stop a mistaken or bot-branch dispatch; a person with write access can still change the workflow on a branch, ... | "publish-templates and goreleaser check out the release tag as `refs/tags/<tag>`" |
| With a host argument, that host is targeted literally. | "`opm registry login` without a host argument resolves the configured registry mapping to its host set" |
| User-facing terminal output (success messages, log prefixes, command descriptions) is governed by the UX principle that users think in terms of modules and is explicitly excluded. | "The identifier rule applies to internal packages only" |
