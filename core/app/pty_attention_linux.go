//go:build linux

package app

import (
	"os"

	"golang.org/x/sys/unix"
)

func terminalInteractiveMode(file *os.File) bool {
	if file == nil {
		return false
	}
	termios, err := unix.IoctlGetTermios(int(file.Fd()), unix.TCGETS)
	if err != nil {
		return false
	}
	return termios.Lflag&unix.ICANON == 0 || termios.Lflag&unix.ECHO == 0
}
