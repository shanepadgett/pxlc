// Command pxlc-crush converts an isolated concept image into PXLC grid source.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shanepadgett/pxlc/internal/crush"
)

const maxSourcePixels = 67_108_864

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("pxlc-crush", flag.ContinueOnError)
	flags.SetOutput(stderr)
	asset := flags.String("asset", "", "PXLC asset name")
	width := flags.Int("width", 64, "logical canvas width")
	height := flags.Int("height", 96, "logical canvas height")
	backgroundText := flags.String("background", "#00ff00", "flat source background as #RRGGBB")
	profile := flags.String("profile", string(crush.ProfileConcept), "conversion profile: concept or sprite")
	flags.Usage = func() {
		if _, err := fmt.Fprintln(stderr, "usage: pxlc-crush [options] INPUT.(png|jpg) OUTPUT.pxl"); err != nil {
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
	if flags.NArg() != 2 || *asset == "" {
		flags.Usage()
		return 2
	}
	background, err := parseColor(*backgroundText)
	if err != nil {
		return report(stderr, err, 2)
	}

	inputPath := flags.Arg(0)
	outputPath := flags.Arg(1)
	input, err := os.Open(inputPath)
	if err != nil {
		return report(stderr, fmt.Errorf("open input: %w", err), 1)
	}
	config, _, err := image.DecodeConfig(input)
	closeErr := input.Close()
	if err != nil {
		return report(stderr, fmt.Errorf("inspect input: %w", err), 1)
	}
	if closeErr != nil {
		return report(stderr, fmt.Errorf("close input: %w", closeErr), 1)
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > maxSourcePixels/config.Height {
		return report(stderr, fmt.Errorf("input exceeds %d pixels", maxSourcePixels), 1)
	}

	input, err = os.Open(inputPath)
	if err != nil {
		return report(stderr, fmt.Errorf("reopen input: %w", err), 1)
	}
	decoded, _, err := image.Decode(input)
	closeErr = input.Close()
	if err != nil {
		return report(stderr, fmt.Errorf("decode input: %w", err), 1)
	}
	if closeErr != nil {
		return report(stderr, fmt.Errorf("close input: %w", closeErr), 1)
	}

	source, err := crush.Convert(decoded, *asset, crush.Options{
		Width:      *width,
		Height:     *height,
		Background: background,
		Profile:    crush.Profile(*profile),
	})
	if err != nil {
		return report(stderr, fmt.Errorf("crush image: %w", err), 1)
	}
	if err := writeFile(outputPath, source); err != nil {
		return report(stderr, err, 1)
	}
	if _, err := fmt.Fprintf(stdout, "wrote %s (%dx%d)\n", outputPath, *width, *height); err != nil {
		return 1
	}
	return 0
}

func parseColor(value string) (color.NRGBA, error) {
	if len(value) != 7 || value[0] != '#' {
		return color.NRGBA{}, fmt.Errorf("background must use #RRGGBB")
	}
	red, err := strconv.ParseUint(value[1:3], 16, 8)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("background must use #RRGGBB")
	}
	green, err := strconv.ParseUint(value[3:5], 16, 8)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("background must use #RRGGBB")
	}
	blue, err := strconv.ParseUint(value[5:7], 16, 8)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("background must use #RRGGBB")
	}
	return color.NRGBA{R: uint8(red), G: uint8(green), B: uint8(blue), A: 0xff}, nil
}

func writeFile(path string, contents []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".pxlc-crush-*")
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

	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write staged output: %w", err)
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

func report(stderr io.Writer, err error, status int) int {
	message := strings.TrimSpace(err.Error())
	if _, writeErr := fmt.Fprintln(stderr, message); writeErr != nil && !errors.Is(writeErr, os.ErrClosed) {
		return 1
	}
	return status
}
