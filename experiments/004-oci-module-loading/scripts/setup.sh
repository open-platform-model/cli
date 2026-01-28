#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXPERIMENT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== OPM Experiment 004: OCI Module Loading Setup ==="
echo ""

# Check dependencies
echo "[1/3] Checking dependencies..."
command -v docker >/dev/null 2>&1 || { echo "Error: docker is required but not installed"; exit 1; }
command -v docker compose >/dev/null 2>&1 || { echo "Error: docker compose is required but not installed"; exit 1; }
command -v cue >/dev/null 2>&1 || { echo "Error: cue is required but not installed"; exit 1; }

echo "  ✓ docker"
echo "  ✓ docker compose"
echo "  ✓ cue"
echo ""

# Start registry
echo "[2/3] Starting local OCI registry..."
cd "$EXPERIMENT_DIR"
docker compose up -d

# Wait for registry to be ready
echo "  Waiting for registry to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:5001/v2/ >/dev/null 2>&1; then
        echo "  ✓ Registry is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "  ✗ Registry failed to start"
        docker compose logs
        exit 1
    fi
    sleep 1
done
echo ""

# Set environment variable
echo "[3/3] Setting CUE_REGISTRY environment variable..."
export CUE_REGISTRY="localhost:5001/cuemodules"
echo "  export CUE_REGISTRY=$CUE_REGISTRY"
echo ""

echo "=== Setup Complete ==="
echo ""
echo "Next steps:"
echo "  1. Run: export CUE_REGISTRY=localhost:5001/cuemodules"
echo "  2. Publish modules: ./scripts/publish-modules.sh"
echo "  3. Run experiment: go run ."
echo ""
echo "To stop registry: docker compose down"
