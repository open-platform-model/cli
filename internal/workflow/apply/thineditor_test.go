package apply

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
	"github.com/open-platform-model/library/opm/module"
)

func operatorOwnedRequest(result *workflowrender.Result) Request {
	return Request{Result: result, Log: output.InstanceLogger("test")}
}

// A local render describes bytes the operator cannot fetch. Refused before any
// write, so the failure costs nothing and leaves the CR untouched.
func TestThinEditor_RefusesLocalSourceModule(t *testing.T) {
	result := &workflowrender.Result{
		Instance:    module.InstanceMetadata{Name: "podinfo", Namespace: "demo"},
		Module:      module.ModuleMetadata{Name: "podinfo"},
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
		Instance: module.InstanceMetadata{Name: "podinfo", Namespace: "demo"},
		Module:   module.ModuleMetadata{},
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
		Instance: module.InstanceMetadata{Name: "podinfo", Namespace: "demo"},
		Module: module.ModuleMetadata{
			ModulePath: "testing.opmodel.dev/modules/cli/podinfo@v0", Name: "podinfo", Version: "0.1.4",
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
		Instance:    module.InstanceMetadata{Name: "podinfo", Namespace: "demo"},
		Module:      module.ModuleMetadata{Name: "podinfo"},
		SourceLocal: true,
	}

	err := previewThinEditor(operatorOwnedRequest(result),
		&inventory.Record{Owner: inventory.OwnerOperator, Name: "podinfo", Namespace: "demo"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "local bytes")
}

// The operator renders an operator-managed instance and never skips, so
// --skip-unprovided is refused as a validation error before any spec write,
// on the real edit and on the dry-run preview alike. The request carries no
// client: reaching a write would panic.
func TestThinEditor_RefusesSkipUnprovided(t *testing.T) {
	result := &workflowrender.Result{
		Instance: module.InstanceMetadata{Name: "hello", Namespace: "demo"},
		Module: module.ModuleMetadata{
			ModulePath: "testing.opmodel.dev/modules/cli/podinfo@v0", Name: "podinfo", Version: "0.1.4",
		},
	}
	req := operatorOwnedRequest(result)
	req.Options.SkipUnprovided = true
	rec := &inventory.Record{Owner: inventory.OwnerOperator, Name: "hello", Namespace: "demo"}

	for name, run := range map[string]func() error{
		"apply":   func() error { return executeThinEditor(context.Background(), req, rec) },
		"dry-run": func() error { return previewThinEditor(req, rec) },
	} {
		t.Run(name, func(t *testing.T) {
			err := run()
			require.Error(t, err)
			var exitErr *opmexit.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
			assert.Equal(t, `--skip-unprovided has no effect on instance "hello": the opm-operator renders it and does not skip provider-fulfilled contracts. Install a provider for the contract instead`, err.Error())
		})
	}
}

// The operator decides what an operator-managed instance prunes, so
// --delete-data only draws a warning there, on the real edit and on the
// dry-run preview, and without the flag nothing mentions it. The local-source
// refusal ends both runs before any write.
func TestThinEditor_DeleteDataWarnsOnOperatorManaged(t *testing.T) {
	result := &workflowrender.Result{
		Instance:    module.InstanceMetadata{Name: "podinfo", Namespace: "demo"},
		Module:      module.ModuleMetadata{Name: "podinfo"},
		SourceLocal: true,
	}
	rec := &inventory.Record{Owner: inventory.OwnerOperator, Name: "podinfo", Namespace: "demo"}

	for _, deleteData := range []bool{true, false} {
		for name, run := range map[string]func(Request) error{
			"apply":   func(req Request) error { return executeThinEditor(context.Background(), req, rec) },
			"dry-run": func(req Request) error { return previewThinEditor(req, rec) },
		} {
			// The request's logger binds the log writer when it is built.
			logBuf := captureLog(t)
			req := operatorOwnedRequest(result)
			req.Options.DeleteData = deleteData
			require.Error(t, run(req), name)
			if deleteData {
				line := logLine(logBuf.String(), DeleteDataOperatorManagedNote)
				require.NotEmpty(t, line, "%s: %s", name, logBuf.String())
				assert.Contains(t, line, "WARN", name)
				continue
			}
			assert.NotContains(t, logBuf.String(), "--delete-data", name)
		}
	}
}

// The note must name the setting the operator obeys. "has no effect" alone
// sent the user nowhere.
func TestDeleteDataOperatorManagedNote_NamesTheDataPolicy(t *testing.T) {
	assert.Contains(t, DeleteDataOperatorManagedNote, "--delete-data does not change what the operator does")
	assert.Contains(t, DeleteDataOperatorManagedNote, "spec.dataPolicy")
	assert.Contains(t, DeleteDataOperatorManagedNote, "PersistentVolumeClaims")
	// No released operator had the field when this was written: the note
	// must not promise that every operator keeps claims.
	assert.Contains(t, DeleteDataOperatorManagedNote, "an older operator deletes them")
}
