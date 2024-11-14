//go:build debug

package modem

import "fmt"

func debugLog(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
