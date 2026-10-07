package game

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var editorFinalPathName = syscall.NewLazyDLL("kernel32.dll").NewProc("GetFinalPathNameByHandleW")

// Resolve the directory itself, not every ancestor's reparse metadata. A
// sandbox may allow this directory while denying inspection of the profile
// above it. The handle follows junctions/symlinks; failure remains a refusal.
func editorPhysicalDirectory(path string) (string, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	h, err := syscall.CreateFile(p, 0, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", err
	}
	defer syscall.CloseHandle(h)
	if err := editorFinalPathName.Find(); err != nil {
		return "", err
	}
	buf := make([]uint16, 32768)
	n, _, callErr := editorFinalPathName.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0)
	if n == 0 {
		return "", fmt.Errorf("resolve physical directory: %w", callErr)
	}
	if n >= uintptr(len(buf)) {
		return "", fmt.Errorf("resolved directory path is too long")
	}
	name := syscall.UTF16ToString(buf[:n])
	if strings.HasPrefix(name, `\\?\UNC\`) {
		name = `\\` + strings.TrimPrefix(name, `\\?\UNC\`)
	} else {
		name = strings.TrimPrefix(name, `\\?\`)
	}
	return filepath.Clean(name), nil
}
