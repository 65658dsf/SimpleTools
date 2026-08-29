//go:build darwin

package platform

import "os/exec"

func launchInstaller(path string) error {
	return exec.Command("open", path).Start()
}
