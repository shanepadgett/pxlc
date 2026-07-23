package artifact

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Transaction stages a set of files before replacing any existing artifact.
type Transaction struct {
	root   string
	stage  string
	paths  []string
	seen   map[string]struct{}
	closed bool
}

// Begin starts an output transaction rooted at root.
func Begin(root string) (*Transaction, error) {
	if root == "" {
		return nil, errors.New("output root is empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory %q: %w", root, err)
	}
	stage, err := os.MkdirTemp(root, ".pxlc-stage-")
	if err != nil {
		return nil, fmt.Errorf("create artifact staging directory: %w", err)
	}
	return &Transaction{root: root, stage: stage, seen: make(map[string]struct{})}, nil
}

// Add writes one relative artifact into the staging area.
func (t *Transaction) Add(file File) error {
	if t.closed {
		return errors.New("artifact transaction is closed")
	}
	relative, err := safeRelativePath(file.Path)
	if err != nil {
		return err
	}
	if _, exists := t.seen[relative]; exists {
		return fmt.Errorf("duplicate artifact path %q", file.Path)
	}
	t.seen[relative] = struct{}{}
	t.paths = append(t.paths, relative)
	path := filepath.Join(t.stage, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create staging directory for %q: %w", file.Path, err)
	}
	if err := os.WriteFile(path, file.Data, 0o644); err != nil {
		return fmt.Errorf("stage artifact %q: %w", file.Path, err)
	}
	return nil
}

// Commit replaces final artifacts in stable path order.
func (t *Transaction) Commit() error {
	if t.closed {
		return errors.New("artifact transaction is closed")
	}
	t.closed = true
	slices.Sort(t.paths)
	for _, relative := range t.paths {
		target := filepath.Join(t.root, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			_ = os.RemoveAll(t.stage)
			return fmt.Errorf("create output directory for %q: %w", filepath.ToSlash(relative), err)
		}
		if err := os.Rename(filepath.Join(t.stage, relative), target); err != nil {
			_ = os.RemoveAll(t.stage)
			return fmt.Errorf("replace artifact %q: %w", filepath.ToSlash(relative), err)
		}
	}
	if err := os.RemoveAll(t.stage); err != nil {
		return fmt.Errorf("remove artifact staging directory: %w", err)
	}
	return nil
}

// Abort discards staged files. It is safe after Commit.
func (t *Transaction) Abort() error {
	if t.closed {
		return nil
	}
	t.closed = true
	if err := os.RemoveAll(t.stage); err != nil {
		return fmt.Errorf("remove artifact staging directory: %w", err)
	}
	return nil
}

// WriteAtomic replaces one output file after its complete contents are written.
func WriteAtomic(path string, data []byte) error {
	if path == "" {
		return errors.New("output path is empty")
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", directory, err)
	}
	temporary, err := os.CreateTemp(directory, ".pxlc-output-")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryPath := temporary.Name()
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set temporary output permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary output: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace output %q: %w", path, err)
	}
	remove = false
	return nil
}

func safeRelativePath(path string) (string, error) {
	if path == "" || filepath.IsAbs(path) {
		return "", fmt.Errorf("artifact path %q must be relative", path)
	}
	relative := filepath.Clean(filepath.FromSlash(path))
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("artifact path %q escapes the output root", path)
	}
	return relative, nil
}
