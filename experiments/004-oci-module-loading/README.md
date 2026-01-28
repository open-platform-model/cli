# Experiment 004: OCI Module Loading

> **📖 Important:** Read [QUICKREF.md](QUICKREF.md) to understand why some commands succeed without a registry!

## Goal

Validate loading OPM modules and providers from:
1. **CUE-based config** at `~/.opm/config.cue` (simulated via `--home` flag)
2. **OCI registries** via CUE's native module system with precedence chain
3. **File path overlays** (multiple paths for dev workflows, e.g., `--path catalog/v0`)

This experiment establishes the foundation for Phase 1 & 2 of the render pipeline with proper module resolution and config-driven provider management.

## 📚 Documentation

- **[QUICKREF.md](QUICKREF.md)** - Quick reference for registry behavior (START HERE!)
- **[TESTING.md](TESTING.md)** - Comprehensive testing guide with scenarios
- **[SUMMARY.md](SUMMARY.md)** - Implementation findings and results
- **[COMPARISON.md](COMPARISON.md)** - Comparison with experiments 001-003
- **[PLAN.md](PLAN.md)** - Original implementation plan

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Config format | CUE module | Fully typed, references providers module, validated at load time |
| Registry precedence | `--registry` > `OPM_REGISTRY` env > `config.registry` | Standard precedence: flag > env > config file |
| Provider resolution | CUE imports from config | Config references `opm.dev/providers@v0`, fails fast if unreachable |
| Path loading | Multi-path overlay | Supports dev workflows where modules are split across directories |
| Registry setup | Local Docker registry | Self-contained testing, no external dependencies |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        004-oci-module-loading                               │
├─────────────────────────────────────────────────────────────────────────────┤
│  Module Source Resolution                                                   │
│  ├─ OCI Registry (via CUE_REGISTRY / modconfig.NewRegistry)                 │
│  │   └─ opm.dev/providers/kubernetes@v1 → fetched from registry             │
│  │   └─ opm.dev/core@v0 → fetched from registry                            │
│  ├─ Local Path Overlay (via load.Config.Overlay)                            │
│  │   └─ --path catalog/v0 → overlays registry modules                      │
│  │   └─ --path ./local-overrides → dev-time overrides                      │
│  └─ Combined: Local paths take precedence over registry                    │
├─────────────────────────────────────────────────────────────────────────────┤
│  Test Module Definition (module.cue)                                        │
│  ├─ import "opm.dev/core@v0"                                               │
│  ├─ import "opm.dev/providers/kubernetes@v1"                               │
│  ├─ import "opm.dev/resources/workload@v0"                                 │
│  └─ #Module with concrete components                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  Output: Loaded cue.Value ready for Phase 3 (Matching)                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Config-Driven Test (Requires Registry)

```bash
# Start local registry and publish modules
./scripts/setup.sh
export OPM_REGISTRY=localhost:5001
./scripts/publish-modules.sh

# Test with simulated config (default home: testdata/.opm)
go run . --module testdata/simple-app

# Expected output:
# [config] Loading config from testdata/.opm
# [config] Config registry: localhost:5001
# [config] Loaded providers: kubernetes
# Registry: localhost:5001 (from config)
# ✓ Module loaded successfully
# Config Providers: kubernetes
```

**Config Structure:**
The simulated config at `testdata/.opm/config.cue` references `opm.dev/providers@v0` and validates provider access at config load time. If the registry is unreachable, config loading fails fast.

### 2. Basic Test (No Registry, No Config)

```bash
# Test with self-contained module (no external imports, no config)
go run . --module testdata/no-deps-app --home ""

# Note: This succeeds without registry because no-deps-app has no deps
# and we disable config loading with --home ""
```

### 3. Registry Precedence Chain

Test the precedence: `--registry` flag > `OPM_REGISTRY` env > `config.registry`

```bash
# Use config registry (localhost:5001)
go run . --module testdata/simple-app

# Override with env var
OPM_REGISTRY=localhost:9999 go run . --module testdata/simple-app
# Registry: localhost:9999 (from OPM_REGISTRY env)

# Override with flag (highest priority)
go run . --registry localhost:8888 --module testdata/simple-app
# Registry: localhost:8888 (from flag)
```

### 4. Path Overlay (Development)

**Note:** Path overlays add files to the loader but don't register modules for import resolution. For dev workflows with local catalog modules, use the registry + publish approach.

```bash
# Overlay adds .cue files but imports still resolve via registry
go run . --path ../../../catalog/v0 -v
```

### 5. Custom Config Home

```bash
# Use a different config directory
go run . --home ./custom-config --module testdata/simple-app
```

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--home` | Config home directory | `testdata/.opm` |
| `--registry` | OCI registry URL override (highest priority) | None |
| `--path` | Local path overlay (repeatable) | None |
| `--module` | Module directory | `testdata/simple-app` |
| `-v, --verbose` | Show loading details | `false` |
| `--dump` | Dump loaded module structure | `false` |

### Registry Resolution Precedence

1. `--registry` flag (highest)
2. `OPM_REGISTRY` environment variable
3. `config.registry` from config.cue
4. `CUE_REGISTRY` environment variable (fallback)

## Success Criteria

| Criteria | Status | Validation | Registry Required? |
|----------|--------|------------|-------------------|
| Basic module loading (no deps) | ✓ | `go run . --module testdata/no-deps-app` | No |
| Registry client creation | ✓ | `modconfig.NewRegistry()` succeeds | No |
| Registry connection (with deps) | ⚠️ | `go run . --module testdata/simple-app` | **Yes** |
| Path overlay file loading | ✓ | Files added to `load.Config.Overlay` | No |
| CUE value extraction | ✓ | Metadata extracted from loaded module | No |
| Error handling | ✓ | Clear errors for missing registry/paths | Varies |

**⚠️ Important:**  
- **no-deps-app** succeeds without registry (no imports to resolve)
- **simple-app** requires registry (has `import "opm.dev/..."` statements)
- Registry client is always created but only used when CUE needs to fetch dependencies

**Note on Import Resolution:**  
Path overlays add files but don't fully replace registry resolution for `import` statements. For true local-only dev workflows, use:
1. Local registry + publish modules (recommended)
2. CUE's module replacement features (future work)

## File Structure

```
004-oci-module-loading/
├── README.md                    # This file
├── PLAN.md                      # Implementation plan
├── SUMMARY.md                   # Findings & results
├── COMPARISON.md                # vs experiments 001-003
├── TESTING.md                   # ⭐ Testing guide (explains registry behavior)
├── docker-compose.yml           # Local OCI registry
├── scripts/
│   ├── setup.sh                 # Start registry + setup
│   └── publish-modules.sh       # Publish catalog to registry
├── loader/
│   ├── config.go                # ⭐ CUE config loader
│   ├── registry.go              # Registry-aware loader with precedence
│   ├── overlay.go               # Path overlay support
│   └── types.go                 # Types and interfaces
├── testdata/
│   ├── .opm/                    # ⭐ Simulated config home
│   │   ├── cue.mod/module.cue   # Config module with providers dep
│   │   └── config.cue           # Config with registry + providers
│   ├── no-deps-app/             # No imports (works without registry)
│   │   ├── cue.mod/module.cue
│   │   └── app.cue
│   └── simple-app/              # Has imports (requires registry)
│       ├── cue.mod/module.cue
│       └── app.cue
├── main.go                      # Experiment CLI
└── go.mod                       # Go module
```

## Key Implementation Details

### Registry Client

Uses `cuelang.org/go/mod/modconfig` for registry access:

```go
reg, err := modconfig.NewRegistry(&modconfig.Config{
    Hosts: func(host string) modconfig.Host {
        if strings.HasPrefix(host, "localhost") {
            return modconfig.Host{Insecure: true}
        }
        return modconfig.Host{}
    },
})
```

### Path Overlay

Uses CUE's `load.Config.Overlay` to inject local files:

```go
overlay := make(map[string]load.Source)
// Walk path, add .cue files to overlay
loadCfg := &load.Config{
    Registry: reg,
    Overlay:  overlay,
}
```

## Spec Requirements Addressed

| Requirement | Status | Implementation |
|-------------|--------|----------------|
| FR-015 (002-cli-spec) OPM_REGISTRY | ✓ | `modconfig.NewRegistry()` respects env var |
| FR-016 (002-cli-spec) Registry fail-fast | ✓ | Clear error on connectivity failure |
| Phase 1 (013-cli-render-spec) Module Loading | ✓ | Loader returns validated cue.Value |
| Phase 2 (013-cli-render-spec) Provider Loading | ✓ | Provider accessible via CUE import |

## Next Steps

After validating this experiment:

1. **Integration with 003-hybrid-render**: Replace `pkg/` local definitions with registry imports
2. **CLI `mod build` integration**: Add `--path` overlay flag to production CLI
3. **Cache management**: Implement module cache invalidation for dev workflows
