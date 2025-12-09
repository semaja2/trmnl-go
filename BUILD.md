# Building TRMNL Virtual Display

This document describes how to build TRMNL Virtual Display for different platforms using `fyne-cross`.

## Quick Start

**Prerequisites:**
- Go 1.22+
- Docker (for fyne-cross cross-compilation)

```bash
# Local development build (current platform only)
go build -o trmnl-go

# Cross-platform build (all platforms)
./build-all.sh

# Or build for specific platform
./build-all.sh macos    # macOS Intel + Apple Silicon
./build-all.sh windows  # Windows x64 + ARM64
./build-all.sh linux    # Linux x64 + ARM64
```

## Why fyne-cross?

`fyne-cross` uses Docker containers to provide consistent build environments for all platforms, handling:
- **Native dependencies**: CoreWLAN (macOS), WLAN API (Windows), wireless drivers (Linux)
- **CGO compilation**: Platform-specific C code for WiFi and battery APIs
- **Cross-compilation**: Build for all platforms from any host OS
- **Consistent builds**: Same build environment regardless of host system

## Build Script

The `build-all.sh` script:
1. Checks for and installs `fyne-cross` if needed
2. Builds for specified platform(s) with proper architectures
3. Outputs binaries to `fyne-cross/dist/`

**Usage:**
```bash
./build-all.sh [all|macos|windows|linux]
```

**Platforms built:**
- **macOS**: `darwin-amd64`, `darwin-arm64` (Intel + Apple Silicon)
- **Windows**: `windows-amd64`, `windows-arm64`
- **Linux**: `linux-amd64`, `linux-arm64`

## Build Output

Cross-platform builds are saved to `fyne-cross/dist/`:

```
fyne-cross/dist/
├── darwin-amd64/
│   └── trmnl-go         # macOS Intel binary
├── darwin-arm64/
│   └── trmnl-go         # macOS Apple Silicon binary
├── windows-amd64/
│   └── trmnl-go.exe     # Windows x64 executable
├── windows-arm64/
│   └── trmnl-go.exe     # Windows ARM64 executable
├── linux-amd64/
│   └── trmnl-go         # Linux x64 binary
└── linux-arm64/
    └── trmnl-go         # Linux ARM64 binary
```

## Native Dependencies Handled by fyne-cross

### macOS
- **CoreWLAN framework**: Native WiFi signal detection (RSSI)
- **IOKit framework**: Native battery percentage detection
- **Automatic frameworks**: Linked via `#cgo LDFLAGS: -framework CoreWLAN -framework IOKit`

### Windows
- **WLAN API**: Native WiFi signal detection via `wlanapi.dll`
- **Windows Power API**: Battery status via `GetSystemPowerStatus()`
- **Libraries**: Automatically linked via `#cgo LDFLAGS: -lwlanapi -lole32`

### Linux
- **No external dependencies**: Uses `/proc/net/wireless` and `/sys/class/power_supply/`
- **Pure Go implementation**: Direct file reading, no CGO required

## Platform-Specific Notes

### macOS

**Running unsigned binaries:**
```bash
# Remove quarantine attribute
xattr -cr trmnl-go

# Or allow in System Settings → Privacy & Security → "Open Anyway"
```

**For distribution**, you'll need an Apple Developer account for code signing and notarization.

### Windows

**Console output:**
- Standard builds (from `./build-all.sh`) show console output when run from cmd/PowerShell
- For GUI-only builds (no console), add `-ldflags="-H=windowsgui"` to build command

**Running:**
```cmd
# From Command Prompt (shows output)
trmnl-go.exe --verbose

# Or just run to see GUI
trmnl-go.exe
```

### Linux

**Running:**
```bash
chmod +x trmnl-go
./trmnl-go --verbose
```

**Display requirements:**
- X11 or Wayland
- No additional packages needed (Fyne handles display automatically)

## Local Development Builds

For quick iteration during development:

```bash
# Build for current platform only (no Docker needed)
go build -o trmnl-go

# Run immediately
./trmnl-go --verbose

# With race detector
go build -race -o trmnl-go
```

Local builds are faster but won't include all platform-specific optimizations that fyne-cross provides.

## Troubleshooting

### Docker not running

**Error:** `Cannot connect to the Docker daemon`

**Solution:** Start Docker Desktop or Docker daemon before running `./build-all.sh`

### fyne-cross not found

The build script automatically installs fyne-cross, but you can install manually:

```bash
go install github.com/fyne-io/fyne-cross@latest
```

### Build failures

**Clean Docker cache:**
```bash
docker system prune -a
```

**Reinstall fyne-cross:**
```bash
go install github.com/fyne-io/fyne-cross@latest
```

## CI/CD Integration

**GitHub Actions example:**

```yaml
name: Build

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Install fyne-cross
        run: go install github.com/fyne-io/fyne-cross@latest

      - name: Build all platforms
        run: ./build-all.sh

      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: binaries
          path: fyne-cross/dist/
```

## Version Management

Version is set in `build-all.sh`:

```bash
VERSION="1.0.0"
```

To release a new version:
1. Update `VERSION` in `build-all.sh`
2. Update `const Version` in `app.go`
3. Run `./build-all.sh`
4. Tag the release: `git tag v1.0.0 && git push --tags`

## Manual fyne-cross Usage

For advanced users who want fine-grained control:

```bash
# Build for specific OS and architecture
fyne-cross darwin -arch=arm64 -app-version=1.0.0 -app-id=net.semaja2.trmnl

# Custom output directory
fyne-cross windows -output=my-builds/

# With additional build flags
fyne-cross linux -ldflags="-s -w" -arch=amd64,arm64
```

See `fyne-cross --help` for all options.
