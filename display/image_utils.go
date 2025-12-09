package display

import (
	"image"
	"image/color"
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
