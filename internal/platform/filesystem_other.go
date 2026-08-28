//go:build !windows

package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func normalizePathAliases(path string) (string, error) { return path, nil }

func isLinkComponent(_ string, info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}

// macOS exposes a few system directories through symlinks into /private
// (notably /var, /tmp, and /etc). These aliases are safe to follow because
// they are root-owned system paths, unlike user-controlled output links.
func isTrustedLinkComponent(path string) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	path = filepath.Clean(path)
	if filepath.Dir(path) != string(filepath.Separator) {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	privateRoot := filepath.Join(string(filepath.Separator), "private")
	relative, err := filepath.Rel(privateRoot, filepath.Clean(resolved))
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
