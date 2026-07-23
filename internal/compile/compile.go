// Package compile validates PXLC declarations and lowers them into concrete raster plans.
package compile

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"

	"github.com/shanepadgett/pxlc/internal/diagnostic"
	"github.com/shanepadgett/pxlc/internal/raster"
	"github.com/shanepadgett/pxlc/internal/syntax"
)

const sourceFormatVersion = 1

const maximumDiagnostics = 100

// Limits bounds work and allocation caused by untrusted source.
type Limits struct {
	MaximumSourceBytes   int
	MaximumTokens        int
	MaximumDimension     int
	MaximumCanvasPixels  int
	MaximumPalettes      int
	MaximumPaletteColors int
	MaximumLayers        int
	MaximumOperations    int
	MaximumPaintedPixels int
	MaximumPreviewPixels int
}

// DefaultLimits returns the fixed safety policy for standalone Phase 1 builds.
func DefaultLimits() Limits {
	return Limits{
		MaximumSourceBytes:   8 << 20,
		MaximumTokens:        1_000_000,
		MaximumDimension:     4_096,
		MaximumCanvasPixels:  16_777_216,
		MaximumPalettes:      256,
		MaximumPaletteColors: 256,
		MaximumLayers:        256,
		MaximumOperations:    1_000_000,
		MaximumPaintedPixels: 67_108_864,
		MaximumPreviewPixels: 67_108_864,
	}
}

// Source is one complete PXLC input with a stable diagnostic path.
type Source struct {
	Path         string
	ArtifactPath string
	Data         []byte
}

// Color records a declared palette entry.
type Color struct {
	Name   string
	Symbol byte
	RGBA   raster.Color
}

// Palette records a validated source palette in declaration order.
type Palette struct {
	Name   string
	Colors []Color
}

// Asset is a validated still asset and its concrete raster plan.
type Asset struct {
	Name          string
	NameSpan      diagnostic.Span
	Width         int
	Height        int
	Palettes      []Palette
	Plan          raster.Plan
	SourcePath    string
	SourceHash    [sha256.Size]byte
	FormatVersion int
}

// SourceFormatVersion returns the source version understood by this compiler.
func SourceFormatVersion() int {
	return sourceFormatVersion
}

// Compile parses, validates, and lowers one source file.
func Compile(source Source, limits Limits) (*Asset, []diagnostic.Diagnostic) {
	if len(source.Data) > limits.MaximumSourceBytes {
		span := startSpan(source.Path)
		d := diagnostic.Error(span, "PXLC-E024", fmt.Sprintf("source exceeds the limit of %d bytes", limits.MaximumSourceBytes))
		return nil, []diagnostic.Diagnostic{d}
	}
	doc, diagnostics := syntax.Parse(source.Path, source.Data, limits.MaximumTokens)
	if len(diagnostics) != 0 {
		return nil, diagnostics
	}
	v := validator{source: source, limits: limits, doc: doc}
	asset := v.validate()
	diagnostic.Sort(v.diagnostics)
	if len(v.diagnostics) != 0 {
		return nil, v.diagnostics
	}
	return asset, nil
}

type validator struct {
	source                  Source
	limits                  Limits
	doc                     *syntax.Document
	diagnostics             []diagnostic.Diagnostic
	width                   int
	height                  int
	paintedPixel            int
	diagnosticLimitReported bool
}

type validatedPalette struct {
	metadata Palette
	byName   map[string]raster.Color
	bySymbol map[byte]raster.Color
}

func (v *validator) validate() *Asset {
	v.validateVersion()
	name := v.validateAssetDeclaration()
	v.validateCanvasDeclaration()
	palettes := v.validatePalettes()
	background := v.validateBackground(palettes)
	layers := v.validateLayers(palettes)

	metadataPalettes := make([]Palette, 0, len(palettes))
	for _, declaration := range v.doc.Palettes {
		if palette, ok := palettes[declaration.Name.Text]; ok {
			metadataPalettes = append(metadataPalettes, palette.metadata)
		}
	}
	return &Asset{
		Name:          name,
		NameSpan:      v.assetNameSpan(),
		Width:         v.width,
		Height:        v.height,
		Palettes:      metadataPalettes,
		Plan:          raster.Plan{Width: v.width, Height: v.height, Background: background, Layers: layers},
		SourcePath:    v.artifactSourcePath(),
		SourceHash:    sha256.Sum256(v.source.Data),
		FormatVersion: sourceFormatVersion,
	}
}

func (v *validator) artifactSourcePath() string {
	if v.source.ArtifactPath != "" {
		return v.source.ArtifactPath
	}
	return v.source.Path
}

func (v *validator) assetNameSpan() diagnostic.Span {
	if len(v.doc.Assets) == 0 {
		return startSpan(v.source.Path)
	}
	return v.doc.Assets[0].Span
}

func (v *validator) validateVersion() {
	version, err := strconv.Atoi(v.doc.Version.Text)
	if err != nil || version != sourceFormatVersion {
		v.error(v.doc.Version.Span, "PXLC-E002", fmt.Sprintf("unsupported source-format version %q; expected %d", v.doc.Version.Text, sourceFormatVersion))
	}
}

func (v *validator) validateAssetDeclaration() string {
	if len(v.doc.Assets) == 0 {
		v.error(startSpan(v.source.Path), "PXLC-E003", "missing asset declaration")
		return ""
	}
	for _, declaration := range v.doc.Assets[1:] {
		v.error(declaration.Span, "PXLC-E010", "duplicate asset declaration")
	}
	v.validateName(v.doc.Assets[0], "asset")
	return v.doc.Assets[0].Text
}

func (v *validator) validateCanvasDeclaration() {
	if len(v.doc.Canvases) == 0 {
		v.error(startSpan(v.source.Path), "PXLC-E003", "missing canvas declaration")
		return
	}
	for _, declaration := range v.doc.Canvases[1:] {
		v.error(declaration.Span, "PXLC-E010", "duplicate canvas declaration")
	}
	canvas := v.doc.Canvases[0]
	width, widthOK := v.positiveInteger(canvas.Width, "canvas width")
	height, heightOK := v.positiveInteger(canvas.Height, "canvas height")
	if !widthOK || !heightOK {
		return
	}
	if width > v.limits.MaximumDimension || height > v.limits.MaximumDimension {
		v.error(canvas.Span, "PXLC-E020", fmt.Sprintf("canvas dimensions exceed the %d pixel dimension limit", v.limits.MaximumDimension))
		return
	}
	if width > v.limits.MaximumCanvasPixels/height {
		v.error(canvas.Span, "PXLC-E024", fmt.Sprintf("canvas exceeds the limit of %d pixels", v.limits.MaximumCanvasPixels))
		return
	}
	v.width = width
	v.height = height
}

func (v *validator) validatePalettes() map[string]validatedPalette {
	palettes := make(map[string]validatedPalette, min(len(v.doc.Palettes), v.limits.MaximumPalettes))
	if len(v.doc.Palettes) == 0 {
		v.error(startSpan(v.source.Path), "PXLC-E003", "at least one palette is required")
		return palettes
	}
	if len(v.doc.Palettes) > v.limits.MaximumPalettes {
		v.error(v.doc.Palettes[v.limits.MaximumPalettes].Span, "PXLC-E024", fmt.Sprintf("asset exceeds the limit of %d palettes", v.limits.MaximumPalettes))
	}
	for paletteIndex, declaration := range v.doc.Palettes {
		if paletteIndex >= v.limits.MaximumPalettes {
			continue
		}
		v.validateName(declaration.Name, "palette")
		if _, exists := palettes[declaration.Name.Text]; exists {
			v.error(declaration.Name.Span, "PXLC-E010", fmt.Sprintf("duplicate palette %q", declaration.Name.Text))
			continue
		}
		palette := v.validatePalette(declaration)
		palettes[declaration.Name.Text] = palette
	}
	return palettes
}

func (v *validator) validatePalette(declaration syntax.Palette) validatedPalette {
	palette := validatedPalette{
		metadata: Palette{Name: declaration.Name.Text, Colors: make([]Color, 0, min(len(declaration.Entries), v.limits.MaximumPaletteColors))},
		byName:   make(map[string]raster.Color, min(len(declaration.Entries), v.limits.MaximumPaletteColors)),
		bySymbol: make(map[byte]raster.Color, min(len(declaration.Entries), v.limits.MaximumPaletteColors)),
	}
	if len(declaration.Entries) == 0 {
		v.error(declaration.Span, "PXLC-E021", fmt.Sprintf("palette %q is empty", declaration.Name.Text))
	}
	maximum := v.limits.MaximumPaletteColors
	if declaration.Maximum != nil {
		declaredMaximum, ok := v.positiveInteger(*declaration.Maximum, "palette maximum")
		if ok {
			if declaredMaximum > v.limits.MaximumPaletteColors {
				v.error(declaration.Maximum.Span, "PXLC-E024", fmt.Sprintf("palette maximum exceeds the compiler limit of %d", v.limits.MaximumPaletteColors))
			} else {
				maximum = declaredMaximum
			}
		}
	}
	if len(declaration.Entries) > maximum {
		v.error(declaration.Span, "PXLC-E021", fmt.Sprintf("palette %q has %d entries; maximum is %d", declaration.Name.Text, len(declaration.Entries), maximum))
	}
	for entryIndex, entry := range declaration.Entries {
		if entryIndex >= v.limits.MaximumPaletteColors {
			continue
		}
		v.validateName(entry.Name, "color")
		color := raster.Color{}
		if entry.Transparent {
			color = raster.Color{A: 0}
		} else if entry.Hex != nil {
			parsed, ok := parseHexColor(entry.Hex.Text)
			if !ok {
				v.error(entry.Hex.Span, "PXLC-E021", fmt.Sprintf("invalid color %q; expected #RRGGBB", entry.Hex.Text))
			} else {
				color = parsed
			}
		}
		if _, exists := palette.byName[entry.Name.Text]; exists {
			v.error(entry.Name.Span, "PXLC-E010", fmt.Sprintf("duplicate color %q in palette %q", entry.Name.Text, declaration.Name.Text))
			continue
		}
		if len(entry.Symbol.Text) != 1 || entry.Symbol.Text[0] < 0x20 || entry.Symbol.Text[0] > 0x7e {
			v.error(entry.Symbol.Span, "PXLC-E021", "palette symbols must be one printable ASCII character")
			continue
		}
		symbol := entry.Symbol.Text[0]
		if _, exists := palette.bySymbol[symbol]; exists {
			v.error(entry.Symbol.Span, "PXLC-E010", fmt.Sprintf("duplicate symbol %q in palette %q", entry.Symbol.Text, declaration.Name.Text))
			continue
		}
		palette.byName[entry.Name.Text] = color
		palette.bySymbol[symbol] = color
		palette.metadata.Colors = append(palette.metadata.Colors, Color{Name: entry.Name.Text, Symbol: symbol, RGBA: color})
	}
	return palette
}

func (v *validator) validateBackground(palettes map[string]validatedPalette) raster.Color {
	if len(v.doc.Backgrounds) == 0 {
		v.error(startSpan(v.source.Path), "PXLC-E003", "missing background declaration")
		return raster.Color{}
	}
	for _, declaration := range v.doc.Backgrounds[1:] {
		v.error(declaration.Span, "PXLC-E010", "duplicate background declaration")
	}
	declaration := v.doc.Backgrounds[0]
	palette, ok := palettes[declaration.Palette.Text]
	if !ok {
		v.error(declaration.Palette.Span, "PXLC-E022", fmt.Sprintf("undeclared palette %q", declaration.Palette.Text))
		return raster.Color{}
	}
	color, ok := palette.byName[declaration.Color.Text]
	if !ok {
		v.error(declaration.Color.Span, "PXLC-E022", fmt.Sprintf("undeclared color %q in palette %q", declaration.Color.Text, declaration.Palette.Text))
		return raster.Color{}
	}
	return color
}

func (v *validator) validateLayers(palettes map[string]validatedPalette) []raster.Layer {
	if len(v.doc.Layers) == 0 {
		v.error(startSpan(v.source.Path), "PXLC-E003", "at least one layer is required")
		return nil
	}
	if len(v.doc.Layers) > v.limits.MaximumLayers {
		v.error(v.doc.Layers[v.limits.MaximumLayers].Span, "PXLC-E024", fmt.Sprintf("asset exceeds the limit of %d layers", v.limits.MaximumLayers))
	}
	seen := make(map[string]struct{}, min(len(v.doc.Layers), v.limits.MaximumLayers))
	layers := make([]raster.Layer, 0, min(len(v.doc.Layers), v.limits.MaximumLayers))
	operationCount := 0
	for layerIndex, declaration := range v.doc.Layers {
		if layerIndex >= v.limits.MaximumLayers {
			continue
		}
		v.validateName(declaration.Name, "layer")
		if _, exists := seen[declaration.Name.Text]; exists {
			v.error(declaration.Name.Span, "PXLC-E010", fmt.Sprintf("duplicate layer %q", declaration.Name.Text))
		} else {
			seen[declaration.Name.Text] = struct{}{}
		}
		palette, ok := palettes[declaration.Palette.Text]
		if !ok {
			v.error(declaration.Palette.Span, "PXLC-E022", fmt.Sprintf("undeclared palette %q", declaration.Palette.Text))
		}
		remainingOperations := max(v.limits.MaximumOperations-operationCount, 0)
		layer := raster.Layer{Operations: make([]raster.Operation, 0, min(len(declaration.Operations), remainingOperations))}
		for _, operation := range declaration.Operations {
			operationCount++
			if operationCount > v.limits.MaximumOperations {
				v.error(operation.Span, "PXLC-E024", fmt.Sprintf("asset exceeds the limit of %d drawing operations", v.limits.MaximumOperations))
				break
			}
			if ok {
				if lowered, valid := v.validateOperation(operation, palette); valid {
					layer.Operations = append(layer.Operations, lowered)
				}
			}
		}
		layers = append(layers, layer)
	}
	return layers
}

func (v *validator) validateOperation(operation syntax.Operation, palette validatedPalette) (raster.Operation, bool) {
	x, xOK := v.integer(operation.X, "x coordinate")
	y, yOK := v.integer(operation.Y, "y coordinate")
	if !xOK || !yOK {
		return raster.Operation{}, false
	}
	if operation.Kind == syntax.OperationGrid {
		return v.validateGrid(operation, palette, x, y)
	}
	color, colorOK := palette.byName[operation.Color.Text]
	if !colorOK {
		v.error(operation.Color.Span, "PXLC-E022", fmt.Sprintf("undeclared color %q in palette %q", operation.Color.Text, palette.metadata.Name))
	}
	width, height := 1, 1
	sizeOK := true
	switch operation.Kind {
	case syntax.OperationHSpan:
		width, sizeOK = v.positiveInteger(operation.Width, "span length")
	case syntax.OperationVSpan:
		height, sizeOK = v.positiveInteger(operation.Width, "span length")
	case syntax.OperationRect:
		width, sizeOK = v.positiveInteger(operation.Width, "rectangle width")
		var heightOK bool
		height, heightOK = v.positiveInteger(operation.Height, "rectangle height")
		sizeOK = sizeOK && heightOK
	}
	if !sizeOK || !v.validExtent(operation.Span, x, y, width, height) {
		return raster.Operation{}, false
	}
	if !v.addPaintedPixels(operation.Span, width, height) {
		return raster.Operation{}, false
	}
	return raster.Operation{Kind: raster.OperationRect, X: x, Y: y, Width: width, Height: height, Color: color}, colorOK
}

func (v *validator) validateGrid(operation syntax.Operation, palette validatedPalette, x, y int) (raster.Operation, bool) {
	if len(operation.Rows) == 0 {
		v.error(operation.Span, "PXLC-E025", "grid must contain at least one row")
		return raster.Operation{}, false
	}
	width := len(operation.Rows[0].Text)
	if width == 0 {
		v.error(operation.Rows[0].Span, "PXLC-E025", "grid rows must not be empty")
		return raster.Operation{}, false
	}
	height := len(operation.Rows)
	if !v.validExtent(operation.Span, x, y, width, height) {
		return raster.Operation{}, false
	}
	if width > int(^uint(0)>>1)/height {
		return raster.Operation{}, false
	}
	valid := true
	unknownSymbols := make(map[byte]diagnostic.Span)
	unknownOrder := make([]byte, 0)
	for _, row := range operation.Rows {
		if len(row.Text) != width {
			v.error(row.Span, "PXLC-E025", fmt.Sprintf("grid row has width %d; expected %d", len(row.Text), width))
			valid = false
			continue
		}
		for i := range len(row.Text) {
			_, ok := palette.bySymbol[row.Text[i]]
			if !ok {
				valid = false
				if _, reported := unknownSymbols[row.Text[i]]; !reported {
					unknownSymbols[row.Text[i]] = row.Span
					unknownOrder = append(unknownOrder, row.Text[i])
				}
			}
		}
	}
	for _, symbol := range unknownOrder {
		v.error(unknownSymbols[symbol], "PXLC-E022", fmt.Sprintf("grid symbol %q is undeclared in palette %q", string(symbol), palette.metadata.Name))
	}
	if !valid {
		return raster.Operation{}, false
	}
	if !v.addPaintedPixels(operation.Span, width, height) {
		return raster.Operation{}, false
	}
	pixels := make([]raster.Color, width*height)
	for rowIndex, row := range operation.Rows {
		for column := range len(row.Text) {
			pixels[rowIndex*width+column] = palette.bySymbol[row.Text[column]]
		}
	}
	return raster.Operation{Kind: raster.OperationGrid, X: x, Y: y, Width: width, Height: height, Pixels: pixels}, true
}

func (v *validator) validExtent(span diagnostic.Span, x, y, width, height int) bool {
	if x < 0 || y < 0 || width <= 0 || height <= 0 || x > v.width-width || y > v.height-height {
		v.error(span, "PXLC-E023", fmt.Sprintf("drawing extent (%d, %d) %dx%d is outside the %dx%d canvas", x, y, width, height, v.width, v.height))
		return false
	}
	return true
}

func (v *validator) addPaintedPixels(span diagnostic.Span, width, height int) bool {
	if width > v.limits.MaximumPaintedPixels/height || v.paintedPixel > v.limits.MaximumPaintedPixels-width*height {
		v.error(span, "PXLC-E024", fmt.Sprintf("asset exceeds the limit of %d painted pixels", v.limits.MaximumPaintedPixels))
		return false
	}
	v.paintedPixel += width * height
	return true
}

func (v *validator) validateName(value syntax.Value, kind string) {
	if !isName(value.Text) {
		v.error(value.Span, "PXLC-E011", fmt.Sprintf("invalid %s name %q", kind, value.Text))
	}
}

func (v *validator) positiveInteger(value syntax.Number, label string) (int, bool) {
	n, ok := v.integer(value, label)
	if ok && n <= 0 {
		v.error(value.Span, "PXLC-E020", label+" must be greater than zero")
		return 0, false
	}
	return n, ok
}

func (v *validator) integer(value syntax.Number, label string) (int, bool) {
	n, err := strconv.Atoi(value.Text)
	if err != nil {
		v.error(value.Span, "PXLC-E001", fmt.Sprintf("%s must be an integer", label))
		return 0, false
	}
	return n, true
}

func (v *validator) error(span diagnostic.Span, code, message string) {
	if len(v.diagnostics) >= maximumDiagnostics {
		if !v.diagnosticLimitReported {
			v.diagnosticLimitReported = true
			v.diagnostics = append(v.diagnostics, diagnostic.Error(span, "PXLC-E024", fmt.Sprintf("source exceeds the limit of %d diagnostics", maximumDiagnostics)))
		}
		return
	}
	v.diagnostics = append(v.diagnostics, diagnostic.Error(span, code, message))
}

func isName(s string) bool {
	if s == "" || !isNameStart(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isNameContinue(s[i]) {
			return false
		}
	}
	return true
}

func isNameStart(b byte) bool {
	return b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

func isNameContinue(b byte) bool {
	return isNameStart(b) || b >= '0' && b <= '9' || b == '-'
}

func parseHexColor(s string) (raster.Color, bool) {
	if len(s) != 7 || s[0] != '#' {
		return raster.Color{}, false
	}
	value, err := strconv.ParseUint(s[1:], 16, 24)
	if err != nil {
		return raster.Color{}, false
	}
	return raster.Color{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: 255}, true
}

func startSpan(path string) diagnostic.Span {
	position := diagnostic.Position{Line: 1, Column: 1}
	return diagnostic.Span{Path: path, Start: position, End: position}
}

func formatRGBA(color raster.Color) string {
	return fmt.Sprintf("#%02x%02x%02x%02x", color.R, color.G, color.B, color.A)
}

// FormatRGBA returns a canonical metadata spelling for a palette color.
func FormatRGBA(color raster.Color) string {
	return strings.ToLower(formatRGBA(color))
}
