# Tasks — cli-login

> Small slice, four groups. The writer lands first with its safety tests; the command composes it.

## 1. internal/dockercfg

- [x] 1.1 `Upsert(path, host, user, secret)` per design: raw-message envelope, single-entry edit, tab-indented, 0600, atomic rename.
- [x] 1.2 Table tests: fresh file; existing file with `credsStore`/`credHelpers`/foreign hosts byte-identical outside the edit; malformed JSON refused; permissions asserted.

## 2. Host resolution + probe

- [x] 2.1 No-arg path: resolver over the resolved mapping → `AllHosts()`; one/many/zero handling per design; `Shadowed` reporting in the resolution line.
- [x] 2.2 Bare-host argument path incl. `+insecure` suffix; scheme selection.
- [x] 2.3 `GET /v2/` probe with basic auth; 401/403 refusal (file untouched), transport failure exit 3.

## 3. Command

- [x] 3.1 New `internal/cmd/registry` package (`registry.go` group + `login.go`): prompts (stderr username, no-echo secret via `x/term` — dependency becomes direct), non-TTY refusal pointing at `docker login`; `$DOCKER_AUTH_CONFIG`-set notice; success message naming file + host. Root registration of the `registry` group.
- [x] 3.2 Constructor + refusal-shape tests; multi-host refusal asserted against the shipped `DefaultRegistry` value.

## 4. End-to-end, specs, record

- [x] 4.1 Hermetic inversion test: login against the in-process registry's basic auth, then the pipeline's authenticated push succeeds using only the written file.
- [x] 4.2 New spec `registry-login`; `CLAUDE.md` command map.
- [x] 4.3 `task fmt vet lint test` green.
- [x] 4.4 Record in `enhancements/0011/`: slice → done + `openspec_ref` + history event (mechanism decision: docker config as D11's named default; device flow / logins.json / credential flags recorded as non-choices with their evidence).
