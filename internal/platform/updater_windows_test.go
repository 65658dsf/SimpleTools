//go:build windows

package platform

import (
	"errors"
	"testing"

	"golang.org/x/sys/windows"
)

func TestLaunchWindowsInstallerUsesRunAs(t *testing.T) {
	path := `C:\Users\test user\AppData\Local\Temp\simpletools-update\simpletools-update-123.exe`
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
		if args != nil {
			t.Fatalf("ShellExecute args = %v, want nil", args)
		}
		if cwd != nil {
			t.Fatalf("ShellExecute cwd = %v, want nil", cwd)
		}
		if showCmd != windows.SW_SHOWNORMAL {
			t.Fatalf("ShellExecute showCmd = %d, want %d", showCmd, windows.SW_SHOWNORMAL)
		}
		return nil
	}

	if err := launchWindowsInstaller(path, shellExecute); err != nil {
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

	err := launchWindowsInstaller(`C:\update.exe`, shellExecute)
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

	if err := launchWindowsInstaller("C:\\update\x00.exe", shellExecute); err == nil {
		t.Fatal("launchWindowsInstaller() error = nil, want invalid path error")
	}
	if called {
		t.Fatal("ShellExecute was called for a path containing NUL")
	}
}
