// Package cli implements the PXLC process boundary and command behavior.
package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/shanepadgett/pxlc/internal/artifact"
	"github.com/shanepadgett/pxlc/internal/compile"
	"github.com/shanepadgett/pxlc/internal/diagnostic"
	"github.com/shanepadgett/pxlc/internal/raster"
)

const (
	exitSuccess     = 0
	exitInvalid     = 1
	exitUsage       = 2
	exitOperational = 3
)

// Run executes one CLI invocation and returns its process exit status.
func Run(args []string, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 {
		if err := printUsage(stderr); err != nil {
			return exitOperational
		}
		return exitUsage
	}
	if len(args) == 1 && (args[0] == "--version" || args[0] == "version") {
		if err := writef(stdout, "pxlc %s (source format %d, metadata schema %d)\n", version, compile.SourceFormatVersion(), artifact.MetadataSchemaVersion()); err != nil {
			return operationalError(stderr, err)
		}
		return exitSuccess
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "help") {
		if err := printUsage(stdout); err != nil {
			return operationalError(stderr, err)
		}
		return exitSuccess
	}

	limits := compile.DefaultLimits()
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr, limits)
	case "build":
		return runBuild(args[1:], stdout, stderr, version, limits)
	case "preview":
		return runPreview(args[1:], stdout, stderr, limits)
	default:
		if err := writef(stderr, "pxlc: unknown command %q\n", args[0]); err != nil {
			return exitOperational
		}
		if err := printUsage(stderr); err != nil {
			return exitOperational
		}
		return exitUsage
	}
}

type commonOptions struct {
	input   string
	quiet   bool
	verbose bool
}

func runCheck(args []string, stdout, stderr io.Writer, limits compile.Limits) int {
	options, err := parseCheck(args)
	if err != nil {
		return usageError(stderr, err)
	}
	inputs, err := loadInputs(options.input)
	if err != nil {
		return inputError(stderr, err)
	}
	diagnostics, err := validateInputs(inputs.files, limits)
	if err != nil {
		return operationalError(stderr, err)
	}
	if len(diagnostics) != 0 {
		if err := printDiagnostics(stderr, diagnostics); err != nil {
			return operationalError(stderr, err)
		}
		return exitInvalid
	}
	if options.verbose {
		for _, input := range inputs.files {
			if err := writef(stdout, "checked %s\n", input.displayPath); err != nil {
				return operationalError(stderr, err)
			}
		}
	} else if !options.quiet {
		if err := writef(stdout, "checked %d asset(s)\n", len(inputs.files)); err != nil {
			return operationalError(stderr, err)
		}
	}
	return exitSuccess
}

type buildOptions struct {
	commonOptions
	output string
}

func runBuild(args []string, stdout, stderr io.Writer, version string, limits compile.Limits) int {
	options, err := parseBuild(args)
	if err != nil {
		return usageError(stderr, err)
	}
	inputs, err := loadInputs(options.input)
	if err != nil {
		return inputError(stderr, err)
	}
	if err = validateArtifactNamespace(inputs.files); err != nil {
		return inputError(stderr, err)
	}
	transaction, err := artifact.Begin(options.output)
	if err != nil {
		return operationalError(stderr, err)
	}
	defer func() {
		_ = transaction.Abort()
	}()
	written := make([]string, 0, len(inputs.files)*2)
	diagnostics := make([]diagnostic.Diagnostic, 0)
	seenAssets := make(map[string]diagnostic.Span, len(inputs.files))
	for _, input := range inputs.files {
		asset, sourceDiagnostics, compileErr := compileInput(input, limits)
		if compileErr != nil {
			return operationalError(stderr, compileErr)
		}
		diagnostics = append(diagnostics, sourceDiagnostics...)
		if asset == nil {
			continue
		}
		if duplicate := duplicateAsset(asset, seenAssets); duplicate != nil {
			diagnostics = append(diagnostics, *duplicate)
			continue
		}
		img, renderErr := raster.Render(asset.Plan)
		if renderErr != nil {
			return operationalError(stderr, fmt.Errorf("render %s: %w", input.displayPath, renderErr))
		}
		files, encodeErr := artifact.Runtime(asset, img, input.relativeStem, version)
		if encodeErr != nil {
			return operationalError(stderr, fmt.Errorf("encode %s: %w", input.displayPath, encodeErr))
		}
		for _, file := range files {
			if addErr := transaction.Add(file); addErr != nil {
				return operationalError(stderr, addErr)
			}
			written = append(written, file.Path)
		}
	}
	if len(diagnostics) != 0 {
		diagnostic.Sort(diagnostics)
		if err = printDiagnostics(stderr, diagnostics); err != nil {
			return operationalError(stderr, err)
		}
		if err = transaction.Abort(); err != nil {
			return operationalError(stderr, err)
		}
		return exitInvalid
	}
	if err = transaction.Commit(); err != nil {
		return operationalError(stderr, err)
	}
	if options.verbose {
		for _, path := range written {
			if err = writef(stdout, "wrote %s\n", joinDisplayPath(options.output, path)); err != nil {
				return operationalError(stderr, err)
			}
		}
	} else if !options.quiet {
		if err = writef(stdout, "built %d asset(s) to %s\n", len(inputs.files), options.output); err != nil {
			return operationalError(stderr, err)
		}
	}
	return exitSuccess
}

type previewOptions struct {
	commonOptions
	output     string
	scale      int
	background raster.Color
}

func runPreview(args []string, stdout, stderr io.Writer, limits compile.Limits) int {
	options, err := parsePreview(args)
	if err != nil {
		return usageError(stderr, err)
	}
	inputs, err := loadInputs(options.input)
	if err != nil {
		return inputError(stderr, err)
	}
	if inputs.directory {
		return usageError(stderr, errors.New("preview requires one .pxl file, not a directory tree"))
	}
	asset, diagnostics, err := compileInput(inputs.files[0], limits)
	if err != nil {
		return operationalError(stderr, err)
	}
	if len(diagnostics) != 0 {
		if err = printDiagnostics(stderr, diagnostics); err != nil {
			return operationalError(stderr, err)
		}
		return exitInvalid
	}
	img, err := raster.Render(asset.Plan)
	if err != nil {
		return operationalError(stderr, fmt.Errorf("render %s: %w", inputs.files[0].displayPath, err))
	}
	preview, err := raster.Preview(img, options.scale, options.background, limits.MaximumPreviewPixels)
	if err != nil {
		return operationalError(stderr, fmt.Errorf("create preview: %w", err))
	}
	data, err := artifact.PNG(preview)
	if err != nil {
		return operationalError(stderr, err)
	}
	if err = artifact.WriteAtomic(options.output, data); err != nil {
		return operationalError(stderr, err)
	}
	if options.verbose {
		if err = writef(stdout, "rendered %s at %dx\n", inputs.files[0].displayPath, options.scale); err != nil {
			return operationalError(stderr, err)
		}
		if err = writef(stdout, "wrote %s\n", options.output); err != nil {
			return operationalError(stderr, err)
		}
	} else if !options.quiet {
		if err = writef(stdout, "wrote preview %s\n", options.output); err != nil {
			return operationalError(stderr, err)
		}
	}
	return exitSuccess
}

func validateInputs(inputs []inputFile, limits compile.Limits) ([]diagnostic.Diagnostic, error) {
	diagnostics := make([]diagnostic.Diagnostic, 0)
	seenAssets := make(map[string]diagnostic.Span, len(inputs))
	for _, input := range inputs {
		asset, sourceDiagnostics, err := compileInput(input, limits)
		if err != nil {
			return nil, err
		}
		diagnostics = append(diagnostics, sourceDiagnostics...)
		if asset != nil {
			if duplicate := duplicateAsset(asset, seenAssets); duplicate != nil {
				diagnostics = append(diagnostics, *duplicate)
			}
		}
	}
	diagnostic.Sort(diagnostics)
	return diagnostics, nil
}

func compileInput(input inputFile, limits compile.Limits) (*compile.Asset, []diagnostic.Diagnostic, error) {
	data, err := readBounded(input.actualPath, limits.MaximumSourceBytes)
	if err != nil {
		return nil, nil, err
	}
	asset, diagnostics := compile.Compile(compile.Source{
		Path:         input.displayPath,
		ArtifactPath: input.sourcePath,
		Data:         data,
	}, limits)
	return asset, diagnostics, nil
}

func duplicateAsset(asset *compile.Asset, seen map[string]diagnostic.Span) *diagnostic.Diagnostic {
	previous, exists := seen[asset.Name]
	if !exists {
		seen[asset.Name] = asset.NameSpan
		return nil
	}
	d := diagnostic.Error(
		asset.NameSpan,
		"PXLC-E010",
		fmt.Sprintf("duplicate asset %q; first declared in %s", asset.Name, previous.Path),
	)
	return &d
}

func printDiagnostics(w io.Writer, diagnostics []diagnostic.Diagnostic) error {
	for _, d := range diagnostics {
		if err := writef(w, "%s\n", d.String()); err != nil {
			return err
		}
	}
	return nil
}

func usageError(stderr io.Writer, err error) int {
	if writeErr := writef(stderr, "pxlc: %v\n", err); writeErr != nil {
		return exitOperational
	}
	if writeErr := printUsage(stderr); writeErr != nil {
		return exitOperational
	}
	return exitUsage
}

func operationalError(stderr io.Writer, err error) int {
	_, _ = fmt.Fprintf(stderr, "pxlc: %v\n", err)
	return exitOperational
}

func inputError(stderr io.Writer, err error) int {
	var usage inputUsageError
	if errors.As(err, &usage) {
		return usageError(stderr, err)
	}
	return operationalError(stderr, err)
}

func printUsage(w io.Writer) error {
	return writef(w, `Usage:
  pxlc check <path> [--quiet | --verbose]
  pxlc build <path> --output <directory> [--quiet | --verbose]
  pxlc preview <file> --output <file.png> [--scale <integer>] [--background transparent|#RRGGBB] [--quiet | --verbose]
  pxlc --version
`)
}

func writef(w io.Writer, format string, arguments ...any) error {
	if _, err := fmt.Fprintf(w, format, arguments...); err != nil {
		return fmt.Errorf("write command output: %w", err)
	}
	return nil
}

func parseCheck(args []string) (commonOptions, error) {
	var options commonOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--quiet":
			options.quiet = true
		case "--verbose":
			options.verbose = true
		case "--help":
			return options, errors.New("check usage requested")
		default:
			if strings.HasPrefix(args[i], "-") {
				return options, fmt.Errorf("unknown check option %q", args[i])
			}
			if options.input != "" {
				return options, errors.New("check accepts one input path")
			}
			options.input = args[i]
		}
	}
	if err := options.validate(); err != nil {
		return options, err
	}
	return options, nil
}

func parseBuild(args []string) (buildOptions, error) {
	var options buildOptions
	for i := 0; i < len(args); i++ {
		argument := args[i]
		switch {
		case argument == "--quiet":
			options.quiet = true
		case argument == "--verbose":
			options.verbose = true
		case argument == "--output":
			value, next, err := optionValue(args, i, "--output")
			if err != nil {
				return options, err
			}
			options.output = value
			i = next
		case strings.HasPrefix(argument, "--output="):
			options.output = strings.TrimPrefix(argument, "--output=")
		case strings.HasPrefix(argument, "-"):
			return options, fmt.Errorf("unknown build option %q", argument)
		default:
			if options.input != "" {
				return options, errors.New("build accepts one input path")
			}
			options.input = argument
		}
	}
	if err := options.validate(); err != nil {
		return options, err
	}
	if options.output == "" {
		return options, errors.New("build requires --output")
	}
	return options, nil
}

func parsePreview(args []string) (previewOptions, error) {
	options := previewOptions{scale: 8}
	for i := 0; i < len(args); i++ {
		argument := args[i]
		switch {
		case argument == "--quiet":
			options.quiet = true
		case argument == "--verbose":
			options.verbose = true
		case argument == "--output" || argument == "--scale" || argument == "--background":
			value, next, err := optionValue(args, i, argument)
			if err != nil {
				return options, err
			}
			if err = options.set(argument, value); err != nil {
				return options, err
			}
			i = next
		case strings.HasPrefix(argument, "--output="):
			options.output = strings.TrimPrefix(argument, "--output=")
		case strings.HasPrefix(argument, "--scale="):
			if err := options.set("--scale", strings.TrimPrefix(argument, "--scale=")); err != nil {
				return options, err
			}
		case strings.HasPrefix(argument, "--background="):
			if err := options.set("--background", strings.TrimPrefix(argument, "--background=")); err != nil {
				return options, err
			}
		case strings.HasPrefix(argument, "-"):
			return options, fmt.Errorf("unknown preview option %q", argument)
		default:
			if options.input != "" {
				return options, errors.New("preview accepts one input path")
			}
			options.input = argument
		}
	}
	if err := options.validate(); err != nil {
		return options, err
	}
	if options.output == "" {
		return options, errors.New("preview requires --output")
	}
	if !strings.EqualFold(filepathExtension(options.output), ".png") {
		return options, errors.New("preview output must have a .png extension")
	}
	return options, nil
}

func (o commonOptions) validate() error {
	if o.input == "" {
		return errors.New("an input path is required")
	}
	if o.quiet && o.verbose {
		return errors.New("--quiet and --verbose cannot be used together")
	}
	return nil
}

func (o *previewOptions) set(option, value string) error {
	switch option {
	case "--output":
		o.output = value
	case "--scale":
		scale, err := strconv.Atoi(value)
		if err != nil || scale <= 0 {
			return fmt.Errorf("preview scale %q must be a positive integer", value)
		}
		o.scale = scale
	case "--background":
		background, err := previewBackground(value)
		if err != nil {
			return err
		}
		o.background = background
	}
	return nil
}

func optionValue(args []string, index int, name string) (string, int, error) {
	if index+1 >= len(args) || args[index+1] == "" {
		return "", index, fmt.Errorf("%s requires a value", name)
	}
	return args[index+1], index + 1, nil
}

func previewBackground(value string) (raster.Color, error) {
	if value == "transparent" {
		return raster.Color{}, nil
	}
	if len(value) != 7 || value[0] != '#' {
		return raster.Color{}, fmt.Errorf("preview background %q must be transparent or #RRGGBB", value)
	}
	n, err := strconv.ParseUint(value[1:], 16, 24)
	if err != nil {
		return raster.Color{}, fmt.Errorf("preview background %q must be transparent or #RRGGBB", value)
	}
	return raster.Color{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: 255}, nil
}

func filepathExtension(path string) string {
	lastSlash := strings.LastIndexAny(path, `/\\`)
	lastDot := strings.LastIndexByte(path, '.')
	if lastDot <= lastSlash {
		return ""
	}
	return path[lastDot:]
}
