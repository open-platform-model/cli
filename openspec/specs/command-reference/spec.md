# Capability: command-reference

## Purpose

The site's CLI reference (`/docs/reference/cli/`) is generated from the `opm` cobra command tree by `task docs:reference` and committed under `docs/site/reference/cli/` in the site page dialect; `task docs:reference:check`, run by `task check` and by a CI job in `pr.yml` and `ci.yml`, fails when the committed pages are stale.

## Requirements

### Requirement: The command reference is generated from the command tree

The repository SHALL generate the site's CLI reference from the cobra command tree that `internal/cmd.NewRootCmd` builds, with cobra's default `completion` command added, and SHALL commit the output under `docs/site/reference/cli/`. `task docs:reference` SHALL write it. The output SHALL be the section page `_index.md` and one page per visible top-level command, named `opm-<command>.md`, holding one entry for that command and one for every visible command under it, depth first in name order. A command SHALL be visible when cobra reports it available (not hidden, not deprecated, not a help topic) and it is not cobra's `help` command. Every generated fact SHALL come from a command's `Use`, `Short`, `Long`, `Example` and `Aliases` fields and its flags.

#### Scenario: A new subcommand appears in the reference

- **WHEN** a developer adds a visible subcommand under `opm module` and runs `task docs:reference`
- **THEN** `docs/site/reference/cli/opm-module.md` SHALL hold an entry headed with the subcommand's full command path
- **AND** the parent's entry SHALL list it under its subcommands with a link to that entry

#### Scenario: Hidden and deprecated commands are left out

- **WHEN** a command is hidden or deprecated
- **THEN** no page SHALL hold an entry for it

#### Scenario: A removed top-level command's page is removed

- **WHEN** a top-level command is removed and `task docs:reference` runs
- **THEN** its generated page SHALL be deleted, and a page in the directory without the generated marker SHALL be left untouched

### Requirement: Every entry has the same parts in the same order

Each entry SHALL consist of, in this order and each only when the command has it: the summary (`Short`); the usage, in a `text` code fence, the command's use line when it is runnable and `<command path> [command]` when it has visible subcommands, followed by its aliases; the description (`Long` without its examples); the flags; the examples; the subcommands. The flags SHALL be a table of name, shorthand, type, default and description listing the command's own flags and the persistent flags of its non-root parents, without hidden flags, deprecated flags and `--help`. The root's persistent flags SHALL appear once, on the section page under "Global flags", and every command page SHALL link there. A default SHALL be shown only where `--help` shows one. The examples SHALL be the lines under an `Examples:` line in `Long`, followed by cobra's `Example` field when set, in an `sh` code fence.

#### Scenario: A flag table entry

- **WHEN** `opm instance build` declares `--output` with shorthand `-o`, type `string`, default `yaml`
- **THEN** its entry's flag table SHALL hold a row with `--output`, `-o`, `string`, `yaml` and the flag's usage text

#### Scenario: Examples written inside the help text

- **WHEN** a command's `Long` ends with an `Examples:` line followed by indented commands
- **THEN** the entry's description SHALL NOT contain them and its examples SHALL hold them, dedented, in an `sh` fence

### Requirement: Generated pages follow the site page dialect

Every generated page SHALL pass the opmodel.dev source lint. Front matter SHALL hold only `title`, `description` (one line), `type` and `weight`: the section page SHALL declare title `CLI Reference`, description `Every opm command and flag, generated from the CLI's cobra commands.`, weight 2 and no `type`; a command page SHALL declare its command path as title, its `Short` as description and `type: reference`. Links SHALL be root-absolute with a trailing slash (`/docs/reference/cli/opm-module/#opm-module-apply`). Every code fence SHALL carry a language tag. Help text SHALL be escaped so that it renders as written: no raw HTML, no Markdown markup it did not intend, and Hugo shortcode delimiters written as `{{</* ... */>}}`.

#### Scenario: A placeholder in the help text

- **WHEN** a command's help text contains `--platform <dir>`
- **THEN** the page SHALL render `<dir>` as text, not as an HTML tag

### Requirement: Output is deterministic and machine-independent

Two runs over the same command tree SHALL write byte-identical pages, on any machine. The output SHALL carry no timestamp, and a flag default under the user's home directory SHALL be written with `~` in place of the home directory.

#### Scenario: Regenerating an unchanged tree

- **WHEN** `task docs:reference` runs twice with no change to the commands
- **THEN** the second run SHALL write nothing

### Requirement: Generated text sits between marker comments

Each page's generated text SHALL sit between the lines `<!-- generated by task docs:reference from the opm cobra commands; DO NOT EDIT -->` and `<!-- end generated -->`, after the front matter, which the generator also owns. Regeneration SHALL keep any text between the front matter and the begin marker, and after the end marker, as written. A page that carries the generated marker but lost its front matter or its end marker SHALL fail generation naming the file, and nothing SHALL be written to it.

#### Scenario: An authored note after the generated block

- **WHEN** an author adds a "See also" section after the end marker of `opm-module.md` and the reference is regenerated
- **THEN** the section SHALL be unchanged in the regenerated page

### Requirement: A stale reference fails the check locally and in CI

`task docs:reference:check` SHALL write nothing and SHALL exit non-zero, naming each file, when any generated page is missing, differs from what the command tree generates, or is no longer generated. Its message SHALL name `task docs:reference` as the fix. `task check` SHALL run it. The workflows `.github/workflows/pr.yml` (every pull request to `main`) and `.github/workflows/ci.yml` (every push) SHALL each run it in a job named `Command Reference (current)` with no `needs` dependency on another job.

#### Scenario: Help text changed without regenerating

- **WHEN** a pull request changes a command's `Short` and does not regenerate the reference
- **THEN** the `Command Reference (current)` job SHALL fail, naming the stale page

#### Scenario: Regenerated in the same change

- **WHEN** the same pull request also commits the output of `task docs:reference`
- **THEN** the job SHALL pass
