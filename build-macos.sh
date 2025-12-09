#!/bin/bash

set -e

# TRMNL-GO macOS Production Build Script
# Builds universal macOS binary (Intel + Apple Silicon)
# Creates proper app bundle with optional code signing and DMG

VERSION="1.0.0"
APP_NAME="TRMNL"
BUNDLE_ID="com.trmnl.virtualdisplay"
BUILD_DIR="build"
DIST_DIR="dist"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
SIGN_APP=false
SIGNING_IDENTITY=""
CREATE_DMG=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --sign)
            SIGN_APP=true
            SIGNING_IDENTITY="$2"
            shift 2
            ;;
        --dmg)
            CREATE_DMG=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--sign \"Developer ID Application: Your Name (TEAMID)\"] [--dmg]"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  TRMNL Virtual Display - macOS Production Build${NC}"
echo -e "${BLUE}  Version: ${VERSION}${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo ""

# Clean previous builds
echo -e "${YELLOW}→${NC} Cleaning previous builds..."
rm -rf "${BUILD_DIR}" "${DIST_DIR}"
mkdir -p "${BUILD_DIR}" "${DIST_DIR}"

# Determine current architecture for macOS builds
CURRENT_ARCH=$(uname -m)
if [ "$CURRENT_ARCH" = "x86_64" ]; then
    NATIVE_ARCH="amd64"
    CROSS_ARCH="arm64"
else
    NATIVE_ARCH="arm64"
    CROSS_ARCH="amd64"
fi

# Build for current macOS architecture (native build with CGO)
echo -e "${YELLOW}→${NC} Building for macOS (${NATIVE_ARCH})..."
GOOS=darwin GOARCH=${NATIVE_ARCH} CGO_ENABLED=1 go build -ldflags "-s -w -X main.Version=${VERSION}" -o "${BUILD_DIR}/trmnl-go-darwin-${NATIVE_ARCH}"
echo -e "${GREEN}✓${NC} macOS ${NATIVE_ARCH} build complete"

# Try to cross-compile for other macOS architecture (may require cross-compilation SDK)
echo -e "${YELLOW}→${NC} Building for macOS (${CROSS_ARCH})..."
if GOOS=darwin GOARCH=${CROSS_ARCH} CGO_ENABLED=1 go build -ldflags "-s -w -X main.Version=${VERSION}" -o "${BUILD_DIR}/trmnl-go-darwin-${CROSS_ARCH}" 2>/dev/null; then
    echo -e "${GREEN}✓${NC} macOS ${CROSS_ARCH} build complete"

    # Create Universal macOS Binary if both architectures built successfully
    echo -e "${YELLOW}→${NC} Creating Universal macOS binary..."
    lipo -create -output "${BUILD_DIR}/trmnl-go-darwin-universal" \
        "${BUILD_DIR}/trmnl-go-darwin-amd64" \
        "${BUILD_DIR}/trmnl-go-darwin-arm64"
    echo -e "${GREEN}✓${NC} Universal binary created"
else
    echo -e "${YELLOW}⚠${NC} macOS ${CROSS_ARCH} cross-compilation not available, using ${NATIVE_ARCH} only"
    cp "${BUILD_DIR}/trmnl-go-darwin-${NATIVE_ARCH}" "${BUILD_DIR}/trmnl-go-darwin-universal"
fi

# Note: Windows and Linux builds require CGO and must be built natively on their respective platforms
# Cross-compilation from macOS is not supported due to Fyne GUI dependencies
echo ""
echo -e "${YELLOW}Note:${NC} Windows and Linux builds require native compilation due to CGO dependencies."
echo -e "${YELLOW}      ${NC}Cross-compilation from macOS is not supported for these platforms."
echo -e "${YELLOW}      ${NC}To build for Windows/Linux, run 'go build' on those platforms directly."

# Create macOS App Bundle
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Creating macOS App Bundle${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"

APP_BUNDLE="${BUILD_DIR}/${APP_NAME}.app"
mkdir -p "${APP_BUNDLE}/Contents/MacOS"
mkdir -p "${APP_BUNDLE}/Contents/Resources"

# Copy universal binary
cp "${BUILD_DIR}/trmnl-go-darwin-universal" "${APP_BUNDLE}/Contents/MacOS/${APP_NAME}"
chmod +x "${APP_BUNDLE}/Contents/MacOS/${APP_NAME}"

# Create Info.plist
cat > "${APP_BUNDLE}/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleExecutable</key>
    <string>${APP_NAME}</string>
    <key>CFBundleIdentifier</key>
    <string>${BUNDLE_ID}</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSSupportsAutomaticGraphicsSwitching</key>
    <true/>
</dict>
</plist>
EOF

echo -e "${GREEN}✓${NC} App bundle created: ${APP_BUNDLE}"

# Code Signing (if requested)
if [ "$SIGN_APP" = true ]; then
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  Code Signing${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"

    if [ -z "$SIGNING_IDENTITY" ]; then
        echo -e "${RED}✗${NC} No signing identity provided"
        echo -e "${YELLOW}Available identities:${NC}"
        security find-identity -v -p codesigning
        exit 1
    fi

    echo -e "${YELLOW}→${NC} Signing with identity: ${SIGNING_IDENTITY}"

    # Sign the binary first
    codesign --force --options runtime --timestamp \
        --sign "${SIGNING_IDENTITY}" \
        "${APP_BUNDLE}/Contents/MacOS/${APP_NAME}"

    # Sign the app bundle
    codesign --force --options runtime --timestamp \
        --sign "${SIGNING_IDENTITY}" \
        "${APP_BUNDLE}"

    # Verify signature
    echo -e "${YELLOW}→${NC} Verifying signature..."
    codesign --verify --verbose "${APP_BUNDLE}"
    spctl --assess --verbose "${APP_BUNDLE}"

    echo -e "${GREEN}✓${NC} Code signing complete"
fi

# Create DMG (if requested)
if [ "$CREATE_DMG" = true ]; then
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  Creating DMG${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"

    DMG_NAME="${APP_NAME}-${VERSION}.dmg"
    DMG_PATH="${DIST_DIR}/${DMG_NAME}"
    TEMP_DMG="${BUILD_DIR}/temp.dmg"

    # Create temporary DMG
    echo -e "${YELLOW}→${NC} Creating disk image..."
    hdiutil create -size 100m -fs HFS+ -volname "${APP_NAME}" "${TEMP_DMG}"

    # Mount it
    hdiutil attach "${TEMP_DMG}" -mountpoint "/Volumes/${APP_NAME}"

    # Copy app bundle
    cp -R "${APP_BUNDLE}" "/Volumes/${APP_NAME}/"

    # Create Applications symlink
    ln -s /Applications "/Volumes/${APP_NAME}/Applications"

    # Unmount
    hdiutil detach "/Volumes/${APP_NAME}"

    # Convert to compressed DMG
    echo -e "${YELLOW}→${NC} Compressing disk image..."
    hdiutil convert "${TEMP_DMG}" -format UDZO -o "${DMG_PATH}"
    rm "${TEMP_DMG}"

    # Sign DMG if we have a signing identity
    if [ "$SIGN_APP" = true ]; then
        echo -e "${YELLOW}→${NC} Signing DMG..."
        codesign --force --sign "${SIGNING_IDENTITY}" "${DMG_PATH}"
    fi

    echo -e "${GREEN}✓${NC} DMG created: ${DMG_PATH}"
fi

# Create distribution archives
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Creating Distribution Archives${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"

# macOS
echo -e "${YELLOW}→${NC} Packaging macOS app..."
cd "${BUILD_DIR}"
zip -r "../${DIST_DIR}/${APP_NAME}-${VERSION}-macos.zip" "${APP_NAME}.app" > /dev/null
cd ..
echo -e "${GREEN}✓${NC} macOS archive: ${DIST_DIR}/${APP_NAME}-${VERSION}-macos.zip"

# Summary
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  Build Complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo ""
echo -e "${BLUE}Distribution files:${NC}"
ls -lh "${DIST_DIR}"
echo ""

if [ "$SIGN_APP" = false ]; then
    echo -e "${YELLOW}Note:${NC} To sign the macOS app, run:"
    echo -e "  ${BLUE}./build-all.sh --sign \"Developer ID Application: Your Name (TEAMID)\"${NC}"
    echo ""
fi

if [ "$CREATE_DMG" = false ]; then
    echo -e "${YELLOW}Note:${NC} To create a DMG installer, add the ${BLUE}--dmg${NC} flag"
    echo ""
fi

echo -e "${GREEN}Build artifacts are in:${NC}"
echo -e "  ${BLUE}${BUILD_DIR}/${NC} - Binaries and app bundle"
echo -e "  ${BLUE}${DIST_DIR}/${NC} - Distribution archives"
echo ""
