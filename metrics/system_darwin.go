//go:build darwin

package metrics

import (
	"context"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/route"
)

// GetMACAddress returns the MAC address of the primary network interface
func GetMACAddress() (string, error) {
	return getMACAddressFromRoute()
}

// GetMACAddressForInterface returns the MAC address for a specific interface
func GetMACAddressForInterface(ifaceName string) (string, error) {
	if ifaceName == "" {
		return getMACAddressFromRoute()
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", err
	}

	return iface.HardwareAddr.String(), nil
}

// GetPrimaryInterfaceName returns the name of the primary network interface
func GetPrimaryInterfaceName() string {
	// Try to get the default route interface
	rib, err := route.FetchRIB(0, route.RIBTypeRoute, 0)
	if err != nil {
		return "en0" // Default fallback
	}

	msgs, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		return "en0"
	}

	for _, msg := range msgs {
		if rm, ok := msg.(*route.RouteMessage); ok {
			if isDefaultRoute(rm) {
				iface, err := net.InterfaceByIndex(rm.Index)
				if err == nil {
					return iface.Name
				}
			}
		}
	}

	return "en0"
}

func getMACAddressFromRoute() (string, error) {
	// Get default route interface
	rib, err := route.FetchRIB(0, route.RIBTypeRoute, 0)
	if err != nil {
		return "", err
	}

	msgs, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		return "", err
	}

	// Find the default route
	for _, msg := range msgs {
		if rm, ok := msg.(*route.RouteMessage); ok {
			if isDefaultRoute(rm) {
				// Get the interface for this route
				iface, err := net.InterfaceByIndex(rm.Index)
				if err != nil {
					continue
				}
				return iface.HardwareAddr.String(), nil
			}
		}
	}

	// Fallback: try en0
	iface, err := net.InterfaceByName("en0")
	if err != nil {
		return "", err
	}

	return iface.HardwareAddr.String(), nil
}

func isDefaultRoute(rm *route.RouteMessage) bool {
	// Check if this is a default route (0.0.0.0/0)
	if len(rm.Addrs) > 0 {
		if dst, ok := rm.Addrs[0].(*route.Inet4Addr); ok {
			// Default route has destination 0.0.0.0
			return dst.IP == [4]byte{0, 0, 0, 0}
		}
	}
	return false
}

// getWiFiSignal returns WiFi signal strength (RSSI in dBm)
func getWiFiSignal() int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport", "-I")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Parse output for "agrCtlRSSI: -50"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "agrCtlRSSI:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				rssiStr := strings.TrimSpace(parts[1])
				if rssi, err := strconv.Atoi(rssiStr); err == nil {
					return rssi
				}
			}
		}
	}

	return 0
}

// getBatteryPercentage returns battery percentage (0-100) or -1 if unavailable
func getBatteryPercentage() float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pmset", "-g", "batt")
	output, err := cmd.Output()
	if err != nil {
		return -1
	}

	// Parse output like: "Now drawing from 'Battery Power'\n -InternalBattery-0 (id=123456789)	95%; discharging; 5:23 remaining"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "InternalBattery") {
			// Extract percentage
			parts := strings.Split(line, "\t")
			if len(parts) > 1 {
				percentStr := strings.TrimSpace(strings.Split(parts[1], ";")[0])
				percentStr = strings.TrimSuffix(percentStr, "%")
				if percent, err := strconv.ParseFloat(percentStr, 64); err == nil {
					return percent
				}
			}
		}
	}

	return -1
}
