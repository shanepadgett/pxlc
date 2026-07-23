// Package crush converts isolated raster concepts into constrained PXLC grids.
package crush

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
)

const (
	backgroundLimit = 12
)

// Profile selects conversion rules for a target asset scale.
type Profile string

const (
	// ProfileConcept preserves more source values for larger concept sprites.
	ProfileConcept Profile = "concept"
	// ProfileSprite flattens values and derives a silhouette outline for small sprites.
	ProfileSprite Profile = "sprite"
)

type swatch struct {
	name   string
	symbol byte
	color  color.NRGBA
}

var mortuaryPalette = []swatch{
	{name: "ink", symbol: 'I', color: color.NRGBA{R: 0x0d, G: 0x0d, B: 0x10, A: 0xff}},
	{name: "outline", symbol: 'K', color: color.NRGBA{R: 0x17, G: 0x16, B: 0x1a, A: 0xff}},
	{name: "deep", symbol: 'D', color: color.NRGBA{R: 0x24, G: 0x21, B: 0x26, A: 0xff}},
	{name: "coat-shadow", symbol: 'Q', color: color.NRGBA{R: 0x30, G: 0x2a, B: 0x31, A: 0xff}},
	{name: "coat", symbol: 'C', color: color.NRGBA{R: 0x44, G: 0x3b, B: 0x42, A: 0xff}},
	{name: "coat-light", symbol: 'H', color: color.NRGBA{R: 0x5b, G: 0x4d, B: 0x50, A: 0xff}},
	{name: "steel-dark", symbol: 'T', color: color.NRGBA{R: 0x49, G: 0x47, B: 0x44, A: 0xff}},
	{name: "steel", symbol: 'S', color: color.NRGBA{R: 0x77, G: 0x73, B: 0x6b, A: 0xff}},
	{name: "steel-light", symbol: 'L', color: color.NRGBA{R: 0xaa, G: 0xa3, B: 0x97, A: 0xff}},
	{name: "linen-dark", symbol: 'A', color: color.NRGBA{R: 0x81, G: 0x76, B: 0x6a, A: 0xff}},
	{name: "linen", symbol: 'B', color: color.NRGBA{R: 0xb4, G: 0xa5, B: 0x8f, A: 0xff}},
	{name: "dried-blood", symbol: 'R', color: color.NRGBA{R: 0x71, G: 0x34, B: 0x3a, A: 0xff}},
	{name: "lamp", symbol: 'G', color: color.NRGBA{R: 0xa6, G: 0x92, B: 0x5f, A: 0xff}},
	{name: "rust", symbol: 'X', color: color.NRGBA{R: 0x64, G: 0x46, B: 0x3d, A: 0xff}},
}

var spritePalette = []swatch{
	{name: "ink", symbol: 'I', color: color.NRGBA{R: 0x12, G: 0x11, B: 0x16, A: 0xff}},
	{name: "garment-dark", symbol: 'D', color: color.NRGBA{R: 0x35, G: 0x2e, B: 0x39, A: 0xff}},
	{name: "garment", symbol: 'C', color: color.NRGBA{R: 0x50, G: 0x43, B: 0x4d, A: 0xff}},
	{name: "garment-light", symbol: 'H', color: color.NRGBA{R: 0x70, G: 0x5b, B: 0x62, A: 0xff}},
	{name: "bone", symbol: 'B', color: color.NRGBA{R: 0xb9, G: 0xa5, B: 0x8e, A: 0xff}},
	{name: "dried-blood", symbol: 'R', color: color.NRGBA{R: 0x75, G: 0x3c, B: 0x3a, A: 0xff}},
	{name: "leather", symbol: 'X', color: color.NRGBA{R: 0x4a, G: 0x37, B: 0x2f, A: 0xff}},
}

// Options controls the logical PXLC canvas and source-background removal.
type Options struct {
	Width      int
	Height     int
	Background color.NRGBA
	Profile    Profile
}

// Convert fits the non-background image bounds into a limited-palette PXLC grid.
func Convert(src image.Image, asset string, options Options) ([]byte, error) {
	if err := validateName(asset); err != nil {
		return nil, err
	}
	configuration, err := configurationFor(options.Profile)
	if err != nil {
		return nil, err
	}
	if options.Width < configuration.margin*2+1 || options.Height < configuration.margin*2+1 {
		minimum := configuration.margin*2 + 1
		return nil, fmt.Errorf("canvas must be at least %dx%d for profile %q", minimum, minimum, configuration.name)
	}
	if options.Width > 4096 || options.Height > 4096 || options.Width > 16_777_216/options.Height {
		return nil, fmt.Errorf("canvas exceeds PXLC source-format limits")
	}

	occupied, ok := occupiedBounds(src, options.Background)
	if !ok {
		return nil, fmt.Errorf("image contains no pixels distinct from the background")
	}

	grid := makeGrid(options.Width, options.Height)
	fitImage(src, occupied, grid, options.Background, configuration)
	fillPinholes(grid)
	smoothDarkOutliers(grid, 2)
	removeIsolated(grid, 2)
	consolidateSingletons(grid, configuration.name == ProfileSprite)
	if configuration.outline {
		deriveOutline(grid)
	}

	return encode(asset, grid, configuration), nil
}

type configuration struct {
	name    Profile
	margin  int
	palette []swatch
	outline bool
	expand  bool
}

func configurationFor(profile Profile) (configuration, error) {
	switch profile {
	case "", ProfileConcept:
		return configuration{name: ProfileConcept, margin: 2, palette: mortuaryPalette, expand: true}, nil
	case ProfileSprite:
		return configuration{name: ProfileSprite, margin: 4, palette: spritePalette, outline: true}, nil
	default:
		return configuration{}, fmt.Errorf("unknown crush profile %q", profile)
	}
}

func validateName(name string) error {
	if name == "" || !nameStart(name[0]) {
		return fmt.Errorf("asset name %q does not match PXLC naming rules", name)
	}
	for i := 1; i < len(name); i++ {
		if !namePart(name[i]) {
			return fmt.Errorf("asset name %q does not match PXLC naming rules", name)
		}
	}
	return nil
}

func nameStart(value byte) bool {
	return value == '_' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func namePart(value byte) bool {
	return nameStart(value) || value == '-' || value >= '0' && value <= '9'
}

func occupiedBounds(src image.Image, background color.NRGBA) (image.Rectangle, bool) {
	bounds := src.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if isBackground(toNRGBA(src.At(x, y)), background) {
				continue
			}
			minX = min(minX, x)
			minY = min(minY, y)
			maxX = max(maxX, x+1)
			maxY = max(maxY, y+1)
		}
	}

	if minX == bounds.Max.X {
		return image.Rectangle{}, false
	}
	return image.Rect(minX, minY, maxX, maxY), true
}

func isBackground(value, background color.NRGBA) bool {
	if value.A < 0x80 {
		return true
	}
	if background.G >= 0xc8 && background.R <= 0x20 && background.B <= 0x20 &&
		value.G >= 0x50 && int(value.G)*2 >= int(value.R)*3 && int(value.G)*2 >= int(value.B)*3 {
		return true
	}
	dr := int(value.R) - int(background.R)
	dg := int(value.G) - int(background.G)
	db := int(value.B) - int(background.B)
	limit := backgroundLimit * backgroundLimit * 3
	return dr*dr+dg*dg+db*db <= limit
}

func makeGrid(width, height int) [][]byte {
	grid := make([][]byte, height)
	for y := range grid {
		grid[y] = bytes.Repeat([]byte{'.'}, width)
	}
	return grid
}

func fitImage(src image.Image, occupied image.Rectangle, grid [][]byte, background color.NRGBA, configuration configuration) {
	height := len(grid)
	width := len(grid[0])
	availableWidth := width - configuration.margin*2
	availableHeight := height - configuration.margin*2
	scale := math.Min(
		float64(availableWidth)/float64(occupied.Dx()),
		float64(availableHeight)/float64(occupied.Dy()),
	)
	destinationWidth := max(1, int(math.Round(float64(occupied.Dx())*scale)))
	destinationHeight := max(1, int(math.Round(float64(occupied.Dy())*scale)))
	offsetX := (width - destinationWidth) / 2
	offsetY := (height - destinationHeight) / 2

	for y := range destinationHeight {
		for x := range destinationWidth {
			sourceMinX := occupied.Min.X + x*occupied.Dx()/destinationWidth
			sourceMaxX := occupied.Min.X + (x+1)*occupied.Dx()/destinationWidth
			sourceMinY := occupied.Min.Y + y*occupied.Dy()/destinationHeight
			sourceMaxY := occupied.Min.Y + (y+1)*occupied.Dy()/destinationHeight
			cell := image.Rect(sourceMinX, sourceMinY, sourceMaxX, sourceMaxY)
			if configuration.name == ProfileSprite {
				symbol, ok := spriteSymbol(src, cell, background, configuration)
				if ok {
					grid[offsetY+y][offsetX+x] = symbol
				}
				continue
			}
			if isBackground(average(src, cell), background) {
				continue
			}

			sample := cell
			if configuration.expand {
				expandX := max(1, cell.Dx()/3)
				expandY := max(1, cell.Dy()/3)
				sample = image.Rect(
					cell.Min.X-expandX,
					cell.Min.Y-expandY,
					cell.Max.X+expandX,
					cell.Max.Y+expandY,
				).Intersect(occupied)
			}
			value, ok := averageForeground(src, sample, background)
			if !ok {
				continue
			}
			grid[offsetY+y][offsetX+x] = nearest(value, configuration)
		}
	}
}

func spriteSymbol(src image.Image, bounds image.Rectangle, background color.NRGBA, configuration configuration) (byte, bool) {
	counts := make(map[byte]int)
	total := bounds.Dx() * bounds.Dy()
	foreground := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			value := toNRGBA(src.At(x, y))
			if isBackground(value, background) {
				continue
			}
			foreground++
			counts[nearest(value, configuration)]++
		}
	}
	if foreground*2 < total {
		return '.', false
	}

	accentMinimum := max(1, foreground/5)
	for _, symbol := range []byte{'B', 'R'} {
		if counts[symbol] >= accentMinimum {
			return symbol, true
		}
	}

	best, bestCount := byte('.'), 0
	for _, candidate := range configuration.palette {
		if counts[candidate.symbol] > bestCount {
			best = candidate.symbol
			bestCount = counts[candidate.symbol]
		}
	}
	return best, bestCount > 0
}

func average(src image.Image, bounds image.Rectangle) color.NRGBA {
	var red, green, blue, alpha, count uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			value := toNRGBA(src.At(x, y))
			red += uint64(value.R)
			green += uint64(value.G)
			blue += uint64(value.B)
			alpha += uint64(value.A)
			count++
		}
	}
	if count == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: uint8(red / count),
		G: uint8(green / count),
		B: uint8(blue / count),
		A: uint8(alpha / count),
	}
}

func averageForeground(src image.Image, bounds image.Rectangle, background color.NRGBA) (color.NRGBA, bool) {
	var red, green, blue, alpha, count uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			value := toNRGBA(src.At(x, y))
			if isBackground(value, background) {
				continue
			}
			red += uint64(value.R)
			green += uint64(value.G)
			blue += uint64(value.B)
			alpha += uint64(value.A)
			count++
		}
	}
	if count == 0 {
		return color.NRGBA{}, false
	}
	return color.NRGBA{
		R: uint8(red / count),
		G: uint8(green / count),
		B: uint8(blue / count),
		A: uint8(alpha / count),
	}, true
}

func toNRGBA(value color.Color) color.NRGBA {
	converted, ok := color.NRGBAModel.Convert(value).(color.NRGBA)
	if !ok {
		panic("color.NRGBAModel returned an unexpected color type")
	}
	return converted
}

func nearest(value color.NRGBA, configuration configuration) byte {
	red := float64(value.R)
	green := float64(value.G)
	blue := float64(value.B)
	if configuration.name == ProfileConcept && green > red*1.22 && green > blue*1.12 && green > 65 {
		return 'G'
	}
	if configuration.name == ProfileConcept && red > 110 && green > 65 && blue < 60 && red > green*1.3 {
		return 'G'
	}
	if red > 80 && red > green*1.5 && red > blue*1.3 {
		return 'R'
	}

	bestSymbol := configuration.palette[0].symbol
	bestDistance := math.MaxFloat64
	for _, candidate := range configuration.palette {
		if candidate.symbol == 'I' || candidate.symbol == 'G' || candidate.symbol == 'R' {
			continue
		}
		deltaRed := red - float64(candidate.color.R)
		deltaGreen := green - float64(candidate.color.G)
		deltaBlue := blue - float64(candidate.color.B)
		distance := deltaRed*deltaRed*0.30 + deltaGreen*deltaGreen*0.59 + deltaBlue*deltaBlue*0.11
		if distance < bestDistance {
			bestDistance = distance
			bestSymbol = candidate.symbol
		}
	}
	return bestSymbol
}

func deriveOutline(grid [][]byte) {
	next := cloneGrid(grid)
	for y := 1; y < len(grid)-1; y++ {
		for x := 1; x < len(grid[y])-1; x++ {
			if !isGarment(grid[y][x]) {
				continue
			}
			if grid[y-1][x] == '.' || grid[y+1][x] == '.' || grid[y][x-1] == '.' || grid[y][x+1] == '.' {
				next[y][x] = 'I'
			}
		}
	}
	copyGrid(grid, next)
}

func isGarment(symbol byte) bool {
	return symbol == 'D' || symbol == 'C' || symbol == 'H'
}

func removeIsolated(grid [][]byte, passes int) {
	for range passes {
		next := cloneGrid(grid)
		for y := 1; y < len(grid)-1; y++ {
			for x := 1; x < len(grid[y])-1; x++ {
				if grid[y][x] == '.' {
					continue
				}
				if opaqueNeighbors(grid, x, y) <= 1 {
					next[y][x] = '.'
				}
			}
		}
		copyGrid(grid, next)
	}
}

func fillPinholes(grid [][]byte) {
	next := cloneGrid(grid)
	for y := 1; y < len(grid)-1; y++ {
		for x := 1; x < len(grid[y])-1; x++ {
			if grid[y][x] != '.' || opaqueNeighbors(grid, x, y) != 8 {
				continue
			}
			best, count := majorityNeighbor(grid, x, y, false)
			if count >= 5 {
				next[y][x] = best
			}
		}
	}
	copyGrid(grid, next)
}

func smoothDarkOutliers(grid [][]byte, passes int) {
	for range passes {
		next := cloneGrid(grid)
		for y := 1; y < len(grid)-1; y++ {
			for x := 1; x < len(grid[y])-1; x++ {
				if !isDark(grid[y][x]) || opaqueNeighbors(grid, x, y) != 8 {
					continue
				}
				darkNeighbors := 0
				for adjacentY := y - 1; adjacentY <= y+1; adjacentY++ {
					for adjacentX := x - 1; adjacentX <= x+1; adjacentX++ {
						if (adjacentX != x || adjacentY != y) && isDark(grid[adjacentY][adjacentX]) {
							darkNeighbors++
						}
					}
				}
				if darkNeighbors > 1 {
					continue
				}
				best, count := majorityNeighbor(grid, x, y, true)
				if count >= 4 {
					next[y][x] = best
				}
			}
		}
		copyGrid(grid, next)
	}
}

func isDark(symbol byte) bool {
	return symbol == 'I' || symbol == 'K' || symbol == 'D'
}

func majorityNeighbor(grid [][]byte, x, y int, excludeDark bool) (byte, int) {
	counts := make(map[byte]int)
	for adjacentY := y - 1; adjacentY <= y+1; adjacentY++ {
		for adjacentX := x - 1; adjacentX <= x+1; adjacentX++ {
			symbol := grid[adjacentY][adjacentX]
			if adjacentX == x && adjacentY == y || symbol == '.' || excludeDark && isDark(symbol) {
				continue
			}
			counts[symbol]++
		}
	}
	best, bestCount := byte('.'), 0
	for _, candidate := range mortuaryPalette {
		if counts[candidate.symbol] > bestCount {
			best = candidate.symbol
			bestCount = counts[candidate.symbol]
		}
	}
	return best, bestCount
}

func consolidateSingletons(grid [][]byte, preserveAccents bool) {
	next := cloneGrid(grid)
	for y := 1; y < len(grid)-1; y++ {
		for x := 1; x < len(grid[y])-1; x++ {
			current := grid[y][x]
			if current == '.' {
				continue
			}
			if preserveAccents && (current == 'B' || current == 'R') {
				continue
			}
			counts := make(map[byte]int)
			for adjacentY := y - 1; adjacentY <= y+1; adjacentY++ {
				for adjacentX := x - 1; adjacentX <= x+1; adjacentX++ {
					if adjacentX == x && adjacentY == y || grid[adjacentY][adjacentX] == '.' {
						continue
					}
					counts[grid[adjacentY][adjacentX]]++
				}
			}
			best, bestCount := current, 0
			for _, candidate := range mortuaryPalette {
				if counts[candidate.symbol] > bestCount {
					best = candidate.symbol
					bestCount = counts[candidate.symbol]
				}
			}
			if counts[current] == 0 && bestCount >= 4 {
				next[y][x] = best
			}
		}
	}
	copyGrid(grid, next)
}

func opaqueNeighbors(grid [][]byte, x, y int) int {
	count := 0
	for adjacentY := y - 1; adjacentY <= y+1; adjacentY++ {
		for adjacentX := x - 1; adjacentX <= x+1; adjacentX++ {
			if (adjacentX != x || adjacentY != y) && grid[adjacentY][adjacentX] != '.' {
				count++
			}
		}
	}
	return count
}

func cloneGrid(grid [][]byte) [][]byte {
	clone := make([][]byte, len(grid))
	for y := range grid {
		clone[y] = append([]byte(nil), grid[y]...)
	}
	return clone
}

func copyGrid(destination, source [][]byte) {
	for y := range destination {
		copy(destination[y], source[y])
	}
}

func encode(asset string, grid [][]byte, configuration configuration) []byte {
	var source []byte
	source = fmt.Appendf(source, "pxlc 1\n\nasset %s\ncanvas %d %d\n\n", asset, len(grid[0]), len(grid))
	source = fmt.Appendf(source, "palette %s max %d {\n", configuration.name, len(configuration.palette)+1)
	source = append(source, "  transparent clear \".\"\n"...)
	for _, value := range configuration.palette {
		source = fmt.Appendf(source, "  color %s %q #%02x%02x%02x\n", value.name, string(value.symbol), value.color.R, value.color.G, value.color.B)
	}
	source = fmt.Appendf(source, "}\n\nbackground %s clear\n\nlayer figure using %s {\n  grid 0 0 {\n", configuration.name, configuration.name)
	for _, row := range grid {
		source = fmt.Appendf(source, "    %q\n", string(row))
	}
	source = append(source, "  }\n}\n"...)
	return source
}
