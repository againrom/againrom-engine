//go:build windows

package game

import (
	"errors"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

const (
	windowsMoveFileReplaceExisting = 0x1
	windowsMoveFileWriteThrough    = 0x8
	windowsPublishFlags            = windowsMoveFileWriteThrough
)

var windowsMoveFileExW = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

// publishSaveFile makes one complete temporary file visible under a final
// name without replacing a path another process already published.
//
// os.Rename on Windows calls MoveFileEx with MOVEFILE_REPLACE_EXISTING. That
// API choice permits two same-second writers to replace each other after both
// selected the same missing name. This call deliberately omits the replacement
// flag. MOVEFILE_WRITE_THROUGH asks Windows not to return before the move has
// completed on disk. The source and destination are siblings, so the operation
// is a same-volume rename and the complete closed file appears as one namespace
// change.
func publishSaveFile(oldPath, newPath string) (savePublishResult, error) {
	oldName, err := syscall.UTF16PtrFromString(oldPath)
	if err != nil {
		return savePublishResult{}, &os.LinkError{Op: "publish", Old: oldPath, New: newPath, Err: err}
	}
	newName, err := syscall.UTF16PtrFromString(newPath)
	if err != nil {
		return savePublishResult{}, &os.LinkError{Op: "publish", Old: oldPath, New: newPath, Err: err}
	}
	ok, _, callErr := windowsMoveFileExW.Call(
		uintptr(unsafe.Pointer(oldName)),
		uintptr(unsafe.Pointer(newName)),
		uintptr(windowsPublishFlags))
	runtime.KeepAlive(oldName)
	runtime.KeepAlive(newName)
	if ok == 0 {
		if callErr == syscall.Errno(0) {
			callErr = syscall.EINVAL
		}
		return savePublishResult{}, &os.LinkError{Op: "publish", Old: oldPath, New: newPath, Err: callErr}
	}
	return savePublishResult{committed: true}, nil
}

func isPublishCollision(err error) bool {
	return errors.Is(err, os.ErrExist) ||
		errors.Is(err, syscall.ERROR_FILE_EXISTS) ||
		errors.Is(err, syscall.ERROR_ALREADY_EXISTS)
}
