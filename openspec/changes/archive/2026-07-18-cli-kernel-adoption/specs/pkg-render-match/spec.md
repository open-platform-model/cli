# Delta: pkg-render-match (cli-kernel-adoption)

The public matching API in `pkg/render` is deleted; matching is kernel `Match` (0006 D9). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Public matching API in pkg/render

**Reason**: `pkg/render` is deleted; the kernel's `Match` is the single matcher (0006 D9).
**Migration**: Kernel `Match`/`Plan` via `kernel-render`.

### Requirement: No CLI dependencies in matching code

**Reason**: Moot — the matching code is deleted.
**Migration**: None needed; the kernel is CLI-free by construction.
