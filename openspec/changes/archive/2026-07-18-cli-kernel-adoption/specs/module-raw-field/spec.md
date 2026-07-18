# Delta: module-raw-field (cli-kernel-adoption)

The CLI `Module` struct (and its `Raw` field) is deleted with kernel adoption (0006 D9).

## REMOVED Requirements

### Requirement: Module exposes evaluated CUE value as public field

**Reason**: The CLI-side `Module` struct is deleted; the kernel's `module.Module.Package` is the evaluated value.
**Migration**: Kernel `module.Module.Package`.
