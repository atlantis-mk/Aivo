//go:build !linux && !darwin

package app

import "os"

func terminalInteractiveMode(_ *os.File) bool { return false }
