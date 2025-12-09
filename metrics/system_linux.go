//go:build linux

package metrics

import (
	"bufio"
	"fmt"
	"net"
	"os"
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

	return "eth0"
}

// getWiFiSignal returns WiFi signal strength (RSSI in dBm) by reading /proc/net/wireless
func getWiFiSignal() int {
	// Try reading from /proc/net/wireless
	file, err := os.Open("/proc/net/wireless")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Skip the first two header lines
	if !scanner.Scan() || !scanner.Scan() {
		return 0
	}

	// Read wireless interface data
	// Format: interface: status link level noise
	// Example: wlan0: 0000   70.  -40.  -256
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// Need at least 4 fields: interface, status, link, level
		if len(fields) >= 4 {
			// The signal level is in the 4th field (index 3)
			// It's typically in dBm and has a trailing dot
			levelStr := strings.TrimSuffix(fields[3], ".")
			if level, err := strconv.ParseFloat(levelStr, 64); err == nil {
				return int(level)
			}
		}
	}

	return 0
}

// getBatteryPercentage returns battery percentage (0-100) or -1 if unavailable
func getBatteryPercentage() float64 {
	// Try reading from /sys/class/power_supply/BAT0/capacity
	data, err := os.ReadFile("/sys/class/power_supply/BAT0/capacity")
	if err != nil {
		// Try BAT1
		data, err = os.ReadFile("/sys/class/power_supply/BAT1/capacity")
		if err != nil {
			return -1
		}
	}

	percentStr := strings.TrimSpace(string(data))
	if percent, err := strconv.ParseFloat(percentStr, 64); err == nil {
		return percent
	}

	return -1
}
