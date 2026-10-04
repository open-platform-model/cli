package operator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/operator/operatortest"
	"github.com/open-platform-model/cli/internal/publish"
)

func TestReadOperatorVersion(t *testing.T) {
	reg := operatortest.Registry(t,
		operatortest.Version{Module: "0.1.0", Operator: "1.0.0-beta.7"},
		operatortest.Version{Module: "0.2.0", OperatorPackage: "-"},
		operatortest.Version{Module: "0.3.0", OperatorPackage: "package operator\n\nImage: tag: \"v1\"\n"},
		operatortest.Version{Module: "0.4.0", Operator: "latest"},
		operatortest.Version{Module: "0.5.0", Operator: "v1.0.0"},
		operatortest.Version{Module: "0.6.0", OperatorPackage: "package operator\n\nVersion: string\n"},
	)
	src := operatortest.Source(t, reg)
	ctx := context.Background()

	got, err := ReadOperatorVersion(ctx, src, "v0.1.0")
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0-beta.7", got)

	refusals := map[string]string{
		"v0.2.0": "has no operator package",
		"v0.3.0": "has no Version",
		"v0.4.0": `"latest" is not a SemVer release`,
		"v0.5.0": `"v1.0.0" is not a SemVer release`,
		"v0.6.0": "is not a concrete string",
	}
	for version, want := range refusals {
		_, err := ReadOperatorVersion(ctx, src, version)
		var verr *VersionError
		require.True(t, errors.As(err, &verr), "%s: want an *VersionError, got %v", version, err)
		assert.Contains(t, err.Error(), want, version)
		assert.Contains(t, err.Error(), OperatorModulePath+" "+version[1:], "the error names the module and its version")
	}
}

func TestReadOperatorVersion_UnservedVersionIsAFetchFailure(t *testing.T) {
	src := operatortest.Source(t, operatortest.Registry(t, operatortest.Version{Module: "0.1.0", Operator: "1.0.0"}))

	_, err := ReadOperatorVersion(context.Background(), src, "v0.9.0")
	var connErr *publish.ConnectivityError
	require.True(t, errors.As(err, &connErr), "got %v", err)
	assert.Contains(t, err.Error(), "opmodel.dev/modules/opm_operator@v0.9.0")
}
