## MODIFIED Requirements

### Requirement: `opm instance build` branches on argument type

The `opm instance build` subcommand SHALL stat its positional argument and choose between the instance-file rendering path and the module-synthesis rendering path based on whether the path resolves to a regular file or a directory.

#### Scenario: Argument is an instance file

- **WHEN** the user runs `opm instance build ./jellyfin_instance.cue` and the path resolves to a regular file
- **THEN** the subcommand SHALL acquire the file's package through the library kernel (`AcquireInstanceFromDir` on the file's directory, with any `-f`/`--values` files layered as values sources) and render it

#### Scenario: Argument is a module directory

- **WHEN** the user runs `opm instance build ./my-module` and the path resolves to a directory
- **THEN** the subcommand SHALL invoke the module-synthesis pipeline, using `-f`/`--values` (or the module's `debugValues`) for values and `--name`/`--namespace` (or defaults) for synthetic metadata

#### Scenario: Argument does not exist

- **WHEN** the positional argument cannot be `os.Stat`'ed
- **THEN** the subcommand SHALL return a clear error naming the missing path
