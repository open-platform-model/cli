# ADR-001: CUE as Configuration Language

## Status

Accepted

## Context

OPM needs a configuration language for defining modules, providers, and user config. YAML is the Kubernetes ecosystem default but lacks type safety — it cannot express provider module references as imports, has no built-in schema validation, and requires external tooling for constraint checking.

The CLI config file (`~/.opm/config.cue`) needs to reference CUE module providers loaded from a registry, which creates a chicken-and-egg problem: providers are loaded from a CUE registry, but the registry URL comes from the config file itself.

Config files may contain sensitive provider settings, making file permissions a practical concern.

## Decision

Use CUE as the single configuration language for modules, providers, instances, and CLI configuration. YAML was considered but rejected because it cannot express provider module references via imports and would require separate schema validation tooling.

To resolve the registry bootstrap problem, config loading uses two phases: the first phase extracts the registry URL with simple parsing before any CUE evaluation, and the second phase loads the full config with the registry available for CUE module imports.

During `config init`, directories are created with permissions `0700` and files with `0600` to protect sensitive provider settings from accidental exposure.

See also ADR-011 for how CUE evaluation is used for metadata extraction.

## Consequences

Using CUE as the single configuration language provides type-safe configuration with built-in validation, allows provider references to be expressed as native CUE imports, and enforces schema constraints at the language level without external tooling. A single language across the entire stack — modules, providers, instances, and config — reduces cognitive overhead.

The two-phase config loading adds complexity to the initialization path. CUE has a steeper learning curve than YAML for new users. Secure default permissions prevent accidental credential exposure but may surprise users expecting standard file permissions.
