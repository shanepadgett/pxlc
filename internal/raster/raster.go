// Package raster executes validated drawing plans with integer pixel semantics.
package raster

import (
	"errors"
	"image"
)

// Color is a straight-alpha RGBA color. Phase 1 accepts alpha values of zero or 255.
type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

// OperationKind identifies a concrete raster operation.
type OperationKind uint8

const (
	// OperationRect fills a rectangular region.
	OperationRect OperationKind = iota + 1
	// OperationGrid draws a rectangular array of colors.
	OperationGrid
)

// Operation is a validated concrete drawing operation.
type Operation struct {
	Kind   OperationKind
	X      int
	Y      int
	Width  int
	Height int
	Color  Color
	Pixels []Color
}

// Layer is an ordered group of operations composited as one transparent surface.
type Layer struct {
	Operations []Operation
}

// Plan describes one complete still image.
type Plan struct {
	Width      int
	Height     int
	Background Color
	Layers     []Layer
}

// Render executes a validated plan and returns a newly owned image.
func Render(plan Plan) (*image.NRGBA, error) {
	if plan.Width <= 0 || plan.Height <= 0 || plan.Width > int(^uint(0)>>1)/plan.Height {
		return nil, errors.New("invalid raster dimensions")
	}
	pixelCount := plan.Width * plan.Height
	if pixelCount > int(^uint(0)>>1)/4 {
		return nil, errors.New("raster dimensions exceed addressable memory")
	}
	output := image.NewNRGBA(image.Rect(0, 0, plan.Width, plan.Height))
	fill(output.Pix, plan.Background)

	layerPixels := make([]byte, pixelCount*4)
	for _, layer := range plan.Layers {
		clear(layerPixels)
		for _, operation := range layer.Operations {
			if err := draw(layerPixels, plan.Width, plan.Height, operation); err != nil {
				return nil, err
			}
		}
		compositeBinary(output.Pix, layerPixels)
	}
	return output, nil
}

func draw(pixels []byte, canvasWidth, canvasHeight int, operation Operation) error {
	if operation.X < 0 || operation.Y < 0 || operation.Width <= 0 || operation.Height <= 0 ||
		operation.X > canvasWidth-operation.Width || operation.Y > canvasHeight-operation.Height {
		return errors.New("raster operation is outside the canvas")
	}
	switch operation.Kind {
	case OperationRect:
		for y := operation.Y; y < operation.Y+operation.Height; y++ {
			for x := operation.X; x < operation.X+operation.Width; x++ {
				setPixel(pixels, canvasWidth, x, y, operation.Color)
			}
		}
	case OperationGrid:
		if operation.Width > int(^uint(0)>>1)/operation.Height || len(operation.Pixels) != operation.Width*operation.Height {
			return errors.New("raster grid has invalid dimensions")
		}
		for y := range operation.Height {
			for x := range operation.Width {
				setPixel(pixels, canvasWidth, operation.X+x, operation.Y+y, operation.Pixels[y*operation.Width+x])
			}
		}
	default:
		return errors.New("unknown raster operation")
	}
	return nil
}

func fill(pixels []byte, color Color) {
	for i := 0; i < len(pixels); i += 4 {
		pixels[i] = color.R
		pixels[i+1] = color.G
		pixels[i+2] = color.B
		pixels[i+3] = color.A
	}
}

func setPixel(pixels []byte, width, x, y int, color Color) {
	i := (y*width + x) * 4
	pixels[i] = color.R
	pixels[i+1] = color.G
	pixels[i+2] = color.B
	pixels[i+3] = color.A
}

func compositeBinary(destination, source []byte) {
	for i := 0; i < len(destination); i += 4 {
		if source[i+3] == 0 {
			continue
		}
		copy(destination[i:i+4], source[i:i+4])
	}
}
