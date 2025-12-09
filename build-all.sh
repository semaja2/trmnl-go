#!/bin/bash
# Cross-platform Build Script using fyne-cross
# Builds for macOS, Windows, and Linux

set -e

# Colors for output
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

VERSION="1.0.0"

echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  TRMNL Virtual Display - Cross-Platform Build${NC}"
echo -e "${BLUE}  Version: ${VERSION}${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo

# Check if fyne-cross is installed
if ! command -v fyne-cross &> /dev/null; then
    echo -e "${YELLOW}fyne-cross not found. Installing...${NC}"
    go install github.com/fyne-io/fyne-cross@latest
fi

# Parse arguments
PLATFORMS="${1:-all}"

build_platform() {
    local platform=$1
    local arch=$2
    echo -e "${YELLOW}→${NC} Building for ${platform}/${arch}..."
    echo "Command: 'fyne-cross ${platform} --arch=${arch} --app-version=${VERSION} --app-id=net.semaja2.trmnl'"
    fyne-cross ${platform} --arch=${arch} --app-version=${VERSION} --app-id=net.semaja2.trmnl
}

case "$PLATFORMS" in
    all)
        echo -e "${YELLOW}→${NC} Building for all platforms..."
        # macOS
        build_platform darwin amd64
        build_platform darwin arm64
        # Windows
        build_platform windows amd64
        build_platform windows arm64
        # Linux
        build_platform linux amd64
        build_platform linux arm64
        ;;
    macos|darwin)
        build_platform darwin amd64
        build_platform darwin arm64
        ;;
    windows)
        build_platform windows amd64
        build_platform windows arm64
        ;;
    linux)
        build_platform linux amd64
        build_platform linux arm64
        ;;
    *)
        echo "Usage: $0 [all|macos|windows|linux]"
        exit 1
        ;;
esac

echo
echo -e "${GREEN}✓${NC} Build complete!"
echo
echo -e "${BLUE}Built artifacts are in:${NC}"
echo -e "  ${BLUE}fyne-cross/dist/${NC}"
echo
ls -lh fyne-cross/dist/ 2>/dev/null || echo "No dist directory found"

echo
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  Build Complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
