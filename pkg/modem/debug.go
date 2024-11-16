//go:build debug

package modem

import "fmt"

func debugLog(format string, args ...any) {
	fmt.Printf(format, args...)
}
