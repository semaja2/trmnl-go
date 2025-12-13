package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Layout constants for screen rendering
const (
	TitleOffsetY       = 40  // Offset from center for title text
	MessageStartY      = 10  // Starting Y offset below center for messages
	MessageLineSpacing = 20  // Vertical spacing between message lines
	BottomMarginY      = 30  // Distance from bottom edge
	MinTextMarginX     = 10  // Minimum horizontal margin for text
	ErrorTitleOffsetY  = 60  // Offset from center for error titles
	ErrorMessageStartY = 20  // Starting Y offset below center for error messages
	MaxLineWrapChars   = 60  // Maximum characters per line for text wrapping
)

// GenerateStartupScreen creates a TRMNL startup/splash screen
func GenerateStartupScreen(width, height int, message string) ([]byte, error) {
	// Create a white background
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	white := color.RGBA{255, 255, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{white}, image.Point{}, draw.Src)

	// Draw TRMNL logo text in center
	drawCenteredText(img, width, height/2-TitleOffsetY, "TRMNL", color.Black)

	// Draw message below (split into lines if needed)
	if message != "" {
		lines := splitLines(message)
		startY := height/2 + MessageStartY
		for i, line := range lines {
			drawCenteredText(img, width, startY+(i*MessageLineSpacing), line, color.RGBA{100, 100, 100, 255})
		}
	}

	// Draw version/info at bottom
	drawCenteredText(img, width, height-BottomMarginY, "Virtual Display", color.RGBA{150, 150, 150, 255})

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode startup screen: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateErrorScreen creates an error message screen
func GenerateErrorScreen(width, height int, errorTitle, errorMessage string) ([]byte, error) {
	// Create a white background
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	white := color.RGBA{255, 255, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{white}, image.Point{}, draw.Src)

	// Draw error icon/title
	drawCenteredText(img, width, height/2-ErrorTitleOffsetY, "⚠ "+errorTitle, color.RGBA{200, 0, 0, 255})

	// Draw error message (split into multiple lines if needed)
	lines := wrapText(errorMessage, MaxLineWrapChars)
	startY := height/2 - ErrorMessageStartY
	for i, line := range lines {
		drawCenteredText(img, width, startY+(i*MessageLineSpacing), line, color.RGBA{80, 80, 80, 255})
	}

	// Draw help text at bottom
	drawCenteredText(img, width, height-BottomMarginY, "Check configuration and try again", color.RGBA{120, 120, 120, 255})

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode error screen: %w", err)
	}

	return buf.Bytes(), nil
}

// drawCenteredText draws text centered horizontally at the given Y position
func drawCenteredText(img *image.RGBA, width, y int, text string, col color.Color) {
	// Use basic font (we'll use a simple monospace font)
	face := basicfont.Face7x13

	// Measure text width
	textWidth := font.MeasureString(face, text).Ceil()

	// Calculate starting X position for centered text
	x := (width - textWidth) / 2
	if x < 0 {
		x = MinTextMarginX // Minimum margin
	}

	// Draw the text
	point := fixed.Point26_6{
		X: fixed.Int26_6(x * 64),
		Y: fixed.Int26_6(y * 64),
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  point,
	}
	d.DrawString(text)
}

// splitLines splits text by newline characters
func splitLines(text string) []string {
	var lines []string
	start := 0

	for i, ch := range text {
		if ch == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}

	// Add the last line
	if start < len(text) {
		lines = append(lines, text[start:])
	}

	return lines
}

// wrapText splits long text into multiple lines
func wrapText(text string, maxChars int) []string {
	if len(text) <= maxChars {
		return []string{text}
	}

	var lines []string
	words := []rune(text)
	start := 0

	for start < len(words) {
		end := start + maxChars
		if end > len(words) {
			end = len(words)
		}

		// Try to break at a space if possible
		if end < len(words) {
			for i := end; i > start; i-- {
				if words[i] == ' ' {
					end = i
					break
				}
			}
		}

		lines = append(lines, string(words[start:end]))
		start = end
		if start < len(words) && words[start] == ' ' {
			start++ // Skip the space
		}
	}

	return lines
}
