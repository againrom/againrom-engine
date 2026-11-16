package cursorclip

import (
	"image"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	cursorUser32    = windows.NewLazySystemDLL("user32.dll")
	cursorGetClient = cursorUser32.NewProc("GetClientRect")
	cursorToScreen  = cursorUser32.NewProc("ClientToScreen")
	cursorIsIconic  = cursorUser32.NewProc("IsIconic")
	cursorClip      = cursorUser32.NewProc("ClipCursor")
)

type System struct{ window windows.HWND }

type cursorPoint struct{ X, Y int32 }
type cursorRect struct{ Left, Top, Right, Bottom int32 }

func (n *System) Rectangle() (image.Rectangle, bool) {
	w := windows.GetForegroundWindow()
	var process uint32
	if w == 0 {
		return image.Rectangle{}, false
	}
	if _, err := windows.GetWindowThreadProcessId(w, &process); err != nil || process != windows.GetCurrentProcessId() {
		return image.Rectangle{}, false
	}
	if iconic, _, _ := cursorIsIconic.Call(uintptr(w)); iconic != 0 {
		return image.Rectangle{}, false
	}
	var class [64]uint16
	if _, err := windows.GetClassName(w, &class[0], int32(len(class))); err != nil || windows.UTF16ToString(class[:]) != "GLFW30" {
		return image.Rectangle{}, false
	}
	var r cursorRect
	if ok, _, _ := cursorGetClient.Call(uintptr(w), uintptr(unsafe.Pointer(&r))); ok == 0 {
		return image.Rectangle{}, false
	}
	lo, hi := cursorPoint{r.Left, r.Top}, cursorPoint{r.Right, r.Bottom}
	if ok, _, _ := cursorToScreen.Call(uintptr(w), uintptr(unsafe.Pointer(&lo))); ok == 0 {
		return image.Rectangle{}, false
	}
	if ok, _, _ := cursorToScreen.Call(uintptr(w), uintptr(unsafe.Pointer(&hi))); ok == 0 {
		return image.Rectangle{}, false
	}
	n.window = w
	return image.Rect(int(lo.X), int(lo.Y), int(hi.X), int(hi.Y)), windows.GetForegroundWindow() == w
}

func (n *System) Clip(r *image.Rectangle) bool {
	var target *cursorRect
	if r != nil {
		if windows.GetForegroundWindow() != n.window {
			return false
		}
		target = &cursorRect{int32(r.Min.X), int32(r.Min.Y), int32(r.Max.X), int32(r.Max.Y)}
	}
	ok, _, _ := cursorClip.Call(uintptr(unsafe.Pointer(target)))
	return ok != 0
}
