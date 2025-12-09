//go:build darwin

package main

import (
	"github.com/semaja2/trmnl-go/config"
	"github.com/semaja2/trmnl-go/display"
)

// createWindow creates the appropriate window for the platform
func createWindow(cfg *config.Config, useFyne bool, verbose bool) DisplayWindow {
	if !useFyne {
		if verbose {
			println("[App] Using native macOS window")
		}
		return display.NewNativeWindow(cfg, verbose)
	}
	if verbose {
		println("[App] Using Fyne window (forced via -use-fyne flag)")
	}
	return display.NewWindow(cfg, verbose)
}
