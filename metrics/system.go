package metrics

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/route"
)

// SystemMetrics holds system information
type SystemMetrics struct {
	BatteryVoltage float64 // Battery percentage (0-100) or voltage equivalent
	RSSI           int     // WiFi signal strength (dBm, typically -30 to -90)
}

// Collect gathers current system metrics
func Collect() SystemMetrics {
	metrics := SystemMetrics{
		BatteryVoltage: 100.0, // Default for desktops without battery
		RSSI:           -50,   // Default decent signal
	}

	// Try to get actual battery percentage
	if battery := getBatteryPercentage(); battery >= 0 {
		metrics.BatteryVoltage = battery
	}

	// Try to get actual WiFi signal strength
	if rssi := getWiFiSignal(); rssi != 0 {
		metrics.RSSI = rssi
	}

	return metrics
}

// getBatteryPercentage returns battery percentage (0-100) or -1 if unavailable
func getBatteryPercentage() float64 {
	switch runtime.GOOS {
	case "darwin": // macOS
		return getMacOSBattery()
	case "linux":
		return getLinuxBattery()
	case "windows":
		return getWindowsBattery()
	default:
		return -1
	}
}

// getMacOSBattery gets battery percentage on macOS using pmset
func getMacOSBattery() float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pmset", "-g", "batt")
	output, err := cmd.Output()
	if err != nil {
		return -1
	}

	// Parse output like: "Now drawing from 'Battery Power'\n -InternalBattery-0 (id=12345) 85%; discharging; 3:27 remaining"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "%") {
			// Extract percentage
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasSuffix(part, "%;") || strings.HasSuffix(part, "%") {
					percentStr := strings.TrimSuffix(strings.TrimSuffix(part, ";"), "%")
					if percent, err := strconv.ParseFloat(percentStr, 64); err == nil {
						return percent
					}
				}
			}
		}
	}

	return -1
}

// getLinuxBattery gets battery percentage on Linux from /sys/class/power_supply
func getLinuxBattery() float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try reading from common battery locations
	cmd := exec.CommandContext(ctx, "cat", "/sys/class/power_supply/BAT0/capacity")
	output, err := cmd.Output()
	if err != nil {
		// Try BAT1
		cmd = exec.CommandContext(ctx, "cat", "/sys/class/power_supply/BAT1/capacity")
		output, err = cmd.Output()
		if err != nil {
			return -1
		}
	}

	percentStr := strings.TrimSpace(string(output))
	if percent, err := strconv.ParseFloat(percentStr, 64); err == nil {
		return percent
	}

	return -1
}

// getWindowsBattery gets battery percentage on Windows using WMIC
func getWindowsBattery() float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wmic", "path", "Win32_Battery", "get", "EstimatedChargeRemaining")
	output, err := cmd.Output()
	if err != nil {
		return -1
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 1 {
		percentStr := strings.TrimSpace(lines[1])
		if percent, err := strconv.ParseFloat(percentStr, 64); err == nil {
			return percent
		}
	}

	return -1
}

// getWiFiSignal returns WiFi signal strength in dBm or 0 if unavailable
func getWiFiSignal() int {
	switch runtime.GOOS {
	case "darwin": // macOS
		return getMacOSWiFiSignal()
	case "linux":
		return getLinuxWiFiSignal()
	case "windows":
		return getWindowsWiFiSignal()
	default:
		return 0
	}
}

// getMacOSWiFiSignal gets WiFi signal on macOS using airport utility
func getMacOSWiFiSignal() int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use the airport utility to get WiFi info
	cmd := exec.CommandContext(ctx, "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport", "-I")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Parse output for "agrCtlRSSI: -XX"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "agrCtlRSSI:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if rssi, err := strconv.Atoi(parts[1]); err == nil {
					return rssi
				}
			}
		}
	}

	return 0
}

// getLinuxWiFiSignal gets WiFi signal on Linux using iwconfig or iw
func getLinuxWiFiSignal() int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try using iw (newer)
	cmd := exec.CommandContext(ctx, "iw", "dev")
	output, err := cmd.Output()
	if err != nil {
		// Fall back to iwconfig
		return getLinuxWiFiSignalIwconfig()
	}

	// Get the interface name first
	lines := strings.Split(string(output), "\n")
	var iface string
	for _, line := range lines {
		if strings.Contains(line, "Interface") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				iface = parts[1]
				break
			}
		}
	}

	if iface == "" {
		return 0
	}

	// Get signal info for the interface
	cmd = exec.CommandContext(ctx, "iw", "dev", iface, "link")
	output, err = cmd.Output()
	if err != nil {
		return 0
	}

	// Parse "signal: -XX dBm"
	lines = strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "signal:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "signal:" && i+1 < len(parts) {
					rssiStr := strings.TrimSpace(parts[i+1])
					if rssi, err := strconv.Atoi(rssiStr); err == nil {
						return rssi
					}
				}
			}
		}
	}

	return 0
}

// getLinuxWiFiSignalIwconfig gets WiFi signal using iwconfig (older Linux)
func getLinuxWiFiSignalIwconfig() int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "iwconfig")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Parse "Signal level=-XX dBm"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Signal level=") {
			parts := strings.Split(line, "Signal level=")
			if len(parts) >= 2 {
				rssiPart := strings.Fields(parts[1])[0]
				rssiStr := strings.TrimSpace(strings.TrimSuffix(rssiPart, "dBm"))
				if rssi, err := strconv.Atoi(rssiStr); err == nil {
					return rssi
				}
			}
		}
	}

	return 0
}

// getWindowsWiFiSignal gets WiFi signal on Windows using netsh
func getWindowsWiFiSignal() int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "netsh", "wlan", "show", "interfaces")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Parse "Signal: XX%"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Signal") && strings.Contains(line, "%") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				signalStr := strings.TrimSpace(strings.TrimSuffix(parts[1], "%"))
				if signal, err := strconv.Atoi(signalStr); err == nil {
					// Convert percentage to approximate dBm
					// 100% ≈ -30 dBm, 0% ≈ -90 dBm
					rssi := -90 + (signal * 60 / 100)
					return rssi
				}
			}
		}
	}

	return 0
}

// String returns a human-readable representation of the metrics
func (m SystemMetrics) String() string {
	return fmt.Sprintf("Battery: %.1f%%, WiFi: %d dBm", m.BatteryVoltage, m.RSSI)
}

// getDefaultRouteInterface returns the interface name used for the default route
func getDefaultRouteInterface() string {
	switch runtime.GOOS {
	case "darwin", "freebsd", "openbsd", "netbsd":
		return getDefaultRouteInterfaceBSD()
	case "linux":
		return getDefaultRouteInterfaceLinux()
	case "windows":
		return getDefaultRouteInterfaceWindows()
	default:
		return ""
	}
}

// getDefaultRouteInterfaceBSD uses golang.org/x/net/route for BSD-like systems (macOS, *BSD)
func getDefaultRouteInterfaceBSD() string {
	rib, err := route.FetchRIB(0, route.RIBTypeRoute, 0)
	if err != nil {
		return ""
	}

	msgs, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		return ""
	}

	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}

		// Look for default route (0.0.0.0/0 or ::/0)
		isDefault := false
		for _, addr := range rm.Addrs {
			if addr == nil {
				continue
			}
			// Check if it's a default destination (0.0.0.0)
			if ipnet, ok := addr.(*route.Inet4Addr); ok {
				if ipnet.IP == [4]byte{0, 0, 0, 0} {
					isDefault = true
					break
				}
			}
		}

		if isDefault && rm.Index > 0 {
			// Get interface by index
			iface, err := net.InterfaceByIndex(rm.Index)
			if err == nil {
				return iface.Name
			}
		}
	}

	return ""
}

// getDefaultRouteInterfaceLinux reads /proc/net/route for Linux
func getDefaultRouteInterfaceLinux() string {
	// For Linux, we'll use a simple fallback approach
	// Read the first non-loopback interface with a valid address
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		// Check if it has a non-link-local address
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() && !ipnet.IP.IsLinkLocalUnicast() {
					return iface.Name
				}
			}
		}
	}

	return ""
}

// getDefaultRouteInterfaceWindows uses net.Interfaces for Windows
func getDefaultRouteInterfaceWindows() string {
	// For Windows, fallback to first valid interface
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		if len(iface.HardwareAddr) > 0 {
			return iface.Name
		}
	}

	return ""
}

// GetMACAddress returns the MAC address of the primary network interface (uppercase)
// If ifaceName is provided, uses that interface; otherwise uses default route interface
func GetMACAddress() (string, error) {
	return GetMACAddressForInterface("")
}

// GetMACAddressForInterface returns the MAC address for a specific interface
func GetMACAddressForInterface(ifaceName string) (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// If no interface specified, try to get default route interface
	if ifaceName == "" {
		ifaceName = getDefaultRouteInterface()
	}

	// If we have a specific interface name, look for it
	if ifaceName != "" {
		for _, iface := range interfaces {
			if iface.Name == ifaceName {
				if len(iface.HardwareAddr) > 0 {
					return strings.ToUpper(iface.HardwareAddr.String()), nil
				}
				return "", fmt.Errorf("interface %s has no MAC address", ifaceName)
			}
		}
		return "", fmt.Errorf("interface %s not found", ifaceName)
	}

	// Fallback: first non-loopback, up interface with a MAC
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		mac := strings.ToUpper(iface.HardwareAddr.String())
		if mac != "" {
			return mac, nil
		}
	}

	return "", fmt.Errorf("no valid network interface found")
}

// GetPrimaryInterfaceName returns the name of the primary network interface
func GetPrimaryInterfaceName() string {
	// Try default route interface first
	if iface := getDefaultRouteInterface(); iface != "" {
		return iface
	}

	// Fallback to first valid interface
	interfaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		if len(iface.HardwareAddr) > 0 {
			return iface.Name
		}
	}

	return "unknown"
}
