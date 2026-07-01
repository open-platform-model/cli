package query

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/charmbracelet/log"
	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
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
	rsf *cmdutil.InstanceSelectorFlags,
	namespace string,
	instanceLog *log.Logger,
) (inv *inventory.InstanceInventoryRecord, live []*unstructured.Unstructured, missing []inventory.InventoryEntry, err error) {
	var invErr error
	switch {
	case rsf.InstanceID != "":
		relName := rsf.InstanceName
		if relName == "" {
			relName = rsf.InstanceID
		}
		inv, invErr = inventory.GetInventory(ctx, client, relName, namespace, rsf.InstanceID)
	case rsf.InstanceName != "":
		inv, invErr = inventory.FindInventoryByInstanceName(ctx, client, rsf.InstanceName, namespace)
	}

	if invErr != nil {
		instanceLog.Error("reading inventory", "error", invErr)
		err = &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("reading inventory: %w", invErr)}
		return nil, nil, nil, err
	}

	if inv == nil {
		name := rsf.InstanceName
		if name == "" {
			name = rsf.InstanceID
		}
		notFound := &kubernetes.InstanceNotFoundError{Name: name, Namespace: namespace}
		instanceLog.Error("instance not found", "name", name, "namespace", namespace)
		err = &opmexit.ExitError{Code: opmexit.ExitNotFound, Err: notFound, Printed: true}
		return nil, nil, nil, err
	}

	liveResources, missingEntries, discoverErr := inventory.DiscoverResourcesFromInventory(ctx, client, inv)
	if discoverErr != nil {
		instanceLog.Error("discovering resources from inventory", "error", discoverErr)
		err = &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("discovering resources: %w", discoverErr)}
		return nil, nil, nil, err
	}

	live = liveResources
	missing = missingEntries
	return inv, live, missing, nil
}

func BuildStatusOptions(namespace string, rsf *cmdutil.InstanceSelectorFlags, outputFormat output.Format, verbose bool, inv *inventory.InstanceInventoryRecord, liveResources []*unstructured.Unstructured, missingEntries []inventory.InventoryEntry) kubernetes.StatusOptions {
	componentMap := make(map[string]string)
	for _, entry := range inv.Inventory.Entries {
		key := entry.Kind + "/" + entry.Namespace + "/" + entry.Name
		componentMap[key] = entry.Component
	}

	statusOpts := kubernetes.StatusOptions{
		Namespace:     namespace,
		InstanceName:  rsf.InstanceName,
		InstanceID:    rsf.InstanceID,
		Version:       inv.ModuleMetadata.Version,
		Owner:         string(inventory.NormalizeCreatedBy(inv.CreatedBy)),
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

func PrintInstanceStatus(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions, logName string) error {
	instanceLog := output.InstanceLogger(logName)

	result, err := kubernetes.GetInstanceStatus(ctx, client, opts)
	if err != nil {
		if kubernetes.IsNoResourcesFound(err) {
			instanceLog.Error("getting status", "error", err)
			return &opmexit.ExitError{Code: opmexit.ExitNotFound, Err: err, Printed: true}
		}
		instanceLog.Error("getting status", "error", err)
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

	formatted, err := kubernetes.FormatStatus(result, opts.OutputFormat)
	if err != nil {
		instanceLog.Error("formatting status", "error", err)
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err, Printed: true}
	}
	output.Println(formatted)

	if result.AggregateStatus != "Ready" && result.AggregateStatus != "Complete" {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf("instance %q: %d resource(s) not ready", opts.InstanceName, result.Summary.NotReady), Printed: true}
	}
	return nil
}
