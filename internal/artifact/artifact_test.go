package artifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/shanepadgett/pxlc/internal/compile"
	"github.com/shanepadgett/pxlc/internal/raster"
)

func TestExampleArtifactsAreDeterministic(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("../../examples/icon.pxl")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	asset, diagnostics := compile.Compile(compile.Source{Path: "examples/icon.pxl", Data: source}, compile.DefaultLimits())
	if len(diagnostics) != 0 {
		t.Fatalf("Compile() diagnostics = %v", diagnostics)
	}
	img, err := raster.Render(asset.Plan)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	first, err := Runtime(asset, img, "icon", "test")
	if err != nil {
		t.Fatalf("Runtime() error = %v", err)
	}
	second, err := Runtime(asset, img, "icon", "test")
	if err != nil {
		t.Fatalf("Runtime() second error = %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("artifact counts differ: %d and %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Path != second[i].Path || !bytes.Equal(first[i].Data, second[i].Data) {
			t.Fatalf("artifact %d is not deterministic", i)
		}
	}

	const wantPNGHash = "75ded2307c4884cbbdeb554ef1bfd725cae9ad38443b9948be0877113dcb24db"
	if got := dataHash(first[0].Data); got != wantPNGHash {
		t.Fatalf("PNG SHA-256 = %s, want %s", got, wantPNGHash)
	}
}

func dataHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
