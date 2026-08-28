//go:build windows

package platform

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestAtomicWriteAcceptsWindowsShortPathAlias(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "directory-with-a-long-name")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	short, err := windowsShortPath(dir)
	if err != nil {
		t.Skipf("short path unavailable: %v", err)
	}
	if strings.EqualFold(short, dir) {
		t.Skip("filesystem did not create a short path alias")
	}
	if err := AtomicWrite(short, "result.txt", func(w io.Writer) error { _, e := w.Write([]byte("ok")); return e }); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "result.txt")); err != nil {
		t.Fatal(err)
	} else if string(got) != "ok" {
		t.Fatalf("got %q", got)
	}
}

func windowsShortPath(path string) (string, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	for size := uint32(260); ; {
		buffer := make([]uint16, size)
		length, err := syscall.GetShortPathName(name, &buffer[0], size)
		if err != nil {
			return "", err
		}
		if length == 0 {
			return path, nil
		}
		if length < size {
			return syscall.UTF16ToString(buffer[:length]), nil
		}
		size = length + 1
	}
}
