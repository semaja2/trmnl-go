# Building TRMNL Virtual Display

This document describes how to build TRMNL Virtual Display for different platforms.

## Quick Build (Current Platform)

For quick local development builds:

```bash
./build.sh
```

This will build for your current platform and automatically code sign on macOS.

## macOS Production Build

The `build-macos.sh` script creates production-ready macOS builds with proper app bundles:

**Basic Build:**

```bash
./build-macos.sh
```

This creates:
- Universal macOS binary (Intel + Apple Silicon)
- macOS App Bundle (`build/TRMNL.app`)
- Distribution ZIP (`dist/TRMNL-1.0.0-macos.zip`)

### With Code Signing

To create a properly signed macOS application:

```bash
./build-macos.sh --sign "Developer ID Application: Your Name (TEAMID)"
```

**Finding your signing identity:**
```bash
security find-identity -v -p codesigning
```

### With DMG Installer

To create a macOS DMG installer:

```bash
./build-macos.sh --sign "Developer ID Application: Your Name (TEAMID)" --dmg
```

This creates `dist/TRMNL-1.0.0.dmg` with:
- App bundle
- Applications folder shortcut
- Proper code signing (if identity provided)

## Platform-Specific Builds

### macOS

**Requirements:**
- Go 1.24+
- Xcode Command Line Tools
- macOS 10.13+

**Build:**
```bash
go build -o trmnl-go
```

**Universal Binary:**
The build script automatically creates a universal binary supporting both Intel and Apple Silicon Macs.

### Windows

**Requirements:**
- Go 1.24+
- Windows 10+
- GCC compiler (MinGW-w64 for CGO support)

**Build:**
```bash
go build -o trmnl-go.exe
```

**Note:** Cross-compilation from macOS/Linux to Windows is not supported due to CGO dependencies (Fyne GUI toolkit). You must build natively on Windows.

### Linux

**Requirements:**
- Go 1.24+
- X11 development libraries

**Ubuntu/Debian:**
```bash
sudo apt-get install libgl1-mesa-dev xorg-dev
go build -o trmnl-go
```

**Fedora/RHEL:**
```bash
sudo dnf install mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel
go build -o trmnl-go
```

**Note:** Cross-compilation from macOS to Linux is not supported due to CGO dependencies.

## Build Output

### Directory Structure

```
trmnl-go/
├── build/              # Build artifacts
│   ├── TRMNL.app      # macOS app bundle
│   ├── trmnl-go-darwin-amd64    # Intel binary
│   ├── trmnl-go-darwin-arm64    # ARM64 binary
│   └── trmnl-go-darwin-universal # Universal binary
└── dist/               # Distribution archives
    ├── TRMNL-1.0.0-macos.zip
    └── TRMNL-1.0.0.dmg (if --dmg flag used)
```

### macOS App Bundle Structure

```
TRMNL.app/
├── Contents/
│   ├── Info.plist          # App metadata
│   ├── MacOS/
│   │   └── TRMNL           # Universal binary
│   └── Resources/          # App resources (empty for now)
```

## Code Signing

### Ad-hoc Signing (Development)

The simple `build.sh` script performs ad-hoc signing automatically:

```bash
codesign --force --deep --sign - trmnl-go
```

This allows the app to run on your local machine without Gatekeeper issues.

### Distribution Signing

For distributing your app, you need:

1. **Apple Developer Account** - $99/year
2. **Developer ID Application Certificate**
3. **Notarization** (for macOS 10.15+)

**Full signing workflow:**

```bash
# 1. Build and sign
./build-macos.sh --sign "Developer ID Application: Your Name (TEAMID)" --dmg

# 2. Notarize (requires Apple Developer account)
xcrun notarytool submit dist/TRMNL-1.0.0.dmg \
    --apple-id "your@email.com" \
    --team-id "TEAMID" \
    --password "app-specific-password" \
    --wait

# 3. Staple notarization ticket
xcrun stapler staple dist/TRMNL-1.0.0.dmg
```

## Troubleshooting

### macOS: "cannot be opened because the developer cannot be verified"

**Solution 1:** Use ad-hoc signing
```bash
./build.sh
```

**Solution 2:** Allow in System Settings
```bash
xattr -cr TRMNL.app
```

Then go to System Settings → Privacy & Security and click "Open Anyway"

### Linux: Missing GL libraries

```bash
# Ubuntu/Debian
sudo apt-get install libgl1-mesa-dev xorg-dev

# Fedora/RHEL
sudo dnf install mesa-libGL-devel libXcursor-devel libXrandr-devel
```

### Windows: CGO errors

Make sure you're building on Windows natively. Cross-compilation is not supported.

## CI/CD

For automated builds in CI/CD pipelines:

**GitHub Actions (macOS):**
```yaml
- name: Build
  run: |
    ./build-macos.sh

- name: Upload artifacts
  uses: actions/upload-artifact@v3
  with:
    name: trmnl-macos
    path: dist/*.zip
```

**Note:** Code signing in CI requires securely storing certificates and credentials.

## Version Management

The version is set in `build-macos.sh`:

```bash
VERSION="1.0.0"
```

This version is:
- Embedded in binaries via `-ldflags "-X main.Version=${VERSION}"`
- Used in the macOS app bundle Info.plist
- Included in distribution filenames

To release a new version:
1. Update `VERSION` in `build-macos.sh`
2. Update `const Version` in `app.go`
3. Run `./build-macos.sh --sign "..." --dmg`
4. Tag the release in git
