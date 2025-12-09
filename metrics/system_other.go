//go:build !darwin

package metrics

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// GetMACAddress returns the MAC address of the primary network interface
func GetMACAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// Find first non-loopback interface with a MAC address
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			if len(iface.HardwareAddr) > 0 {
				return iface.HardwareAddr.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no network interface found")
}

// GetMACAddressForInterface returns the MAC address for a specific interface
func GetMACAddressForInterface(ifaceName string) (string, error) {
	if ifaceName == "" {
		return GetMACAddress()
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", err
	}

	return iface.HardwareAddr.String(), nil
}

// GetPrimaryInterfaceName returns the name of the primary network interface
func GetPrimaryInterfaceName() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "eth0"
	}

	// Find first non-loopback interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			return iface.Name
		}
	}

	if runtime.GOOS == "windows" {
		return "Ethernet"
	}
	return "eth0"
}

// getWiFiSignal returns WiFi signal strength (stub for non-macOS platforms)
func getWiFiSignal() int {
	// Platform-specific implementation would go here
	// For now, return a default value
	return 0
}

// getBatteryPercentage returns battery percentage (0-100) or -1 if unavailable
func getBatteryPercentage() float64 {
	switch runtime.GOOS {
	case "linux":
		// Try reading from /sys/class/power_supply/BAT0/capacity
		output, err := exec.Command("cat", "/sys/class/power_supply/BAT0/capacity").Output()
		if err != nil {
			// Try BAT1
			output, err = exec.Command("cat", "/sys/class/power_supply/BAT1/capacity").Output()
			if err != nil {
				return -1
			}
		}
		percentStr := strings.TrimSpace(string(output))
		if percent, err := strconv.ParseFloat(percentStr, 64); err == nil {
			return percent
		}
		return -1

	case "windows":
		// Use WMIC to get battery status
		output, err := exec.Command("WMIC", "Path", "Win32_Battery", "Get", "EstimatedChargeRemaining").Output()
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

	default:
		return -1
	}
}
