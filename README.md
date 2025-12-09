# TRMNL Virtual Display

Virtual TRMNL device in Go that renders dashboard content to a desktop window. Reports system metrics like battery and WiFi signal.

## Features

- Cross-platform (macOS, Windows, Linux)
- Works with usetrmnl.com cloud and self-hosted servers
- **One-command setup** via `/api/setup` endpoint
- **Mirror mode** for viewing current screen across devices
- API key or Device ID authentication
- Native macOS window (borderless available via Fyne fallback)
- Always-on-top mode (macOS only)
- Dark mode
- Auto MAC address detection
- **Realistic battery reporting** (Li-ion voltage curve 3.0V-4.08V)
- **WiFi signal strength** (RSSI) reporting
- **Startup splash screen** and **error screen rendering**
- Predefined device models (TRMNL, virtual, waveshare, etc.)

## Installation

**Prerequisites:**
- Go 1.22+
- Docker (for cross-compilation with fyne-cross)

```bash
git clone https://github.com/semaja2/trmnl-go.git
cd trmnl-go

# Local development build
go build -o trmnl-go

# Cross-platform build (all platforms)
./build-all.sh

# Or build for specific platform
./build-all.sh macos    # macOS only
./build-all.sh windows  # Windows only
./build-all.sh linux    # Linux only
```

**Note:** fyne-cross will be automatically installed if not present. It uses Docker containers to cross-compile for all platforms with native dependencies (CoreWLAN, WLAN API, etc.).

## Quick Start

```bash
# First run (automatic setup - no configuration needed!)
./trmnl-go

# That's it! The app will automatically:
# 1. Detect your MAC address
# 2. Register with usetrmnl.com
# 3. Save your API key
# 4. Start displaying

# Subsequent runs just use saved config
./trmnl-go

# Or manually trigger setup
./trmnl-go -setup

# Using API key directly
./trmnl-go -api-key YOUR_API_KEY

# Self-hosted server
./trmnl-go -device-id YOUR_DEVICE_ID -base-url https://your-server.com

# Mirror mode (shows current screen instead of device-specific content)
./trmnl-go -mirror

# Save configuration for future runs
./trmnl-go -api-key YOUR_KEY -width 1024 -height 768 -save
```

## Command-Line Options

```
  -api-key string       TRMNL API key
  -device-id string     Device ID (self-hosted)
  -base-url string      API base URL (default: https://trmnl.app)
  -setup                Run setup to retrieve API key via MAC address
  -mirror               Use mirror mode (show current screen)
  -model string         Device model (e.g., TRMNL, virtual-hd, virtual-fhd)
  -list-models          List available device models
  -interface string     Network interface for MAC address (e.g., en0, eth0)
  -width int            Window width (overrides model default)
  -height int           Window height (overrides model default)
  -dark                 Enable dark mode
  -always-on-top        Keep window on top (macOS only)
  -use-fyne             Force Fyne GUI (default: native on macOS)
  -verbose              Enable verbose logging
  -version              Show version
  -save                 Save settings to config
```

## Predefined Models

The application includes predefined device models that automatically set screen dimensions and model identifier:

```bash
# List available models
./trmnl-go -list-models

# Use a specific model
./trmnl-go -api-key YOUR_KEY -model virtual-hd
./trmnl-go -api-key YOUR_KEY -model TRMNL
./trmnl-go -api-key YOUR_KEY -model waveshare-9.7

# Override model dimensions
./trmnl-go -api-key YOUR_KEY -model virtual-hd -width 1280 -height 720
```

**Available models:**
- `TRMNL` - TRMNL e-ink display (800x480)
- `virtual` - Virtual display (800x480)
- `virtual-hd` - Virtual display HD (1024x768)
- `virtual-fhd` - Virtual display Full HD (1920x1080)
- `virtual-portrait` - Virtual display portrait (480x800)
- `waveshare-7.5` - Waveshare 7.5" e-ink (800x480)
- `waveshare-9.7` - Waveshare 9.7" e-ink (1200x825)

## Configuration

Config stored at `~/.config/trmnl/config.json`:

```json
{
  "api_key": "YOUR_API_KEY",
  "device_id": "AA:BB:CC:DD:EE:FF",
  "friendly_id": "My TRMNL Device",
  "base_url": "https://trmnl.app",
  "model": "virtual-hd",
  "window_width": 1024,
  "window_height": 768,
  "dark_mode": false,
  "always_on_top": false,
  "mirror_mode": false,
  "verbose": false
}
```

**Priority:** CLI flags > Environment variables > Config file > Defaults

## Environment Variables

- `TRMNL_API_KEY`: API key
- `TRMNL_DEVICE_ID`: Device ID
- `TRMNL_BASE_URL`: Custom API URL

## How It Works

1. **Startup**: Shows splash screen with device info
2. **MAC Detection**: Auto-detects MAC address from default route interface (or generates random MAC)
3. **Setup (optional)**: Can retrieve API key from `/api/setup` using MAC address
4. **Authentication**: Connects using API key or Device ID
5. **Metrics Collection**: Gathers battery level and WiFi signal strength
6. **Display Fetch**: Requests content from `/api/display` (or `/api/current_screen` in mirror mode)
7. **Image Rendering**: Downloads and displays PNG image
8. **Error Handling**: Shows error screens for network/API failures
9. **Auto-Refresh**: Updates at server-specified intervals

## System Metrics

Real device metrics are collected and reported to the server:

- **MAC Address**: Auto-detected from default route interface, or random MAC if unavailable
- **Battery Percentage**: Real battery level via OS (0-100%, defaults to 100% if no battery)
- **Battery Voltage**: Calculated using realistic Li-ion discharge curve (3.0V-4.08V)
  - Linear curve from 1-83%: V = 3.0 + (percentage × 0.012)
  - Plateau from 83-100%: 3.996V → 4.08V
- **WiFi Signal (RSSI)**: Real signal strength in dBm (-40 to -90 dBm, defaults to -50 dBm)
- **Screen Dimensions**: Sent to server in Width/Height headers

## API Endpoints

### GET /api/setup
Retrieves API key and device registration info via MAC address.

**Request headers:**
```
ID: AA:BB:CC:DD:EE:FF (MAC address)
```

**Response:**
```json
{
  "status": 200,
  "api_key": "your-api-key",
  "friendly_id": "Device Name",
  "image_url": "https://...",
  "message": "Success"
}
```

### GET /api/display
Fetches device-specific display content.

**Request headers:**
```
Access-Token: [API key] or ID: [MAC address]
Content-Type: application/json
percent_charged: 85.00
Battery-Voltage: 4.02
RSSI: -50
FW-Version: 1.0.0
Model: virtual
Width: 800
Height: 480
Refresh-Rate: 60
```

**Response:**
```json
{
  "image_url": "https://...",
  "filename": "2025-01-10-plugin-T00:00:00",
  "refresh_rate": 60,
  "status": 200,
  "error": ""
}
```

### GET /api/current_screen (Mirror Mode)
Fetches the current screen (same across all devices). Uses same headers and response format as `/api/display`.

### GET /api/models
Retrieves available device models with specifications (width, height, colors, etc.). Response includes array of model objects with display capabilities.

## Development

```bash
# Local development build
go build -o trmnl-go

# Run with verbose logging
./trmnl-go --verbose

# Cross-platform builds using fyne-cross (requires Docker)
./build-all.sh

# fyne-cross handles all platform-specific dependencies:
# - macOS: CoreWLAN, IOKit frameworks
# - Windows: WLAN API, Windows Power API
# - Linux: /proc/net/wireless, /sys/class/power_supply
```

## Build Output

Cross-platform builds are output to `fyne-cross/dist/`:
- `darwin-amd64/` - macOS Intel binaries
- `darwin-arm64/` - macOS Apple Silicon binaries
- `windows-amd64/` - Windows x64 executables
- `windows-arm64/` - Windows ARM64 executables
- `linux-amd64/` - Linux x64 binaries
- `linux-arm64/` - Linux ARM64 binaries

## Console Output on Windows

Windows builds support two modes:
- **Standard build** (`go build`): Shows console output when run from cmd/PowerShell
- **GUI build** (`-ldflags="-H=windowsgui"`): No console window (cleaner UX)

The `./build-all.sh` script creates standard builds that show console output for debugging.

## License

To be determined, all rights reserved by semaja2

## References

- [TRMNL](https://usetrmnl.com)
- [TRMNL Display](https://github.com/usetrmnl/trmnl-display)
- [Fyne](https://fyne.io)
