# Research: Go CUE SDK Validation Patterns

**Source**: Timoni v0.x codebase (`timoni mod vet` implementation)  
**Purpose**: Extract Go CUE SDK patterns for OPM validation without external `cue` binary

## Key Imports

```go
import (
    "cuelang.org/go/cue"
    "cuelang.org/go/cue/cuecontext"
    "cuelang.org/go/cue/load"
    "cuelang.org/go/cue/errors"
)
```

## Core Patterns

### 1. Creating a CUE Context

Always create a fresh context per operation to avoid memory bloat:

```go
ctx := cuecontext.New()
```

### 2. Loading CUE Instances

Use `load.Config` to configure how CUE loads modules:

```go
cfg := &load.Config{
    ModuleRoot: moduleRoot,    // Path to cue.mod parent directory
    Package:    pkgName,       // Package name (default: "main")
    Dir:        moduleRoot,    // Working directory for relative paths
    DataFiles:  true,          // Include data files (.yaml, .json)
}

instances := load.Instances([]string{}, cfg)
if len(instances) == 0 {
    return errors.New("no CUE instances found")
}

inst := instances[0]
if inst.Err != nil {
    return fmt.Errorf("load error: %w", inst.Err)
}
```

### 3. Building Values from Instances

```go
value := ctx.BuildInstance(inst)
if value.Err() != nil {
    return value.Err()
}
```

### 4. Schema Validation

Basic validation (syntax and schema constraints):

```go
if err := value.Validate(); err != nil {
    return err
}
```

### 5. Concrete Validation

Require all values to be fully resolved (no open fields):

```go
if err := value.Validate(cue.Concrete(true), cue.Final()); err != nil {
    return err
}
```

- `cue.Concrete(true)` - requires all values to be concrete (not just types)
- `cue.Final()` - disallows further unification

### 6. Looking Up Paths

Extract specific values from the CUE tree:

```go
// Look up a definition
moduleSchema := value.LookupPath(cue.ParsePath("#Module"))
if moduleSchema.Err() != nil {
    return errors.New("does not conform to #Module schema")
}

// Look up a field
configValue := value.LookupPath(cue.ParsePath("config"))
if !configValue.Exists() {
    return errors.New("config field not found")
}
```

### 7. Iterating Fields

```go
iter, err := value.Fields(cue.Concrete(true))
if err != nil {
    return err
}

for iter.Next() {
    name := iter.Selector().String()
    fieldValue := iter.Value()
    
    if fieldValue.Err() != nil {
        // Handle error
    }
    
    // Process field
}
```

### 8. Checking Value Properties

```go
// Check if value exists
if !value.Exists() { ... }

// Check if null
if value.IsNull() { ... }

// Get underlying Go type
switch value.Kind() {
case cue.StructKind:
    // Handle struct
case cue.ListKind:
    // Handle list
case cue.StringKind:
    str, _ := value.String()
}
```

## Error Handling

### Extracting Error Details

CUE errors contain position information:

```go
import "cuelang.org/go/cue/errors"

func formatCUEError(err error) string {
    var msgs []string
    
    for _, e := range errors.Errors(err) {
        pos := e.Position()
        if pos.IsValid() {
            msgs = append(msgs, fmt.Sprintf("%s:%d:%d: %s",
                pos.Filename(), pos.Line(), pos.Column(), e.Message()))
        } else {
            msgs = append(msgs, e.Message())
        }
    }
    
    return strings.Join(msgs, "\n")
}
```

### Common Error Types

1. **Load errors** (`inst.Err`) - Missing files, invalid imports
2. **Build errors** (`value.Err()`) - Syntax errors, unification conflicts
3. **Validation errors** (`value.Validate()`) - Schema violations, non-concrete values

## Values Merging

To merge multiple values files:

```go
// Load base values
baseValue := ctx.CompileString(`{ foo: "bar" }`)

// Load overlay values
overlayValue := ctx.CompileString(`{ foo: "baz", extra: true }`)

// Unify (merge)
merged := baseValue.Unify(overlayValue)
if merged.Err() != nil {
    return merged.Err()  // Conflict if values disagree
}
```

For files:

```go
// Compile from file bytes
data, _ := os.ReadFile("values.cue")
value := ctx.CompileBytes(data, cue.Filename("values.cue"))
```

## Tag Injection

Inject runtime values via CUE tags:

```go
cfg := &load.Config{
    Tags: []string{
        "name=" + instanceName,
        "namespace=" + namespace,
    },
    TagVars: map[string]load.TagVar{
        "version": {
            Func: func() (ast.Expr, error) {
                return ast.NewString("1.0.0"), nil
            },
        },
    },
}
```

In CUE:

```cue
metadata: {
    name:      string @tag(name)
    namespace: string @tag(namespace)
    version:   string @tag(version, var=version)
}
```

## OPM-Specific Adaptations

### Differences from Timoni

1. **No Kubernetes extraction** - OPM validates CUE definitions only, not K8s objects
2. **Schema targets** - Validate `#Module`, `#Component`, `#Scope` instead of Timoni instance
3. **Optional concrete mode** - Schema validation by default, concrete with `--concrete`
4. **Multi-package support** - Handle `components/`, `scopes/` subdirectories

### Recommended Package Structure

```
cli/internal/validator/
├── module.go       # ModuleValidator
├── config.go       # ConfigValidator
├── errors.go       # Error formatting with Charm
├── result.go       # ValidationResult, ConfigResult
└── validator_test.go
```

## Dependencies

| Package | Purpose | Version |
|---------|---------|---------|
| `cuelang.org/go/cue` | CUE evaluation | v0.11+ |
| `cuelang.org/go/cue/cuecontext` | Context creation | v0.11+ |
| `cuelang.org/go/cue/load` | Module loading | v0.11+ |
| `cuelang.org/go/cue/errors` | Error handling | v0.11+ |
