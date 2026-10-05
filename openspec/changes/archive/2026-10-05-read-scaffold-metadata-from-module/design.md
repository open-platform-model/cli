## Context

`Kernel.AcquireModuleFromDir` (library `v1.0.0-beta.4`, `opm/kernel/acquire.go`) loads the tree and calls `module.NewModuleFromValue`, which decodes the value's `metadata` struct into `*schema.ModuleMetadata` and fails the acquire when `metadata` is absent or does not decode. So by the time either scaffold site holds `mod`, `mod.Metadata` has already decoded. The library documents `Metadata` as "may be nil", so the CLI still guards for nil.

## Goals / Non-Goals

**Goals:**
- Both scaffold sites read identity from `mod.Metadata`.
- Refusal headlines, evidence rows and exit codes stay the same for every input the e2e and unit tests cover.

**Non-Goals:**
- The other d2 sites (render, vet, instance init). They need library accessors that do not exist yet.
- Any new library API.

## Decisions

### Shape of `assertDerives`

```go
func assertDerives(ctx context.Context, k *kernel.Kernel, dir, newPath string) error {
	mod, err := k.AcquireModuleFromDir(ctx, dir)
	if err != nil {
		return fmt.Errorf("internal error: scaffolded tree does not load: %w", err)
	}
	if mod.Metadata == nil {
		return errors.New("internal error: scaffolded tree decoded no metadata")
	}
	for _, c := range []struct{ field, got, want string }{
		{"modulePath", mod.Metadata.ModulePath, newPath},
		{"version", mod.Metadata.Version, InitialVersion},
	} {
		if c.got != c.want {
			return &RefusalError{ /* unchanged refusal, keyed on c.field */ }
		}
	}
	return nil
}
```

The slice gives a fixed check order. Today's map makes it random which field is reported when both mismatch. That is the only intended observable difference, and it only shows when a donor fails both checks.

### Shape of `statedVersion`

```go
version := ""
if mod.Metadata != nil {
	version = mod.Metadata.Version
}
if version == "" {
	return "", refuse("not stated")
}
```

The `cueedit.CheckVersion` gate after it is unchanged.

## Research & Decisions

### Does reading `Metadata` change which inputs refuse?

**Context**: `LookupPath(...).String()` errors on an absent, non-concrete or non-string field. `Metadata` was filled by `Decode` into a struct of strings.

**Explored**: library `opm/module/module.go` (`NewModuleFromValue`, `decodeModuleMetadata`), `opm/kernel/acquire.go` and `opm/internal/loader/shape.go` (`ModuleSpec`, `requireConcrete`) at `v1.0.0-beta.4`, plus a hermetic `kernel.New()` run in the new `TestDetectRepair` subtests.

**Findings**:
- The acquire's shape gate requires `kind: "Module"` and concrete, non-empty `metadata.name`, `metadata.modulePath` and `metadata.version` (`ModuleSpec.RequiredConcreteFields`). An absent, empty or non-concrete field fails the acquire with "missing required field", whether or not the tree embeds core `#Module`. A non-string value passes the gate but fails `Decode`, so the acquire fails too.
- Both sites already handle acquire failure ("tree does not load"), before and after this change. So by the time either site reads a field, both are non-empty strings, and the old "does not evaluate" branch, the `statedVersion` "not stated" branch and the new nil-`Metadata` branch are all unreachable guards.
- An earlier draft of this design (and the plan review) assumed an absent field decodes to `""` for a donor outside core `#Module` and reaches the donor-defect refusal. The hermetic test shows the shape gate refuses that tree first, so no such behaviour change exists.

**Options considered**:
1. Read `Metadata`, nil-guard only (the plan entry's reading).
2. Drop the guards entirely, trusting the shape gate. The library documents `Metadata` as "may be nil", so a guard costs nothing and survives a gate change.

**Decision**: Option 1.

**Rationale**: It matches the owner's decision and the plan entry. The only observable difference is that a donor failing both checks is now reported on `modulePath` every time.

## Risks / Trade-offs

- [Risk] A future library change could fill `Metadata` from a different source than `Package`. → The library documents `Metadata` as a cache of `Package`'s `metadata` subtree ("Package wins"), and the e2e tests compare the result against the identity package.

## Error handling

Unchanged: an acquire failure is an internal error in `assertDerives` and a "tree does not load" refusal in `statedVersion`, a mismatch is the existing `RefusalError` (exit 2), and the new nil-`Metadata` branch is an internal error (exit 1). No command syntax or flags change; output changes only in which field a donor failing both checks is reported on.
