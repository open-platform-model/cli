package query

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

func ParseEventsOptions(since, eventType, outputFmt string, watchMode bool) (kubernetes.EventsOptions, error) {
	if eventType != "" && eventType != "Normal" && eventType != "Warning" {
		return kubernetes.EventsOptions{}, &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid --type %q (valid: Normal, Warning)", eventType),
		}
	}

	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatWide || outputFormat == output.FormatDir {
		return kubernetes.EventsOptions{}, &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, json, yaml)", outputFmt),
		}
	}
	if watchMode && outputFormat != output.FormatTable {
		return kubernetes.EventsOptions{}, &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("watch mode only supports table output"),
		}
	}

	sinceCutoff, err := kubernetes.ParseSince(since)
	if err != nil {
		return kubernetes.EventsOptions{}, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}

	return kubernetes.EventsOptions{Since: sinceCutoff, EventType: eventType, OutputFormat: outputFormat}, nil
}

func PrintEvents(ctx context.Context, client *kubernetes.Client, opts kubernetes.EventsOptions, logName string) error {
	releaseLog := output.ReleaseLogger(logName)
	result, err := kubernetes.GetModuleEvents(ctx, client, opts)
	if err != nil {
		if kubernetes.IsNoResourcesFound(err) {
			releaseLog.Error("getting events", "error", err)
			return &oerrors.ExitError{Code: oerrors.ExitNotFound, Err: err, Printed: true}
		}
		releaseLog.Error("getting events", "error", err)
		return &oerrors.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

	formatted, err := kubernetes.FormatEvents(result, opts.OutputFormat)
	if err != nil {
		releaseLog.Error("formatting events", "error", err)
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err, Printed: true}
	}

	output.Println(formatted)
	return nil
}

func WatchEvents(ctx context.Context, client *kubernetes.Client, opts kubernetes.EventsOptions, logName string) error { //nolint:gocyclo
	releaseLog := output.ReleaseLogger(logName)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	children, err := kubernetes.DiscoverChildren(ctx, client, opts.InventoryLive, opts.Namespace)
	if err != nil {
		releaseLog.Error("discovering children", "error", err)
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("discovering children: %w", err)}
	}

	uidSet := make(map[types.UID]bool)
	for _, res := range opts.InventoryLive {
		uidSet[res.GetUID()] = true
	}
	for _, child := range children {
		uidSet[child.GetUID()] = true
	}

	watcher, err := client.Clientset.CoreV1().Events(opts.Namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		releaseLog.Error("starting event watch", "error", err)
		return &oerrors.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: fmt.Errorf("starting event watch: %w", err)}
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return nil
			}
			if event.Type != watch.Added && event.Type != watch.Modified {
				continue
			}
			ev, ok := event.Object.(*corev1.Event)
			if !ok || !uidSet[ev.InvolvedObject.UID] {
				continue
			}
			if opts.EventType != "" && ev.Type != opts.EventType {
				continue
			}
			if !opts.Since.IsZero() {
				evTime := ev.LastTimestamp.Time
				if evTime.IsZero() {
					evTime = ev.CreationTimestamp.Time
				}
				if evTime.Before(opts.Since) {
					continue
				}
			}
			output.Println(kubernetes.FormatSingleEventLine(ev))
		}
	}
}
