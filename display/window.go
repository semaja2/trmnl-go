package display

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/semaja2/trmnl-go/config"
)

// Window represents the display window
type Window struct {
	app             fyne.App
	window          fyne.Window
	imageWidget     *canvas.Image
	statusLabel     *widget.Label
	config          *config.Config
	verbose         bool
	refreshCallback func()
	rotateCallback  func()
}

// NewWindow creates a new display window
func NewWindow(cfg *config.Config, verbose bool) *Window {
	w := &Window{
		app:     app.New(),
		config:  cfg,
		verbose: verbose,
	}

	w.window = w.app.NewWindow("TRMNL Virtual Display")

	// Set fullscreen or windowed mode
	if cfg.Fullscreen {
		w.window.SetFullScreen(true)
	} else {
		w.window.Resize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))
		w.window.SetFixedSize(true)
	}

	// Set as master window for proper app behavior (shows dock icon on macOS)
	w.window.SetMaster()

	// Create image widget
	w.imageWidget = canvas.NewImageFromImage(nil)
	w.imageWidget.FillMode = canvas.ImageFillContain
	w.imageWidget.SetMinSize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))

	// Create status label
	w.statusLabel = widget.NewLabel("Initializing...")
	w.statusLabel.Alignment = fyne.TextAlignCenter

	// Simple layout with status bar at bottom
	content := container.NewBorder(
		nil,           // top
		w.statusLabel, // bottom
		nil,           // left
		nil,           // right
		w.imageWidget, // center
	)

	w.window.SetContent(content)

	// Set up keyboard shortcuts using Canvas shortcut handler
	// Cmd+R / Ctrl+R for refresh
	w.window.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyR,
		Modifier: fyne.KeyModifierControl | fyne.KeyModifierSuper,
	}, func(shortcut fyne.Shortcut) {
		if w.refreshCallback != nil {
			w.refreshCallback()
		}
	})

	// Cmd+T / Ctrl+T for rotate
	w.window.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyT,
		Modifier: fyne.KeyModifierControl | fyne.KeyModifierSuper,
	}, func(shortcut fyne.Shortcut) {
		if w.rotateCallback != nil {
			w.rotateCallback()
		}
	})

	return w
}

// Show displays the window
func (w *Window) Show() {
	w.window.Show()
	// Start the main event loop (blocks until window closes)
	w.app.Run()
}

// UpdateImage updates the displayed image from byte data
func (w *Window) UpdateImage(imageData []byte) error {
	if w.verbose {
		fmt.Printf("[Display] Decoding image (%d bytes)\n", len(imageData))
	}

	// Decode image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Apply rotation if enabled
	if w.config.Rotation != 0 {
		img = rotateImage(img, w.config.Rotation)
		if w.verbose {
			fmt.Printf("[Display] Applied rotation: %d degrees\n", w.config.Rotation)
		}
	}

	// Apply dark mode if enabled
	if w.config.DarkMode {
		img = invertImage(img)
		if w.verbose {
			fmt.Println("[Display] Applied dark mode inversion")
		}
	}

	// Update the image on the UI thread using Fyne's thread-safe method
	fyne.Do(func() {
		w.imageWidget.Image = img
		w.imageWidget.Refresh()
	})

	if w.verbose {
		fmt.Printf("[Display] Image updated: %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())
	}

	return nil
}

// UpdateStatus updates the status label text
// This is called from a goroutine, so we need to be careful about threading
func (w *Window) UpdateStatus(status string) {
	// Use fyne.Do to ensure UI updates happen on the main thread
	fyne.Do(func() {
		w.statusLabel.SetText(status)
	})
}

// SetOnClosed sets the callback for when the window is closed
func (w *Window) SetOnClosed(callback func()) {
	w.window.SetOnClosed(callback)
}

// SetOnRefresh sets the callback for manual refresh (Cmd+R / Ctrl+R)
func (w *Window) SetOnRefresh(callback func()) {
	w.refreshCallback = callback
}

// SetOnRotate sets the callback for manual rotate (Cmd+T / Ctrl+T)
func (w *Window) SetOnRotate(callback func()) {
	w.rotateCallback = callback
}

// Close closes the window
func (w *Window) Close() {
	w.window.Close()
}

// GetApp returns the Fyne app instance
func (w *Window) GetApp() interface{} {
	return w.app
}

// SetMenuItemsEnabled is a no-op for Fyne window (shortcuts handled via callbacks)
func (w *Window) SetMenuItemsEnabled(enabled bool) {
	// No-op - Fyne shortcuts are already guarded in the callback
}
