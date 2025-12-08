#!/bin/bash
set -e

APP_NAME="TRMNL"
VERSION="1.0.0"
BUNDLE_ID="net.semaja2.trmnl.virtualdisplay"
BINARY="trmnl-go"

echo "Building macOS app bundle..."

# Build the binary
echo "Building binary..."
go build -o "$BINARY"

# Code sign
echo "Signing binary..."
codesign --force --deep --sign - "$BINARY"

# Create app bundle structure
APP_DIR="$APP_NAME.app"
rm -rf "$APP_DIR"
mkdir -p "$APP_DIR/Contents/MacOS"
mkdir -p "$APP_DIR/Contents/Resources"

# Copy binary
cp "$BINARY" "$APP_DIR/Contents/MacOS/"

# Create Info.plist
cat > "$APP_DIR/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>$BINARY</string>
    <key>CFBundleIdentifier</key>
    <string>$BUNDLE_ID</string>
    <key>CFBundleName</key>
    <string>$APP_NAME</string>
    <key>CFBundleDisplayName</key>
    <string>TRMNL Virtual Display</string>
    <key>CFBundleVersion</key>
    <string>$VERSION</string>
    <key>CFBundleShortVersionString</key>
    <string>$VERSION</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.utilities</string>
    <key>NSSupportsAutomaticTermination</key>
    <true/>
    <key>NSSupportsSuddenTermination</key>
    <true/>
</dict>
</plist>
EOF

# Create simple icon (you can replace this with a real .icns file)
# For now, just create a placeholder
touch "$APP_DIR/Contents/Resources/AppIcon.icns"

echo "✅ App bundle created: $APP_DIR"
echo ""
echo "To run: open $APP_DIR"
echo "To install: cp -r $APP_DIR /Applications/"
