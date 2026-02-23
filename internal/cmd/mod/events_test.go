package mod

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/kubernetes"
)

func TestNewModEventsCmd_FlagsExist(t *testing.T) {
	cfg := &config.GlobalConfig{}
	cmd := NewModEventsCmd(cfg)

	// Verify all expected flags exist.
	flags := []string{
		"release-name", "release-id", "namespace",
		"kubeconfig", "context",
		"since", "type", "watch", "output",
	}
	for _, name := range flags {
		f := cmd.Flags().Lookup(name)
		assert.NotNilf(t, f, "flag --%s should exist", name)
	}
}

func TestNewModEventsCmd_FlagDefaults(t *testing.T) {
	cfg := &config.GlobalConfig{}
	cmd := NewModEventsCmd(cfg)

	assert.Equal(t, "1h", cmd.Flags().Lookup("since").DefValue)
	assert.Equal(t, "", cmd.Flags().Lookup("type").DefValue)
	assert.Equal(t, "false", cmd.Flags().Lookup("watch").DefValue)
	assert.Equal(t, "table", cmd.Flags().Lookup("output").DefValue)
}

func TestNewModEventsCmd_OutputShorthand(t *testing.T) {
	cfg := &config.GlobalConfig{}
	cmd := NewModEventsCmd(cfg)

	f := cmd.Flags().ShorthandLookup("o")
	assert.NotNil(t, f, "flag -o shorthand should exist")
	assert.Equal(t, "output", f.Name)
}

// TestRunEvents_ValidationErrors tests early-return validation paths in runEvents.
// These paths execute before any Kubernetes calls, so no fake client is needed.
func TestRunEvents_ValidationErrors(t *testing.T) {
	cfg := &config.GlobalConfig{}
	kf := &cmdutil.K8sFlags{}

	tests := []struct {
		name      string
		rsf       cmdutil.ReleaseSelectorFlags
		typeFlag  string
		output    string
		since     string
		expectErr string
	}{
		{
			name:      "mutual exclusivity: both selectors",
			rsf:       cmdutil.ReleaseSelectorFlags{ReleaseName: "app", ReleaseID: "uuid"},
			expectErr: "mutually exclusive",
		},
		{
			name:      "neither selector provided",
			rsf:       cmdutil.ReleaseSelectorFlags{},
			expectErr: "either --release-name or --release-id is required",
		},
		{
			name:      "invalid --type",
			rsf:       cmdutil.ReleaseSelectorFlags{ReleaseName: "app", Namespace: "ns"},
			typeFlag:  "Error",
			expectErr: "invalid --type",
		},
		{
			name:      "invalid --output",
			rsf:       cmdutil.ReleaseSelectorFlags{ReleaseName: "app", Namespace: "ns"},
			output:    "xml",
			expectErr: "invalid output format",
		},
		{
			name:      "invalid --since",
			rsf:       cmdutil.ReleaseSelectorFlags{ReleaseName: "app", Namespace: "ns"},
			since:     "foo",
			expectErr: "invalid --since",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputFmt := tt.output
			if outputFmt == "" {
				outputFmt = "table"
			}
			sinceFmt := tt.since
			if sinceFmt == "" {
				sinceFmt = "1h"
			}
			err := runEvents(nil, cfg, &tt.rsf, kf, sinceFmt, tt.typeFlag, false, outputFmt)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectErr)
		})
	}
}

// TestRunEventsWatch_ContextCancellation verifies that runEventsWatch exits
// cleanly (returns nil) when the context is canceled, matching the signal
// handling contract (SIGINT/SIGTERM → exit code 0).
func TestRunEventsWatch_ContextCancellation(t *testing.T) {
	clientset := fake.NewSimpleClientset() //nolint:staticcheck // matches existing test patterns
	client := &kubernetes.Client{Clientset: clientset}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately — simulates SIGINT

	opts := kubernetes.EventsOptions{
		Namespace:     "default",
		InventoryLive: nil, // no resources — watch will start on empty set
	}

	err := runEventsWatch(ctx, client, opts, "test")
	assert.NoError(t, err, "runEventsWatch should return nil on context cancellation")
}
