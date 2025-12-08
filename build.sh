#!/bin/bash

# Build script for TRMNL virtual display
# Automatically handles code signing on macOS

set -e

echo "Building trmnl-go..."
go build -o trmnl-go

# Code sign on macOS (required for macOS 15.x+)
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Code signing for macOS..."
    codesign --force --deep --sign - trmnl-go
    echo "✅ Build complete and signed!"
else
    echo "✅ Build complete!"
fi

echo ""
echo "Run with: ./trmnl-go -api-key YOUR_API_KEY"
