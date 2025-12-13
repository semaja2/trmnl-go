package display

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
)

// rotateImage rotates an image by the specified degrees (90, 180, 270)
func rotateImage(img image.Image, degrees int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	switch degrees {
	case 90:
		// Rotate 90 degrees clockwise
		rotated := image.NewRGBA(image.Rect(0, 0, height, width))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				rotated.Set(height-1-y, x, img.At(x, y))
			}
		}
		return rotated

	case 180:
		// Rotate 180 degrees
		rotated := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				rotated.Set(width-1-x, height-1-y, img.At(x, y))
			}
		}
		return rotated

	case 270:
		// Rotate 270 degrees clockwise (or 90 counter-clockwise)
		rotated := image.NewRGBA(image.Rect(0, 0, height, width))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				rotated.Set(y, width-1-x, img.At(x, y))
			}
		}
		return rotated

	default:
		// No rotation or invalid angle
		return img
	}
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

// applyImageTransformations applies rotation, dark mode, and e-paper transformations to image data
// Returns the transformed image data as PNG bytes
func applyImageTransformations(imageData []byte, rotation int, darkMode bool, ePaperMode bool) ([]byte, error) {
	// If no transformations needed, return original data
	if rotation == 0 && !darkMode && !ePaperMode {
		return imageData, nil
	}

	// Decode image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Apply e-paper effect first (before rotation/inversion for best results)
	if ePaperMode {
		img = applyEPaperEffect(img)
	}

	// Apply rotation
	if rotation != 0 {
		img = rotateImage(img, rotation)
	}

	// Apply dark mode (invert after e-paper effect)
	if darkMode {
		img = invertImage(img)
	}

	// Re-encode image to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

// applyEPaperEffect simulates an e-paper/e-ink display appearance
// - Converts to grayscale
// - Reduces to 4-bit color depth (16 shades of gray)
// - Applies Floyd-Steinberg dithering for smoother gradients
// - Adds pronounced texture to simulate e-paper grain
// - Adds warm tint for realistic off-white background
func applyEPaperEffect(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Convert to grayscale and create error diffusion matrix
	grayscale := image.NewGray(bounds)
	errorMap := make([][]float64, height)
	for i := range errorMap {
		errorMap[i] = make([]float64, width)
	}

	// First pass: convert to grayscale
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			r, g, b, _ := originalColor.RGBA()

			// Convert to grayscale using luminance formula
			gray := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
			gray = gray / 256.0 // Normalize to 0-255 range

			grayscale.SetGray(x, y, color.Gray{Y: uint8(gray)})
		}
	}

	// Second pass: Apply Floyd-Steinberg dithering and reduce to 4-bit (16 levels)
	resultRGBA := image.NewRGBA(bounds)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			oldPixel := float64(grayscale.GrayAt(x, y).Y)

			// Add accumulated error from previous pixels
			oldPixel += errorMap[y][x]

			// Clamp to valid range
			if oldPixel < 0 {
				oldPixel = 0
			}
			if oldPixel > 255 {
				oldPixel = 255
			}

			// Quantize to 16 levels (4-bit)
			newPixel := math.Round(oldPixel/17.0) * 17.0 // 255/15 ≈ 17

			// Add more pronounced texture noise (simulate e-paper grain)
			noise := (rand.Float64() - 0.5) * 8.0 // ±4 intensity (increased from ±1.5)
			newPixel += noise

			// Clamp after noise
			if newPixel < 0 {
				newPixel = 0
			}
			if newPixel > 255 {
				newPixel = 255
			}

			grayValue := uint8(newPixel)

			// Apply warm tint for e-paper look (slightly yellowish/beige background)
			// E-paper displays have an off-white background, not pure white
			r := grayValue
			g := grayValue
			b := uint8(math.Max(0, float64(grayValue)-12)) // Reduce blue for warm tint

			// Add slight yellow tint to whites/light grays
			if grayValue > 200 {
				tintStrength := (float64(grayValue) - 200.0) / 55.0 // 0 to 1 for pixels 200-255
				g = uint8(math.Min(255, float64(g)+tintStrength*8))  // Add yellow
			}

			resultRGBA.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})

			// Calculate quantization error
			quantError := oldPixel - newPixel

			// Distribute error to neighboring pixels (Floyd-Steinberg)
			if x+1 < width {
				errorMap[y][x+1] += quantError * 7.0 / 16.0
			}
			if y+1 < height {
				if x > 0 {
					errorMap[y+1][x-1] += quantError * 3.0 / 16.0
				}
				errorMap[y+1][x] += quantError * 5.0 / 16.0
				if x+1 < width {
					errorMap[y+1][x+1] += quantError * 1.0 / 16.0
				}
			}
		}
	}

	return resultRGBA
}
