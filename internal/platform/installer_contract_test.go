package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWindowsInstallerWaitsBeforeReplacingExecutable(t *testing.T) {
	t.Parallel()

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate installer contract test")
	}
	installerPath := filepath.Join(filepath.Dir(sourceFile), "..", "..", "build", "windows", "installer", "project.nsi")
	contents, err := os.ReadFile(installerPath)
	if err != nil {
		t.Fatal(err)
	}
	script := string(contents)

	required := []string{
		`${GetOptions} $R0 "/UPDATEPID=" $UpdateProcessID`,
		`kernel32::OpenProcess(i 0x00100000, i 0, i r0) p.r1 ?e`,
		`${If} $2 == 87`,
		`kernel32::WaitForSingleObject`,
		`WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"`,
	}
	for _, fragment := range required {
		if !strings.Contains(script, fragment) {
			t.Errorf("installer is missing required update contract %q", fragment)
		}
	}

	waitIndex := strings.Index(script, "Call WaitForUpdateProcess")
	filesIndex := strings.Index(script, "!insertmacro wails.files")
	if waitIndex < 0 || filesIndex < 0 || waitIndex >= filesIndex {
		t.Errorf("installer must wait for the application before writing files: wait=%d files=%d", waitIndex, filesIndex)
	}

	timeoutIndex := strings.Index(script, "${If} $2 == 258")
	if timeoutIndex < 0 || !strings.Contains(script[timeoutIndex:], "Abort") {
		t.Error("installer must abort when the application does not exit before the timeout")
	}
}
