## REMOVED Requirements

### Requirement: Module Gate validates consumer values against #module.#config

**Reason**: The CLI carries no gate system. Values are validated by the library kernel: `Kernel.ValidateConfigDetailed` for `opm module vet` and the synthesized build for `opm module build` and `opm module apply`. `LoadModuleInstanceFromValue` has no implementation in the tree, and the last artifact this capability named, `ConfigError`, is deleted by this change.

**Migration**: None for users. `opm module vet` and `opm module build` report the same `#config` violations through the kernel; the contract lives in the `mod-vet` and `instance-building` specs.

### Requirement: ConfigError provides structured field errors

**Reason**: `ConfigError` is deleted with the CLI's copy of the kernel validator. The grouped presentation reads positions from the kernel's raw CUE error tree (see the `errors-domain` and `cmdutil` specs).

**Migration**: Walk the CUE error tree with `cuelang.org/go/cue/errors.Errors`, or group it with `pkg/errors.GroupedErrorsFromError`.

### Requirement: Post-gate concreteness check

**Reason**: Concreteness is asserted by the kernel, on the merged values in `ValidateConfigDetailed` and on the built instance during synthesis. There is no CLI-side post-gate check.

**Migration**: None.
