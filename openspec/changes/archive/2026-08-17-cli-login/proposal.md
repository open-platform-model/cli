# Proposal — cli-login

> Slice `cli-login` of enhancement `0011`. Decisions D11 and D24. Independent of everything else in the entry; `catalogs-publish-cutover` depends on it.

## Why

The push is performed by CUE's resolver, which reads exactly two credential stores: the standard OCI/docker credential file and CUE's `logins.json`. A credential written anywhere else is invisible at the moment it is needed — which is why D11 rejects an OPM-private store outright and fixes the command's contract as: target the registry `ResolveRegistry` produces, write where CUE reads, and publish itself never prompts.

Exploration closed the mechanism fork D11 left open, decisively. Wrapping `cue login`'s OAuth device flow fails against reality: GHCR — the only registry any OPM artifact publishes to today — answers the device-authorization endpoint with 405 (documented in CUE's own source, which tells users to run `docker login` instead), GitHub tokens fail `cue login --token`'s format regex, and `cue login` refuses multi-host mappings — which OPM's *default* registry value is. Meanwhile `logins.json` turns out to be a Bearer-token-only path in the resolver's transport. D11 already named the answer: "the default is the standard OCI credential file", with a `docker-credential-opm` helper as the recorded upgrade path. So `opm registry login` writes docker `config.json` — the store every CI publish in the workspace already populates by hand.

## What Changes

- **New `opm registry login [host]`** — a new `registry` root command group, `login` its first subcommand (D24's rename of D11's `opm login`: the command names the thing authenticated to, since a bare `opm login` reads as logging into an OPM service that does not exist). With an argument: a bare registry host (docker-login semantics). Without: resolve the configured registry mapping (`--registry` > `OPM_REGISTRY` > `config.registry` — and report shadowed sources, the first consumer of `ResolveRegistry.Shadowed`) to its host set via the public resolver; a single host proceeds; multiple hosts refuse, listing each with a runnable `opm registry login <host>` action.
- **Interactive credential entry**: username prompt on stderr, secret via no-echo terminal read (`x/term`, becoming a direct dependency); a non-interactive invocation refuses with a pointer to `docker login` (the CI path, which already works and needs nothing from this command).
- **Verify before write**: a `GET /v2/` probe against the host with the entered credential (`https`, or `http` for `+insecure` hosts); an authentication failure refuses without touching the file.
- **The write**: read-modify-write of the docker config file (`$DOCKER_CONFIG/config.json`, else `~/.docker/config.json`) setting `auths[host].auth` (base64 `user:password`) — preserving every other host, `credHelpers`, and `credsStore` byte-for-byte; created `0600` when absent; atomic same-directory rename. No public writer exists in the ecosystem; the contract is small and pinned by tests.
- **Out of scope, recorded**: OAuth device flow (revisitable behind the same command if Zot verifies the endpoint — D11's own framing), `logins.json` writing (Bearer-only path, no current registry needs it), a `--token`/`--password-stdin` flag (CI has `docker login`; YAGNI until a concrete consumer), logout.

## Capabilities

### New Capabilities

- `registry-login`: credential entry into the store CUE's push and pull both read.

### Modified Capabilities

<!-- none -->

## Impact

- **SemVer: MINOR.** New command; no existing behavior changes.
- **Commands**: new `internal/cmd/registry` package — the `registry` group (seventh root command) plus `login.go`; does not set `SkipConfigLoadAnnotation` (needs the resolved registry; config load does no Kubernetes work).
- **Packages**: new `internal/dockercfg` (the read-modify-write writer + tests); `golang.org/x/term` indirect → direct.
- **Testing**: hermetic end-to-end via the public in-process registry's `AuthConfig` (basic auth) — run login against it, then assert the file it wrote makes the pipeline's authenticated push succeed (the inverse of the landed `TestPush_Authenticated`).
- **Consumers**: `catalogs-publish-cutover`; humans publishing from laptops. The recorded upgrade path (`docker-credential-opm` via `credHelpers`) requires no change to this command's contract.
