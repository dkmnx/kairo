package ui

import (
	"os"

	"golang.org/x/sys/windows"
)

func supportsANSI() bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}

	handle := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if handle == windows.InvalidHandle {
		return false
	}

	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return false
	}

	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if err := windows.SetConsoleMode(handle, mode); err != nil {
		return false
	}

	return true
}
