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

**Explored**: library `opm/module/module.go` (`NewModuleFromValue`, `decodeModuleMetadata`) and `opm/kernel/acquire.go` at `v1.0.0-beta.4`.

**Findings**:
- A non-concrete or non-string `metadata.version` or `metadata.modulePath` fails `Decode`, so the acquire fails first. Both sites already handle acquire failure ("tree does not load"), before and after this change.
- An absent field decodes to `""`, but only in a donor that does not embed core `#Module`. Core declares `modulePath!` and `version!` required, so in a `#Module` donor an absent field is non-concrete, `Decode` fails, and the acquire fails ("tree does not load"), unchanged. In `statedVersion` an empty version already refused as "not stated", and an absent one errored into the same refusal, so nothing changes.
- In `assertDerives`, for a non-`#Module` donor (such as the e2e `litdonor` fixture shape), an absent field used to produce the "does not evaluate" internal error. Now it is `""` and fails the comparison, so it produces the donor-defect refusal with an empty `metadata.<field>` evidence row. That is the more accurate message for a clone source whose metadata does not state the field. Official templates cannot reach this path (publish gates enforce the derivation).

**Options considered**:
1. Read `Metadata`, nil-guard only (the plan entry's reading).
2. Also treat `""` as "does not evaluate", to keep the old internal error for an absent field. This keeps an error that blames the CLI for a defect in the donor.

**Decision**: Option 1.

**Rationale**: It matches the owner's decision and the plan entry. The only inputs that behave differently are a non-`#Module` donor with no `metadata.modulePath` or `metadata.version`, which now gets the donor-defect refusal (the accurate one), and a donor failing both checks, which is now reported on `modulePath` every time.

## Risks / Trade-offs

- [Risk] A future library change could fill `Metadata` from a different source than `Package`. → The library documents `Metadata` as a cache of `Package`'s `metadata` subtree ("Package wins"), and the e2e tests compare the result against the identity package.

## Error handling

Unchanged: an acquire failure is an internal error in `assertDerives` and a "tree does not load" refusal in `statedVersion`, a mismatch is the existing `RefusalError` (exit 2), and the new nil-`Metadata` branch is an internal error (exit 1). No command syntax or flags change; output changes only for the two donor cases in the Findings above.
