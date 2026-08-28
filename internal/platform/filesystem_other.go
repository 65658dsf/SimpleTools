//go:build !windows

package platform

import "os"

func normalizePathAliases(path string) (string, error) { return path, nil }

func isLinkComponent(_ string, info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}
