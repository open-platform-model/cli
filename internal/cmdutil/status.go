package cmdutil

import (
	"context"
	"fmt"

	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ParseStatusOutputFormat validates formats supported by status commands.
func ParseStatusOutputFormat(outputFmt string) (output.Format, error) {
	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatDir {
		return "", &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, wide, yaml, json)", outputFmt),
		}
	}
	return outputFormat, nil
}

// BuildStatusOptions constructs kubernetes status options from inventory data.
func BuildStatusOptions(namespace string, rsf *ReleaseSelectorFlags, outputFormat output.Format, verbose bool, inv *inventory.InventorySecret, liveResources []*unstructured.Unstructured, missingEntries []inventory.InventoryEntry) kubernetes.StatusOptions {
	componentMap := make(map[string]string)
	var version string
	if len(inv.Index) > 0 {
		if change, ok := inv.Changes[inv.Index[0]]; ok {
			version = change.Source.Version
			for _, entry := range change.Inventory.Entries {
				key := entry.Kind + "/" + entry.Namespace + "/" + entry.Name
				componentMap[key] = entry.Component
			}
		}
	}

	statusOpts := kubernetes.StatusOptions{
		Namespace:     namespace,
		ReleaseName:   rsf.ReleaseName,
		ReleaseID:     rsf.ReleaseID,
		Version:       version,
		ComponentMap:  componentMap,
		OutputFormat:  outputFormat,
		InventoryLive: liveResources,
		Wide:          outputFormat == output.FormatWide,
		Verbose:       verbose,
	}
	for _, m := range missingEntries {
		statusOpts.MissingResources = append(statusOpts.MissingResources, kubernetes.MissingResource{
			Kind:      m.Kind,
			Namespace: m.Namespace,
			Name:      m.Name,
		})
	}
	return statusOpts
}

// PrintReleaseStatus fetches and displays the current status.
func PrintReleaseStatus(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions, logName string) error {
	releaseLog := output.ReleaseLogger(logName)

	result, err := kubernetes.GetReleaseStatus(ctx, client, opts)
	if err != nil {
		if kubernetes.IsNoResourcesFound(err) {
			releaseLog.Error("getting status", "error", err)
			return &oerrors.ExitError{Code: oerrors.ExitNotFound, Err: err, Printed: true}
		}
		releaseLog.Error("getting status", "error", err)
		return &oerrors.ExitError{Code: ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

	formatted, err := kubernetes.FormatStatus(result, opts.OutputFormat)
	if err != nil {
		releaseLog.Error("formatting status", "error", err)
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err, Printed: true}
	}
	output.Println(formatted)

	if result.AggregateStatus != "Ready" && result.AggregateStatus != "Complete" {
		return &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: fmt.Errorf("release %q: %d resource(s) not ready", opts.ReleaseName, result.Summary.NotReady), Printed: true}
	}
	return nil
}

// LogResolvedKubernetesConfig emits the resolved Kubernetes config at debug level.
func LogResolvedKubernetesConfig(k8sConfigNamespace, kubeconfig, contextName string) {
	output.Debug("resolved kubernetes config",
		"kubeconfig", kubeconfig,
		"context", contextName,
		"namespace", k8sConfigNamespace,
	)
}
