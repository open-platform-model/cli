#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXPERIMENT_DIR="$(dirname "$SCRIPT_DIR")"
CATALOG_DIR="$EXPERIMENT_DIR/../../../catalog/v0"

echo "=== Publishing OPM Modules to Local Registry ==="
echo ""

# Check if CUE_REGISTRY is set
if [ -z "${CUE_REGISTRY:-}" ]; then
    echo "Error: CUE_REGISTRY environment variable is not set"
    echo "Run: export CUE_REGISTRY=localhost:5001/cuemodules"
    exit 1
fi

echo "Registry: $CUE_REGISTRY"
echo ""

# Check if catalog exists
if [ ! -d "$CATALOG_DIR" ]; then
    echo "Error: Catalog directory not found at $CATALOG_DIR"
    exit 1
fi

# Publish order (respects dependencies)
MODULES=(
    "core"
    "schemas"
    "resources"
    "traits"
    "policies"
    "providers"
)

for module in "${MODULES[@]}"; do
    MODULE_PATH="$CATALOG_DIR/$module"
    
    if [ ! -d "$MODULE_PATH" ]; then
        echo "⊘ Skipping $module (not found)"
        continue
    fi
    
    echo "Publishing $module..."
    cd "$MODULE_PATH"
    
    # Ensure module is tidy
    if ! cue mod tidy; then
        echo "  ✗ Failed to tidy $module"
        exit 1
    fi
    
    # Publish to registry
    if ! cue mod publish v0.1.0; then
        echo "  ✗ Failed to publish $module"
        exit 1
    fi
    
    echo "  ✓ Published $module@v0.1.0"
    echo ""
done

echo "=== All Modules Published ==="
echo ""
echo "Verify with: curl http://localhost:5001/v2/_catalog"
