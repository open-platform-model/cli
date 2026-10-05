package query

import (
	"context"
	"fmt"

	"github.com/open-platform-model/library/opm/k8s/health"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"

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

// ResolveInventory reads the instance's ModuleInstance record and discovers
// the live state of every resource it tracks. A failed record read exits 1 and
// a missing record exits 5. Each tracked resource is returned as live, missing
// (NotFound) or unreadable (any other read error); ResolveInventory prints
// nothing about unreadable entries, since delete and the read-only commands
// report them differently.
func ResolveInventory(
	ctx context.Context,
	client *kubernetes.Client,
	rsf *cmdutil.InstanceSelectorFlags,
	namespace string,
	instanceLog *log.Logger,
) (inv *inventory.Record, live []*unstructured.Unstructured, missing []k8sinventory.Entry, unreadable []inventory.UnreadableEntry, err error) {
	var invErr error
	switch {
	case rsf.InstanceID != "":
		// Instance-id selectors resolve by listing ModuleInstance CRs and
		// matching status.instanceUUID.
		inv, invErr = inventory.FindRecordByInstanceUUID(ctx, client, namespace, rsf.InstanceID)
	case rsf.InstanceName != "":
		// Name selectors resolve by a direct ModuleInstance GET.
		inv, invErr = inventory.GetRecord(ctx, client, rsf.InstanceName, namespace)
	}

	if invErr != nil {
		instanceLog.Error("reading inventory", "error", invErr)
		err = &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("reading inventory: %w", invErr)}
		return nil, nil, nil, nil, err
	}

	if inv == nil {
		name := rsf.InstanceName
		if name == "" {
			name = rsf.InstanceID
		}
		notFound := &kubernetes.InstanceNotFoundError{Name: name, Namespace: namespace}
		instanceLog.Error("instance not found", "name", name, "namespace", namespace)
		err = &opmexit.ExitError{Code: opmexit.ExitNotFound, Err: notFound, Printed: true}
		return nil, nil, nil, nil, err
	}

	live, missing, unreadable, discoverErr := inventory.DiscoverResourcesFromInventory(ctx, client, inv)
	if discoverErr != nil {
		instanceLog.Error("discovering resources from inventory", "error", discoverErr)
		err = &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("discovering resources: %w", discoverErr)}
		return nil, nil, nil, nil, err
	}

	return inv, live, missing, unreadable, nil
}

// WarnUnreadable logs one warning per tracked resource that could not be read,
// naming it and the read error. The read-only instance commands call it so a
// failed read is never silent.
func WarnUnreadable(logger *log.Logger, unreadable []inventory.UnreadableEntry) {
	for _, u := range unreadable {
		logger.Warn("could not read tracked resource",
			"kind", u.Entry.Kind, "namespace", u.Entry.Namespace, "name", u.Entry.Name, "error", u.Err)
	}
}

// BuildStatusOptions assembles the status options from a resolved inventory.
// Missing entries become "Missing" rows and unreadable entries "Unknown" rows.
func BuildStatusOptions(namespace string, rsf *cmdutil.InstanceSelectorFlags, outputFormat output.Format, verbose bool, inv *inventory.Record, liveResources []*unstructured.Unstructured, missingEntries []k8sinventory.Entry, unreadable []inventory.UnreadableEntry) kubernetes.StatusOptions {
	componentMap := make(map[string]string)
	for _, entry := range inv.Inventory.Entries {
		key := entry.Kind + "/" + entry.Namespace + "/" + entry.Name
		componentMap[key] = entry.Component
	}

	statusOpts := kubernetes.StatusOptions{
		Namespace:           namespace,
		InstanceName:        rsf.InstanceName,
		InstanceID:          rsf.InstanceID,
		Version:             inv.ModuleVersion,
		Owner:               inventory.DisplayOwner(inv.Owner),
		ComponentMap:        componentMap,
		OutputFormat:        outputFormat,
		InventoryLive:       liveResources,
		Wide:                outputFormat == output.FormatWide,
		Verbose:             verbose,
		UnreadableResources: inventory.UnreadableResources(unreadable),
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

	if !health.IsHealthy(result.AggregateStatus) {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf("instance %q: %d resource(s) not ready", opts.InstanceName, result.Summary.NotReady), Printed: true}
	}
	return nil
}
