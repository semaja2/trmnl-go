# Quick Start

## Build

```bash
git clone https://github.com/semaja2/trmnl-go.git
cd trmnl-go
./build.sh
```

**macOS 15.x:** Code signing is required. The `build.sh` script handles this automatically.

## Run

```bash
./trmnl-go -api-key YOUR_API_KEY
```

Get your API key from [usetrmnl.com/plugins](https://usetrmnl.com/plugins)

## Save Configuration

```bash
./trmnl-go -api-key YOUR_KEY -width 1024 -height 768 -save
```

Then just run `./trmnl-go`

## Device Identification

The app auto-detects your MAC address from the primary network interface and uses it as the Device ID for TRMNL cloud licensing.

```bash
./trmnl-go -api-key YOUR_KEY -verbose
```

Output shows:
```
Auto-detected Device ID from en0: xx:xx:xx:xx:xx:xx
Network: en0 (xx:xx:xx:xx:xx:xx)
```

## Troubleshooting

**Crash on macOS 15.x with "Code Signature Invalid"**

Re-sign the binary:
```bash
codesign --force --deep --sign - trmnl-go
```

**Window not appearing?**

1. Ensure you're not running via SSH
2. Verify API key is correct
3. Run with `-verbose` to see output

**Configure window size:**
```bash
./trmnl-go -width 1024 -height 768
```

## Done

Your virtual TRMNL display will:
- Connect to TRMNL API
- Fetch dashboard images
- Auto-refresh based on server settings
- Report MAC address, battery, and WiFi metrics
