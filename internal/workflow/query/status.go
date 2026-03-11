package query

import (
	"context"
	"fmt"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/charmbracelet/log"
	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	pkginventory "github.com/opmodel/cli/pkg/inventory"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ParseStatusOutputFormat(outputFmt string) (output.Format, error) {
	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatDir {
		return "", &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, wide, yaml, json)", outputFmt),
		}
	}
	return outputFormat, nil
}

func ResolveInventory(
	ctx context.Context,
	client *kubernetes.Client,
	rsf *cmdutil.ReleaseSelectorFlags,
	namespace string,
	releaseLog *log.Logger,
) (inv *inventory.InventorySecret, live []*unstructured.Unstructured, missing []inventory.InventoryEntry, err error) {
	var invErr error
	switch {
	case rsf.ReleaseID != "":
		relName := rsf.ReleaseName
		if relName == "" {
			relName = rsf.ReleaseID
		}
		inv, invErr = inventory.GetInventory(ctx, client, relName, namespace, rsf.ReleaseID)
	case rsf.ReleaseName != "":
		inv, invErr = inventory.FindInventoryByReleaseName(ctx, client, rsf.ReleaseName, namespace)
	}

	if invErr != nil {
		releaseLog.Error("reading inventory", "error", invErr)
		err = &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("reading inventory: %w", invErr)}
		return nil, nil, nil, err
	}

	if inv == nil {
		name := rsf.ReleaseName
		if name == "" {
			name = rsf.ReleaseID
		}
		notFound := &kubernetes.ReleaseNotFoundError{Name: name, Namespace: namespace}
		releaseLog.Error("release not found", "name", name, "namespace", namespace)
		err = &opmexit.ExitError{Code: opmexit.ExitNotFound, Err: notFound, Printed: true}
		return nil, nil, nil, err
	}

	liveResources, missingEntries, discoverErr := inventory.DiscoverResourcesFromInventory(ctx, client, inv)
	if discoverErr != nil {
		releaseLog.Error("discovering resources from inventory", "error", discoverErr)
		err = &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("discovering resources: %w", discoverErr)}
		return nil, nil, nil, err
	}

	live = liveResources
	missing = missingEntries
	return inv, live, missing, nil
}

func BuildStatusOptions(namespace string, rsf *cmdutil.ReleaseSelectorFlags, outputFormat output.Format, verbose bool, inv *inventory.InventorySecret, liveResources []*unstructured.Unstructured, missingEntries []inventory.InventoryEntry) kubernetes.StatusOptions {
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
		Owner:         string(pkginventory.NormalizeCreatedBy(inv.ReleaseMetadata.CreatedBy)),
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

func PrintReleaseStatus(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions, logName string) error {
	releaseLog := output.ReleaseLogger(logName)

	result, err := kubernetes.GetReleaseStatus(ctx, client, opts)
	if err != nil {
		if kubernetes.IsNoResourcesFound(err) {
			releaseLog.Error("getting status", "error", err)
			return &opmexit.ExitError{Code: opmexit.ExitNotFound, Err: err, Printed: true}
		}
		releaseLog.Error("getting status", "error", err)
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

	formatted, err := kubernetes.FormatStatus(result, opts.OutputFormat)
	if err != nil {
		releaseLog.Error("formatting status", "error", err)
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err, Printed: true}
	}
	output.Println(formatted)

	if result.AggregateStatus != "Ready" && result.AggregateStatus != "Complete" {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf("release %q: %d resource(s) not ready", opts.ReleaseName, result.Summary.NotReady), Printed: true}
	}
	return nil
}
