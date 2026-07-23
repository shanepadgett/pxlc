// Package syntax tokenizes and parses PXLC source without applying semantic rules.
package syntax

import "github.com/shanepadgett/pxlc/internal/diagnostic"

// Value is source text with its location.
type Value struct {
	Text string
	Span diagnostic.Span
}

// Number is an integer spelling with its location.
type Number = Value

// Document is the unvalidated declaration tree for one source file.
type Document struct {
	Version     Number
	Assets      []Value
	Canvases    []Canvas
	Palettes    []Palette
	Backgrounds []Background
	Layers      []Layer
}

// Canvas declares logical image dimensions.
type Canvas struct {
	Width  Number
	Height Number
	Span   diagnostic.Span
}

// Palette declares named colors and their grid symbols.
type Palette struct {
	Name    Value
	Maximum *Number
	Entries []PaletteEntry
	Span    diagnostic.Span
}

// PaletteEntry declares an opaque or transparent color.
type PaletteEntry struct {
	Name        Value
	Symbol      Value
	Hex         *Value
	Transparent bool
	Span        diagnostic.Span
}

// Background selects a palette color for untouched canvas pixels.
type Background struct {
	Palette Value
	Color   Value
	Span    diagnostic.Span
}

// Layer declares an ordered group of drawing operations.
type Layer struct {
	Name       Value
	Palette    Value
	Operations []Operation
	Span       diagnostic.Span
}

// OperationKind identifies a Phase 1 drawing primitive.
type OperationKind uint8

const (
	// OperationPixel draws one pixel.
	OperationPixel OperationKind = iota + 1
	// OperationHSpan draws a left-to-right span.
	OperationHSpan
	// OperationVSpan draws a top-to-bottom span.
	OperationVSpan
	// OperationRect draws a filled rectangle.
	OperationRect
	// OperationGrid draws a rectangular literal grid.
	OperationGrid
)

// Operation is an unvalidated drawing declaration.
type Operation struct {
	Kind   OperationKind
	X      Number
	Y      Number
	Width  Number
	Height Number
	Color  Value
	Rows   []Value
	Span   diagnostic.Span
}
