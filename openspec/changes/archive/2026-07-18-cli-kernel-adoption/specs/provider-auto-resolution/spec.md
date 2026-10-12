# Delta: provider-auto-resolution (cli-kernel-adoption)

Provider auto-resolution is retired with the provider concept (enhancement 0006, "`~/.opm` becomes two import-free data files: `config.cue` drops `providers`/`cacheDir`, D21's local default platform is the sibling `~/.opm/platform.cue`; folded into C2" and "Platform-source flag surface (single `--platform <file>` override + automatic precedence, with source-provenance reporting); resolves OQ14"). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Auto-select provider when single provider configured

**Reason**: No providers exist to auto-select; platform-source precedence replaces the selection problem (enhancement 0006, "Platform-source flag surface (single `--platform <file>` override + automatic precedence, with source-provenance reporting); resolves OQ14").
**Migration**: `platform-resolution` precedence (`--platform` > cluster CR > local default).

### Requirement: Provider auto-resolution visible in verbose output

**Reason**: Replaced by platform-source provenance reporting on every command.
**Migration**: `platform-resolution` provenance requirement.
