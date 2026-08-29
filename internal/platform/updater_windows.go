//go:build windows

package platform

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
)

type shellExecuteFunc func(
	windows.Handle,
	*uint16,
	*uint16,
	*uint16,
	*uint16,
	int32,
) error

func launchInstaller(path string) error {
	// CreateProcess cannot launch a requireAdministrator executable from a
	// non-elevated process. ShellExecute with runas brokers the UAC prompt.
	return launchWindowsInstaller(path, windows.ShellExecute)
}

func launchWindowsInstaller(path string, shellExecute shellExecuteFunc) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("installer path is empty")
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return fmt.Errorf("encode installer elevation verb: %w", err)
	}
	installer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("encode installer path: %w", err)
	}
	if err := shellExecute(0, verb, installer, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("launch installer with elevation: %w", err)
	}
	return nil
}
