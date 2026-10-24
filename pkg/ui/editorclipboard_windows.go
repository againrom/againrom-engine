package ui

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

var (
	clipboardUser   = syscall.NewLazyDLL("user32.dll")
	clipboardKernel = syscall.NewLazyDLL("kernel32.dll")
	clipboardOpen   = clipboardUser.NewProc("OpenClipboard")
	clipboardClose  = clipboardUser.NewProc("CloseClipboard")
	clipboardGet    = clipboardUser.NewProc("GetClipboardData")
	clipboardSet    = clipboardUser.NewProc("SetClipboardData")
	clipboardEmpty  = clipboardUser.NewProc("EmptyClipboard")
	clipboardAlloc  = clipboardKernel.NewProc("GlobalAlloc")
	clipboardFree   = clipboardKernel.NewProc("GlobalFree")
	clipboardLock   = clipboardKernel.NewProc("GlobalLock")
	clipboardUnlock = clipboardKernel.NewProc("GlobalUnlock")
	clipboardSize   = clipboardKernel.NewProc("GlobalSize")
	clipboardMove   = clipboardKernel.NewProc("RtlMoveMemory")
)

func withEditorClipboard(f func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var last error
	for attempt := 0; ; attempt++ {
		ok, _, err := clipboardOpen.Call(0)
		if ok != 0 {
			break
		}
		last = err
		if attempt == 24 {
			return fmt.Errorf("open clipboard: %w", last)
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer clipboardClose.Call()
	return f()
}

func clipboardBytes(format uintptr) ([]byte, error) {
	h, _, err := clipboardGet.Call(format)
	if h == 0 {
		return nil, fmt.Errorf("read clipboard: %w", err)
	}
	n, _, _ := clipboardSize.Call(h)
	if n == 0 || n > 16<<20 {
		return nil, fmt.Errorf("clipboard data size %d is unsupported", n)
	}
	p, _, err := clipboardLock.Call(h)
	if p == 0 {
		return nil, fmt.Errorf("lock clipboard: %w", err)
	}
	defer clipboardUnlock.Call(h)
	b := make([]byte, int(n))
	clipboardMove.Call(uintptr(unsafe.Pointer(&b[0])), p, n)
	runtime.KeepAlive(b)
	return b, nil
}

func putClipboardBytes(format uintptr, b []byte) error {
	h, _, err := clipboardAlloc.Call(2, uintptr(len(b)))
	if h == 0 {
		return fmt.Errorf("allocate clipboard: %w", err)
	}
	p, _, err := clipboardLock.Call(h)
	if p == 0 {
		clipboardFree.Call(h)
		return fmt.Errorf("lock clipboard: %w", err)
	}
	clipboardMove.Call(p, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	runtime.KeepAlive(b)
	clipboardUnlock.Call(h)
	if ok, _, err := clipboardSet.Call(format, h); ok == 0 {
		clipboardFree.Call(h)
		return fmt.Errorf("write clipboard: %w", err)
	}
	return nil
}

func (systemEditorClipboard) read() (s string, err error) {
	err = withEditorClipboard(func() error {
		var readErr error
		s, readErr = readEditorClipboardText()
		return readErr
	})
	return
}

func readEditorClipboardText() (string, error) {
	available, _, _ := clipboardUser.NewProc("IsClipboardFormatAvailable").Call(13)
	if available == 0 {
		return "", nil
	}
	b, err := clipboardBytes(13)
	if err != nil {
		return "", err
	}
	var words []uint16
	for i := 0; i+1 < len(b); i += 2 {
		v := uint16(b[i]) | uint16(b[i+1])<<8
		if v == 0 {
			break
		}
		words = append(words, v)
	}
	return string(utf16.Decode(words)), nil
}

func (systemEditorClipboard) write(s string) error {
	return withEditorClipboard(func() error { return writeEditorClipboardText(s) })
}

func writeEditorClipboardText(s string) error {
	u := append(utf16.Encode([]rune(s)), 0)
	b := make([]byte, 2*len(u))
	for i, w := range u {
		b[2*i], b[2*i+1] = byte(w), byte(w>>8)
	}
	if ok, _, err := clipboardEmpty.Call(); ok == 0 {
		return fmt.Errorf("empty clipboard: %w", err)
	}
	return putClipboardBytes(13, b)
}
