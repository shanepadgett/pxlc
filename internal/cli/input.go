package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const maximumInputFiles = 10_000

type inputSet struct {
	files     []inputFile
	directory bool
}

type inputFile struct {
	actualPath   string
	displayPath  string
	sourcePath   string
	relativeStem string
}

type inputUsageError struct {
	message string
}

func (e inputUsageError) Error() string {
	return e.message
}

func loadInputs(inputPath string) (inputSet, error) {
	info, err := os.Stat(inputPath)
	if err != nil {
		return inputSet{}, fmt.Errorf("inspect input %q: %w", inputPath, err)
	}
	if info.IsDir() {
		files, loadErr := loadDirectory(inputPath)
		return inputSet{files: files, directory: true}, loadErr
	}
	if !info.Mode().IsRegular() || filepath.Ext(inputPath) != ".pxl" {
		return inputSet{}, inputUsageError{message: fmt.Sprintf("input %q must be a .pxl file or directory", inputPath)}
	}
	name := filepath.Base(inputPath)
	stem := strings.TrimSuffix(name, ".pxl")
	return inputSet{files: []inputFile{{
		actualPath:   inputPath,
		displayPath:  stableDisplayPath(inputPath, filepath.Dir(inputPath)),
		sourcePath:   filepath.ToSlash(name),
		relativeStem: stem,
	}}}, nil
}

func loadDirectory(root string) ([]inputFile, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".pxl" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		paths = append(paths, path)
		if len(paths) > maximumInputFiles {
			return inputUsageError{message: fmt.Sprintf("input exceeds the limit of %d .pxl files", maximumInputFiles)}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk input %q: %w", root, err)
	}
	if len(paths) == 0 {
		return nil, inputUsageError{message: fmt.Sprintf("input directory %q contains no regular .pxl files", root)}
	}
	slices.Sort(paths)
	inputs := make([]inputFile, 0, len(paths))
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return nil, fmt.Errorf("make input path relative: %w", err)
		}
		sourcePath := filepath.ToSlash(relative)
		inputs = append(inputs, inputFile{
			actualPath:   path,
			displayPath:  stableDisplayPath(path, root),
			sourcePath:   sourcePath,
			relativeStem: strings.TrimSuffix(sourcePath, ".pxl"),
		})
	}
	return inputs, nil
}

func validateArtifactNamespace(inputs []inputFile) error {
	paths := make(map[string]string, len(inputs)*2)
	for _, input := range inputs {
		for _, path := range []string{input.relativeStem + ".png", input.relativeStem + ".pxlc.json"} {
			key := strings.ToLower(filepath.ToSlash(path))
			if previous, exists := paths[key]; exists {
				return inputUsageError{message: fmt.Sprintf("generated artifact path %q conflicts with %q", path, previous)}
			}
			paths[key] = path
		}
	}
	keys := make([]string, 0, len(paths))
	for key := range paths {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		path := paths[key]
		parts := strings.Split(key, "/")
		for i := 1; i < len(parts); i++ {
			ancestor := strings.Join(parts[:i], "/")
			if filePath, exists := paths[ancestor]; exists {
				return inputUsageError{message: fmt.Sprintf("generated artifact path %q requires file %q to be a directory", path, filePath)}
			}
		}
	}
	return nil
}

func readBounded(path string, maximumBytes int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open input %q: %w", path, err)
	}
	info, statErr := file.Stat()
	if statErr != nil {
		_ = file.Close()
		return nil, fmt.Errorf("inspect open input %q: %w", path, statErr)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, fmt.Errorf("open input %q is not a regular file", path)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, int64(maximumBytes)+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read input %q: %w", path, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close input %q: %w", path, closeErr)
	}
	return data, nil
}

func stableDisplayPath(path, root string) string {
	if !filepath.IsAbs(path) {
		return filepath.ToSlash(filepath.Clean(path))
	}
	if cwd, err := os.Getwd(); err == nil {
		if relative, relErr := filepath.Rel(cwd, path); relErr == nil && !escapes(relative) {
			return filepath.ToSlash(relative)
		}
	}
	parent := filepath.Dir(root)
	relative, err := filepath.Rel(parent, path)
	if err == nil && !escapes(relative) {
		return filepath.ToSlash(relative)
	}
	return filepath.Base(path)
}

func escapes(relative string) bool {
	return relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func joinDisplayPath(root, relative string) string {
	return filepath.ToSlash(filepath.Join(root, filepath.FromSlash(relative)))
}
