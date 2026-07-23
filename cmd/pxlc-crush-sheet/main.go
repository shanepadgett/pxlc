// Command pxlc-crush-sheet reduces a generated raster sprite sheet to fixed logical cells.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shanepadgett/pxlc/internal/crush"
)

const maxSourcePixels = 67_108_864

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("pxlc-crush-sheet", flag.ContinueOnError)
	flags.SetOutput(stderr)
	columns := flags.Int("columns", 4, "number of source sheet columns")
	rows := flags.Int("rows", 3, "number of source sheet rows")
	cellSize := flags.Int("cell-size", 32, "logical width and height of each output cell")
	cellWidth := flags.Int("cell-width", 0, "logical cell width; overrides cell-size when positive")
	cellHeight := flags.Int("cell-height", 0, "logical cell height; overrides cell-size when positive")
	previewPath := flags.String("preview", "", "optional nearest-neighbor preview PNG")
	previewScale := flags.Int("preview-scale", 6, "integer preview scale")
	flags.Usage = func() {
		if _, err := fmt.Fprintln(stderr, "usage: pxlc-crush-sheet [options] INPUT.(png|jpg) OUTPUT.png"); err != nil {
			return
		}
		flags.PrintDefaults()
	}

	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 2 {
		flags.Usage()
		return 2
	}

	logicalWidth := *cellSize
	if *cellWidth > 0 {
		logicalWidth = *cellWidth
	}
	logicalHeight := *cellSize
	if *cellHeight > 0 {
		logicalHeight = *cellHeight
	}

	decoded, err := decodeImage(flags.Arg(0))
	if err != nil {
		return report(stderr, err)
	}
	reduced, err := crush.ReduceSheetCells(decoded, *columns, *rows, logicalWidth, logicalHeight)
	if err != nil {
		return report(stderr, fmt.Errorf("reduce sheet: %w", err))
	}
	if err := writePNG(flags.Arg(1), reduced); err != nil {
		return report(stderr, err)
	}

	if *previewPath != "" {
		preview, previewErr := crush.EnlargeNearest(reduced, *previewScale)
		if previewErr != nil {
			return report(stderr, fmt.Errorf("create preview: %w", previewErr))
		}
		if err := writePNG(*previewPath, preview); err != nil {
			return report(stderr, err)
		}
	}

	if _, err := fmt.Fprintf(stdout, "wrote %s (%dx%d cells at %dx%d)\n", flags.Arg(1), *columns, *rows, logicalWidth, logicalHeight); err != nil {
		return 1
	}
	return 0
}

func decodeImage(path string) (image.Image, error) {
	input, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open input: %w", err)
	}
	config, _, err := image.DecodeConfig(input)
	closeErr := input.Close()
	if err != nil {
		return nil, fmt.Errorf("inspect input: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close input: %w", closeErr)
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > maxSourcePixels/config.Height {
		return nil, fmt.Errorf("input exceeds %d pixels", maxSourcePixels)
	}

	input, err = os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reopen input: %w", err)
	}
	decoded, _, err := image.Decode(input)
	closeErr = input.Close()
	if err != nil {
		return nil, fmt.Errorf("decode input: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close input: %w", closeErr)
	}
	return decoded, nil
}

func writePNG(path string, value image.Image) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".pxlc-crush-sheet-*")
	if err != nil {
		return fmt.Errorf("create staged output: %w", err)
	}
	temporaryPath := temporary.Name()
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := png.Encode(temporary, value); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("encode staged output: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close staged output: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace output: %w", err)
	}
	remove = false
	return nil
}

func report(stderr io.Writer, err error) int {
	message := strings.TrimSpace(err.Error())
	if _, writeErr := fmt.Fprintln(stderr, message); writeErr != nil && !errors.Is(writeErr, os.ErrClosed) {
		return 1
	}
	return 1
}
