# Blueprint Module - Important Note

**Status:** Conceptual Example

This example demonstrates how to use OPM Blueprints, but requires the `opmodel.dev/blueprints@v0` package to be published to the OPM registry.

## Current Limitation

The blueprints catalog package is not yet published. You can view the blueprint definitions in the catalog source code at:

```
/path/to/catalog/v0/blueprints/
├── workload/
│   ├── stateless_workload.cue
│   ├── stateful_workload.cue
│   └── ...
└── data/
    └── simple_database.cue
```

## To Use This Example

Once the blueprints package is published, this example will work as-is. Until then, you can:

1. **Reference manual composition examples** — See `blog/`, `jellyfin/`, `multi-tier-module/` for working examples
2. **Copy blueprint definitions locally** — Copy the blueprint `.cue` files into your module (not recommended for production)
3. **Wait for catalog publication** — The OPM team is working on publishing the full catalog

## What This Example Would Demonstrate

When functional, this example shows:
- Using `#StatelessWorkload` blueprint (40% less code than manual composition)
- Using `#SimpleDatabase` blueprint (auto-generates DB config from engine type)
- Combining blueprints with additional traits (`#Expose` not in blueprint)

See the README.md for full documentation of the blueprint pattern.
