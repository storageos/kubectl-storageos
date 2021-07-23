package logger

import (
	"fmt"
)

var (
	quiet = false
)

// SetQuiet sets the value for quiet
func SetQuiet(s bool) {
	quiet = s
}

func Printf(format string, args ...interface{}) {
	if quiet {
		return
	}
	fmt.Printf(format, args...)
}
