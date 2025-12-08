# TRMNL Virtual Display

Virtual TRMNL device in Go that renders dashboard content to a desktop window. Reports system metrics like battery and WiFi signal.

## Features

- Cross-platform (macOS, Windows, Linux)
- Works with usetrmnl.com cloud and self-hosted servers
- API key or Device ID authentication
- Native macOS window (borderless available via Fyne fallback)
- Always-on-top mode (macOS only)
- Dark mode
- Auto MAC address detection

## Installation

**Prerequisites:** Go 1.22+

```bash
git clone https://github.com/semaja2/trmnl-go.git
cd trmnl-go
./build.sh
```

**macOS App Bundle:**
```bash
./bundle-macos.sh
open TRMNL.app
# Or install: cp -r TRMNL.app /Applications/
```

**macOS 15.x+:** Code signing is required. The build scripts handle this automatically.

## Quick Start

```bash
# Using API key
./trmnl-go -api-key YOUR_API_KEY

# Self-hosted server
./trmnl-go -device-id YOUR_DEVICE_ID -base-url https://your-server.com

# Save configuration
./trmnl-go -api-key YOUR_KEY -width 1024 -height 768 -save
```

## Command-Line Options

```
  -api-key string       TRMNL API key
  -device-id string     Device ID (self-hosted)
  -base-url string      API base URL (default: usetrmnl.com)
  -width int            Window width (default: 800)
  -height int           Window height (default: 480)
  -dark                 Enable dark mode
  -always-on-top        Keep window on top (macOS only)
  -use-fyne             Force Fyne GUI (default: native on macOS)
  -verbose              Enable verbose logging
  -version              Show version
  -save                 Save settings to config
```

## Configuration

Config stored at `~/.config/trmnl/config.json`:

```json
{
  "api_key": "YOUR_API_KEY",
  "base_url": "https://usetrmnl.com",
  "window_width": 800,
  "window_height": 480,
  "dark_mode": false,
  "always_on_top": false,
  "verbose": false
}
```

**Priority:** CLI flags > Environment variables > Config file > Defaults

## Environment Variables

- `TRMNL_API_KEY`: API key
- `TRMNL_DEVICE_ID`: Device ID
- `TRMNL_BASE_URL`: Custom API URL

## How It Works

1. Auto-detects MAC address from default route interface (or generates random MAC if unavailable)
2. Authenticates with TRMNL API
3. Sends screen dimensions and system info in request headers
4. Fetches display content from `/api/display`
5. Downloads and renders image
6. Reports system metrics (battery, WiFi)
7. Auto-refreshes at server-specified interval

## System Metrics

- **MAC Address**: Auto-detected from default route interface, or random MAC if unavailable
- **Battery**: Real percentage via OS (100% if no battery)
- **WiFi Signal**: Real dBm via OS (-50 dBm if no WiFi)
- **Screen Dimensions**: Sent to server in X-Screen-Width/X-Screen-Height headers

## API Headers

**Request headers sent to server:**
```
access-token / ID: Authentication
battery-voltage: 0-100
rssi: WiFi signal (dBm)
User-Agent: trmnl-go-virtual/1.0.0
X-Device-Type: virtual
X-OS: darwin/linux/windows
X-Arch: amd64/arm64
X-Screen-Width: Window width
X-Screen-Height: Window height
```

**Optional response fields from server:**
```json
{
  "image_url": "https://...",
  "filename": "display.png",
  "refresh_rate": 60,
  "width": 800,
  "height": 480
}
```

## Development

```bash
# Build
go build -o trmnl-go

# Cross-compile
GOOS=darwin GOARCH=arm64 go build
GOOS=linux GOARCH=amd64 go build
GOOS=windows GOARCH=amd64 go build
```

## License

To be determined, all rights reserved by semaja2

## References

- [TRMNL](https://usetrmnl.com)
- [TRMNL Display](https://github.com/usetrmnl/trmnl-display)
- [Fyne](https://fyne.io)
