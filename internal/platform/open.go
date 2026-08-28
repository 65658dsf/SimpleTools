package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

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
