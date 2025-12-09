package metrics

import (
	"fmt"
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

// String returns a human-readable representation of the metrics
func (m SystemMetrics) String() string {
	return fmt.Sprintf("Battery: %.1f%%, WiFi: %d dBm", m.BatteryVoltage, m.RSSI)
}
