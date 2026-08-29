//go:build !windows && !darwin

package platform

import (
	"fmt"
	"runtime"
)

func launchInstaller(string) error {
	return fmt.Errorf("unsupported update platform %s", runtime.GOOS)
}
