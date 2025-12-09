package models

import "fmt"

// DeviceModel represents a TRMNL device model with its specifications
type DeviceModel struct {
	Name   string // Model identifier sent in API headers
	Width  int    // Screen width in pixels
	Height int    // Screen height in pixels
	Desc   string // Human-readable description
}

// Predefined TRMNL device models
var (
	// Physical TRMNL devices
	TRMNL = DeviceModel{
		Name:   "TRMNL",
		Width:  800,
		Height: 480,
		Desc:   "TRMNL e-ink display (800x480)",
	}

	// Virtual display models
	Virtual = DeviceModel{
		Name:   "virtual",
		Width:  800,
		Height: 480,
		Desc:   "Virtual display (800x480)",
	}

	VirtualHD = DeviceModel{
		Name:   "virtual-hd",
		Width:  1024,
		Height: 768,
		Desc:   "Virtual display HD (1024x768)",
	}

	VirtualFHD = DeviceModel{
		Name:   "virtual-fhd",
		Width:  1920,
		Height: 1080,
		Desc:   "Virtual display Full HD (1920x1080)",
	}

	VirtualPortrait = DeviceModel{
		Name:   "virtual-portrait",
		Width:  480,
		Height: 800,
		Desc:   "Virtual display portrait (480x800)",
	}

	// Common e-ink display sizes
	Waveshare75 = DeviceModel{
		Name:   "waveshare-7.5",
		Width:  800,
		Height: 480,
		Desc:   "Waveshare 7.5\" e-ink (800x480)",
	}

	Waveshare97 = DeviceModel{
		Name:   "waveshare-9.7",
		Width:  1200,
		Height: 825,
		Desc:   "Waveshare 9.7\" e-ink (1200x825)",
	}
)

// AllModels returns all predefined device models
func AllModels() []DeviceModel {
	return []DeviceModel{
		TRMNL,
		Virtual,
		VirtualHD,
		VirtualFHD,
		VirtualPortrait,
		Waveshare75,
		Waveshare97,
	}
}

// GetModel returns a device model by name (case-insensitive)
func GetModel(name string) (DeviceModel, error) {
	for _, model := range AllModels() {
		if model.Name == name {
			return model, nil
		}
	}
	return DeviceModel{}, fmt.Errorf("unknown model: %s", name)
}

// ListModels returns a formatted string of all available models
func ListModels() string {
	result := "Available device models:\n"
	for _, model := range AllModels() {
		result += fmt.Sprintf("  %-20s %s\n", model.Name, model.Desc)
	}
	return result
}
