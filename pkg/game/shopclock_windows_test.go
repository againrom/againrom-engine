//go:build windows

package game

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var windowsGetThreadTimes = syscall.NewLazyDLL("kernel32.dll").NewProc("GetThreadTimes")

// windowsCurrentThread is the pseudo-handle GetCurrentThread answers.
const windowsCurrentThread = ^uintptr(1)

// shopStockThreadTime answers the CPU time act spends on the calling thread,
// kernel plus user time from GetThreadTimes. Other processes on a loaded
// machine delay act's wall time, not this.
func shopStockThreadTime(act func()) time.Duration {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	before := windowsThreadTime()
	act()
	return windowsThreadTime() - before
}

func windowsThreadTime() time.Duration {
	var creation, exit, kernel, user syscall.Filetime
	ok, _, err := windowsGetThreadTimes.Call(windowsCurrentThread,
		uintptr(unsafe.Pointer(&creation)), uintptr(unsafe.Pointer(&exit)),
		uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user)))
	if ok == 0 {
		panic(err)
	}
	ticks := func(ft syscall.Filetime) int64 { return int64(ft.HighDateTime)<<32 | int64(ft.LowDateTime) }
	return time.Duration(ticks(kernel)+ticks(user)) * 100
}
