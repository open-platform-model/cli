# Module Import Experiment — Document Index

Quick navigation for all experiment files.

## Documentation

| File | Purpose | Audience |
|------|---------|----------|
| **[FINDINGS.md](FINDINGS.md)** | Final results and recommendations | Everyone — start here |
| **[README.md](README.md)** | Complete experiment documentation | Developers implementing the solution |
| **[SUMMARY.md](SUMMARY.md)** | Executive summary | Decision makers |
| **[VISUAL_SUMMARY.md](VISUAL_SUMMARY.md)** | Diagrams and visual explanations | Visual learners |
| **[IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md)** | Step-by-step how-to | Module authors & CLI developers |

## Code

| File | Purpose |
|------|---------|
| **[module_import_test.go](module_import_test.go)** | Test implementation (all tests pass) |
| **[go.mod](go.mod)** | Go module dependencies |

## Test Data

| Directory | Purpose |
|-----------|---------|
| **[testdata/simple_module/](testdata/simple_module/)** | Module WITHOUT values.cue (works ✓) |
| **[testdata/module_with_values/](testdata/module_with_values/)** | Module WITH values.cue (breaks importability) |

## Quick Links

- **What's the solution?** → [FINDINGS.md](FINDINGS.md#the-solution-ifdev)
- **How do I implement it?** → [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md#module-author-checklist)
- **Why `@if(dev)`?** → [README.md](README.md#option-d-ifdev-build-tag-recommended)
- **What were the alternatives?** → [README.md](README.md#alternative-approaches-to-valuescue-exclusion)
- **See the tests** → [module_import_test.go](module_import_test.go)

## TL;DR

**Question**: Can OPM modules be flattened AND importable?  
**Answer**: YES — use `@if(dev)` on `values.cue`

```cue
@if(dev)

package jellyfin

values: { ... }
```

All tests pass. Production-ready.
