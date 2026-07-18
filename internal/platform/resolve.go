package platform

import (
	"context"
	"fmt"
	"os"

	"github.com/open-platform-model/library/opm/helper/synth"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
)

// Source identifies which precedence step produced the resolved platform.
type Source string

const (
	// SourceFlagFile is the explicit --platform <file> override.
	SourceFlagFile Source = "flag"
	// SourceClusterCR is the cluster Platform CR spec.
	SourceClusterCR Source = "cluster"
	// SourceLocalDefault is the local default ~/.opm/platform.cue.
	SourceLocalDefault Source = "local"
)

// Resolution reports where the platform came from — the provenance every
// render-bearing command surfaces (D21: the fallback warns, it never
// silently swaps platforms).
type Resolution struct {
	// Source is the precedence step that produced the platform.
	Source Source
	// Location names the concrete origin (file path or CR name).
	Location string
	// Warning is non-empty when resolution fell back from the cluster CR
	// to the local default.
	Warning string
}

// Describe returns the one-line provenance description for command output.
func (r Resolution) Describe() string {
	switch r.Source {
	case SourceFlagFile:
		return "platform: " + r.Location + " (--platform)"
	case SourceClusterCR:
		return "platform: cluster Platform CR " + r.Location
	case SourceLocalDefault:
		return "platform: " + r.Location + " (local default)"
	default:
		return "platform: unknown source"
	}
}

// ClusterSpecGetter fetches the cluster Platform CR's spec. It returns
// (spec, name, "", nil) on success and ("", unavailable-reason, nil) when the
// CR is absent or unreadable in a way that permits warn-fallback (NotFound,
// Forbidden — D21). Any other error is fatal to resolution.
type ClusterSpecGetter func(ctx context.Context) (spec map[string]any, name string, unavailable string, err error)

// ResolveOptions selects the platform sources for one command invocation.
type ResolveOptions struct {
	// PlatformFlag is the --platform flag value (highest precedence).
	PlatformFlag string
	// ConfigPath is the resolved config file path; the local default
	// platform file is its sibling platform.cue.
	ConfigPath string
	// Cluster is the cluster CR getter. nil means the command is offline
	// (build/render) and MUST NOT read the cluster (D17/D21).
	Cluster ClusterSpecGetter
}

// Resolve resolves the platform spec by precedence and returns the typed
// kernel input plus provenance. It performs no materialization (registry
// I/O stays a separate, caller-driven step — see Materialize).
func Resolve(ctx context.Context, opts ResolveOptions) (synth.PlatformInput, Resolution, error) {
	// 1. Explicit local override.
	if opts.PlatformFlag != "" {
		in, err := DecodeFile(opts.PlatformFlag)
		if err != nil {
			return synth.PlatformInput{}, Resolution{}, err
		}
		return in, Resolution{Source: SourceFlagFile, Location: opts.PlatformFlag}, nil
	}

	// 2. Cluster Platform CR (cluster-facing commands only).
	fallbackWarning := ""
	if opts.Cluster != nil {
		spec, name, unavailable, err := opts.Cluster(ctx)
		if err != nil {
			return synth.PlatformInput{}, Resolution{}, fmt.Errorf("reading cluster Platform: %w", err)
		}
		if unavailable == "" {
			in, err := DecodeCRSpec(spec, name)
			if err != nil {
				return synth.PlatformInput{}, Resolution{}, err
			}
			return in, Resolution{Source: SourceClusterCR, Location: name}, nil
		}
		fallbackWarning = "cluster Platform not used (" + unavailable + ") — falling back to the local default platform"
		output.Warn(fallbackWarning)
	}

	// 3. Local default.
	localPath := config.PlatformFilePath(opts.ConfigPath)
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return synth.PlatformInput{}, Resolution{}, fmt.Errorf(
			"no platform source available: no --platform flag%s and %s does not exist — run 'opm config init' to seed a local default platform",
			clusterCloseParen(opts.Cluster != nil), localPath)
	}
	in, err := DecodeFile(localPath)
	if err != nil {
		return synth.PlatformInput{}, Resolution{}, err
	}
	return in, Resolution{Source: SourceLocalDefault, Location: localPath, Warning: fallbackWarning}, nil
}

// clusterCloseParen phrases the no-source error for cluster-facing vs
// offline commands.
func clusterCloseParen(clusterTried bool) string {
	if clusterTried {
		return ", no readable cluster Platform,"
	}
	return ""
}
