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
// on the cluster, or timeout runs out. The timeout is a fresh --timeout that
// starts with this wait, not what the apply's CustomResourceDefinition wait
// left over. It uses the shared poll loop in internal/kubernetes, so a resource that
// disappears after being applied fails the wait at once and a timeout names
// every resource still pending.
// The resources are already applied and recorded when it runs; a failure here
// leaves them in place, and the error says how to inspect them.
func waitForHealthy(ctx context.Context, req Request, timeout time.Duration, instanceLog *log.Logger) error {
	resources := req.Result.Resources
	if len(resources) == 0 {
		return nil
	}

	start := time.Now()
	instanceLog.Info(fmt.Sprintf("waiting for %d resource(s) to become healthy", len(resources)), "timeout", timeout)

	waitCtx, cancel := context.WithDeadline(ctx, start.Add(timeout))
	defer cancel()

	if err := kubernetes.Wait(waitCtx, req.K8sClient, resources, kubernetes.HealthyPredicate, start); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf(
			"instance %q was applied but is not healthy: %w\nThe resources are in place. Inspect them with:\n  opm instance status %s -n %s",
			req.Result.Instance.Name, err, req.Result.Instance.Name, req.Result.Instance.Namespace)}
	}

	output.Println(output.FormatCheckmark("Instance healthy"))
	return nil
}
