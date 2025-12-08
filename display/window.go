package display

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/semaja2/trmnl-go/config"
)

// Window represents the display window
type Window struct {
	app         fyne.App
	window      fyne.Window
	imageWidget *canvas.Image
	statusLabel *widget.Label
	config      *config.Config
	verbose     bool
}

// NewWindow creates a new display window
func NewWindow(cfg *config.Config, verbose bool) *Window {
	w := &Window{
		app:     app.New(),
		config:  cfg,
		verbose: verbose,
	}

	w.window = w.app.NewWindow("TRMNL Virtual Display")
	w.window.Resize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))
	w.window.SetFixedSize(true)

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
	if w.statusLabel != nil {
		// Use fyne.Do to ensure UI updates happen on the main thread
		fyne.Do(func() {
			w.statusLabel.SetText(status)
		})
	}
}

// SetOnClosed sets the callback for when the window is closed
func (w *Window) SetOnClosed(callback func()) {
	w.window.SetOnClosed(callback)
}

// Close closes the window
func (w *Window) Close() {
	if w.window != nil {
		w.window.Close()
	}
}

// GetApp returns the Fyne app instance
func (w *Window) GetApp() interface{} {
	return w.app
}

// invertImage inverts the colors of an image for dark mode
func invertImage(img image.Image) image.Image {
	bounds := img.Bounds()
	inverted := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			r, g, b, a := originalColor.RGBA()

			// Invert RGB channels (keep alpha)
			invertedColor := color.RGBA{
				R: uint8(255 - (r >> 8)),
				G: uint8(255 - (g >> 8)),
				B: uint8(255 - (b >> 8)),
				A: uint8(a >> 8),
			}

			inverted.Set(x, y, invertedColor)
		}
	}

	return inverted
}

// CreatePlaceholderImage creates a placeholder image with text
func CreatePlaceholderImage(width, height int, text string) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with light gray background
	gray := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{gray}, image.Point{}, draw.Src)

	// Draw a border
	border := color.RGBA{R: 100, G: 100, B: 100, A: 255}
	drawBorder(img, border, 2)

	return img
}

// drawBorder draws a border around the image
func drawBorder(img *image.RGBA, col color.Color, thickness int) {
	bounds := img.Bounds()

	// Top and bottom borders
	for i := 0; i < thickness; i++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.Set(x, bounds.Min.Y+i, col)
			img.Set(x, bounds.Max.Y-1-i, col)
		}
	}

	// Left and right borders
	for i := 0; i < thickness; i++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			img.Set(bounds.Min.X+i, y, col)
			img.Set(bounds.Max.X-1-i, y, col)
		}
	}
}
