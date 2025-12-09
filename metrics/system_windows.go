//go:build windows

package metrics

/*
#cgo LDFLAGS: -lsetupapi -lwlanapi -lole32
#include <windows.h>
#include <wlanapi.h>

double getWindowsBatteryLevel() {
    SYSTEM_POWER_STATUS status;
    if (GetSystemPowerStatus(&status)) {
        if (status.BatteryLifePercent <= 100) {
            return (double)status.BatteryLifePercent;
        }
    }
    return -1.0;
}

int getWindowsWiFiRSSI() {
    HANDLE hClient = NULL;
    DWORD dwMaxClient = 2;
    DWORD dwCurVersion = 0;
    DWORD dwResult = 0;
    int rssi = 0;

    // Open handle to WLAN API
    dwResult = WlanOpenHandle(dwMaxClient, NULL, &dwCurVersion, &hClient);
    if (dwResult != ERROR_SUCCESS) {
        return 0;
    }

    PWLAN_INTERFACE_INFO_LIST pIfList = NULL;
    PWLAN_INTERFACE_INFO pIfInfo = NULL;

    // Enumerate wireless interfaces
    dwResult = WlanEnumInterfaces(hClient, NULL, &pIfList);
    if (dwResult != ERROR_SUCCESS) {
        WlanCloseHandle(hClient, NULL);
        return 0;
    }

    // Get the first interface
    if (pIfList->dwNumberOfItems > 0) {
        pIfInfo = &pIfList->InterfaceInfo[0];

        // Get connection attributes
        PWLAN_CONNECTION_ATTRIBUTES pConnectInfo = NULL;
        DWORD connectInfoSize = sizeof(WLAN_CONNECTION_ATTRIBUTES);
        WLAN_OPCODE_VALUE_TYPE opCode = wlan_opcode_value_type_invalid;

        dwResult = WlanQueryInterface(
            hClient,
            &pIfInfo->InterfaceGuid,
            wlan_intf_opcode_current_connection,
            NULL,
            &connectInfoSize,
            (PVOID*)&pConnectInfo,
            &opCode
        );

        if (dwResult == ERROR_SUCCESS && pConnectInfo != NULL) {
            // Get RSSI value (signal quality is 0-100, convert to dBm approximation)
            // RSSI = -100 + (signalQuality / 2)
            ULONG signalQuality = pConnectInfo->wlanAssociationAttributes.wlanSignalQuality;
            rssi = -100 + (signalQuality / 2);
            WlanFreeMemory(pConnectInfo);
        }
    }

    if (pIfList != NULL) {
        WlanFreeMemory(pIfList);
    }

    WlanCloseHandle(hClient, NULL);
    return rssi;
}
*/
import "C"
import (
	"fmt"
	"net"
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
		return "Ethernet"
	}

	// Find first non-loopback interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			return iface.Name
		}
	}

	return "Ethernet"
}

// getWiFiSignal returns WiFi signal strength (RSSI in dBm) using Windows WLAN API
func getWiFiSignal() int {
	return int(C.getWindowsWiFiRSSI())
}

// getBatteryPercentage returns battery percentage (0-100) or -1 if unavailable using Windows API
func getBatteryPercentage() float64 {
	return float64(C.getWindowsBatteryLevel())
}
