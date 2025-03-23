#!/bin/bash

set -ex

echo "Building LaQueue CLI..."

# Check if go is installed
which go
go version

# Create output directory
mkdir -p bin

# Check the directory
ls -la bin/

# Build for the current platform
echo "Building for $(go env GOOS)/$(go env GOARCH)..."
echo "Building from $(pwd)"

# Verify the source files exist
ls -la cmd/laqueue/

# Build with verbose output
go build -v -o bin/laqueue cmd/laqueue/main.go

echo "Build complete! Binary available at bin/laqueue"
ls -la bin/ 