package apply

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
	pkgmodule "github.com/open-platform-model/cli/pkg/module"
)

func operatorOwnedRequest(result *workflowrender.Result) Request {
	return Request{Result: result, Log: output.InstanceLogger("test")}
}

// A local render describes bytes the operator cannot fetch. Refused before any
// write, so the failure costs nothing and leaves the CR untouched.
func TestThinEditor_RefusesLocalSourceModule(t *testing.T) {
	result := &workflowrender.Result{
		Instance:    pkgmodule.InstanceMetadata{Name: "podinfo", Namespace: "demo"},
		Module:      pkgmodule.ModuleMetadata{Name: "podinfo"},
		SourceLocal: true,
	}

	err := executeThinEditor(context.Background(), operatorOwnedRequest(result),
		&inventory.Record{Owner: inventory.OwnerOperator, Name: "podinfo", Namespace: "demo"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "local bytes")
	assert.Contains(t, err.Error(), "publish the module")
}

func TestThinEditor_RefusesIncompleteModuleReference(t *testing.T) {
	result := &workflowrender.Result{
		Instance: pkgmodule.InstanceMetadata{Name: "podinfo", Namespace: "demo"},
		Module:   pkgmodule.ModuleMetadata{},
	}

	err := executeThinEditor(context.Background(), operatorOwnedRequest(result),
		&inventory.Record{Owner: inventory.OwnerOperator, Name: "podinfo", Namespace: "demo"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "module reference")
}

// A dry-run against an operator-owned instance must preview the spec edit, not
// the CLI-executor render-and-apply it will never perform.
func TestPreviewThinEditor_DescribesTheSpecEditOnly(t *testing.T) {
	result := &workflowrender.Result{
		Instance: pkgmodule.InstanceMetadata{Name: "podinfo", Namespace: "demo"},
		Module: pkgmodule.ModuleMetadata{
			ModulePath: "opmodel.dev/modules", Name: "podinfo", Version: "0.1.3",
		},
	}

	err := previewThinEditor(operatorOwnedRequest(result),
		&inventory.Record{Owner: inventory.OwnerOperator, Name: "podinfo", Namespace: "demo"})

	require.NoError(t, err)
}

// The preview runs the same gates as the real edit, so a local-bytes render is
// rejected at dry-run rather than deferred to the apply that follows it.
func TestPreviewThinEditor_RefusesLocalSourceModule(t *testing.T) {
	result := &workflowrender.Result{
		Instance:    pkgmodule.InstanceMetadata{Name: "podinfo", Namespace: "demo"},
		Module:      pkgmodule.ModuleMetadata{Name: "podinfo"},
		SourceLocal: true,
	}

	err := previewThinEditor(operatorOwnedRequest(result),
		&inventory.Record{Owner: inventory.OwnerOperator, Name: "podinfo", Namespace: "demo"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "local bytes")
}
