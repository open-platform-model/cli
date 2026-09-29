package cuemod

import (
	"errors"
	"net"
	"strings"
)

// transportFailure is how the OCI registry client prefixes a request that
// never got an HTTP response (connection refused, DNS failure, timeout). A
// registry that answered, even with 404 or 401, does not carry it. cmd/cue
// flattens the cause into a plain string, so the text is the only signal; a
// test pins it against the embedded CUE version.
const transportFailure = "cannot do HTTP request"

// IsConnectivityError reports whether a Tidy error means the registry could
// not be reached, as opposed to a module that does not resolve.
func IsConnectivityError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return strings.Contains(err.Error(), transportFailure)
}
