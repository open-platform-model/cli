## Context

OPM CLI commands (`delete`, `status`) discover cluster resources via label selectors. Currently:

1. **Union behavior**: When both `--name` and `--release-id` are provided, the system queries with BOTH selectors and returns the union—resources matching either. This is surprising: a typo in `--release-id` combined with correct `--name` still finds and deletes resources.

2. **Silent success on no match**: When no resources match, commands log "no resources found" and exit 0. This masks user errors (wrong namespace, typo in name).

## Goals / Non-Goals

**Goals:**

- Make selector behavior predictable: one selector type per invocation
- Surface errors when no resources match (fail-fast)
- Add `--release-id` flag to `status` command for parity with `delete`

**Non-Goals:**

- Implementing `--ignore-not-found` flag (documented as TODO)
- Changing `diff` command (it renders locally, different discovery model)

## Decisions

### D1: Mutual exclusivity of `--name` and `--release-id`

**Decision**: Error if both flags provided. User must choose one selector type.

**Rationale**:

- Eliminates union confusion
- Makes intent explicit
- Simpler mental model: "I'm selecting by name" OR "I'm selecting by release-id"

**Implementation**: Validate in command layer (`mod_delete.go`, `mod_status.go`) before calling kubernetes package.

### D2: Fail on no resources found

**Decision**: Return error (non-zero exit) when selector matches zero resources.

**Error message format**:

```
Error: no resources found for module "<name>" in namespace "<namespace>"
```

or

```
Error: no resources found for release-id "<uuid>" in namespace "<namespace>"
```

**Rationale**:

- Matches kubectl behavior (errors on not found by default)
- Catches typos and misconfigurations early
- Scripts/automation can detect failures

**Future**: `--ignore-not-found` flag will provide idempotent semantics when needed.

### D3: `--namespace` always required

**Decision**: Keep `--namespace` flag required for both selector types.

**Rationale**:

- Safety: prevents accidental cluster-wide operations
- Consistency: same flag requirements regardless of selector type
- Even though release-id is globally unique, scoping to namespace is a reasonable guard

## Risks / Trade-offs

### R1: Breaking change for existing scripts

**Risk**: Scripts using both `--name` and `--release-id` will fail.
**Mitigation**: Project is v0.x, breaking changes expected. Document in CHANGELOG.

### R2: Fail-on-no-match breaks idempotent workflows

**Risk**: Running `delete` twice fails on second run.
**Mitigation**: Document `--ignore-not-found` as planned feature in TODO.md. Users can check `status` first if needed.
