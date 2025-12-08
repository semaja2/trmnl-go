#!/bin/bash

# Code sign all binaries (required on macOS for Go-built apps with CGO/Fyne)
# This fixes "Code Signature Invalid" crashes on macOS 15.x

echo "Signing binaries for macOS..."

for binary in trmnl-go test-window test-simple; do
    if [ -f "$binary" ]; then
        echo "Signing $binary..."
        codesign --force --deep --sign - "$binary"
    fi
done

echo "Done! Binaries are now signed and should run without crashing."
