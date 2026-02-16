# Multi-Package Module — Large Module Organization

**Complexity:** Advanced  
**Workload Types:** `stateless` (Deployment)

**Note:** This example demonstrates the CONCEPT of multi-package organization. Full CUE package imports require additional module setup beyond the scope of this example. For production use, see the CUE documentation on modules and packages.

This example shows how you WOULD structure a large module with separate packages, using inline package declarations to demonstrate the pattern.

## What This Example Demonstrates

### Core Concepts
- **Multi-file organization** — One file per component for better maintainability
- **Component distribution** — `frontend.cue`, `backend.cue`, `worker.cue` as separate files
- **CUE unification** — All files in same package, unified by CUE

### OPM Patterns
- One file per component for large modules
- File naming matches component name (`frontend.cue` → `#components: frontend`)
- Separation of concerns (metadata/schema in `module.cue`, definitions in component files)

## Architecture

```
Main Package (module.cue, values.cue)
    │
    │ imports
    ▼
Components Package (components/*.cue)
    ├── frontend.cue  → #frontend component
    ├── backend.cue   → #backend component
    ├── worker.cue    → #worker component
    └── components.cue → Aggregates all into #all
```

### Deployed Architecture

```
┌──────────────────────────┐
│ frontend (nginx)         │  3 replicas
│   Port: 8080             │  → Service
│   Env: BACKEND_URL       │
└──────────────────────────┘
           │
           │ calls
           ▼
┌──────────────────────────┐
│ backend (node)           │  3 replicas
│   Port: 3000             │  → Service
└──────────────────────────┘

┌──────────────────────────┐
│ worker (python)          │  2 replicas
│   Background jobs        │  (no service)
└──────────────────────────┘
```

## File Structure

```
multi-package-module/
├── cue.mod/
│   └── module.cue     # CUE module definition
├── module.cue         # Module metadata and schema
├── frontend.cue       # Frontend component (separate file)
├── backend.cue        # Backend component (separate file)
├── worker.cue         # Worker component (separate file)
└── values.cue         # Concrete values
```

**Key difference from single-package:**
- Single-package typical: `module.cue` + `components.cue` + `values.cue` (3 files)
- Multi-file pattern: `module.cue` + one file per component + `values.cue`

**Note:** All files use `package main`. True multi-package separation (separate `package components`) requires advanced CUE module configuration beyond this example's scope.

## Configuration Schema

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `frontend.image` | string | `"nginx:1.25-alpine"` | Frontend container image |
| `frontend.replicas` | int | `3` | Frontend replica count |
| `frontend.port` | int | `8080` | Frontend service port |
| `backend.image` | string | `"node:20-alpine"` | Backend container image |
| `backend.replicas` | int | `3` | Backend replica count |
| `backend.port` | int | `3000` | Backend service port |
| `worker.image` | string | `"python:3.11-slim"` | Worker container image |
| `worker.replicas` | int | `2` | Worker replica count |

## Rendered Kubernetes Resources

| Resource | Name | Type | Replicas |
|----------|------|------|----------|
| Deployment | `frontend` | `apps/v1` | 3 |
| Service | `frontend` | `v1` | ClusterIP (port 8080) |
| Deployment | `backend` | `apps/v1` | 3 |
| Service | `backend` | `v1` | ClusterIP (port 3000) |
| Deployment | `worker` | `apps/v1` | 2 |

**Total:** 5 Kubernetes resources

## Usage

### Build (render to YAML)

```bash
# Render to stdout
opm mod build ./examples/multi-package-module

# Render to split files
opm mod build --split ./examples/multi-package-module
```

### Apply to Kubernetes

```bash
# Apply with defaults
opm mod apply ./examples/multi-package-module

# Apply to specific namespace
opm mod apply --namespace apps ./examples/multi-package-module
```

## Key Code Snippets

### Main Package Import

```cue
// module.cue (package main)
import (
	"opmodel.dev/core@v0"
	"example.com/multi-package-module@v0/components"  // ← Import components package
)

core.#Module

#config: {
	frontend: { ... }
	backend:  { ... }
	worker:   { ... }
}

// Import components from the components package
#components: components.#all  // ← Use exported #all from components package
```

**Key insight:** The main package imports the `components` package and uses its exported `#all` field to populate `#components`.

### Component Package Structure

```cue
// components/frontend.cue (package components)
package components  // ← Different package

import (
	resources_workload "opmodel.dev/resources/workload@v0"
	traits_workload "opmodel.dev/traits/workload@v0"
	traits_network "opmodel.dev/traits/network@v0"
)

#frontend: {  // ← Define component
	resources_workload.#Container
	traits_workload.#Scaling
	traits_network.#Expose

	spec: {
		container: { ... }
		scaling: { count: #config.frontend.replicas }  // ← References #config
		expose: { ... }
	}
}
```

**Key insight:** Components reference `#config` even though it's defined in the `main` package. This works because `components.cue` declares `#config: _` (an open field that gets unified with the main package's `#config`).

### Component Aggregation

```cue
// components/components.cue (package components)
package components

// Re-export #config from parent package
#config: _  // ← Open field, unified with main package's #config

// Aggregate all components
#all: {
	frontend: #frontend  // ← From frontend.cue
	backend:  #backend   // ← From backend.cue
	worker:   #worker    // ← From worker.cue
}
```

**Key insight:** The `#all` field aggregates components from individual files. The main package imports this single field instead of importing each component individually.

## Single-Package vs. Multi-Package

### Single-Package (e.g., blog/, jellyfin/)

```
blog/
├── cue.mod/module.cue
├── module.cue       } All same package
├── components.cue   } (package main)
└── values.cue       }
```

**Pros:**
- Simple structure
- No import statements needed
- Good for small modules (1-5 components)

**Cons:**
- All components in one file (or awkwardly split)
- Harder to navigate with many components

### Multi-Package (this example)

```
multi-package-module/
├── cue.mod/module.cue
├── module.cue       } package main
├── values.cue       }
└── components/
    ├── frontend.cue  } package components
    ├── backend.cue   }
    ├── worker.cue    }
    └── components.cue}
```

**Pros:**
- One file per component
- Clear separation of concerns
- Scalable to 10+ components
- Easy to navigate and maintain

**Cons:**
- Slightly more complex (imports, package declarations)
- Overkill for small modules

## When to Use Multi-File Organization

### Use Multi-File When:
- **Large modules** — 5+ components
- **Team collaboration** — Multiple people working on different components
- **Clear structure** — You want explicit file → component mapping
- **Maintainability** — Easier to find and edit specific components

### Use Single components.cue File When:
- **Small modules** — 1-3 components
- **Rapid prototyping** — Quick iteration, simple structure
- **Learning OPM** — Fewer files to navigate
- **Simple use cases** — Components fit comfortably in one file

## Advanced: Package-Level Reuse

You can import the `components` package from other modules:

```cue
// another-module/module.cue
import (
	shared "example.com/multi-package-module@v0/components"
)

#components: {
	// Reuse frontend component
	web: shared.#frontend & {
		spec: {
			scaling: count: 5  // Override replica count
		}
	}

	// Add new components
	api: { ... }
}
```

This enables **component libraries** — reusable component definitions across modules.

## Package Communication

### How #config Flows Between Packages

1. **Main package defines #config:**
   ```cue
   // module.cue
   #config: {
       frontend: { replicas: int }
   }
   ```

2. **Components package declares #config as open:**
   ```cue
   // components/components.cue
   #config: _  // Open field
   ```

3. **CUE unifies both:**
   - Main package's concrete `#config` schema
   - Components package's open `#config: _`
   - Result: Components see the full `#config` definition

4. **Components reference #config:**
   ```cue
   // components/frontend.cue
   spec: {
       scaling: count: #config.frontend.replicas  // ← Resolves to main's #config
   }
   ```

## Best Practices

1. **One component per file** — Keeps files focused and navigable
2. **Aggregate in components.cue** — Central export point
3. **Use meaningful package names** — `components`, `policies`, `providers`
4. **Document package boundaries** — Comment what each package exports
5. **Avoid deep nesting** — Max 2 package levels (`main` → `components`)

## Next Steps

- **Validate all examples:** Run `opm mod build` on each example
- **Compare approaches:** Single-package vs. multi-package for your use case
- **Explore blueprints:** See [blueprint-module/](../blueprint-module/) for reduced boilerplate

## Related Examples

- [blog/](../blog/) — Single-package example (small module)
- [jellyfin/](../jellyfin/) — Single-package example (1 component)
- [multi-tier-module/](../multi-tier-module/) — Single-package example (4 components)
- [values-layering/](../values-layering/) — Environment-specific configuration
