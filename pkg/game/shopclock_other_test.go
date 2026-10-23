//go:build !windows

package game

import "time"

// shopStockThreadTime answers act's wall time; the release tests that use it
// run on Windows, where it is the thread's CPU time.
func shopStockThreadTime(act func()) time.Duration {
	start := time.Now()
	act()
	return time.Since(start)
}
