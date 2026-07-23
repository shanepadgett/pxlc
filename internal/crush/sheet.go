package crush

import (
	"fmt"
	"image"
	"image/color"
)

// ReduceSheet splits src into equal cells and area-averages each cell to a
// fixed logical size. The result retains the source background and is opaque.
func ReduceSheet(src image.Image, columns, rows, cellSize int) (*image.NRGBA, error) {
	return ReduceSheetCells(src, columns, rows, cellSize, cellSize)
}

// ReduceSheetCells splits src into equal cells and area-averages each cell to
// a fixed logical width and height. The result retains the source background
// and is opaque.
func ReduceSheetCells(src image.Image, columns, rows, cellWidth, cellHeight int) (*image.NRGBA, error) {
	if columns < 1 || rows < 1 || cellWidth < 1 || cellHeight < 1 {
		return nil, fmt.Errorf("columns, rows, cell width, and cell height must be positive")
	}
	if columns > 4096 || rows > 4096 || cellWidth > 4096 || cellHeight > 4096 {
		return nil, fmt.Errorf("sheet dimensions exceed limits")
	}
	if columns > 67_108_864/cellWidth || rows > 67_108_864/cellHeight {
		return nil, fmt.Errorf("reduced sheet exceeds pixel limits")
	}
	width := columns * cellWidth
	height := rows * cellHeight
	if width > 67_108_864/height {
		return nil, fmt.Errorf("reduced sheet exceeds pixel limits")
	}

	bounds := src.Bounds()
	if bounds.Dx() < columns*cellWidth || bounds.Dy() < rows*cellHeight {
		return nil, fmt.Errorf("source cells must be at least %dx%d pixels", cellWidth, cellHeight)
	}
	if bounds.Dx()%columns != 0 || bounds.Dy()%rows != 0 {
		return nil, fmt.Errorf("source %dx%d does not divide into %dx%d cells", bounds.Dx(), bounds.Dy(), columns, rows)
	}

	result := image.NewNRGBA(image.Rect(0, 0, width, height))
	sourceCellWidth := bounds.Dx() / columns
	sourceCellHeight := bounds.Dy() / rows
	for cellY := range rows {
		for cellX := range columns {
			cell := image.Rect(
				bounds.Min.X+cellX*sourceCellWidth,
				bounds.Min.Y+cellY*sourceCellHeight,
				bounds.Min.X+(cellX+1)*sourceCellWidth,
				bounds.Min.Y+(cellY+1)*sourceCellHeight,
			)
			for y := range cellHeight {
				for x := range cellWidth {
					sample := image.Rect(
						cell.Min.X+x*cell.Dx()/cellWidth,
						cell.Min.Y+y*cell.Dy()/cellHeight,
						cell.Min.X+(x+1)*cell.Dx()/cellWidth,
						cell.Min.Y+(y+1)*cell.Dy()/cellHeight,
					)
					result.SetNRGBA(cellX*cellWidth+x, cellY*cellHeight+y, averageCell(src, sample))
				}
			}
		}
	}
	return result, nil
}

// EnlargeNearest scales src by an integer without interpolation.
func EnlargeNearest(src *image.NRGBA, scale int) (*image.NRGBA, error) {
	if scale < 1 || scale > 64 {
		return nil, fmt.Errorf("preview scale must be between 1 and 64")
	}
	bounds := src.Bounds()
	if bounds.Dx() > 67_108_864/scale || bounds.Dy() > 67_108_864/scale {
		return nil, fmt.Errorf("preview exceeds pixel limits")
	}
	width := bounds.Dx() * scale
	height := bounds.Dy() * scale
	if width > 67_108_864/height {
		return nil, fmt.Errorf("preview exceeds pixel limits")
	}

	result := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range bounds.Dy() {
		for x := range bounds.Dx() {
			value := src.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			for destinationY := y * scale; destinationY < (y+1)*scale; destinationY++ {
				for destinationX := x * scale; destinationX < (x+1)*scale; destinationX++ {
					result.SetNRGBA(destinationX, destinationY, value)
				}
			}
		}
	}
	return result, nil
}

func averageCell(src image.Image, bounds image.Rectangle) color.NRGBA {
	var red, green, blue, count uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := src.At(x, y).RGBA()
			red += uint64(r >> 8)
			green += uint64(g >> 8)
			blue += uint64(b >> 8)
			count++
		}
	}
	return color.NRGBA{
		R: uint8(red / count),
		G: uint8(green / count),
		B: uint8(blue / count),
		A: 0xff,
	}
}
