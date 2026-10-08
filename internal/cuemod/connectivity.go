package cuemod

import (
	"context"
	"errors"
	"net"

	liberrors "github.com/open-platform-model/library/opm/errors"
)

// The readers below decide a registry failure from the library's typed
// classification (liberrors.Classify), never from the error's text. Classify
// reads the typed chain first and owns the one text fallback for the forms
// cue/load and cmd/cue flatten into a string; the library pins those forms
// against the embedded CUE. Source: 0021:D8:R12.

// IsConnectivityError reports whether err means the registry could not be
// reached: no HTTP response at all (a refused connection, a DNS or TLS
// failure, a timeout or an expired deadline). A registry that answered, with
// any status, is not a connectivity failure.
//
// Classify leaves a cancellation unclassified, since the caller's own
// cancellation is not a fetch failure. A canceled HTTP request
// (*url.Error wrapping context.Canceled) is a net.Error and has always
// counted as connectivity here, so the cancellation clause keeps that
// answer, and a bare context.Canceled stays not connectivity.
func IsConnectivityError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		var netErr net.Error
		return errors.As(err, &netErr)
	}
	return fetchKindIs(err, liberrors.FetchUnreachable)
}

// IsUnauthorized reports whether err means the registry answered and refused
// the caller: a 401 (the credentials are missing or wrong) or a 403 (the
// credentials may not do this) that reaches the library as such. CUE's
// registry client reports a 403 answer to a tag lookup as not found, so that
// case is IsFetchNotFound, not this.
func IsUnauthorized(err error) bool {
	return fetchKindIs(err, liberrors.FetchUnauthorized)
}

// IsFetchNotFound reports whether err is a registry answer that it does not
// hold what was asked for: a module version (including a 403 tag lookup,
// which CUE reports as not found), a package a path@version load names, or a
// blob behind a held tag.
func IsFetchNotFound(err error) bool {
	return fetchKindIs(err, liberrors.FetchNotFound)
}

// IsVersionNotHeld reports whether err from a module version fetch means the
// registry does not hold that version: FetchNotFound with no HTTP status.
// CUE's registry client reports the tag lookup's 404 and 403 as
// modregistry.ErrNotFound, without the status; a 404 on the module archive
// blob behind a held tag carries status 404 and is not "not held".
func IsVersionNotHeld(err error) bool {
	var fe *liberrors.FetchError
	return errors.As(liberrors.Classify(err), &fe) && fe.Kind == liberrors.FetchNotFound && fe.Status == 0
}

// fetchKindIs reports whether the library classifies err as a fetch failure
// of kind.
func fetchKindIs(err error, kind liberrors.FetchKind) bool {
	var fe *liberrors.FetchError
	return errors.As(liberrors.Classify(err), &fe) && fe.Kind == kind
}
