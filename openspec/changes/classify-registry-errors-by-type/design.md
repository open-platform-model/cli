## Context

See proposal.md for why. The library half is in v1.0.0-beta.6: `opm/errors.Classify` reads the typed chain first (`context.DeadlineExceeded`, `ociregistry.HTTPError` status, `modregistry.ErrNotFound`, the ociregistry not-found and unauthorized codes, `net.Error`), then a text fallback for the forms `cue/load` and `cmd/cue` flatten. It returns a `*FetchError{Kind, Coordinate, Status, Err}` or the error unchanged, and it passes `context.Canceled` through unclassified. The library's own measurements (archived change `type-fetch-errors-and-check-cancellation`, "Spike findings") give the forms against CUE v0.17.1, which the cli also embeds:

| Failure | typed fetch (`reg.Fetch`, `FetchArtifact`) | flattened (`cue/load`, `cmd/cue`) | Kind |
| --- | --- | --- | --- |
| version absent (404 tag) | `modregistry.ErrNotFound` | `cannot fetch P@V: module P@V: module not found` | `FetchNotFound`, Status 0 |
| 403 on the tag | `modregistry.ErrNotFound` | `module not found` | `FetchNotFound`, Status 0 |
| standalone `P@vX.Y.Z` load: version or package absent | | `cannot find module providing package P@vX.Y.Z` | `FetchNotFound` |
| undeclared import, own-path package missing | | `cannot find module providing package P` (no exact version) | unclassified |
| refused connection, DNS, TLS | `net.Error` | `cannot do HTTP request: ...` | `FetchUnreachable` |
| 401 | `HTTPError` 401 | `401 Unauthorized: ...` | `FetchUnauthorized`, 401 |
| 429 | `HTTPError` 429 | `429 Too Many Requests: ...` | `FetchOther`, 429 |
| 500, 503 | `HTTPError` | `503 Service Unavailable: ...` | `FetchOther`, 5xx (transient) |
| expired deadline | `net.Error` + `DeadlineExceeded` | | `FetchUnreachable` |
| cancelled context | `*url.Error{context.Canceled}` (a `net.Error`) | | unclassified |

The cli's exit codes for these sites: a `*publish.ConnectivityError` exits 3 everywhere (`initError`, `checkError`, `publishError`); `ErrNotPublished` exits 5 in `opm catalog registry check`; anything else from `opm instance init` exits 1.

## Goals / Non-Goals

**Goals:**

- The three probes and the platform hint's registry branch decide by the library's kinds.
- Every exit code stays what it is today for every row above. Tests pin each row before the swap and must pass unchanged after it.

**Non-Goals:**

- Using `ErrTransient` at any cli site. It is not the exact equivalent of any probe (see D1).
- New exit codes or a retry anywhere. Making `AcquireModuleFromRegistry` failures exit 3 in the render-bearing commands (`internal/workflow/render/module.go:174-180`) would change exit codes and is left out.
- cli#310 (record-read exit codes).

## Decisions

### D1. `IsConnectivityError` is `FetchUnreachable`, plus the cancelled request

```go
// IsConnectivityError reports whether err means the registry could not be
// reached: no HTTP response at all. A registry that answered, with any
// status, is not a connectivity failure.
func IsConnectivityError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		var netErr net.Error
		return errors.As(err, &netErr)
	}
	var fe *liberrors.FetchError
	return errors.As(liberrors.Classify(err), &fe) && fe.Kind == liberrors.FetchUnreachable
}
```

**Why not `errors.Is(Classify(err), ErrTransient)`**: `ErrTransient` also holds for a 5xx answer. Today a 5xx is not connectivity (no `net.Error`, no `cannot do HTTP request`), so `instance init` exits 1 on it. `ErrTransient` would move that to 3. `Kind == FetchUnreachable` matches today's answer on every row of the table: `net.Error` and `cannot do HTTP request` are exactly the inputs that `Classify` turns into `FetchUnreachable` (a `DeadlineExceeded` is a `net.Error` too).

**The cancellation clause.** `Classify` leaves `context.Canceled` unclassified, but today a cancelled request (`*url.Error{Err: context.Canceled}`) is a `net.Error` and counts as connectivity. The clause keeps that answer for that form, and keeps a plain `context.Canceled` (no `net.Error`) as not connectivity, as today. No cli command cancels the context on these paths today (no signal-bound context reaches `instance init`), so the clause is about parity, not a user-visible case. Its pin rows are unit rows. The clause is a typed check, so it is not a text probe.

Alternatives: drop the clause and record the cancellation change (rejected: the rule is that no exit code moves, and the cost is three lines); `ErrTransient` (rejected above).

### D2. `fetchPublishedTree`: `FetchNotFound` from the version lookup is `ErrNotPublished`

```go
loc, err := reg.Fetch(ctx, mv)
if err != nil {
	if cuemod.IsVersionNotHeld(err) {
		return "", fmt.Errorf("%s@%s: %w", repo, version, ErrNotPublished)
	}
	return "", &ConnectivityError{Op: fmt.Sprintf("fetching %s", mv), Err: err}
}
```

`cuemod.IsVersionNotHeld(err)` is `FetchNotFound` with `Status == 0`. The tag lookup's 404 and 403 reach the cli as `modregistry.ErrNotFound`, which CUE builds without the HTTP status, so they carry `Status` 0. The one `FetchNotFound` that carries a status is a 404 on a blob: a registry that holds the tag but not the module archive. Today its text has no lowercase `not found`, so it is a `*ConnectivityError` (exit 3), and the `Status` test keeps it there. This is an assumption to verify, not a measured fact: section 2 pins the blob row through a registry that drops the archive blob, and section 3 adjusts `IsVersionNotHeld` to whatever typed signal tells that row apart (the `Status`, or `ociregistry.ErrBlobUnknown` in the chain) if the pin shows otherwise. If no typed signal separates it, the implementer stops and reports, and does not fall back to text.

The same helper serves D3.

### D3. `loadPublishedPackage`: absent only when the probed version or package is missing

`compat.go` loads `repo/pkgPath@version` standalone. Today the text `cannot find module providing package` means absent (`found=false`), and every other failure is a `*ConnectivityError`. Under the kinds, both "absent" and "a dependency of the probed build is missing" are `FetchNotFound`, and the flattened `cue/load` error carries no coordinate (`Coordinate` is empty for a resolution failure). The library's archived change `type-fetch-errors-and-check-cancellation` (design.md, D2) leaves this mapping to the cli.

Decision: on a load error whose kind is `FetchNotFound`, ask by type which one it is.

```go
if err := insts[0].Err; err != nil {
	if cuemod.IsFetchNotFound(err) {
		absent, aerr := probedPackageAbsent(opts, repo, pkgPath, version)
		if aerr != nil {
			return cue.Value{}, false, aerr // *ConnectivityError
		}
		if absent {
			return cue.Value{}, false, nil
		}
	}
	return cue.Value{}, false, &ConnectivityError{Op: "loading " + pattern, Err: err}
}
```

`probedPackageAbsent` calls `fetchPublishedTree` for `repo@version`. It goes through the same on-disk module cache the load just used, so a present build costs no second download. `ErrNotPublished` means the version is absent: absent. A `*ConnectivityError` is returned as is. A fetched tree with no `.cue` file directly in `<tree>/<pkgPath>` means the package is absent at that version: absent. Otherwise the probed package is present, so the not-found concerned a dependency: a `*ConnectivityError`, as today.

`loadPublishedPackage` takes `repo` and `pkgPath` separately instead of the joined import path. The walk has no context today (`load.Instances` takes none), so the extra fetch uses `context.Background()` for the same reason.

Alternatives considered:
1. Accept the edge and let a missing dependency read as "absent" with a pinning test (the first option in that library design). Rejected: it moves an exit code from 3 to a silent negative signal, so a broken predecessor would let a compat refusal pass unseen.
2. Fetch first on every probe, before the load. Rejected: the same answer, but an extra registry round trip on the hot path, where the failure path is enough.
3. Keep the text match for this one site. Rejected: it is one of the three probes the owner decision removes.

### D4. Platform build hint: the registry branch reads the kind

`internal/config/platform.go:97-110` `platformBuildHint` picks a hint, not an exit code. The "pin a published build" case matches `module not found`, `cannot find package` and `cannot expand module graph`. Only the first is a registry answer, and `FetchNotFound` covers it. The other two are author defects that the library leaves unclassified by design (an undeclared import, a malformed dependency module file), so they stay text matches and say why in a comment, and so does the `#registry` branch. The case becomes `cuemod.IsFetchNotFound(err) || strings.Contains(msg, "cannot find package") || strings.Contains(msg, "cannot expand module graph")`. Section 2 pins the hint for each form the kernel produces on a platform directory (unpublished pin, undeclared import, `#registry` key mismatch, wrong kind, unreachable registry).

The three readers live together in `internal/cuemod/connectivity.go` beside `IsConnectivityError`: `IsFetchNotFound(err)` (kind `FetchNotFound`, any status) and `IsVersionNotHeld(err)` (D2). `internal/cuemod` imports no other cli package, and `internal/publish` already imports it, so `internal/config` and `internal/publish` both read them with no import cycle (checked with `go list -deps`).

### D5. Library bump and the objectset swap ride together, first

`go get github.com/open-platform-model/library@v1.0.0-beta.6` moves only that line (`cuelang.org/go` stays v0.17.1; checked with a scratch modfile, `go build ./...` passes). `opm/k8s/object` has the same `Duplicates` and `DuplicateIdentitiesError` as the deprecated `opm/helper/objectset`. The cli builds the error itself (`render.go:318-319`) and matches it with `errors.As` (`validation.go:60`), so both move in one commit and the refusal text does not change. Keeping objectset would leave two SA1019 findings in `task lint`, so the swap cannot wait for the later tier-adoption change.

beta.6 moves the kernel's default core to `opmodel.dev/core@v2.0.0-beta.4`. The cli's templates, `hack/platform` and fixtures pin core v2.0.0-beta.1, which the cascade moves separately. Section 1 runs the whole unit suite on the bump alone; a failure there is fixed in section 1 (or reported) before any probe changes.

### Test harness

The pin tests need registries that answer with a chosen status. `internal/cuemod/cuemodtest` gains `StatusRegistry(t, status) string` (an `httptest.Server` that answers every request with `status`) and a selective form that fronts an `ociserver` over `ocimem` and answers a chosen repository, or blob requests, with a chosen status. The absent version, the absent package and the missing dependency use a plain in-memory registry with the right content pushed (`rawPush` in `internal/publish/check_test.go` already does this). The unreachable registry is `cuemodtest.UnreachableRegistry`.

## Risks / Trade-offs

- [The blob-404 row is not measured yet] → D2: section 2 pins it before section 3 relies on it; no typed signal means stop and report, never text.
- [`Classify`'s text fallback reads CUE's text, so a CUE bump can still move a classification] → the library pins each form against the embedded CUE (`TestCUEFailureForms`) and fails on a change. The cli's pin tests run through real registries too, so a moved form also fails here, before a release.
- [The disambiguating fetch in D3 adds a registry call on the not-found path] → it only runs on a failure, and the module cache makes it free for a present build.
- [beta.6 brings unrelated library changes into the same PR] → section 1 isolates them; the PR body names it so the reviewer reads section 1's diff on its own.
- [A 5xx during `instance init`'s acquire exits 1 though the library calls it transient] → kept on purpose (no exit code moves). Changing it to 3 is a separate, visible decision for the owner.

## Migration Plan

None. No exit code or flag changes; one help line gains the exit code 5 the command already returns. Rollback is a revert of the PR; the library bump can stay.
