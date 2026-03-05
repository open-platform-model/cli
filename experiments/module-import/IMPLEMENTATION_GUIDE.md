# Implementation Guide: `@if(dev)` for Importable Modules

Quick reference for converting OPM modules to use the `@if(dev)` pattern.

## Module Author Checklist

### 1. Change package declaration

```diff
- package main
+ package jellyfin  // or whatever your module name is
```

### 2. Add `@if(dev)` to values.cue

```diff
+ @if(dev)
+
  package jellyfin
  
  values: {
      image: { tag: "latest" }
      replicas: 1
  }
```

**Important**: The `@if(dev)` line must be the **first line** of the file, before the package clause.

### 3. Ensure #config has defaults

For any field in `values.cue`, make sure `#config` has a default:

```cue
#config: {
    replicas: int | *1        // ← default if values.cue not loaded
    image: {
        tag: string | *"latest"
    }
}
```

This ensures the module works even when `values.cue` is excluded (on import).

### 4. Test locally

```bash
cd your-module
cue eval -t dev .   # Should include values.cue
cue eval .          # Should exclude values.cue
```

### 5. Publish

When published to a CUE registry, the `@if(dev)` tag ensures `values.cue` is excluded when the module is imported as a dependency.

---

## CLI Implementation Checklist

### 1. Pass `-t dev` when loading local modules

```go
// When loading a module from the local filesystem
instances := load.Instances([]string{"."}, &load.Config{
    Dir:  modulePath,
    Tags: []string{"dev"},  // ← Enable dev tag
})
```

### 2. Don't pass tags when loading imported modules

```go
// When loading a module as a dependency
instances := load.Instances([]string{"."}, &load.Config{
    Dir: modulePath,
    // No Tags field → @if(dev) files excluded
})
```

### 3. Update module templates

The `opm mod init` command should generate `values.cue` with `@if(dev)` already included:

```cue
@if(dev)

package {{ .PackageName }}

values: {
    // Add your default values here
}
```

---

## Verification

### Test Case 1: Local Load (with tag)

```go
func TestLocalLoadWithTag(t *testing.T) {
    ctx := cuecontext.New()
    
    instances := load.Instances([]string{"."}, &load.Config{
        Dir:  "testdata/my-module",
        Tags: []string{"dev"},
    })
    
    val := ctx.BuildInstance(instances[0])
    
    // values.cue should be included
    values := val.LookupPath(cue.ParsePath("values"))
    assert.True(t, values.Exists())
}
```

### Test Case 2: Import Load (no tag)

```go
func TestImportLoadWithoutTag(t *testing.T) {
    ctx := cuecontext.New()
    
    instances := load.Instances([]string{"."}, &load.Config{
        Dir: "testdata/my-module",
        // No Tags → @if(dev) not active
    })
    
    val := ctx.BuildInstance(instances[0])
    
    // values.cue should be excluded
    values := val.LookupPath(cue.ParsePath("values"))
    assert.False(t, values.Exists())
    
    // Module should still unify with #Module
    moduleDef := catalogVal.LookupPath(cue.ParsePath("#Module"))
    unified := moduleDef.Unify(val)
    assert.NoError(t, unified.Err())
}
```

### Test Case 3: #ModuleRelease Integration

```go
func TestModuleReleaseIntegration(t *testing.T) {
    // Load module WITHOUT tag (simulates import)
    moduleVal := loadModule(t, "testdata/my-module", nil)
    
    // Create release
    release := releaseSchema
    release = release.FillPath(cue.ParsePath("#module"), moduleVal)
    release = release.FillPath(cue.ParsePath("values"), userValues)
    
    assert.NoError(t, release.Err())
    
    // Components should resolve with user values
    components := release.LookupPath(cue.ParsePath("components"))
    assert.True(t, components.Exists())
}
```

---

## Migration Path

For existing modules:

1. **Phase 1**: Add `@if(dev)` to values.cue, keep current CLI behavior
2. **Phase 2**: Update CLI to pass `-t dev` for local loads
3. **Phase 3**: Convert modules to named packages (`package <name>`)
4. **Phase 4**: Publish to registry, test imports

---

## Common Issues

### Issue 1: Forgot `@if(dev)`

**Symptom**: Import fails with `#Module.values: field not allowed`

**Fix**: Add `@if(dev)` as first line of `values.cue`

### Issue 2: `@if(dev)` in wrong position

**Symptom**: CUE parse error

**Fix**: Ensure `@if(dev)` is **before** the package clause:

```cue
@if(dev)       ← Must be first

package foo    ← Package clause second
```

### Issue 3: Module works locally but not on import

**Symptom**: Module fails validation when imported

**Fix**: Ensure `#config` has defaults for all fields in `values.cue`:

```cue
#config: {
    field: type | *defaultValue
}
```

### Issue 4: CLI not passing tag

**Symptom**: `values.cue` not loaded even locally

**Fix**: Check `load.Config.Tags` includes `"dev"`:

```go
&load.Config{
    Dir:  path,
    Tags: []string{"dev"},  // ← Required
}
```

---

## References

- CUE build tags: https://cuelang.org/docs/reference/command/cue-help-injection/
- Module import experiment: `experiments/module-import/`
- Test implementation: `experiments/module-import/module_import_test.go`
