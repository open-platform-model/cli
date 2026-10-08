## Context

See proposal.md. `opm operator install [flags]` gets no new flag and no changed behaviour; this
change edits two main specs.

`PlanInstall` (`internal/operator/plan_install.go`) reads every object install applies three
times before any write, and `installError` (`internal/cmd/operator/install.go`) assigns the exit
code by error type:

| Read | Where | Error | Exit |
| --- | --- | --- | --- |
| 1 | wait for terminating objects (`terminatingObjects`, `internal/operator/install.go`) | the API error, wrapped | by the read error: 4 Forbidden or Unauthorized, 3 server timeout or service unavailable, 1 otherwise |
| 2 | migration proof (`getLive`, `internal/operator/migration_plan.go`) | `MigrationReadError` | by the read error, same mapping |
| 3 | apply guard (`inventory.PreApplyExistenceCheck`), only without a record | `GuardError` | 2 |

Read 1 runs for every object of the plan, so an object that is unreadable from the start is
always refused there. Exit 2 needs reads 1 and 2 to be answered and read 3 to fail.

## Goals / Non-Goals

**Goals:** the specs state the table above.

**Non-Goals:** one exit code for all three reads; any code change.

## Decisions

### Where the rule is stated

The full rule goes into `operator-lifecycle`, in the requirement that already lists install's
checks before its first write. `apply-pruning` keeps only what belongs to the existence check:
code 2 for a read that check fails, and a pointer to `operator-lifecycle` for the rest.

Alternative considered: correct only the one scenario line in `apply-pruning`. Rejected: the
requirement sentence above it states the same wrong rule, and the reader of the install spec
would still find no exit code for an unreadable object.

The scenario "Operator install refuses an unreadable resource" keeps its name, because OpenSpec
refuses a MODIFIED requirement that drops a scenario of the main spec; its content now states
exit 4.

## Research & Decisions

### Every statement of this exit code

**Context**: the task asks for every spec and docs page, not one line.
**Explored**: `grep -rn -i` for "unreadable", "cannot read", "cannot be read", "could not be
read" and "guard" over `openspec/specs`, `docs`, `README.md` and `QUICKSTART.md`, and for exit
codes in `openspec/specs/operator-lifecycle/spec.md`, `openspec/specs/operator-migration/spec.md`
and `docs/site`.
**Found**:
1. `openspec/specs/apply-pruning/spec.md`, requirement "Pre-apply existence check on first
   install": the sentence "`opm operator install` SHALL refuse with the exit code of its other
   apply-guard refusals (2)" and the scenario "Operator install refuses an unreadable resource".
   Both wrong for the common case; both corrected.
2. `openspec/specs/operator-lifecycle/spec.md`, "Every check that can refuse install runs before
   its first write": names the three checks, states no exit code for a failed read. Extended.
3. `openspec/specs/operator-migration/spec.md`, scenario "Unreadable object refuses the install":
   states no exit code and is correct as written. Not changed.
4. `docs/`: no page states an exit code for this case.
**Decision**: change 1 and 2.
**Rationale**: 3 says nothing false, and the rule is now stated once, in 2.

### Evidence that the code does this

`TestPlanInstall_UnreadableObjectRefuses` (`internal/operator/unreadable_test.go`) denies the
read at each of the three positions and checks the error type; `TestInstallErrorMapping`
(`internal/cmd/operator/operator_test.go`) maps those types to 4, 4 and 2 for a Forbidden read.
Codes 3 and 1 for reads 1 and 2 are read from `cmdutil.ExitCodeFromK8sError`; no install test
covers them.

## Risks / Trade-offs

- [The spec now fixes exit 4 and 2 for the same cause in different checks] → that is the owner's
  decision; a later unification is a behaviour change with its own change.
