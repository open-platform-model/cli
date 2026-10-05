package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/controller/controllertest"
	"github.com/open-platform-model/cli/internal/publish"
)

func TestReadControllerVersion(t *testing.T) {
	reg := controllertest.Registry(t,
		controllertest.Version{Module: "0.1.0", Controller: "1.0.0-beta.7"},
		controllertest.Version{Module: "0.2.0", ControllerPackage: "-"},
		controllertest.Version{Module: "0.3.0", ControllerPackage: "package controller\n\nImage: tag: \"v1\"\n"},
		controllertest.Version{Module: "0.4.0", Controller: "latest"},
		controllertest.Version{Module: "0.5.0", Controller: "v1.0.0"},
		controllertest.Version{Module: "0.6.0", ControllerPackage: "package controller\n\nVersion: string\n"},
	)
	src := controllertest.Source(t, reg)
	ctx := context.Background()

	got, err := ReadControllerVersion(ctx, src, "v0.1.0")
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0-beta.7", got)

	refusals := map[string]string{
		"v0.2.0": "has no controller package",
		"v0.3.0": "has no Version",
		"v0.4.0": `"latest" is not a SemVer release`,
		"v0.5.0": `"v1.0.0" is not a SemVer release`,
		"v0.6.0": "is not a concrete string",
	}
	for version, want := range refusals {
		_, err := ReadControllerVersion(ctx, src, version)
		var verr *VersionError
		require.True(t, errors.As(err, &verr), "%s: want an *VersionError, got %v", version, err)
		assert.Contains(t, err.Error(), want, version)
		assert.Contains(t, err.Error(), ControllerModulePath+" "+version[1:], "the error names the module and its version")
	}
}

func TestReadControllerVersion_UnservedVersionIsAFetchFailure(t *testing.T) {
	src := controllertest.Source(t, controllertest.Registry(t, controllertest.Version{Module: "0.1.0", Controller: "1.0.0"}))

	_, err := ReadControllerVersion(context.Background(), src, "v0.9.0")
	var connErr *publish.ConnectivityError
	require.True(t, errors.As(err, &connErr), "got %v", err)
	assert.Contains(t, err.Error(), "opmodel.dev/modules/opm_controller@v0.9.0")
}
