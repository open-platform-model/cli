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

`cuemod.IsVersionNotHeld(err)` is `FetchNotFound` with `Status == 0`. The tag lookup's 404 and 403 reach the cli as `modregistry.ErrNotFound`, which CUE builds without the HTTP status, so they carry `Status` 0. The one `FetchNotFound` that carries a status is a 404 on a blob: a registry that holds the tag but not the module archive. Today its text has no lowercase `not found`, so it is a `*ConnectivityError` (exit 3), and the `Status` test keeps it there. The spike measured it: the fetch fails with `404 Not Found: blob unknown`, and today's answer is a `*ConnectivityError` (exit 3), pinned. The `Status` test keeps it there; if section 3 shows the status missing from the chain, `IsVersionNotHeld` reads `ociregistry.ErrBlobUnknown` instead, never text.

The same helper serves D3.

### D3. `loadPublishedPackage`: absent only when the probed version or package is missing

`compat.go` loads `repo/pkgPath@version` standalone. Today the text `cannot find module providing package` means absent (`found=false`), and every other failure is a `*ConnectivityError`. Under the kinds, both "absent" and "a dependency of the probed build is missing" are `FetchNotFound`, and the flattened `cue/load` error carries no coordinate (`Coordinate` is empty for a resolution failure). The library's archived change `type-fetch-errors-and-check-cancellation` (design.md, D2) leaves this mapping to the cli.

Decision: on a load error whose kind is `FetchNotFound`, ask by type which one it is.

```go
if err := insts[0].Err; err != nil {
	switch {
	case cuemod.IsFetchNotFound(err):
		absent, aerr := probedPackageAbsent(opts, repo, pkgPath, version)
		if aerr != nil {
			return cue.Value{}, false, aerr // *ConnectivityError
		}
		if absent {
			return cue.Value{}, false, nil
		}
	case unprovidedImport(err):
		return cue.Value{}, false, nil // see "The unversioned form" below
	}
	return cue.Value{}, false, &ConnectivityError{Op: "loading " + pattern, Err: err}
}
```

`probedPackageAbsent` calls `fetchPublishedTree` for `repo@version`. It goes through the same on-disk module cache the load just used, so a present build costs no second download. `ErrNotPublished` means the version is absent: absent. A `*ConnectivityError` is returned as is. Any other error from the disambiguating fetch (`fetchPublishedTree` also fails without asking the registry: the registry mapping does not build, the version does not form, the fetched source has no filesystem root) is not returned: the original load failure is, as `&ConnectivityError{Op: "loading "+pattern, Err: err}`, which is today's answer for that load (exit 3). A fetched tree with no `.cue` file directly in `<tree>/<pkgPath>` means the package is absent at that version: absent. Otherwise the probed package is present, so the not-found concerned a dependency: a `*ConnectivityError`, as today.

`loadPublishedPackage` takes `repo` and `pkgPath` separately instead of the joined import path. The walk has no context today (`load.Instances` takes none), so the extra fetch uses `context.Background()` for the same reason.

Alternatives considered:
1. Accept the edge and let a missing dependency read as "absent" with a pinning test (the first option in that library design). Rejected: it moves an exit code from 3 to a silent negative signal, so a broken predecessor would let a compat refusal pass unseen.
2. Fetch first on every probe, before the load. Rejected: the same answer, but an extra registry round trip on the hot path, where the failure path is enough.
3. Keep the text match for this one site. Rejected: it is one of the three probes the owner decision removes.

**The unversioned `cannot find module providing package P` form.** The library classifies only the form whose `P` carries an exact version (a standalone `path@vX.Y.Z` load); the unversioned form, which reports an import that no module of the build provides (an import its fetched dependency does not provide, or a missing package under its own module path), is an author defect and stays unclassified (library spec `fetch-error-classification`). The spike measured both variants: today they read as absent (`found=false`). Under the rule above they would fall through to a `*ConnectivityError` (exit 3), an exit-code change.

No typed signal tells them apart from other unclassified load failures (a parse error, an import cycle, a dependency whose module file does not parse), which exit 3 today: CUE reports the missing import as `modpkgload.ImportMissingError`, an internal type, and `cue/load` flattens it into a string. So the cli keeps today's answer with one local recognition, applied only after the library's classification found nothing:

```go
// unprovidedImport reports an import that no module of the loaded build
// provides: an author defect the library leaves unclassified on purpose,
// because no registry interaction failed.
func unprovidedImport(err error) bool {
	var fe *liberrors.FetchError
	return !errors.As(liberrors.Classify(err), &fe) &&
		strings.Contains(err.Error(), "cannot find module providing package")
}
```

This is not a registry-failure probe: every registry answer is decided by the library's kinds first, and the match runs only on what the library calls an author defect, which is the rule D4 already applies to the platform hint (`cannot find package`). The local mapping for the compat walk belongs to this change (the library leaves the cli its exit-code mapping, including this edge). Whether a predecessor that does not load because of a broken import should abort the walk (exit 3) instead of reading as absent is an owner question; until it is answered, the pin holds today's answer, and the answer is one clause to delete.

### D4. Platform build hint: the registry branch reads the kind

`internal/config/platform.go:97-110` `platformBuildHint` picks a hint, not an exit code. The "pin a published build" case matches `module not found`, `cannot find package` and `cannot expand module graph`. Only the first is a registry answer, and `FetchNotFound` covers it. The other two are author defects that the library leaves unclassified by design (an undeclared import, a malformed dependency module file), so they stay text matches and say why in a comment, and so does the `#registry` branch. The case becomes `cuemod.IsFetchNotFound(err) || strings.Contains(msg, "cannot find package") || strings.Contains(msg, "cannot expand module graph")`. Section 1 pins the hint for each form the kernel produces on a platform directory (unpublished pin, a pinned catalog whose archive blob answers 404, undeclared import, `#registry` key mismatch, wrong kind, unreachable registry). `IsFetchNotFound` is broader than the text `module not found` (it also covers a `404 Not Found` status and the versioned `cannot find module providing package`) and narrower where `cannot do HTTP request` appears in the same message (unreachable wins), so the hint can change for those forms; a hint is not an exit code, and the pin records what the 404 form gets.

The three readers live together in `internal/cuemod/connectivity.go` beside `IsConnectivityError`: `IsFetchNotFound(err)` (kind `FetchNotFound`, any status) and `IsVersionNotHeld(err)` (D2). `internal/cuemod` imports no other cli package, and `internal/publish` already imports it, so `internal/config` and `internal/publish` both read them with no import cycle (checked with `go list -deps`).

### D5. Library bump and the objectset swap ride together, after the pins

`go get github.com/open-platform-model/library@v1.0.0-beta.6` moves only that line (`cuelang.org/go` stays v0.17.1; checked with a scratch modfile, `go build ./...` passes). `opm/k8s/object` has the same `Duplicates`, `Duplicate`, `Identity`, `Producer` and `DuplicateIdentitiesError` as the deprecated `opm/helper/objectset`; the two render files and their two test files move. The cli builds the error itself (`render.go:318-319`) and matches it with `errors.As` (`validation.go:60`), so both move in one commit and the refusal text does not change. Keeping objectset would leave two SA1019 findings in `task lint`, so the swap cannot wait for the later tier-adoption change.

beta.6 moves the kernel's default core to `opmodel.dev/core@v2.0.0-beta.4`. The cli's templates, `hack/platform` and fixtures pin core v2.0.0-beta.1, which the cascade moves separately. Section 2 runs the whole unit suite on the bump alone, with the section 1 pins unchanged; a failure there is fixed in section 2 (or reported) before any probe changes.

### Test harness

The pin tests need registries that answer with a chosen status. `internal/cuemod/cuemodtest` gains `StatusRegistry(t, status) string` (an `httptest.Server` that answers every request with `status`) and a selective form that fronts an `ociserver` over `ocimem` and answers a chosen repository, or blob requests, with a chosen status. The absent version, the absent package and the missing dependency use a plain in-memory registry with the right content pushed (`rawPush` in `internal/publish/check_test.go` already does this). The unreachable registry is `cuemodtest.UnreachableRegistry`.

## Risks / Trade-offs

- [The unversioned `cannot find module providing package` form keeps one local text recognition] → D3: it runs only on what the library leaves unclassified, the pin fails if CUE rewords it, and whether the form should abort as exit 3 is an open owner question.
- [`Classify`'s text fallback reads CUE's text, so a CUE bump can still move a classification] → the library pins each form against the embedded CUE (`TestCUEFailureForms`) and fails on a change. The cli's pin tests run through real registries too, so a moved form also fails here, before a release.
- [The disambiguating fetch in D3 adds a registry call on the not-found path] → it only runs on a failure, and the module cache makes it free for a present build.
- [beta.6 brings unrelated library changes into the same PR] → section 2 isolates them, after section 1 pinned the answers on beta.4; the PR body names it so the reviewer reads section 2's diff on its own.
- [A 5xx during `instance init`'s acquire exits 1 though the library calls it transient] → kept on purpose (no exit code moves). Changing it to 3 is a separate, visible decision for the owner.

## Spike findings

Measured on library v1.0.0-beta.4 and CUE v0.17.1 through real in-memory registries (`cuemodtest.StatusRegistry`, `cuemodtest.Fronted`), before any probe changed. Each row is pinned by a section 1 test; every answer is today's.

| Site | Failure | Today's answer (exit) |
| --- | --- | --- |
| `IsConnectivityError` (init) | refused connection, through `Tidy` and the kernel acquire | true (3) |
| | module or dependency not held, 403, 401, 429, 503, through `Tidy` and the kernel acquire | false (1) |
| | a fetch with an expired deadline; a bare `context.DeadlineExceeded` (it is itself a `net.Error`) | true |
| | `*url.Error{context.Canceled}` | true; a bare `context.Canceled` false |
| `fetchPublishedTree` (check) | version not held; 403 on every request | `ErrNotPublished` (5) |
| | 401, 429, 503, refused, tag held but archive blob 404 (`404 Not Found: blob unknown`) | `*ConnectivityError` (3) |
| | registry mapping does not parse | plain error (1) |
| `loadPublishedPackage` (compat) | version not held; package absent at a held version; 403 on the probed repository (a versioned `cannot find module providing package P@vX.Y.Z`) | absent |
| | an import the fetched dependency does not provide; an import of a missing own-path package (unversioned `cannot find module providing package P`) | absent |
| | dependency not held, dependency 403, dependency archive blob 404 | `*ConnectivityError` (3) |
| | probed archive blob 404; 401, 429, 503 on the probed repository; refused; registry mapping does not parse | `*ConnectivityError` (3) |
| `platformBuildHint` | unpublished pin, pinned build's archive blob 404, undeclared import, refused registry (all carry `cannot find package`) | "Pin a published build" |
| | not a `#Platform` | "single package embedding core.#Platform" |

A 503 answer without a body reads `503 Service Unavailable: malformed error response`; the status is still in the chain. The 401/429/5xx answers of `opm instance init` stay test pins and are not stated in the spec; whether a 5xx should exit 3 there is an owner question.

## Migration Plan

None. No exit code, flag or help text changes. Rollback is a revert of the PR; the library bump can stay.
