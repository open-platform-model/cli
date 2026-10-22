package controller

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// The names of the earlier install of the controller, which this CLI no
// longer manages. Install only reads the Deployment, to point the user to the
// docs page; it never writes, adopts or migrates the earlier install.
const (
	earlierControllerNamespace  = "opm-operator-system"
	earlierControllerDeployment = "opm-operator-controller-manager"

	// earlierInstallPage is the title of the docs page that says how to move
	// off the earlier controller.
	earlierInstallPage = "Install over the earlier controller"
)

// earlierInstallNotice is the one line install prints when the earlier
// controller is on the cluster. It points to the docs page and gives no
// deletion advice.
const earlierInstallNotice = "the cluster still runs the controller of the earlier install (Deployment " +
	earlierControllerDeployment + " in namespace " + earlierControllerNamespace +
	"); this install does not migrate it, and the opm docs page \"" + earlierInstallPage + "\" says how to move to this one"

// earlierNotice returns the notice when a successful read finds the
// earlier controller's Deployment, and "" when it is absent or the read
// fails for any reason: the notice is advice, so an unknown answer prints
// nothing and changes no outcome.
func earlierNotice(ctx context.Context, client *kubernetes.Client) string {
	probe := &unstructured.Unstructured{}
	probe.SetAPIVersion("apps/v1")
	probe.SetKind(kindDeployment)
	probe.SetName(earlierControllerDeployment)
	probe.SetNamespace(earlierControllerNamespace)
	if _, err := getObject(ctx, client, probe); err != nil {
		return ""
	}
	return earlierInstallNotice
}

// HintedError is a refusal followed by one hint line. It keeps the refusal in
// the chain, so the exit code and every errors.As on the refusal are unchanged.
type HintedError struct {
	Err  error
	Hint string
}

func (e *HintedError) Error() string { return e.Err.Error() + "\nhint: " + e.Hint }
func (e *HintedError) Unwrap() error { return e.Err }

// HintEarlierInstall adds the hint line to a refusal of the apply guard when
// the earlier controller is on the cluster. Any other error, and a refusal
// with no earlier Deployment found, is returned as it is.
func HintEarlierInstall(ctx context.Context, client *kubernetes.Client, err error) error {
	var guardErr *GuardError
	if !errors.As(err, &guardErr) {
		return err
	}
	notice := earlierNotice(ctx, client)
	if notice == "" {
		return err
	}
	return &HintedError{Err: err, Hint: notice}
}

// WarnEarlierInstall prints the notice as one warning, for an install whose
// checks all passed and that has not written yet.
func WarnEarlierInstall(ctx context.Context, client *kubernetes.Client) {
	if notice := earlierNotice(ctx, client); notice != "" {
		output.Warn(notice)
	}
}
