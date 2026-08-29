package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultOutputDirectory returns the writable output directory used when the
// caller does not choose one explicitly. It is rooted next to the running
// executable so packaged builds and development binaries have deterministic
// behavior on every supported desktop platform.
//
// The directory is created on demand. Callers should still validate the
// returned path before using it for a job, since the executable directory may
// be read-only (for example, an app installed under a protected system path).
func DefaultOutputDirectory() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	executable, err = filepath.Abs(filepath.Clean(executable))
	if err != nil {
		return "", fmt.Errorf("resolve executable directory: %w", err)
	}
	// Resolve a launcher symlink where possible so output follows the actual
	// binary location. If resolution fails, the absolute executable path is
	// still a useful and deterministic fallback.
	if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
		executable = resolved
	}
	directory := filepath.Join(filepath.Dir(executable), "output")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("create default output directory: %w", err)
	}
	return filepath.Abs(filepath.Clean(directory))
}

// OpenDirectory asks the host shell to reveal an existing directory. The path
// is passed as an argument, never interpolated into a shell command.
func OpenDirectory(path string) error {
	path, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("path is not a directory")
	}
	var command string
	switch runtime.GOOS {
	case "windows":
		command = "explorer.exe"
	case "darwin":
		command = "open"
	case "linux":
		command = "xdg-open"
	default:
		return fmt.Errorf("opening directories is unsupported on %s", runtime.GOOS)
	}
	return exec.Command(command, path).Start()
}

// RevealFiles opens the host file manager and selects the first output file.
// On macOS Finder accepts multiple paths and can select all of them; Windows
// Explorer has a single-file /select contract, so only the first path is used
// there. Linux desktop environments generally lack a portable select API, and
// the containing directory is opened instead. Every path is validated before
// launching a process and is passed as an argument rather than shell text.
func RevealFiles(paths []string) error {
	files, err := normalizeRevealPaths(paths)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("no output files to reveal")
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer.exe", "/select,"+files[0]).Start()
	case "darwin":
		args := make([]string, 1, len(files)+1)
		args[0] = "-R"
		args = append(args, files...)
		return exec.Command("open", args...).Start()
	case "linux":
		return exec.Command("xdg-open", filepath.Dir(files[0])).Start()
	default:
		return fmt.Errorf("revealing files is unsupported on %s", runtime.GOOS)
	}
}

func normalizeRevealPaths(paths []string) ([]string, error) {
	files := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		absolute, err := filepath.Abs(filepath.Clean(path))
		if err != nil {
			return nil, fmt.Errorf("resolve output file %q: %w", path, err)
		}
		info, err := os.Stat(absolute)
		if err != nil {
			return nil, fmt.Errorf("inspect output file %q: %w", absolute, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("output path %q is not a regular file", absolute)
		}
		if resolved, resolveErr := filepath.EvalSymlinks(absolute); resolveErr == nil {
			absolute = filepath.Clean(resolved)
		}
		if _, duplicate := seen[absolute]; duplicate {
			continue
		}
		seen[absolute] = struct{}{}
		files = append(files, absolute)
	}
	return files, nil
}
