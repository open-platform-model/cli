package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCUEVersionCompatible(t *testing.T) {
	tests := []struct {
		name       string
		sdkVersion string
		binVersion string
		expected   bool
	}{
		{
			name:       "exact match",
			sdkVersion: "v0.11.0",
			binVersion: "v0.11.0",
			expected:   true,
		},
		{
			name:       "patch version differs",
			sdkVersion: "v0.11.0",
			binVersion: "v0.11.5",
			expected:   true,
		},
		{
			name:       "minor version differs",
			sdkVersion: "v0.11.0",
			binVersion: "v0.12.0",
			expected:   false,
		},
		{
			name:       "major version differs",
			sdkVersion: "v0.11.0",
			binVersion: "v1.0.0",
			expected:   false,
		},
		{
			name:       "without v prefix sdk",
			sdkVersion: "0.11.0",
			binVersion: "v0.11.0",
			expected:   true,
		},
		{
			name:       "without v prefix binary",
			sdkVersion: "v0.11.0",
			binVersion: "0.11.0",
			expected:   true,
		},
		{
			name:       "without v prefix both",
			sdkVersion: "0.11.0",
			binVersion: "0.11.0",
			expected:   true,
		},
		{
			name:       "pre-release version compatible",
			sdkVersion: "v0.15.0",
			binVersion: "v0.15.3-beta.1",
			expected:   true,
		},
		{
			name:       "invalid sdk version",
			sdkVersion: "invalid",
			binVersion: "v0.11.0",
			expected:   false,
		},
		{
			name:       "invalid binary version",
			sdkVersion: "v0.11.0",
			binVersion: "invalid",
			expected:   false,
		},
		{
			name:       "empty sdk version",
			sdkVersion: "",
			binVersion: "v0.11.0",
			expected:   false,
		},
		{
			name:       "empty binary version",
			sdkVersion: "v0.11.0",
			binVersion: "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CUEVersionCompatible(tt.sdkVersion, tt.binVersion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompatibilityMessage(t *testing.T) {
	tests := []struct {
		name       string
		sdkVersion string
		binVersion string
		contains   string
	}{
		{
			name:       "compatible",
			sdkVersion: "v0.11.0",
			binVersion: "v0.11.5",
			contains:   "compatible",
		},
		{
			name:       "major mismatch",
			sdkVersion: "v0.11.0",
			binVersion: "v1.0.0",
			contains:   "MAJOR",
		},
		{
			name:       "minor mismatch",
			sdkVersion: "v0.11.0",
			binVersion: "v0.12.0",
			contains:   "MINOR",
		},
		{
			name:       "invalid format",
			sdkVersion: "invalid",
			binVersion: "v0.11.0",
			contains:   "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := CompatibilityMessage(tt.sdkVersion, tt.binVersion)
			assert.Contains(t, msg, tt.contains)
		})
	}
}

func TestGetInfo(t *testing.T) {
	info := GetInfo()

	assert.NotEmpty(t, info.Version)
	assert.NotEmpty(t, info.GoVersion)
	assert.NotEmpty(t, info.CUESDKVersion)
}

func TestInfoString(t *testing.T) {
	info := Info{
		Version:       "v1.0.0",
		GitCommit:     "abc123",
		BuildDate:     "2026-01-01",
		GoVersion:     "go1.22.0",
		CUESDKVersion: "v0.15.0",
	}

	str := info.String()
	assert.Contains(t, str, "v1.0.0")
	assert.Contains(t, str, "abc123")
	assert.Contains(t, str, "v0.15.0")
}

func TestCUEBinaryInfoString(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		info := CUEBinaryInfo{Found: false}
		str := info.String()
		assert.Contains(t, str, "not found")
	})

	t.Run("found and compatible", func(t *testing.T) {
		info := CUEBinaryInfo{
			Found:      true,
			Version:    "v0.15.0",
			Path:       "/usr/local/bin/cue",
			Compatible: true,
		}
		str := info.String()
		assert.Contains(t, str, "v0.15.0")
		assert.Contains(t, str, "compatible")
		assert.Contains(t, str, "/usr/local/bin/cue")
	})

	t.Run("found but incompatible", func(t *testing.T) {
		info := CUEBinaryInfo{
			Found:      true,
			Version:    "v0.12.0",
			Path:       "/usr/local/bin/cue",
			Compatible: false,
			Message:    "incompatible - MINOR version mismatch",
		}
		str := info.String()
		assert.Contains(t, str, "v0.12.0")
		assert.Contains(t, str, "incompatible")
	})
}
