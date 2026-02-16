#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building plugin-morphemap-go WASM binary..."
GOOS=wasip1 GOARCH=wasm go build -o ../dist/morphemap-go-v1.0.0.wasm ../cmd/plugin/main.go

echo "Build complete: dist/morphemap-go-v1.0.0.wasm"
