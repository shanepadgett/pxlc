package cli

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildFailurePreservesArtifactsAndPreviewScales(t *testing.T) {
	t.Parallel()
	temporary := t.TempDir()
	sourcePath := filepath.Join(temporary, "icon.pxl")
	outputPath := filepath.Join(temporary, "art")
	previewPath := filepath.Join(temporary, "preview.png")
	valid := []byte(`pxlc 1
asset icon
canvas 2 1
palette p { transparent clear "." color ink "K" #123456 }
background p clear
layer body using p { grid 0 0 { "K." } }
`)
	if err := os.WriteFile(sourcePath, valid, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	if status := Run([]string{"build", sourcePath, "--output", outputPath}, &stdout, &stderr, "test"); status != exitSuccess {
		t.Fatalf("build status = %d, stderr = %s", status, stderr.String())
	}
	runtimePath := filepath.Join(outputPath, "icon.png")
	before, err := os.ReadFile(runtimePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if err := os.WriteFile(sourcePath, []byte("pxlc 1\nasset broken\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() invalid source error = %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if status := Run([]string{"build", sourcePath, "--output", outputPath}, &stdout, &stderr, "test"); status != exitInvalid {
		t.Fatalf("invalid build status = %d, stderr = %s", status, stderr.String())
	}
	after, err := os.ReadFile(runtimePath)
	if err != nil {
		t.Fatalf("ReadFile() preserved artifact error = %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("invalid build replaced an existing runtime artifact")
	}

	if err := os.WriteFile(sourcePath, valid, 0o644); err != nil {
		t.Fatalf("WriteFile() restored source error = %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if status := Run([]string{"preview", sourcePath, "--scale", "3", "--output", previewPath}, &stdout, &stderr, "test"); status != exitSuccess {
		t.Fatalf("preview status = %d, stderr = %s", status, stderr.String())
	}
	file, err := os.Open(previewPath)
	if err != nil {
		t.Fatalf("Open() preview error = %v", err)
	}
	preview, decodeErr := png.Decode(file)
	closeErr := file.Close()
	if decodeErr != nil {
		t.Fatalf("Decode() preview error = %v", decodeErr)
	}
	if closeErr != nil {
		t.Fatalf("Close() preview error = %v", closeErr)
	}
	if got := preview.Bounds().Size(); got.X != 6 || got.Y != 3 {
		t.Fatalf("preview size = %v, want (6,3)", got)
	}
}

func TestBuildRejectsArtifactFileDirectoryCollision(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	nested := filepath.Join(root, "foo.png")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	source := func(name string) []byte {
		return []byte("pxlc 1\nasset " + name + `
canvas 1 1
palette p { transparent clear "." }
background p clear
layer base using p { pixel 0 0 clear }
`)
	}
	if err := os.WriteFile(filepath.Join(root, "foo.pxl"), source("foo"), 0o644); err != nil {
		t.Fatalf("WriteFile() foo error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "bar.pxl"), source("bar"), 0o644); err != nil {
		t.Fatalf("WriteFile() bar error = %v", err)
	}
	var stdout, stderr bytes.Buffer
	status := Run([]string{"build", root, "--output", filepath.Join(root, "output")}, &stdout, &stderr, "test")
	if status != exitUsage {
		t.Fatalf("build status = %d, want %d; stderr = %s", status, exitUsage, stderr.String())
	}
}
