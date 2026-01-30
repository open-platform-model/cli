package version

import (
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

func TestInfoString(t *testing.T) {
	info := Info{
		Version:       "v1.0.0",
		GitCommit:     "abc123",
		BuildDate:     "2026-01-29",
		GoVersion:     "go1.25",
		CUESDKVersion: "v0.15.0",
	}

	str := info.String()

	assert.Contains(t, str, "v1.0.0")
	assert.Contains(t, str, "abc123")
	assert.Contains(t, str, "2026-01-29")
	assert.Contains(t, str, "go1.25")
	assert.Contains(t, str, "v0.15.0")
}

func TestCUEVersionCompatible(t *testing.T) {
	tests := []struct {
		name       string
		sdkVersion string
		binVersion string
		want       bool
	}{
		{
			name:       "exact match",
			sdkVersion: "v0.15.0",
			binVersion: "v0.15.0",
			want:       true,
		},
		{
			name:       "patch version different",
			sdkVersion: "v0.15.0",
			binVersion: "v0.15.1",
			want:       true,
		},
		{
			name:       "minor version different",
			sdkVersion: "v0.15.0",
			binVersion: "v0.14.0",
			want:       false,
		},
		{
			name:       "major version different",
			sdkVersion: "v0.15.0",
			binVersion: "v1.15.0",
			want:       false,
		},
		{
			name:       "without v prefix",
			sdkVersion: "0.15.0",
			binVersion: "0.15.1",
			want:       true,
		},
		{
			name:       "mixed v prefix",
			sdkVersion: "v0.15.0",
			binVersion: "0.15.0",
			want:       true,
		},
		{
			name:       "empty sdk version",
			sdkVersion: "",
			binVersion: "v0.15.0",
			want:       false,
		},
		{
			name:       "empty bin version",
			sdkVersion: "v0.15.0",
			binVersion: "",
			want:       false,
		},
		{
			name:       "invalid sdk format",
			sdkVersion: "invalid",
			binVersion: "v0.15.0",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CUEVersionCompatible(tt.sdkVersion, tt.binVersion)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractMajorMinor(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"v0.15.0", "0.15"},
		{"0.15.0", "0.15"},
		{"v1.2.3", "1.2"},
		{"1.2.3", "1.2"},
		{"v0.15.0-beta", "0.15"},
		{"invalid", ""},
		{"", ""},
		{"v1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := extractMajorMinor(tt.version)
			assert.Equal(t, tt.want, got)
		})
	}
}
