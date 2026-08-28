//go:build windows

package platform

import (
	"os"
	"syscall"
)

// normalizePathAliases expands Windows 8.3 components while preserving
// symbolic-link and junction components. This lets EvalSymlinks output be
// compared without mistaking a short path alias for a redirect.
func normalizePathAliases(path string) (string, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	for size := uint32(260); ; {
		buffer := make([]uint16, size)
		length, err := syscall.GetLongPathName(name, &buffer[0], size)
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

func isLinkComponent(path string, info os.FileInfo) bool {
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	// Windows junctions are reparse points but are not reported as
	// ModeSymlink by Go. Readlink handles both symbolic links and junctions.
	_, err := os.Readlink(path)
	return err == nil
}
