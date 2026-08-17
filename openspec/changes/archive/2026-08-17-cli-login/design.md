# Design — cli-login

## Overview

One small command over one careful file writer. The design's center of gravity is *not writing more than it must*: the docker config file is shared property (docker, podman, other tools, `credsStore`/`credHelpers` indirections), and `opm registry login` edits exactly one `auths` entry.

## Research & Decisions

### Name: `opm registry login`, not top-level `opm login`

**Context**: D11 drafted the command as top-level `opm login`. On review of this slice's artifacts the name read as ambiguous — "do I log in to OPM or to something else?" — because `opm` is a platform tool with no hosted OPM service behind it, unlike docker/podman/oras/cue where the registry is the only possible referent of "login".
**Decision**: `opm registry login [host]` — a new `registry` root command group, `login` its first subcommand. Recorded as 0011 D24 (amends D11); flags, defaults, and the whole contract are unchanged. Helm is the precedent: a package tool pushing to OCI registries it does not operate, answering the same ambiguity with `helm registry login`. The capability name (`registry-login`) and the existing `opm catalog registry check` subgroup already use the noun.
**Rationale**: the command names what you authenticate to, and the group gives logout and the `docker-credential-opm` helper flow a home when they arrive.

### Store: docker config, not `logins.json`, not device flow

**Context**: D11 names both stores CUE reads and calls the OCI/docker file "the default"; the device-flow wrap was "not chosen now" pending Zot verification.
**Explored**: GHCR 405s the device endpoint (documented in CUE's own `login.go`, which redirects users to `docker login`); GitHub tokens fail `cue login --token`'s `appv1_`-style regex; `cue login` hard-refuses multi-host mappings — OPM's default value is multi-host; the resolver's transport consults `logins.json` first but only as a Bearer source — basic/htpasswd registries are served by the docker path alone; every workspace CI publish does `docker login ghcr.io`.
**Decision**: write `auths[host].auth` in the docker config. `logins.json` and the device flow are recorded non-choices, revisitable behind the unchanged command contract (D11's own upgrade-path framing).
**Rationale**: it is the one store that serves every registry OPM actually pushes to, with the mechanism every existing credential in the fleet already uses.

### Host resolution: bare host argument; mapping resolved only for the no-arg case

**Context**: D11 fixes the no-arg target as "the registry `ResolveRegistry` produces" — a CUE_REGISTRY *mapping*, not a host; nothing in 0011 bridges that gap. `cue login`'s answer to multi-host is refusal.
**Decision**: `opm registry login ghcr.io` takes the host literally (docker-login semantics — no mapping parse). `opm registry login` with no argument builds the public resolver over the resolved mapping and takes `AllHosts()`: one host → proceed; several → refuse, listing each host with `opm registry login <host>` as the action (the house refusal shape); zero (no registry configured) → refuse pointing at `opm config init`. The resolution report names the source and any shadowed values — `ResolveRegistry.Shadowed`'s first consumer, closing D11's "an override is reportable rather than silent" claim.
**Rationale**: no magic: picking the first-party prefix's host silently would authenticate users against a host they didn't name. The refusal is one copy-paste away from correct, which is the same standard the publish refusals hold.

### Verify-then-write

**Decision**: before writing, probe `GET /v2/` with the entered basic credential — `https://` normally, `http://` when the resolved mapping marks the host `+insecure` (or the bare-host form is given the literal `+insecure` suffix). 401/403 → refusal, file untouched; other transport failure → connectivity error (exit 3); 200/404-with-auth-accepted → write. Success message names the file written and the host. Implemented through CUE's own `ociauth` transport rather than a raw basic GET — GHCR answers `/v2/` with a bearer challenge regardless of credential validity, so only the challenge-following transport verifies for real; basic-auth registries behave identically either way.
**Rationale**: a login that stores a bad credential moves the failure to the next publish, blamed on the wrong command. The probe is one request and mirrors what the transport will do anyway.

### The writer (`internal/dockercfg`)

**Decision**: `Upsert(path, host, user, secret)` — read the existing JSON (absent → start empty), decode into a raw `map[string]json.RawMessage` at the top level so **unknown keys pass through byte-identical** (`credsStore`, `credHelpers`, `plugins`, anything future), replace only `auths.<host>` with `{"auth": base64(user+":"+secret)}`, re-encode with tab indentation, write `0600` via same-dir temp + rename. No file locking (docker itself does none; the atomic rename bounds the damage).
**Rationale**: the raw-message envelope is what makes the writer safe to run against a file docker owns — a typed struct would silently drop fields added by other tools.

### Input UX

**Decision**: username prompt via `output.Prompt` (stderr), secret via `term.ReadPassword` on the TTY (no echo, newline after). Non-TTY stdin → refuse with the `docker login <host>` pointer (CI never needs this command). No flags for credentials in this slice.
**Rationale**: matches the repo's one interactive precedent (refuse-on-non-TTY) and D11's "publish itself never prompts" division: prompting is *this* command's whole job, so it is unapologetically interactive.

## Technical Notes

- Command: new `internal/cmd/registry` package — `registry.go` (the group) plus `login.go` — with the group registered in `root.go`; no `SkipConfigLoadAnnotation` (registry resolution requires config load; config load performs no Kubernetes I/O).
- Path resolution: `$DOCKER_CONFIG/config.json` else `~/.docker/config.json` — mirroring the read side's order so the write lands where the push will look. (`$DOCKER_AUTH_CONFIG` inline JSON and the podman path are read-side extras; login does not write them and says so if `$DOCKER_AUTH_CONFIG` is set, since it would shadow the written entry.)
- Exit codes: 0 written; 2 refusal (multi-host, no registry, bad credential, non-TTY); 3 connectivity.
- Tests: `dockercfg` table tests (fresh file 0600; existing file with `credsStore`+foreign hosts preserved byte-for-byte outside the edited entry; invalid JSON refused); command tests with `t.Setenv(DOCKER_CONFIG)`; the end-to-end inversion — login against `modregistrytest.NewServer` basic auth, then the pipeline's push succeeds using only the written file; multi-host refusal against the shipped default mapping.
- Recorded for later: device-flow upgrade blocked on Zot endpoint verification; `docker-credential-opm` helper as the `credHelpers` path; logout.
