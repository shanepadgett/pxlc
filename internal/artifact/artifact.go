// Package artifact encodes and stages deterministic compiler output.
package artifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"path/filepath"
	"strings"

	"github.com/shanepadgett/pxlc/internal/compile"
)

const metadataSchemaVersion = 1

// File is one encoded artifact at a slash-separated path relative to an output root.
type File struct {
	Path string
	Data []byte
}

// MetadataSchemaVersion returns the generated metadata version.
func MetadataSchemaVersion() int {
	return metadataSchemaVersion
}

// PNG encodes an image with fixed settings and no incidental metadata.
func PNG(img image.Image) ([]byte, error) {
	var output bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&output, img); err != nil {
		return nil, fmt.Errorf("encode PNG: %w", err)
	}
	return output.Bytes(), nil
}

// Runtime encodes a still image and its versioned metadata.
func Runtime(asset *compile.Asset, img image.Image, relativeStem, compilerVersion string) ([]File, error) {
	pngData, err := PNG(img)
	if err != nil {
		return nil, err
	}
	imagePath := filepath.ToSlash(relativeStem + ".png")
	metadataPath := filepath.ToSlash(relativeStem + ".pxlc.json")
	metadata := metadataDocument{
		Schema:  "pxlc.asset",
		Version: metadataSchemaVersion,
		Asset:   asset.Name,
		Canvas: metadataCanvas{
			Width:  asset.Width,
			Height: asset.Height,
		},
		Image: filepath.Base(imagePath),
		Source: metadataSource{
			Path:          asset.SourcePath,
			FormatVersion: asset.FormatVersion,
			SHA256:        hex.EncodeToString(asset.SourceHash[:]),
		},
		Compiler: metadataCompiler{
			Version: compilerVersion,
		},
		Content: metadataContent{
			ImageSHA256: hashHex(pngData),
		},
		Palettes: make([]metadataPalette, 0, len(asset.Palettes)),
	}
	for _, palette := range asset.Palettes {
		encodedPalette := metadataPalette{Name: palette.Name, Colors: make([]metadataColor, 0, len(palette.Colors))}
		for _, color := range palette.Colors {
			encodedPalette.Colors = append(encodedPalette.Colors, metadataColor{
				Name:   color.Name,
				Symbol: string(color.Symbol),
				RGBA:   compile.FormatRGBA(color.RGBA),
			})
		}
		metadata.Palettes = append(metadata.Palettes, encodedPalette)
	}
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode metadata: %w", err)
	}
	metadataData = append(metadataData, '\n')
	return []File{{Path: imagePath, Data: pngData}, {Path: metadataPath, Data: metadataData}}, nil
}

type metadataDocument struct {
	Schema   string            `json:"schema"`
	Version  int               `json:"version"`
	Asset    string            `json:"asset"`
	Canvas   metadataCanvas    `json:"canvas"`
	Image    string            `json:"image"`
	Palettes []metadataPalette `json:"palettes"`
	Source   metadataSource    `json:"source"`
	Compiler metadataCompiler  `json:"compiler"`
	Content  metadataContent   `json:"content"`
}

type metadataCanvas struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type metadataPalette struct {
	Name   string          `json:"name"`
	Colors []metadataColor `json:"colors"`
}

type metadataColor struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	RGBA   string `json:"rgba"`
}

type metadataSource struct {
	Path          string `json:"path"`
	FormatVersion int    `json:"formatVersion"`
	SHA256        string `json:"sha256"`
}

type metadataCompiler struct {
	Version string `json:"version"`
}

type metadataContent struct {
	ImageSHA256 string `json:"imageSHA256"`
}

func hashHex(data []byte) string {
	hash := sha256.Sum256(data)
	return strings.ToLower(hex.EncodeToString(hash[:]))
}
