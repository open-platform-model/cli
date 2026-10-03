package apply

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// waitForHealthy blocks until every resource the apply rendered reads healthy
// on the cluster, or deadline passes: the end of the command's --timeout
// budget, which the apply's CustomResourceDefinition wait has already drawn
// on. It uses the shared poll loop in internal/kubernetes, so a resource that
// disappears after being applied fails the wait at once and a timeout names
// every resource still pending.
// The resources are already applied and recorded when it runs; a failure here
// leaves them in place, and the error says how to inspect them.
func waitForHealthy(ctx context.Context, req Request, deadline time.Time, instanceLog *log.Logger) error {
	resources := req.Result.Resources
	if len(resources) == 0 {
		return nil
	}

	start := time.Now()
	instanceLog.Info(fmt.Sprintf("waiting for %d resource(s) to become healthy", len(resources)), "timeout", deadline.Sub(start).Round(time.Second))

	waitCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	if err := kubernetes.Wait(waitCtx, req.K8sClient, resources, kubernetes.HealthyPredicate, start); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf(
			"instance %q was applied but is not healthy: %w\nThe resources are in place. Inspect them with:\n  opm instance status %s -n %s",
			req.Result.Instance.Name, err, req.Result.Instance.Name, req.Result.Instance.Namespace)}
	}

	output.Println(output.FormatCheckmark("Instance healthy"))
	return nil
}
