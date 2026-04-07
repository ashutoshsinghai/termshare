//go:build darwin

package cmd

import "os/exec"

func exec_xattr(path string) error {
	return exec.Command("xattr", "-d", "com.apple.quarantine", path).Run()
}
