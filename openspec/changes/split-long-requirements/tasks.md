## 1. cli-B1: operator-lifecycle (17), apply-pruning (11), 28 requirements

- [x] 1.1 Re-read both specs from the branch and list the 28 long requirements by name; confirm the count against `openspec validate --specs --strict --json`
- [x] 1.2 Write the delta files `specs/operator-lifecycle/spec.md` and `specs/apply-pruning/spec.md` with the forms of `design.md`, and append the ledger subsections to `design.md`
- [x] 1.3 Grep the repo (not `openspec/changes/archive/`) for each renamed or removed header; run the byte check once per rewritten requirement
- [x] 1.4 Run the gate with N=28 and the per-spec validation; `task lint` and `task test` green, then commit `docs(openspec): split the long requirements of operator-lifecycle and apply-pruning`

## 2. cli-B2: deploy (14), platform-resolution (9), artifact-publishing (5), 28 requirements

- [x] 2.1 List the 28 long requirements of the three specs and confirm the count
- [x] 2.2 Write the delta files for the three specs and append their ledger subsections
- [x] 2.3 Grep for old headers; run the byte check per rewritten requirement
- [x] 2.4 Run the gate with N=56 and the per-spec validation; `task lint` and `task test:unit` ran (the cluster-backed `task test` did not run), then commit `docs(openspec): split the long requirements of deploy, platform-resolution and artifact-publishing`

## 3. cli-B3: release-workflow (8), kernel-render (7), deps-cascade (6), instance-init (6), authoring-commands (1), 28 requirements

- [x] 3.1 List the 28 long requirements of the five specs and confirm the count
- [x] 3.2 Write the delta files for the five specs and append their ledger subsections
- [x] 3.3 Grep for old headers; run the byte check per rewritten requirement
- [x] 3.4 Run the gate with N=84 and the per-spec validation; `task lint` and `task test:unit` ran (the cluster-backed `task test` did not run), then commit `docs(openspec): split the long requirements of release-workflow, kernel-render, deps-cascade, instance-init and authoring-commands`

## 4. cli-B4: instance-inventory (6), mod-status (6), release-cascade-wiring (6), cmdutil (5), platform-check (5), 28 requirements

- [x] 4.1 List the 28 long requirements of the five specs and confirm the count (unit 4a, `instance-inventory` 6 and `mod-status` 6: 12 long requirements; unit 4b, `release-cascade-wiring` 6, `cmdutil` 5 and `platform-check` 5: 16 long requirements)
- [x] 4.2 Write the delta files for the five specs and append their ledger subsections (unit 4a, `instance-inventory` and `mod-status`; unit 4b, `release-cascade-wiring`, `cmdutil` and `platform-check`)
- [x] 4.3 Grep for old headers; run the byte check per rewritten requirement (unit 4a, `instance-inventory` and `mod-status`; unit 4b, `release-cascade-wiring`, `cmdutil` and `platform-check`)
- [x] 4.4 Run the gate with N=112 and the per-spec validation; `task lint` and `task test:unit` ran (the cluster-backed `task test` did not run), then commit `docs(openspec): split the long requirements of instance-inventory, mod-status, release-cascade-wiring, cmdutil and platform-check` (unit 4a: gate with N=96, per-spec validation of `instance-inventory` and `mod-status`, commit `docs(openspec): split the long requirements of instance-inventory and mod-status`; unit 4b: gate with N=112, per-spec validation of `release-cascade-wiring`, `cmdutil` and `platform-check`, commit `docs(openspec): split the long requirements of release-cascade-wiring, cmdutil and platform-check`)

## 5. cli-B5: test-fixture-lineage (5), cmd-structure (4), e2e-cluster-preconditions (4), inst-commands (4), kind-cluster-tasks (4), mod-vet (4), catalog-compatibility (3), 28 requirements

- [x] 5.1 List the 28 long requirements of the seven specs and confirm the count (unit 5a, `test-fixture-lineage` 5, `cmd-structure` 4 and `e2e-cluster-preconditions` 4: 13 long requirements; unit 5b, `inst-commands` 4, `kind-cluster-tasks` 4, `mod-vet` 4 and `catalog-compatibility` 3: 15 long requirements)
- [x] 5.2 Write the delta files for the seven specs and append their ledger subsections (unit 5a, `test-fixture-lineage`, `cmd-structure` and `e2e-cluster-preconditions`; unit 5b, `inst-commands`, `kind-cluster-tasks`, `mod-vet` and `catalog-compatibility`)
- [x] 5.3 Grep for old headers; run the byte check per rewritten requirement (unit 5a, `test-fixture-lineage`, `cmd-structure` and `e2e-cluster-preconditions`; unit 5b, `inst-commands`, `kind-cluster-tasks`, `mod-vet` and `catalog-compatibility`)
- [x] 5.4 Run the gate with N=140 and the per-spec validation; `task lint` and `task test` green, then commit `docs(openspec): split the long requirements of seven test, command and compatibility specs` (unit 5a ran the gate with N=125, the per-spec validation of `test-fixture-lineage`, `cmd-structure` and `e2e-cluster-preconditions`, `task lint` and `task test:unit`, and committed `docs(openspec): split the long requirements of test-fixture-lineage, cmd-structure and e2e-cluster-preconditions`; unit 5b ran the gate with N=140, the per-spec validation of `inst-commands`, `kind-cluster-tasks`, `mod-vet` and `catalog-compatibility`, `task lint` and `task test:unit`, and committed `docs(openspec): split the long requirements of inst-commands, kind-cluster-tasks, mod-vet and catalog-compatibility`)

## 6. cli-B6: published-module-resolution (4), template-modules (4), dependency-tidy (3), e2e-cluster-workflow (3), errors-domain (3), flag-conventions (3), inventory-ownership (3), log-output-style (3), apply-preflight-gates (2), 28 requirements

- [ ] 6.1 List the 28 long requirements of the nine specs and confirm the count
- [ ] 6.2 Write the delta files for the nine specs and append their ledger subsections
- [ ] 6.3 Grep for old headers; run the byte check per rewritten requirement
- [ ] 6.4 Run the gate with N=168 and the per-spec validation; `task lint` and `task test` green, then commit `docs(openspec): split the long requirements of nine resolution, convention and workflow specs`

## 7. cli-B7: pr-workflow (3), catalog-registry-check, ci-workflow, config, config-commands, config-types, docs-bundle, health-evaluation, instance-building, k8s-helpers, mod-apply, module-synthetic-instance, release-gates (2 each), command-reference (1), 28 requirements

- [ ] 7.1 List the 28 long requirements of the fourteen specs and confirm the count
- [ ] 7.2 Write the delta files for the fourteen specs and append their ledger subsections
- [ ] 7.3 Grep for old headers; run the byte check per rewritten requirement
- [ ] 7.4 Run the gate with N=196 and the per-spec validation; `task lint` and `task test` green, then commit `docs(openspec): split the long requirements of fourteen config, workflow and helper specs`

## 8. cli-B8: repo-automation, resource-conversion, resource-discovery, status-verbose, workflow-security (2 each), instance-identity-labeling, mod-instance-optional, mod-list, pkg-types, platform-pull, refactoring-requirements, registry-login, secret-inventory-migration, status-wide-output (1 each), 19 requirements

- [ ] 8.1 List the 19 long requirements of the fourteen specs and confirm the count
- [ ] 8.2 Write the delta files for the fourteen specs and append their ledger subsections
- [ ] 8.3 Grep for old headers; run the byte check per rewritten requirement
- [ ] 8.4 Run the gate with N=215 and the per-spec validation; `task lint` and `task test` green, then commit `docs(openspec): split the long requirements of the last fourteen specs`

## 9. Restore --strict

- [ ] 9.1 In `Taskfile.yml`, make `openspec:check` run `openspec validate --all --strict --no-interactive` and drop the "--strict is off" text from its `desc`
- [ ] 9.2 In `AGENTS.md`, drop "`--strict` is off until the long-requirement rewrite lands" from the `task openspec:check` row and show `--strict` in its command
- [ ] 9.3 In `openspec/config.yaml`, drop the same text from Validation Gates line 4
- [ ] 9.4 Confirm `ci.yml` and `pr.yml` call `task openspec:check` and need no edit; confirm no other file in the repo says that `--strict` is off (grep outside `openspec/changes/archive/`)
- [ ] 9.5 `task openspec:check`, `task lint` and `task test` green, then commit `ci(openspec): validate with --strict again`

## 10. Final gate

- [ ] 10.1 Run the gate of `design.md` with `LAST=1` in a scratch copy: change validates strict, `validate --specs --strict` exits 0 after archive, no "Archive would refuse" line, no new ERROR item
- [ ] 10.2 Run `task check` in the worktree and read that `openspec:check` runs with `--strict` and exits 0
- [ ] 10.3 Check the scenario ledger of `design.md` against the 215 requirements: one subsection per rewritten requirement, no old scenario name missing
- [ ] 10.4 Commit any fix the gate found as `docs(openspec): fix <what the gate found>`; with a clean tree this task is ticked with no commit
