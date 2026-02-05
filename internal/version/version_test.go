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
