# Delta: instance-file-loading (cli-kernel-adoption)

`LoadInstanceFile` survives for instance-argument metadata extraction; the parse/process orchestration it fed is deleted (enhancement 0006, "The CLI replaces its own render/match pipeline entirely with the `library` kernel").

## REMOVED Requirements

### Requirement: Internal instance-file inspection returns raw parse data

**Reason**: `internal/instancefile` is deleted; render loads via the kernel.
**Migration**: Kernel `LoadInstancePackage` + `ProcessModuleInstance`.

### Requirement: Instance metadata must be concrete during parse-only extraction

**Reason**: The parse-only extraction path (`internal/instancefile`) is deleted; `cmdutil.resolveInstanceArgFromFile` extracts name/namespace via `LoadInstanceFile` directly.
**Migration**: `cmdutil` instance-arg extraction.

### Requirement: Workflow orchestration calls ParseModuleInstance then ProcessModuleInstance

**Reason**: The orchestration is kernel calls now.
**Migration**: `kernel-render` capability.
