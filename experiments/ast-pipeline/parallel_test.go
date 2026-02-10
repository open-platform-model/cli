package astpipeline

import (
	"fmt"
	"sync"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Hypothesis 3: Parallel evaluation from shared AST
// ---------------------------------------------------------------------------

// Input CUE (loaded once from testdata/test-module/):
//
//	package testmodule
//	metadata: { name: "test-module", ... }
//	#config:     { ... }
//	#components: { ... }
//
// Concurrency: 10 goroutines, each calling cuecontext.New() + ctx.BuildInstance(inst)
// on the shared build.Instance. Each reads metadata.name → "test-module".
func TestParallel_SharedASTIndependentContexts(t *testing.T) {
	// Load module → get inst. Spawn N goroutines each doing:
	//   cuecontext.New() + ctx.BuildInstance(inst)
	// No panics.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	inst := instances[0]
	const numWorkers = 10

	var wg sync.WaitGroup
	errors := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := cuecontext.New()
			val := ctx.BuildInstance(inst)
			if val.Err() != nil {
				errors <- fmt.Errorf("worker %d: build error: %w", id, val.Err())
				return
			}
			// Read a field to prove the value is usable
			name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
			if err != nil {
				errors <- fmt.Errorf("worker %d: lookup error: %w", id, err)
				return
			}
			if name != "test-module" {
				errors <- fmt.Errorf("worker %d: unexpected name %q", id, name)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// Input CUE (loaded once from testdata/test-module/):
//
//	#config: { image: string, replicas: int & >=1, ... }
//	#components: {
//	    web: {
//	        spec: { container: image: #config.image, replicas: #config.replicas }
//	    }
//	    ...
//	}
//
// Concurrency: 10 goroutines, each filling #config with unique values:
//
//	worker N → #config: { image: "worker-N:latest", replicas: N+1, ... }
//
// Each verifies its own values propagated through #components.web.spec
func TestParallel_FillPathConcurrent(t *testing.T) {
	// Each goroutine does BuildInstance + FillPath on its own Value.
	// This tests the key operation: injecting values concurrently.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	inst := instances[0]
	const numWorkers = 10

	type result struct {
		id       int
		image    string
		replicas int64
		err      error
	}

	results := make(chan result, numWorkers)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := cuecontext.New()
			val := ctx.BuildInstance(inst)
			if val.Err() != nil {
				results <- result{id: id, err: val.Err()}
				return
			}

			// Each worker injects different values into #config
			configVal := ctx.CompileString(fmt.Sprintf(`{
				image: "worker-%d:latest"
				replicas: %d
				port: 8080
				debug: false
			}`, id, id+1))
			if configVal.Err() != nil {
				results <- result{id: id, err: configVal.Err()}
				return
			}

			filled := val.FillPath(cue.ParsePath("#config"), configVal)
			if filled.Err() != nil {
				results <- result{id: id, err: filled.Err()}
				return
			}

			// Read back the value through #components.web.spec.container.image
			// (which references #config.image)
			img, err := filled.LookupPath(cue.ParsePath("#components.web.spec.container.image")).String()
			if err != nil {
				results <- result{id: id, err: fmt.Errorf("lookup image: %w", err)}
				return
			}
			rep, err := filled.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
			if err != nil {
				results <- result{id: id, err: fmt.Errorf("lookup replicas: %w", err)}
				return
			}

			results <- result{id: id, image: img, replicas: rep}
		}(i)
	}

	wg.Wait()
	close(results)

	for r := range results {
		require.NoError(t, r.err, "worker %d", r.id)
		expected := fmt.Sprintf("worker-%d:latest", r.id)
		assert.Equal(t, expected, r.image, "worker %d image", r.id)
		assert.Equal(t, int64(r.id+1), r.replicas, "worker %d replicas", r.id)
	}
}

// Input CUE (loaded once from testdata/test-module/):
//
//	#config:     { image: string, replicas: int & >=1, ... }
//	#components: { web: { spec: { ... } }, ... }
//
// 3 transformer templates (compiled as CUE strings):
//
//	DeploymentTransformer: { #transform: { #component: _, #context: {...}, output: { kind: "Deployment", ... } } }
//	ServiceTransformer:    { #transform: { #component: _, #context: {...}, output: { kind: "Service", ... } } }
//	ConfigMapTransformer:  { #transform: { #component: _, #context: {...}, output: { kind: "ConfigMap", ... } } }
//
// Concurrency: 3 goroutines (one per transformer), each:
//   - builds from shared inst, fills #config, extracts web component
//   - compiles transformer, injects #component + #context via FillPath
//   - extracts output → verifies correct kind
func TestParallel_TransformerSimulation(t *testing.T) {
	// Simulate the executor: shared module, multiple transformer jobs
	// running concurrently with separate contexts.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	inst := instances[0]

	// Define transformer templates (as CUE strings for simplicity)
	transformers := []struct {
		name string
		cue  string
	}{
		{
			name: "DeploymentTransformer",
			cue: `{
				#transform: {
					#component: _
					#context: { name: string, namespace: string, ... }
					output: {
						apiVersion: "apps/v1"
						kind: "Deployment"
						metadata: {
							name: #context.name
							namespace: #context.namespace
						}
						spec: replicas: #component.spec.replicas
					}
				}
			}`,
		},
		{
			name: "ServiceTransformer",
			cue: `{
				#transform: {
					#component: _
					#context: { name: string, namespace: string, ... }
					output: {
						apiVersion: "v1"
						kind: "Service"
						metadata: {
							name: #context.name
							namespace: #context.namespace
						}
						spec: type: "ClusterIP"
					}
				}
			}`,
		},
		{
			name: "ConfigMapTransformer",
			cue: `{
				#transform: {
					#component: _
					#context: { name: string, namespace: string, ... }
					output: {
						apiVersion: "v1"
						kind: "ConfigMap"
						metadata: {
							name: #context.name
							namespace: #context.namespace
						}
					}
				}
			}`,
		},
	}

	type jobResult struct {
		transformer string
		kind        string
		name        string
		err         error
	}

	results := make(chan jobResult, len(transformers))
	var wg sync.WaitGroup

	for _, tf := range transformers {
		wg.Add(1)
		go func(tfName, tfCUE string) {
			defer wg.Done()

			// Each goroutine gets its own context
			ctx := cuecontext.New()

			// Build module from shared instance
			moduleVal := ctx.BuildInstance(inst)
			if moduleVal.Err() != nil {
				results <- jobResult{transformer: tfName, err: moduleVal.Err()}
				return
			}

			// Fill #config to make components concrete
			configVal := ctx.CompileString(`{
				image: "nginx:1.25"
				replicas: 2
				port: 8080
				debug: false
			}`)
			filled := moduleVal.FillPath(cue.ParsePath("#config"), configVal)
			if filled.Err() != nil {
				results <- jobResult{transformer: tfName, err: filled.Err()}
				return
			}

			// Get the web component
			component := filled.LookupPath(cue.ParsePath("#components.web"))
			if !component.Exists() {
				results <- jobResult{transformer: tfName, err: fmt.Errorf("component not found")}
				return
			}

			// Build transformer
			tfVal := ctx.CompileString(tfCUE)
			if tfVal.Err() != nil {
				results <- jobResult{transformer: tfName, err: tfVal.Err()}
				return
			}

			// Inject component and context
			transform := tfVal.LookupPath(cue.ParsePath("#transform"))
			unified := transform.FillPath(cue.ParsePath("#component"), component)
			unified = unified.FillPath(cue.ParsePath("#context.name"), ctx.Encode("web"))
			unified = unified.FillPath(cue.ParsePath("#context.namespace"), ctx.Encode("default"))

			if unified.Err() != nil {
				results <- jobResult{transformer: tfName, err: unified.Err()}
				return
			}

			// Extract output
			output := unified.LookupPath(cue.ParsePath("output"))
			if !output.Exists() {
				results <- jobResult{transformer: tfName, err: fmt.Errorf("no output")}
				return
			}

			var obj map[string]any
			if err := output.Decode(&obj); err != nil {
				results <- jobResult{transformer: tfName, err: err}
				return
			}

			results <- jobResult{
				transformer: tfName,
				kind:        obj["kind"].(string),
				name:        obj["metadata"].(map[string]any)["name"].(string),
			}
		}(tf.name, tf.cue)
	}

	wg.Wait()
	close(results)

	// Collect results
	kinds := make(map[string]string) // transformer -> kind
	for r := range results {
		require.NoError(t, r.err, "transformer %s", r.transformer)
		kinds[r.transformer] = r.kind
		assert.Equal(t, "web", r.name, "all should target 'web' component")
		t.Logf("%s produced %s/%s", r.transformer, r.kind, r.name)
	}

	assert.Equal(t, "Deployment", kinds["DeploymentTransformer"])
	assert.Equal(t, "Service", kinds["ServiceTransformer"])
	assert.Equal(t, "ConfigMap", kinds["ConfigMapTransformer"])
}

// Input CUE (loaded once from testdata/test-module/):
//
//	#config:     { image: string, replicas: int & >=1, ... }
//	#components: { web: { spec: { container: image: #config.image, ... } } }
//
// Runs same job 5 times sequentially, then 5 times in parallel.
// Each job N fills: #config: { image: "img-N:latest", replicas: N+1, ... }
// Verifies: parallel results are identical to sequential results per job ID.
func TestParallel_ResultsMatchSequential(t *testing.T) {
	// Run the same jobs sequentially vs parallel.
	// Assert identical results.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	inst := instances[0]

	runJob := func(ctx *cue.Context, id int) (string, int64, error) {
		val := ctx.BuildInstance(inst)
		if val.Err() != nil {
			return "", 0, val.Err()
		}

		configVal := ctx.CompileString(fmt.Sprintf(`{
			image: "img-%d:latest"
			replicas: %d
			port: 8080
			debug: false
		}`, id, id+1))

		filled := val.FillPath(cue.ParsePath("#config"), configVal)
		if filled.Err() != nil {
			return "", 0, filled.Err()
		}

		img, err := filled.LookupPath(cue.ParsePath("#components.web.spec.container.image")).String()
		if err != nil {
			return "", 0, err
		}
		rep, err := filled.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
		if err != nil {
			return "", 0, err
		}
		return img, rep, nil
	}

	// Sequential
	seqResults := make(map[int][2]any) // id -> [image, replicas]
	for i := 0; i < 5; i++ {
		ctx := cuecontext.New()
		img, rep, err := runJob(ctx, i)
		require.NoError(t, err, "sequential job %d", i)
		seqResults[i] = [2]any{img, rep}
	}

	// Parallel
	parResults := make(map[int][2]any)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := cuecontext.New()
			img, rep, err := runJob(ctx, id)
			require.NoError(t, err, "parallel job %d", id)
			mu.Lock()
			parResults[id] = [2]any{img, rep}
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	// Compare
	for id, seq := range seqResults {
		par, ok := parResults[id]
		require.True(t, ok, "parallel should have result for job %d", id)
		assert.Equal(t, seq, par, "job %d results should match", id)
	}
}

// Input CUE (loaded once from testdata/test-module/):
//
//	module.cue → package testmodule; metadata: { name: "test-module", ... }; ...
//	values.cue → package testmodule; values: { ... }
//
// Fallback approach: format each inst.Files entry to bytes (safe to share),
// then each of 5 goroutines re-parses from bytes → BuildFile → Unify.
// Verifies: metadata.name = "test-module" in each goroutine.
func TestParallel_RebuildFromFiles(t *testing.T) {
	// Fallback path: if BuildInstance can't be shared,
	// re-parse from the AST source in each goroutine.
	// This tests parsing → building independently per goroutine.
	modulePath := testModulePath(t)
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	inst := instances[0]

	// Capture formatted source of each file (bytes are safe to share)
	type fileSource struct {
		filename string
		content  []byte
	}
	var sources []fileSource
	for _, file := range inst.Files {
		b, err := format.Node(file)
		require.NoError(t, err)
		sources = append(sources, fileSource{
			filename: file.Filename,
			content:  b,
		})
	}

	const numWorkers = 5
	var wg sync.WaitGroup
	errors := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Parse files fresh in this goroutine
			var files []*ast.File
			for _, src := range sources {
				f, err := parser.ParseFile(src.filename, src.content, parser.ParseComments)
				if err != nil {
					errors <- fmt.Errorf("worker %d: parse %s: %w", id, src.filename, err)
					return
				}
				files = append(files, f)
			}

			// Build value from parsed files
			ctx := cuecontext.New()

			// BuildFile only takes a single file; for multiple files
			// we need to unify them
			if len(files) == 0 {
				errors <- fmt.Errorf("worker %d: no files", id)
				return
			}

			val := ctx.BuildFile(files[0])
			for _, f := range files[1:] {
				val2 := ctx.BuildFile(f)
				val = val.Unify(val2)
			}

			if val.Err() != nil {
				errors <- fmt.Errorf("worker %d: build error: %w", id, val.Err())
				return
			}

			name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
			if err != nil {
				errors <- fmt.Errorf("worker %d: lookup error: %w", id, err)
				return
			}
			if name != "test-module" {
				errors <- fmt.Errorf("worker %d: unexpected name %q", id, name)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}
