package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultOutputDirectory(t *testing.T) {
	directory, err := DefaultOutputDirectory()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(directory) != "output" {
		t.Fatalf("expected output directory basename, got %q", directory)
	}
	info, err := os.Stat(directory)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("default output path is not a directory: %q", directory)
	}
}

func TestNormalizeRevealPaths(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "result.txt")
	if err := os.WriteFile(file, []byte("result"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths, err := normalizeRevealPaths([]string{"", file, file})
	if err != nil {
		t.Fatal(err)
	}
	expected, err := filepath.EvalSymlinks(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != filepath.Clean(expected) {
		t.Fatalf("unexpected normalized paths: %#v", paths)
	}
}

func TestNormalizeRevealPathsRejectsMissingAndDirectories(t *testing.T) {
	root := t.TempDir()
	if _, err := normalizeRevealPaths([]string{filepath.Join(root, "missing.txt")}); err == nil {
		t.Fatal("expected missing output path to be rejected")
	}
	if _, err := normalizeRevealPaths([]string{root}); err == nil {
		t.Fatal("expected directory path to be rejected")
	}
}

func TestRevealFilesRejectsEmptyInput(t *testing.T) {
	if err := RevealFiles(nil); err == nil {
		t.Fatal("expected empty reveal request to fail")
	}
}
