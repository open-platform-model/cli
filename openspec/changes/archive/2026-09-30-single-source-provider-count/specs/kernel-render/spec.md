## ADDED Requirements

### Requirement: A platform pinning an older core is refused as a re-pin

When the kernel refuses a render because the platform module pins a core release older than the first one deriving a field the kernel reads (the library's core-floor refusal, raised before the render is staged), every render-bearing command SHALL fail as a validation failure printing the library's message, which names the platform, the missing field and the core release required. When the platform is a directory the user named with `--platform`, the command SHALL add a hint naming that directory and the exact `cue mod get opmodel.dev/core@<version>` command to run in it, where `<version>` is the core release the kernel was verified against (which satisfies every such floor, not only the one that fired). The CLI SHALL NOT fall back to another platform source, retry against a generated platform, or render with a count of its own. A platform the CLI generates (from the cluster Platform or from the render's own dependency pins) pins the kernel's verified core, so it is never refused this way.

#### Scenario: An older-core --platform directory is refused with the re-pin command

- **WHEN** `opm instance build` runs with `--platform <dir>` whose `cue.mod/module.cue` pins `opmodel.dev/core@v2` at `v2.0.0-alpha.11`
- **THEN** the command exits as a validation failure, prints `render failed` with the library's message naming `providedBy` and `2.0.0-alpha.12`, prints a hint naming `<dir>` and `cue mod get opmodel.dev/core@v2.0.0-alpha.12`, and prints no manifest

#### Scenario: A generated platform is never refused for its core

- **WHEN** a render-bearing command runs with no `--platform` and the platform is generated from the cluster Platform or from the render's dependency pins
- **THEN** the generated platform pins the kernel's verified core release and the render is not refused for an older core
