# Render Pipeline Test Module

This directory contains a test module for validating the hybrid Go+CUE render pipeline.

## Structure

- `core_defs.cue` - Minimal core OPM definitions (inlined for testing)
- `provider.cue` - Simple Kubernetes provider with one transformer
- `module.cue` - Test module with multiple components
- `cue.mod/` - CUE module configuration

## Test Cases

### Provider
- **Name**: test-kubernetes
- **Version**: 0.1.0
- **Transformers**: 
  - `deployment` - Transforms stateless workloads to Kubernetes Deployments

### Components
- `web` - Stateless workload (matches deployment transformer)
- `api` - Stateless workload (matches deployment transformer)
- `worker` - Stateless workload (matches deployment transformer)
- `database` - Stateful workload (does NOT match - tests error handling)

## Usage

### Build the module
```bash
cd cli/testdata/render-test
opm mod build
```

### Verbose output
```bash
opm mod build --verbose
```

### Different output formats
```bash
# YAML to stdout (default)
opm mod build

# YAML to file
opm mod build --output-file output.yaml

# JSON
opm mod build -o json

# Directory (separate files)
opm mod build -o dir --output-dir manifests/
```

## Expected Results

**Successful renders**: 3 components (web, api, worker)
- Each generates a Kubernetes Deployment
- Labels include OPM tracking labels
- Resources sorted alphabetically (api, web, worker)

**Excluded**: 1 component (database)
- Has `stateful` label, doesn't match `stateless` transformer
- Demonstrates proper matching logic

## Validation

The rendered YAML can be validated with kubectl:
```bash
opm mod build | kubectl apply --dry-run=client -f -
```

## Performance Characteristics

- **Total render time**: ~4ms for 3 components
- **Parallel execution**: ~177µs for 3 workers
- **Per-worker time**: ~150µs average
- **Overhead**: ~3.8ms (loading, matching, aggregation)

## Testing Features

This test validates:
- ✅ 5-phase hybrid pipeline
- ✅ CUE-computed matching via `#Matches`
- ✅ Parallel worker execution
- ✅ AST transport for thread safety
- ✅ TransformerContext injection with hidden fields
- ✅ Deterministic output (FR-023)
- ✅ Fail-on-end error aggregation (FR-024)
- ✅ Multiple output formats (YAML, JSON, dir)
- ✅ OPM tracking labels
- ✅ Source comments in output
