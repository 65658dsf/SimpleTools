package platform

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestUniqueName(t *testing.T) {
	d := t.TempDir()
	first := filepath.Join(d, "photo.png")
	if err := os.WriteFile(first, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := UniqueName(d, "photo.jpg", ".png")
	if filepath.Base(got) != "photo-1.png" {
		t.Fatalf("got %s", got)
	}
}

func TestAtomicWrite(t *testing.T) {
	d := t.TempDir()
	if err := AtomicWrite(d, "result.txt", func(w io.Writer) error { _, e := w.Write([]byte("ok")); return e }); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(d, "result.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "ok" {
		t.Fatalf("got %q", b)
	}
}

func TestAtomicWriteRejectsPathOutsideDirectory(t *testing.T) {
	d := t.TempDir()
	outside := filepath.Join(filepath.Dir(d), "outside.txt")
	err := AtomicWrite(d, outside, func(w io.Writer) error { _, e := w.Write([]byte("nope")); return e })
	if err == nil {
		t.Fatal("expected path containment error")
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("outside file was created: %v", err)
	}
}

func TestAtomicWriteRejectsSymlinkedDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	linked := filepath.Join(root, "nested")
	if err := os.Symlink(outside, linked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	err := AtomicWrite(linked, "result.txt", func(w io.Writer) error { _, e := w.Write([]byte("nope")); return e })
	if err == nil {
		t.Fatal("expected symlinked output directory to be rejected")
	}
	if _, err := os.Stat(filepath.Join(outside, "result.txt")); !os.IsNotExist(err) {
		t.Fatalf("symlink target was written: %v", err)
	}
}

func TestAtomicWriteRejectsSymlinkedAncestorDirectory(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	dir := filepath.Join(alias, "target")
	err := AtomicWrite(dir, "result.txt", func(w io.Writer) error { _, e := w.Write([]byte("nope")); return e })
	if err == nil {
		t.Fatal("expected symlinked ancestor output directory to be rejected")
	}
	if _, err := os.Stat(filepath.Join(target, "result.txt")); !os.IsNotExist(err) {
		t.Fatalf("symlink target was written: %v", err)
	}
}
