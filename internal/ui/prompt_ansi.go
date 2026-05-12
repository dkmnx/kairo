//go:build !windows

package ui

import "os"

func supportsANSI() bool {
	return os.Getenv("TERM") != "dumb"
}
