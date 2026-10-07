package mod

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var finalPathName = syscall.NewLazyDLL("kernel32.dll").NewProc("GetFinalPathNameByHandleW")

func resolvedPath(name string) (string, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	p, err := syscall.UTF16PtrFromString(abs)
	if err != nil {
		return "", err
	}
	h, err := syscall.CreateFile(p, 0, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", err
	}
	defer syscall.CloseHandle(h)
	if err := finalPathName.Find(); err != nil {
		return "", err
	}
	buf := make([]uint16, 32768)
	n, _, callErr := finalPathName.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0)
	if n == 0 {
		return "", fmt.Errorf("resolve physical path: %w", callErr)
	}
	if n >= uintptr(len(buf)) {
		return "", fmt.Errorf("resolved path is too long")
	}
	name = syscall.UTF16ToString(buf[:n])
	if strings.HasPrefix(name, `\\?\UNC\`) {
		name = `\\` + strings.TrimPrefix(name, `\\?\UNC\`)
	} else {
		name = strings.TrimPrefix(name, `\\?\`)
	}
	return filepath.Clean(name), nil
}
