## REMOVED Requirements

### Requirement: Instance file loader lives in `pkg/loader/`

**Reason**: The loader duplicated the library kernel's instance acquisition and set `CUE_REGISTRY` on the process to pass the registry. Every CLI load path now goes through a kernel acquire verb (`kernel-render` spec), including the cluster-query commands' path argument.

**Migration**: Load an instance package with `Kernel.AcquireInstanceFromDir(ctx, dir)` on a kernel constructed with the resolved registry; the instance name and namespace are on the acquired instance's metadata, and the underlying value is its package field.
