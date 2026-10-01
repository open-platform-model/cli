package version

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	info := Get()

	// Verify struct is populated
	require.NotEmpty(t, info.GoVersion, "GoVersion should be populated")
	require.NotEmpty(t, info.CUESDKVersion, "CUESDKVersion should be populated")
}

func TestCUESDKVersionFrom(t *testing.T) {
	tests := []struct {
		name string
		bi   *debug.BuildInfo
		want string
	}{
		{"nil build info", nil, "unknown"},
		{"no deps", &debug.BuildInfo{}, "unknown"},
		{
			"other deps only",
			&debug.BuildInfo{Deps: []*debug.Module{{Path: "example.com/x", Version: "v1.0.0"}}},
			"unknown",
		},
		{
			"plain dependency",
			&debug.BuildInfo{Deps: []*debug.Module{
				{Path: "example.com/x", Version: "v1.0.0"},
				{Path: "cuelang.org/go", Version: "v0.17.1"},
			}},
			"v0.17.1",
		},
		{
			"replaced by another version",
			&debug.BuildInfo{Deps: []*debug.Module{{
				Path: "cuelang.org/go", Version: "v0.17.1",
				Replace: &debug.Module{Path: "example.com/fork", Version: "v0.18.0-rc.1"},
			}}},
			"v0.18.0-rc.1",
		},
		{
			"replaced by local path",
			&debug.BuildInfo{Deps: []*debug.Module{{
				Path: "cuelang.org/go", Version: "v0.17.1",
				Replace: &debug.Module{Path: "../cue"},
			}}},
			"v0.17.1 (replaced)",
		},
		{
			"dependency without version",
			&debug.BuildInfo{Deps: []*debug.Module{{Path: "cuelang.org/go"}}},
			"unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cueSDKVersionFrom(tt.bi))
		})
	}
}

func TestInfoString(t *testing.T) {
	info := Info{
		Version:       "v1.0.0",
		GitCommit:     "abc123",
		BuildDate:     "2026-01-29",
		GoVersion:     "go1.25",
		CUESDKVersion: "v0.17.1",
	}

	str := info.String()

	assert.Contains(t, str, "v1.0.0")
	assert.Contains(t, str, "abc123")
	assert.Contains(t, str, "2026-01-29")
	assert.Contains(t, str, "go1.25")
	assert.Contains(t, str, "v0.17.1")
}
