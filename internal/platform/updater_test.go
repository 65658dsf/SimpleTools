package platform

import "testing"

func TestUpdateTempPatternKeepsInstallerExtension(t *testing.T) {
	tests := []struct {
		name string
		goos string
		want string
	}{
		{name: "windows", goos: "windows", want: "simpletools-update-*.exe"},
		{name: "windows case insensitive", goos: " WINDOWS ", want: "simpletools-update-*.exe"},
		{name: "macOS", goos: "darwin", want: "simpletools-update-*.dmg"},
		{name: "other", goos: "linux", want: "simpletools-update-*"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := updateTempPattern(test.goos); got != test.want {
				t.Fatalf("updateTempPattern(%q) = %q, want %q", test.goos, got, test.want)
			}
		})
	}
}
