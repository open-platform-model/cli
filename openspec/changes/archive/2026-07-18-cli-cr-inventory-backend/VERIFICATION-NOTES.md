# Verification notes — cli-cr-inventory-backend

Status as of 2026-07-18. All 28 tasks are checked off in `tasks.md`; the code is
complete and green (`go vet`, unit tests across 26 packages, and CI-pinned
golangci-lint v2.11.3 all clean). This document records what was **actually
verified on a live cluster**, what **did not work**, and what is **left to do**
before archive. It is deliberately blunt so nothing is over-claimed.

---

## 1. What was verified live (kind-opm-dev + opm-operator v1.0.0-alpha.4)

The operator manifest (`internal/operator/dist/install.yaml`, pinned to
`v1.0.0-alpha.4`) was applied to `kind-opm-dev`; CRDs Established, operator pod
1/1 Running. Integration programs run via `go run` (they build their own client,
they do not load `~/.opm/config.cue`):

| Program | Result |
| --- | --- |
| `tests/integration/gates` | PASS — CRD presence, field floor, ceiling-skip (no Platform), status-RBAC allowed |
| `tests/integration/migration` | PASS — legacy Secret (rev 4) → CR (rev 5), `cm-stale` pruned, Secret deleted **after** the status write, Secret-only instance invisible to `GetRecord`/`ListRecords` |
| `tests/integration/deploy` | PASS |
| `tests/integration/inventory-apply` | PASS |
| `tests/integration/inventory-ops` | PASS |
| `tests/integration/inst-list` | PASS |
| `tests/integration/inst-tree` | PARTIAL — 9/10 steps (all inventory ops) pass; step 10 fails (see §2.2) |
| `tests/integration/module-apply` | BLOCKED before its inventory logic (see §2.1) |

**Operator-version ceiling (task 7.1), verified end-to-end:** a `Platform`
(`name: cluster`) was applied; the operator reconciled it and stamped
`status.operatorVersion: v1.0.0-alpha.4` (`Materialized`). Probing
`GateOperatorVersionCeiling` against that live Platform:

- `1.0.0-alpha.3`, `0.9.0` (older) → **refused** with the upgrade-the-CLI error
- `1.0.0-alpha.4` (equal), `2.0.0` (newer) → **pass**
- `dev` → **skip** with a warning

Net: every CR inventory operation (create, read, revision continuation,
rename-prune, delete-CR-last, Secret→CR migration) and the full gate battery
were confirmed against a real API server and a real operator.

---

## 2. What did NOT work

### 2.1 The real `opm` CLI commands can't load `~/.opm/config.cue` (environment, not this change)

`opm operator install`, `opm instance apply`, and `opm module apply` all load
the global config, which imports provider packages from the registry
(`opmodel.dev/.../providers@v1`, …). Against the configured
`localhost:5000+insecure` registry, that resolution fails:

```
configuration error
  Location: /var/home/emil/.opm/config.cue
  import failed: … missing ',' in argument list (and 1 more errors)
```

- This is **pre-existing and unrelated to the Secret→CR rewiring** — it is a
  registry-content / config-resolution problem (a published provider package
  the local registry serves does not parse, or the local registry is missing
  the versions the config pins).
- Consequence: I could **not** exercise the user-facing command path
  end-to-end on the cluster (config load → render a real module from the
  registry → apply → CR write). I deployed the operator by applying the
  embedded manifest directly instead of via `opm operator install`.
- The inventory building blocks that those commands call **were** verified
  directly by the integration programs; the **full command flow through the
  real binary** was not.

### 2.2 `inst-tree` step 10 — testdata `#ModuleInstance` resolution (fixture, not this change)

`inst-tree` step 10 calls `cmdutil.ResolveInstanceArg(<file path>)`, which loads
`tests/integration/inst-tree/testdata/instance.cue`. That file imports
`opmodel.dev/core/v1alpha1/modulerelease@v1` (pinned `core@v1 v1.3.10`,
`language.version v0.15.0`) and references `mr.#ModuleInstance`. It fails with:

```
building instance file: undefined field: #ModuleInstance
```

- The failing code path (`ResolveInstanceArg` → `loader.LoadInstanceFile` → CUE
  build) is **untouched by this change** — the `inst-tree` rewiring only swapped
  inventory calls. Steps 1–9 (apply, discover, resolve-by-name, resolve-by-uuid,
  and all inventory assertions) pass.
- Likely cause: a `core@v1 v1.3.10` / `language v0.15.0` vs CUE v0.17.1-SDK
  resolution quirk for this fixture, or the local registry not serving a
  `modulerelease@v1` that exports `#ModuleInstance`. A standalone `cue eval` of
  the same import did **not** raise "undefined field", so this is
  fixture/registry-specific, not a schema deletion.

### 2.3 Provenance `source: local` annotation — write + SSA-removal not verified live

The provenance detection (`HasLocalModuleReplacement`) and the SSA write/omit
mechanism are unit-tested, but **no live test exercises the annotation**:

- The integration programs write CRs via a `writeInventoryCR` helper that does
  not set `SourceLocal`, so the annotation is never stamped in these runs.
- The only path that stamps it is a real `opm instance apply` / `opm module
  apply` of a locally-replaced module — which is blocked by §2.1.
- Therefore the D7 scenarios "local render stamps the annotation" and "registry
  re-apply clears it (SSA field-ownership removal)" have **not** been confirmed
  end-to-end on a cluster. This is the one behavior in the change with only
  unit-level coverage.

### 2.4 `inst-tree` namespace-termination flakiness (test hygiene, not this change)

Running `inst-tree` back-to-back fails because its own `cleanup()` deletes the
`opm-tree-test` namespace and Step 1 recreates resources before termination
finishes (`namespace … is being terminated`). It passes on a clean cluster.
Pre-existing test-isolation weakness; worth adding a wait-for-terminated guard.

---

## 3. What is left

Nothing blocks the code, but before archive:

1. **Resolve the local-registry/config breakage (§2.1)** so the real CLI
   commands and `module-apply`/`inst-tree` fixtures run. Either republish the
   provider + `core` packages the config/testdata pin to `localhost:5000`, fix
   the offending published provider package, or point the registry at the public
   GHCR mirror. This is an environment task, tracked separately from this change.
2. **End-to-end the provenance annotation (§2.3)** once §2.1 is fixed: a real
   `opm module apply` of a module with a `cue.mod/local-module.cue` replaceWith
   should stamp `module-instance.opmodel.dev/source: local`, and a subsequent
   registry-resolved apply should remove it (SSA field ownership). Add this to
   `tests/integration` or a `tests/e2e` case.
3. **Full-command e2e** (optional but recommended): a real `opm instance apply
   <file>` → CR create/status, `opm instance status/list/delete`, exercising the
   config-load + render path the integration programs bypass.
4. **`inst-tree` fixture fix (§2.2)** and the namespace-terminated guard (§2.4).
5. **`lastAppliedSourceDigest` semantics** — currently a digest of the canonical
   module reference (`path@version`), an interim identity value. Kernel adoption
   (slice C2) should replace it with the operator's module-source-bytes digest
   (or the spec should record the interim as intended). See `apply.go`
   `sourceDigest`.
6. **opm-operator housekeeping**: the A6 change
   (`operator-platform-status-operator-version`) is merged and released
   (v1.0.0-alpha.3 / alpha.4) but its OpenSpec change dir is **not archived** in
   the opm-operator repo. Archive it there.
7. **Regenerate the public command reference** in `opmodel.dev/` from the updated
   `instance apply` / `module apply` help text (this change updated the source
   help strings; the generated site pages are downstream).
8. **Archive this change** (`openspec archive`): on archive, record the
   `history` event's `slice: cli/<archive-date>-cli-cr-inventory-backend` in
   `enhancements/0006/config.yaml` (the event body is already appended; only the
   archive-date slice ref is pending), and flip
   `enhancements/0006` implementation status if appropriate.
9. **Commit the branch** `feat/cli-cr-inventory-backend` — nothing is committed
   yet.

---

## 4. Environment reset

The `kind-opm-dev` cluster was returned to a clean state after testing: the
`Platform`, the operator (Deployment/RBAC/Service), all three opmodel CRDs, the
`opm-operator-system` namespace, the leftover `opm-module-apply-itest`
namespace, and the temporary ceiling-probe program + built test binary were all
removed. No opmodel CRDs, opm namespaces, or `opm-operator` cluster RBAC remain.
