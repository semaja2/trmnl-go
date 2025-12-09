//go:build !darwin

package main

import (
	"github.com/semaja2/trmnl-go/config"
	"github.com/semaja2/trmnl-go/display"
)

// createWindow creates the appropriate window for the platform
func createWindow(cfg *config.Config, useFyne bool, verbose bool) DisplayWindow {
	// On non-macOS platforms, always use Fyne
	return display.NewWindow(cfg, verbose)
}
