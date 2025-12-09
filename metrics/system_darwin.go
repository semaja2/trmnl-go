//go:build darwin

package metrics

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreWLAN -framework Foundation -framework IOKit
#import <CoreWLAN/CoreWLAN.h>
#import <IOKit/ps/IOPowerSources.h>
#import <IOKit/ps/IOPSKeys.h>

int getWiFiRSSI() {
	@autoreleasepool {
		CWWiFiClient *client = [CWWiFiClient sharedWiFiClient];
		if (!client) {
			return 0;
		}

		CWInterface *interface = [client interface];
		if (!interface) {
			return 0;
		}

		NSInteger rssi = [interface rssiValue];
		return (int)rssi;
	}
}

double getBatteryLevel() {
	@autoreleasepool {
		CFTypeRef powerSourcesInfo = IOPSCopyPowerSourcesInfo();
		if (!powerSourcesInfo) {
			return -1.0;
		}

		CFArrayRef powerSources = IOPSCopyPowerSourcesList(powerSourcesInfo);
		if (!powerSources) {
			CFRelease(powerSourcesInfo);
			return -1.0;
		}

		double batteryLevel = -1.0;
		CFIndex count = CFArrayGetCount(powerSources);

		for (CFIndex i = 0; i < count; i++) {
			CFTypeRef powerSource = CFArrayGetValueAtIndex(powerSources, i);
			CFDictionaryRef description = IOPSGetPowerSourceDescription(powerSourcesInfo, powerSource);

			if (description) {
				// Check if this is a battery (not an external power source)
				CFStringRef transportType = CFDictionaryGetValue(description, CFSTR(kIOPSTransportTypeKey));
				if (transportType && CFStringCompare(transportType, CFSTR(kIOPSInternalType), 0) == kCFCompareEqualTo) {
					// Get current capacity
					CFNumberRef currentCapacity = CFDictionaryGetValue(description, CFSTR(kIOPSCurrentCapacityKey));
					CFNumberRef maxCapacity = CFDictionaryGetValue(description, CFSTR(kIOPSMaxCapacityKey));

					if (currentCapacity && maxCapacity) {
						int current = 0, max = 0;
						CFNumberGetValue(currentCapacity, kCFNumberIntType, &current);
						CFNumberGetValue(maxCapacity, kCFNumberIntType, &max);

						if (max > 0) {
							batteryLevel = (double)current / (double)max * 100.0;
							break;
						}
					}
				}
			}
		}

		CFRelease(powerSources);
		CFRelease(powerSourcesInfo);
		return batteryLevel;
	}
}
*/
import "C"
import (
	"net"

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

// getWiFiSignal returns WiFi signal strength (RSSI in dBm) using CoreWLAN framework
func getWiFiSignal() int {
	return int(C.getWiFiRSSI())
}

// getBatteryPercentage returns battery percentage (0-100) or -1 if unavailable using IOKit framework
func getBatteryPercentage() float64 {
	return float64(C.getBatteryLevel())
}
