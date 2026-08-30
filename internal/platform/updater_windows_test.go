//go:build windows

package platform

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/sys/windows"
)

func TestLaunchWindowsInstallerUsesRunAs(t *testing.T) {
	path := `C:\Users\test user\AppData\Local\Temp\simpletools-update\simpletools-update-123.exe`
	installDirectory := `C:\Program Files\LemonDev\SimpleTools`
	processID := 12345
	called := false
	shellExecute := func(hwnd windows.Handle, verb, file, args, cwd *uint16, showCmd int32) error {
		called = true
		if hwnd != 0 {
			t.Fatalf("ShellExecute hwnd = %v, want 0", hwnd)
		}
		if got := windows.UTF16PtrToString(verb); got != "runas" {
			t.Fatalf("ShellExecute verb = %q, want %q", got, "runas")
		}
		if got := windows.UTF16PtrToString(file); got != path {
			t.Fatalf("ShellExecute file = %q, want %q", got, path)
		}
		want := "/UPDATEPID=12345 /D=" + installDirectory
		if got := windows.UTF16PtrToString(args); got != want {
			t.Fatalf("ShellExecute args = %q, want %q", got, want)
		}
		if cwd != nil {
			t.Fatalf("ShellExecute cwd = %v, want nil", cwd)
		}
		if showCmd != windows.SW_SHOWNORMAL {
			t.Fatalf("ShellExecute showCmd = %d, want %d", showCmd, windows.SW_SHOWNORMAL)
		}
		return nil
	}

	if err := launchWindowsInstaller(path, installDirectory, processID, shellExecute); err != nil {
		t.Fatalf("launchWindowsInstaller() error = %v", err)
	}
	if !called {
		t.Fatal("ShellExecute was not called")
	}
}

func TestLaunchWindowsInstallerWrapsCancelledElevation(t *testing.T) {
	shellExecute := func(windows.Handle, *uint16, *uint16, *uint16, *uint16, int32) error {
		return windows.ERROR_CANCELLED
	}

	err := launchWindowsInstaller(`C:\update.exe`, `D:\Apps\SimpleTools`, 12345, shellExecute)
	if !errors.Is(err, windows.ERROR_CANCELLED) {
		t.Fatalf("launchWindowsInstaller() error = %v, want wrapped ERROR_CANCELLED", err)
	}
}

func TestLaunchWindowsInstallerRejectsEmbeddedNUL(t *testing.T) {
	called := false
	shellExecute := func(windows.Handle, *uint16, *uint16, *uint16, *uint16, int32) error {
		called = true
		return nil
	}

	if err := launchWindowsInstaller("C:\\update\x00.exe", `D:\Apps\SimpleTools`, 12345, shellExecute); err == nil {
		t.Fatal("launchWindowsInstaller() error = nil, want invalid path error")
	}
	if called {
		t.Fatal("ShellExecute was called for a path containing NUL")
	}
}

func TestLaunchWindowsInstallerRejectsEmbeddedNULInInstallDirectory(t *testing.T) {
	called := false
	shellExecute := func(windows.Handle, *uint16, *uint16, *uint16, *uint16, int32) error {
		called = true
		return nil
	}

	if err := launchWindowsInstaller(`C:\update.exe`, "D:\\Apps\x00SimpleTools", 12345, shellExecute); err == nil {
		t.Fatal("launchWindowsInstaller() error = nil, want invalid directory error")
	}
	if called {
		t.Fatal("ShellExecute was called for an install directory containing NUL")
	}
}

func TestLaunchWindowsInstallerWithoutRegisteredDirectory(t *testing.T) {
	var arguments string
	shellExecute := func(_ windows.Handle, _, _, args, _ *uint16, _ int32) error {
		arguments = windows.UTF16PtrToString(args)
		return nil
	}
	if err := launchWindowsInstaller(`C:\update.exe`, "", 9876, shellExecute); err != nil {
		t.Fatal(err)
	}
	if arguments != "/UPDATEPID=9876" {
		t.Fatalf("ShellExecute args = %q, want PID only", arguments)
	}
}

func TestMatchingInstallDirectoryRequiresCurrentExecutable(t *testing.T) {
	directory := t.TempDir()
	executable := filepath.Join(directory, "SimpleTools.exe")
	if err := os.WriteFile(executable, []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := matchingInstallDirectory(directory, executable); got != directory {
		t.Fatalf("matchingInstallDirectory() = %q, want %q", got, directory)
	}
	if got := matchingInstallDirectory(filepath.Dir(directory), executable); got != "" {
		t.Fatalf("matchingInstallDirectory() accepted unrelated directory %q", got)
	}
}

func TestInstallDirectoryFromDisplayIcon(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "\u81ea\u5b9a\u4e49 install,dir")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(directory, "SimpleTools.exe")
	if err := os.WriteFile(executable, []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, value := range []string{
		executable,
		executable + ",0",
		`"` + executable + `"`,
		`"` + executable + `",0`,
	} {
		if got := installDirectoryFromDisplayIcon(value, executable); got != directory {
			t.Errorf("installDirectoryFromDisplayIcon(%q) = %q, want %q", value, got, directory)
		}
	}

	unrelated := filepath.Join(t.TempDir(), "SimpleTools.exe")
	if got := installDirectoryFromDisplayIcon(unrelated, executable); got != "" {
		t.Fatalf("installDirectoryFromDisplayIcon() accepted unrelated executable directory %q", got)
	}
}

func TestUninstallRegistryPathMatchesWailsConfig(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate updater test")
	}
	contents, err := os.ReadFile(filepath.Join(filepath.Dir(sourceFile), "..", "..", "wails.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Info struct {
			CompanyName string `json:"companyName"`
			ProductName string `json:"productName"`
		} `json:"info"`
	}
	if err := json.Unmarshal(contents, &config); err != nil {
		t.Fatal(err)
	}
	want := `Software\Microsoft\Windows\CurrentVersion\Uninstall\` + config.Info.CompanyName + config.Info.ProductName
	if uninstallRegistryPath != want {
		t.Fatalf("uninstallRegistryPath = %q, want %q from wails.json", uninstallRegistryPath, want)
	}
}
