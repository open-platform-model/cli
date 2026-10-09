package instance

import (
	"github.com/open-platform-model/cli/internal/cmdutil"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// unservedKindHint is printed when a delete failed on a recorded kind the
// cluster does not serve. It names the ways out that exist.
const unservedKindHint = "The cluster does not serve a recorded kind, so its objects cannot be checked.\n" +
	"Install the definition of that kind again, or, if only the API version was removed,\n" +
	"apply the instance again so the record takes the served version; then re-run."

// deleteFailureExitCode is the exit code of a delete with per-resource
// errors: 1, unless a failed API discovery request is among them. Then the
// cluster gave no answer at all, and the code is that of the request's
// failure (4 denied, 3 timed out or unavailable, 1 otherwise).
func deleteFailureExitCode(result *kubernetes.DeleteResult) int {
	for i := range result.Errors {
		if kubernetes.IsDiscoveryFailure(result.Errors[i].Err) {
			return cmdutil.ExitCodeFromK8sError(result.Errors[i].Err)
		}
	}
	return opmexit.ExitGeneralError
}

// hintUnservedKinds prints unservedKindHint when a per-resource error of
// result is a kind the cluster does not serve.
func hintUnservedKinds(result *kubernetes.DeleteResult) {
	for i := range result.Errors {
		if kubernetes.IsKindNotServed(result.Errors[i].Err) {
			output.Details(unservedKindHint)
			return
		}
	}
}
