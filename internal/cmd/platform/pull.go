package platformcmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/library/opm/helper/platformmodule"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
)

// NewPlatformPullCmd creates the platform pull command.
func NewPlatformPullCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var forceFlag bool

	c := &cobra.Command{
		Use:   "pull <dir>",
		Short: "Write the cluster's platform module to a directory",
		Long: `Reproduce the cluster's platform package on this machine.

Reads the cluster Platform CR, generates the platform module the operator
generated from the same inputs -- the effective registry it recorded on
status.registry, or the CR's subscriptions when it has recorded none -- and
writes that module's files to <dir>. Building against the result renders
what the cluster renders:

  opm module build --platform <dir> ./my-module

The module is the generated one unchanged: the reserved module path, the
pinned dependency closure and one importing registry entry per catalog. The
command writes nothing to the cluster and publishes nothing.

An existing non-empty <dir> is refused unless --force is given, which
replaces its contents.

Exit codes:

  0   the module was written
  1   <dir> is not empty and --force was not given; nothing was written
  2   the recorded registry does not generate -- an unpublished pin, or a
      cluster Platform predating the scalar-version subscription shape
  3   the cluster could not be reached
  5   no readable Platform CR: there is no cluster package to pull

Examples:
  # Pull the cluster's platform module next to a repository
  opm platform pull ./cluster-platform

  # Refresh it after the cluster's registry changed
  opm platform pull ./cluster-platform --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
				Config:         cfg,
				KubeconfigFlag: kf.Kubeconfig,
				ContextFlag:    kf.Context,
			})
			if err != nil {
				return &opmexit.ExitError{
					Code: opmexit.ExitGeneralError,
					Err:  fmt.Errorf("resolving kubernetes config: %w", err),
				}
			}
			k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
			if err != nil {
				return &opmexit.ExitError{
					Code: opmexit.ExitConnectivityError,
					Err:  fmt.Errorf("connecting to cluster: %w", err),
				}
			}
			return RunPlatformPull(c.Context(), cfg, PullOptions{
				Dir:     args[0],
				Force:   forceFlag,
				Cluster: platform.ClusterPlatformGetterFor(k8sClient.Dynamic),
			})
		},
	}

	kf.AddTo(c)
	c.Flags().BoolVar(&forceFlag, "force", false,
		"Replace the contents of a non-empty target directory")

	return c
}

// PullOptions are the inputs RunPlatformPull needs that do not come from the
// global config: the target directory, the overwrite decision and the two
// seams a test drives the command through without a cluster or a registry.
type PullOptions struct {
	// Dir is the directory the generated module is written to.
	Dir string
	// Force replaces the contents of a non-empty Dir.
	Force bool
	// Cluster reads the Platform CR. Never nil: pull has no offline arm.
	Cluster platform.ClusterPlatformGetter
	// ModFiles serves published module files for the closure derivation.
	// Nil constructs one from the configured registry; a test injects a
	// fixture graph.
	ModFiles platformmodule.ModFileSource
}

// RunPlatformPull resolves the cluster's platform, writes the generated
// module to opts.Dir and prints what it reproduced.
//
// The target is checked before the cluster is read: a refusal that depends
// only on the local filesystem should not cost a round trip, and it must
// leave the directory as it found it either way.
func RunPlatformPull(ctx context.Context, cfg *config.GlobalConfig, opts PullOptions) error {
	if err := checkPullTarget(opts.Dir, opts.Force); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}

	// Resolution decodes the document and raises the stale-status and
	// not-Ready warnings; the report needs the rows behind it, so the
	// getter is wrapped rather than called a second time.
	var doc *platform.ClusterPlatform
	capture := func(ctx context.Context) (*platform.ClusterPlatform, string, error) {
		d, unavailable, err := opts.Cluster(ctx)
		doc = d
		return d, unavailable, err
	}

	dir, res, err := platform.Resolve(ctx, platform.ResolveOptions{
		ConfigPath:      cfg.ConfigPath,
		Cluster:         capture,
		NoLocalFallback: true,
		Registry:        cfg.Registry,
		ModFiles:        opts.ModFiles,
	})
	if err != nil {
		return pullResolveError(err)
	}

	spec, eff, err := platform.DecodeCR(doc)
	if err != nil {
		// Unreachable: Resolve decoded the same document to get here.
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

	if err := copyModule(dir, opts.Dir); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}

	output.Println(pullReport(res, doc, spec, eff, opts.Dir))
	return nil
}

// pullResolveError maps a resolution failure onto the command's exit codes.
// Everything that is neither an unreadable CR nor an unreachable cluster
// happened after the document was read, so it is the recorded registry
// failing to generate.
func pullResolveError(err error) error {
	switch {
	case errors.Is(err, platform.ErrNoClusterPlatform):
		return &opmexit.ExitError{
			Code: opmexit.ExitNotFound,
			Err: fmt.Errorf("cluster Platform %q not found or not readable: nothing to pull (%w)",
				inventory.PlatformSingletonName, err),
		}
	case errors.Is(err, platform.ErrClusterRead):
		return &opmexit.ExitError{Code: opmexit.ExitConnectivityError, Err: err}
	default:
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}
}

// checkPullTarget refuses a target that is a file, or a non-empty directory
// without --force. A missing directory is fine: the copy creates it.
func checkPullTarget(dir string, force bool) error {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("checking target directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target %s is a file, not a directory", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading target directory %s: %w", dir, err)
	}
	if len(entries) > 0 && !force {
		return fmt.Errorf("target directory %s is not empty; pass --force to replace its contents", dir)
	}
	return nil
}

// copyModule replaces dst with a copy of the generated module at src. The
// files are read back out of the cache and written through the library's own
// writer, the one that produced the cache entry, so dst is the generated
// module exactly: real files, never links, and a self-contained module a
// repository can commit.
func copyModule(src, dst string) error {
	files, err := readModuleFiles(src)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("replacing %s: %w", dst, err)
	}
	if err := files.WriteTo(dst); err != nil {
		return fmt.Errorf("writing the platform module to %s: %w", dst, err)
	}
	return nil
}

// readModuleFiles reads every file of the generated module at dir, keyed by
// its slash-separated path relative to dir. Reads are root-scoped, so a
// symlink planted in the cache cannot walk the copy out of it.
func readModuleFiles(dir string) (platformmodule.Files, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("reading the generated platform module at %s: %w", dir, err)
	}
	defer func() { _ = root.Close() }()

	files := platformmodule.Files{}
	err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := root.ReadFile(filepath.FromSlash(path))
		if readErr != nil {
			return readErr
		}
		files[path] = data
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading the generated platform module at %s: %w", dir, err)
	}
	return files, nil
}

// pullReport is what the command prints: the provenance line resolution
// produced, the generations and operator the status recorded, every registry
// entry with its source, and the build command that consumes the directory.
func pullReport(res platform.Resolution, doc *platform.ClusterPlatform,
	spec platform.Spec, eff *platform.Effective, dir string,
) string {
	out := res.Describe() + "\n" + describeGeneration(doc, eff) + "\n\n"

	entries, sources := spec.Entries, map[string]string(nil)
	if res.RegistryOrigin == platform.RegistryOriginEffective {
		entries, sources = eff.Entries, eff.Sources
	}
	out += "registry: " + strconv.Itoa(len(entries)) + " " + pluralEntries(len(entries)) + "\n"
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		source := sources[e.Path]
		if source == "" {
			// The spec half records no source: every subscription is one.
			source = platform.EntrySourceSubscription
		}
		rows = append(rows, []string{e.Path, e.Version, enabledLabel(e.Enable), source})
	}
	out += output.AlignColumns("  ", rows)

	out += "\n\nwrote " + dir + "\n"
	out += "next: opm module build --platform " + dir + " <module-dir>"
	return out
}

// describeGeneration names the CR generation, the generation the operator
// observed and the operator that recorded it — the three facts that say how
// far the pulled package is from the spec on the cluster.
func describeGeneration(doc *platform.ClusterPlatform, eff *platform.Effective) string {
	line := "generation " + strconv.FormatInt(doc.Generation, 10)
	if eff == nil {
		return line + ", no operator generation recorded"
	}
	line += " (observed " + strconv.FormatInt(eff.ObservedGeneration, 10) + ")"
	if eff.OperatorVersion == "" {
		return line + ", operator version not recorded"
	}
	return line + ", operator " + eff.OperatorVersion
}

func enabledLabel(enable bool) string {
	if enable {
		return "enabled"
	}
	return "disabled"
}

func pluralEntries(n int) string {
	if n == 1 {
		return "entry"
	}
	return "entries"
}
