# Delta: module-instance-type (cli-kernel-adoption)

The CLI-side prepared-instance type is retired; the library kernel's `module.Instance` is the prepared-instance type (0006 D9). `pkg/module` keeps only the decoded metadata shapes.

## REMOVED Requirements

### Requirement: module.Instance represents a fully prepared module instance

**Reason**: The kernel's `module.Instance` is the prepared instance; the CLI type had no callers after kernel adoption.
**Migration**: `github.com/open-platform-model/library/opm/module.Instance`.

### Requirement: Config accessed via Module.Config

**Reason**: The CLI `Module` struct is deleted; the kernel's `Module`/`Instance.ConfigSchema()` carry the schema.
**Migration**: Kernel `module.Module` / `Instance.ConfigSchema()`.

### Requirement: Instance exposes MatchComponents accessor

**Reason**: Matching is kernel-internal; the accessor had no callers.
**Migration**: Kernel `module.Instance.MatchComponents`.

### Requirement: No ExecuteComponents method

**Reason**: Moot — the type it constrains is deleted.
**Migration**: None needed.

### Requirement: Old constructor removed

**Reason**: Moot — the Instance type itself is deleted; there is no constructor surface to constrain (caught during archive sync).
**Migration**: None needed.
