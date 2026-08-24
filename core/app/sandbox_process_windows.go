//go:build windows

package app

import (
	"os"
	"os/exec"
)

func setProcessGroup(_ *exec.Cmd) {}

func killProcessGroup(process *os.Process) error {
	if process == nil {
		return nil
	}
	return process.Kill()
}

func terminateProcessGroup(process *os.Process) error {
	if process == nil {
		return nil
	}
	return process.Kill()
}
