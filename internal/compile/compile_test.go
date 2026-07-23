package compile

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shanepadgett/pxlc/internal/raster"
)

func TestCompileAndRenderLayerSemantics(t *testing.T) {
	t.Parallel()
	source := []byte(`pxlc 1
asset sample
canvas 3 2
palette p {
  transparent clear "."
  color red "R" #ff0000
  color blue "B" #0000ff
}
background p clear
layer base using p {
  rect 0 0 3 2 red
  pixel 1 1 clear
}
layer top using p {
  hspan 0 0 2 blue
}
`)
	asset, diagnostics := Compile(Source{Path: "sample.pxl", Data: source}, DefaultLimits())
	if len(diagnostics) != 0 {
		t.Fatalf("Compile() diagnostics = %v", diagnostics)
	}
	img, err := raster.Render(asset.Plan)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	assertPixel(t, img.Pix, img.Stride, 0, 0, raster.Color{B: 255, A: 255})
	assertPixel(t, img.Pix, img.Stride, 2, 0, raster.Color{R: 255, A: 255})
	assertPixel(t, img.Pix, img.Stride, 1, 1, raster.Color{})
}

func TestDiagnosticLimitPreservesSourceOrder(t *testing.T) {
	t.Parallel()
	var source strings.Builder
	source.WriteString("pxlc 1\nasset sample\n")
	for i := range 99 {
		fmt.Fprintf(&source, "asset duplicate%d\n", i)
	}
	source.WriteString(`canvas 4 1
palette p { transparent clear "." }
background p clear
layer base using p { grid 0 0 { "ABCD" } }
`)
	_, diagnostics := Compile(Source{Path: "limited.pxl", Data: []byte(source.String())}, DefaultLimits())
	if len(diagnostics) != maximumDiagnostics+1 {
		t.Fatalf("diagnostic count = %d, want %d", len(diagnostics), maximumDiagnostics+1)
	}
	messages := make([]string, 0, len(diagnostics))
	for _, d := range diagnostics {
		messages = append(messages, d.Message)
	}
	joined := strings.Join(messages, "\n")
	if !strings.Contains(joined, `grid symbol "A"`) || strings.Contains(joined, `grid symbol "B"`) {
		t.Fatalf("diagnostics did not preserve first-encountered symbol:\n%s", joined)
	}
}

func TestCompileRejectsOutOfBoundsExtent(t *testing.T) {
	t.Parallel()
	source := []byte(`pxlc 1
asset bad
canvas 2 2
palette p { color red "R" #ff0000 }
background p red
layer base using p { rect 1 1 2 1 red }
`)
	asset, diagnostics := Compile(Source{Path: "bad.pxl", Data: source}, DefaultLimits())
	if asset != nil {
		t.Fatal("Compile() returned an asset for invalid source")
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "PXLC-E023" {
		t.Fatalf("Compile() diagnostics = %#v, want one PXLC-E023", diagnostics)
	}
}

func assertPixel(t *testing.T, pixels []byte, stride, x, y int, want raster.Color) {
	t.Helper()
	offset := y*stride + x*4
	got := raster.Color{R: pixels[offset], G: pixels[offset+1], B: pixels[offset+2], A: pixels[offset+3]}
	if got != want {
		t.Fatalf("pixel (%d, %d) = %#v, want %#v", x, y, got, want)
	}
}
