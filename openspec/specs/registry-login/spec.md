## Purpose

`opm registry login [host]` — interactive credential entry into the standard OCI/docker credential file, the store CUE's resolver reads for both push and pull. Enhancement 0011 D11 (mechanism: docker config as the named default) and D24 (the `registry` command group names what is authenticated to). Verify-then-write: the credential is probed against the registry before anything lands, and the shared file is edited one `auths` entry at a time.

## Requirements

### Requirement: Login writes where the push reads

`opm registry login [host]` SHALL store a credential in the standard OCI/docker credential file — the store CUE's resolver reads for both push and pull. With a host argument, that host is targeted literally. Without one, the command SHALL resolve the configured registry mapping (flag > environment > config, reporting the winning source and any shadowed values) to its host set: exactly one host proceeds; several refuse with each host listed as a runnable `opm registry login <host>` action; none refuses pointing at configuration. The credential SHALL be verified against the registry before writing; a failed verification leaves the file untouched. The write SHALL modify only the targeted host's entry, preserving every other key content-identically (whitespace canonicalized to the writer's tab indentation), creating the file with owner-only permissions when absent.

#### Scenario: Single-host mapping logs in

- **WHEN** the resolved mapping names one host and the entered credential verifies
- **THEN** the file's entry for that host is written and the push subsequently authenticates with no further configuration

#### Scenario: Multi-host mapping refuses with actions

- **WHEN** the resolved mapping names several hosts and no argument is given
- **THEN** the command refuses, listing each host as a runnable login command

#### Scenario: Bad credential never lands

- **WHEN** the registry rejects the entered credential
- **THEN** the command refuses and the credential file is unmodified

#### Scenario: Shared file preserved

- **WHEN** the file already carries other hosts, a credsStore, or credHelpers
- **THEN** everything outside the targeted host's entry survives content-identically, whitespace canonicalized to the writer's tab indentation

#### Scenario: Non-interactive refuses

- **WHEN** the command runs without a terminal
- **THEN** it refuses and points at `docker login` for scripted use
