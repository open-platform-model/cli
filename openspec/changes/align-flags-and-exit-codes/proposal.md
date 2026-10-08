## Why

The cli goes to v1.0.0 soon; after that a flag or an exit code is a contract. An audit of the
command surface found the same word meaning different things and the same thing spelled
differently: `--force` skips the prompt of `instance delete` but overrides a refusal
everywhere else, `instance delete` has no `--yes` while `module init` has, `module vet` names
the synthetic instance with `--instance-name` while `module build` and `module apply` use
`--name`, `opm version --output json` is an unknown flag, the exit code of a usage error is
stated in five specs and nowhere a user can read it, and the help of two command groups is
printed with stray indentation.

## What Changes

- `opm instance delete` gains `--yes`/`-y` to skip the confirmation prompt. `--force` stays
  there as a deprecated alias: it still skips the prompt, it is hidden from help, and it
  prints one deprecation line on standard error.
- `--force` has one meaning on the commands that keep it (`instance apply`, `module apply`,
  `platform pull`, `config init`): do what the command otherwise refuses in order to protect
  existing state. It never answers a prompt.
- `opm module vet` gains `--name`, as `module build` and `module apply` have.
  `--instance-name` stays there as a deprecated alias with the same effect.
- The exit code of a usage error stays 1, as the main specs state it today. `opm --help` now
  prints the exit code table, and a test pins the code for an unknown command, an unknown
  flag and a wrong argument count. `opm instance status --help` states that command's own
  codes, which the root table points at.
- `opm version` and `opm module template list` gain `-o`/`--output` (`json`, `yaml`, and the
  text or table they print today as the default).
- The help of every command is free of tab-indented lines; `opm catalog` and `opm registry`
  were the visible cases.

No old spelling is removed and no exit code moves, so nothing here is breaking.

SemVer class after GA: MINOR (new flags with defaults, two deprecations). Beta ships it as
the next `1.0.0-beta.N`.

## Capabilities

### New Capabilities

- `flag-conventions`: the meaning of `--force` and `--yes`, how a deprecated flag behaves, the
  exit code of a usage error and where it is documented, `-o`/`--output` on `opm version` and
  `opm module template list`, and the layout rule for help text.

### Modified Capabilities

- `deploy`: the requirement "Instance delete of an instance that deploys the operator is
  guarded" names `--yes` as the prompt flag; its deprecated alias bypasses the guard no more
  than `--yes` does.
- `mod-vet`: the requirement "mod vet command flags and syntax" names `--name` as the flag
  for the synthesized instance's name and `--instance-name` as its deprecated alias.

## Impact

- Commands: `opm instance delete`, `opm module vet`, `opm version`,
  `opm module template list`, `opm` (root help), and the help text of any command whose long
  description carries tab-indented lines.
- Packages: `internal/cmd`, `internal/cmd/instance`, `internal/cmd/module`,
  `internal/cmd/catalog`, `internal/cmd/registry`, `cmd/opm`. No change under
  `internal/kubernetes` or `internal/workflow`.
- The generated command reference (docs bundle `cli`) follows the cobra tree and changes
  with the next release; nothing generated is committed in this repo.
- No new dependency.
