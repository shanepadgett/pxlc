package raster

import (
	"errors"
	"image"
)

// Preview returns an integer nearest-neighbor enlargement over the requested background.
// A transparent background preserves source alpha.
func Preview(source *image.NRGBA, scale int, background Color, maximumPixels int) (*image.NRGBA, error) {
	if source == nil || scale <= 0 {
		return nil, errors.New("invalid preview input")
	}
	width := source.Bounds().Dx()
	height := source.Bounds().Dy()
	if width > int(^uint(0)>>1)/scale || height > int(^uint(0)>>1)/scale {
		return nil, errors.New("preview dimensions exceed addressable memory")
	}
	previewWidth := width * scale
	previewHeight := height * scale
	if previewWidth > int(^uint(0)>>1)/previewHeight || previewWidth*previewHeight > maximumPixels {
		return nil, errors.New("preview exceeds the configured pixel limit")
	}
	preview := image.NewNRGBA(image.Rect(0, 0, previewWidth, previewHeight))
	for y := range previewHeight {
		for x := range previewWidth {
			sourceOffset := source.PixOffset(x/scale, y/scale)
			color := Color{
				R: source.Pix[sourceOffset],
				G: source.Pix[sourceOffset+1],
				B: source.Pix[sourceOffset+2],
				A: source.Pix[sourceOffset+3],
			}
			if color.A == 0 {
				color = background
			}
			setPixel(preview.Pix, previewWidth, x, y, color)
		}
	}
	return preview, nil
}
