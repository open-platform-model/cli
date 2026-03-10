package release

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// NewReleaseStatusCmd creates the release status command.
func NewReleaseStatusCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		outputFlag  string
		detailsFlag bool
	)

	c := &cobra.Command{
		Use:   "status <file|name|uuid>",
		Short: "Show resource status for a release",
		Long: `Show status of resources deployed by an OPM release.

Arguments:
  file         Path to a release.cue file or directory containing one.
               The release name and namespace are read from the file's metadata.
               --namespace overrides the namespace found in the file.
  name         Release name (use -n / --namespace to scope by namespace).
  uuid         Release UUID.

Examples:
  # Identify by release.cue file in the current directory
  opm release status .

  # Identify by release.cue file path
  opm release status ./releases/jellyfin/release.cue

  # Identify by name
  opm release status jellyfin -n media

  # Wide output
  opm release status jellyfin -n media -o wide`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseStatus(args[0], cfg, &kf, namespace, outputFlag, detailsFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace (default: from config)")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, wide, yaml, json)")
	c.Flags().BoolVar(&detailsFlag, "details", false, "Show pod-level diagnostics for unhealthy workloads")

	return c
}

func runReleaseStatus(identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag, outputFmt string, verbose bool) error {
	ctx := context.Background()

	ra, err := cmdutil.ResolveReleaseArg(identifier, cfg)
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}
	rsf := ra.ToSelectorFlags(namespaceFlag)

	if err := rsf.Validate(); err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  ra.EffectiveNamespace(namespaceFlag),
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}
	if err := cmdutil.RequireNamespace(k8sConfig); err != nil {
		return err
	}

	namespace := k8sConfig.Namespace.Value
	cmdutil.LogResolvedKubernetesConfig(namespace, k8sConfig.Kubeconfig.Value, k8sConfig.Context.Value)

	logName := rsf.LogName()
	releaseLog := output.ReleaseLogger(logName)

	outputFormat, err := cmdutil.ParseStatusOutputFormat(outputFmt)
	if err != nil {
		return err
	}

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	inv, liveResources, missingEntries, err := cmdutil.ResolveInventory(ctx, k8sClient, rsf, namespace, releaseLog)
	if err != nil {
		return err
	}

	statusOpts := cmdutil.BuildStatusOptions(namespace, rsf, outputFormat, verbose, inv, liveResources, missingEntries)
	return cmdutil.PrintReleaseStatus(ctx, k8sClient, statusOpts, logName)
}
