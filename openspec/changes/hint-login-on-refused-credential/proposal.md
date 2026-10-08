## Why

When a platform module does not build because the registry refuses the credentials, the cli
prints the hint "Pin a published build in <dir>/cue.mod/module.cue, then try again" and exits
2. That hint is the fix for a missing version. No other pin cures a refused credential: the
user must log in. The cli already reads the failure as a typed "unauthorized" answer at the
place that picks the hint, and publish and `module vet` already answer the same refusal with
`opm registry login <host>` and exit code 4.

## What Changes

- A platform module build that fails on a refused registry credential (a 401, or a 403 that
  reaches the cli as a refusal) gets the hint
  `Log in to the registry, then retry:  opm registry login <host>`, the text publish prints.
  The host is named when the configured registry mapping holds exactly one host. With several
  hosts, or none, the hint is the bare `opm registry login`, which lists the hosts itself.
- `opm platform check` exits 4 for that failure. Before, it exited 2. Its help states the
  code.
- Every other cause keeps its hint and exit code 2: a missing version, a registry that gives
  no response, an unresolved dependency, a wrong package shape, a `#registry` key mismatch and
  the default.
- Known limits, each pinned by a test: a 403 answer to the tag lookup, and a token endpoint
  that answers 403, reach the cli as "not found", so they keep the pin hint and exit 2. A
  token endpoint that answers 401 reaches the cli as "no response" and keeps the pin hint and
  exit 2 too. The reading of registry error text belongs to the library; the cli follows when
  the library release that reads these answers as refusals is pinned.

A script that reads exit 2 from `opm platform check` as "bad credentials" now sees 4. The
code moves to its documented meaning (4 is "permission denied"), as it did for publish and
`module vet`, so this is a fix, not a breaking change.

SemVer class after GA: PATCH. Beta ships it as the next `1.0.0-beta.N`.

Not in this change: the render path (`opm module build`, `opm instance apply` and the other
commands that build a platform to render against it). It prints no hint today and exits 1 for
every platform build failure; it is listed as a question for the owner.

## Capabilities

### New Capabilities

### Modified Capabilities

- `errors-domain`: the requirement "Platform module build failure hints" gains the login hint
  for a refused credential, and no longer fixes the exit code at 2 for that cause.
- `platform-check`: the requirement "A platform that cannot build fails with its build
  diagnostic" gains exit code 4 for a refused registry credential.

## Impact

- Command: `opm platform check`, the only command that builds a platform through
  `config.BuildPlatformModule`.
- Packages: `internal/config` (the hint and the cause), `internal/cmd/platform` (the exit
  code and the help), `internal/cmdutil` (the login hint text moves to `internal/config` so
  both callers print one text; publish and `module vet` print what they printed before).
- Docs: the generated `opm platform check` reference page.
- No new dependency.
