//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const uninstallRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Uninstall\LemonDevSimpleTools`

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
	return launchWindowsInstaller(path, currentInstallDirectory(), os.Getpid(), windows.ShellExecute)
}

func currentInstallDirectory() string {
	executable, err := os.Executable()
	if err != nil {
		return ""
	}
	executable, err = filepath.Abs(filepath.Clean(executable))
	if err != nil {
		return ""
	}
	if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
		executable = resolved
	}

	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		key, openErr := registry.OpenKey(root, uninstallRegistryPath, registry.QUERY_VALUE|registry.WOW64_64KEY)
		if openErr != nil {
			continue
		}
		directory := registeredInstallDirectory(key, executable)
		_ = key.Close()
		if directory != "" {
			return directory
		}
	}
	return ""
}

func registeredInstallDirectory(key registry.Key, executable string) string {
	if value, _, err := key.GetStringValue("InstallLocation"); err == nil {
		if directory := matchingInstallDirectory(value, executable); directory != "" {
			return directory
		}
	}
	if value, _, err := key.GetStringValue("DisplayIcon"); err == nil {
		return installDirectoryFromDisplayIcon(value, executable)
	}
	return ""
}

func installDirectoryFromDisplayIcon(value, executable string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, `"`) {
		if end := strings.Index(value[1:], `"`); end >= 0 {
			value = value[1 : end+1]
		}
	} else if comma := strings.LastIndex(value, ","); comma >= 0 {
		if _, err := strconv.Atoi(strings.TrimSpace(value[comma+1:])); err == nil {
			value = value[:comma]
		}
	}
	return matchingInstallDirectory(filepath.Dir(value), executable)
}

func matchingInstallDirectory(directory, executable string) string {
	directory = strings.Trim(strings.TrimSpace(directory), `"`)
	if directory == "" || !filepath.IsAbs(directory) {
		return ""
	}
	directory = filepath.Clean(directory)
	candidate := filepath.Join(directory, filepath.Base(executable))
	if resolved, err := filepath.EvalSymlinks(candidate); err == nil {
		candidate = resolved
	}
	if !strings.EqualFold(filepath.Clean(candidate), filepath.Clean(executable)) {
		return ""
	}
	info, err := os.Stat(candidate)
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	return directory
}

func launchWindowsInstaller(path, installDirectory string, processID int, shellExecute shellExecuteFunc) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("installer path is empty")
	}
	if processID <= 0 {
		return errors.New("update process ID is invalid")
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return fmt.Errorf("encode installer elevation verb: %w", err)
	}
	installer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("encode installer path: %w", err)
	}
	arguments := "/UPDATEPID=" + strconv.Itoa(processID)
	if strings.TrimSpace(installDirectory) != "" {
		// NSIS requires /D to be the final argument and forbids quoting its
		// value, even when the absolute directory contains spaces.
		arguments += " /D=" + installDirectory
	}
	parameters, err := windows.UTF16PtrFromString(arguments)
	if err != nil {
		return fmt.Errorf("encode install directory: %w", err)
	}
	if err := shellExecute(0, verb, installer, parameters, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("launch installer with elevation: %w", err)
	}
	return nil
}
