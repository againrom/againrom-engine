package cursorclip

import (
	"image"
	"runtime"
	"structs"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	cursorMainQueue, cursorMainQueueError = purego.Dlsym(purego.RTLD_DEFAULT, "_dispatch_main_q")
	cursorMainSync                        func(uintptr, objc.Block)
	cursorConfineSelector                 = objc.RegisterName("setMouseConfinementRect:")
)

func init() {
	if address, err := purego.Dlsym(purego.RTLD_DEFAULT, "dispatch_sync"); err == nil && cursorMainQueueError == nil {
		purego.RegisterFunc(&cursorMainSync, address)
	}
}

type System struct{ window objc.ID }

type cursorCocoaRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

// cursorOnMain runs AppKit calls on its window thread without cgo.
func cursorOnMain(f func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if objc.Send[bool](objc.ID(objc.GetClass("NSThread")), objc.RegisterName("isMainThread")) {
		f()
		return
	}
	if cursorMainSync != nil {
		block := objc.NewBlock(func(_ objc.Block) { f() })
		defer block.Release()
		cursorMainSync(cursorMainQueue, block)
	}
}

func (n *System) Rectangle() (r image.Rectangle, focused bool) {
	cursorOnMain(func() {
		w := cursorCocoaWindow()
		if w == 0 {
			return
		}
		bounds := objc.Send[cursorCocoaRect](w.Send(objc.RegisterName("contentView")), objc.RegisterName("bounds"))
		if bounds.W <= 0 || bounds.H <= 0 {
			return
		}
		if n.window != w {
			n.set(nil)
			n.window = w.Send(objc.RegisterName("retain"))
		}
		r = image.Rect(int(bounds.X), int(bounds.Y), int(bounds.X+bounds.W), int(bounds.Y+bounds.H))
		focused = true
	})
	return
}

func cursorCocoaWindow() objc.ID {
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	w := app.Send(objc.RegisterName("keyWindow"))
	if w == 0 || !objc.Send[bool](app, objc.RegisterName("isActive")) ||
		objc.Send[bool](w, objc.RegisterName("isMiniaturized")) ||
		!objc.Send[bool](w, objc.RegisterName("respondsToSelector:"), cursorConfineSelector) {
		return 0
	}
	return w
}

func (n *System) Clip(r *image.Rectangle) (ok bool) {
	cursorOnMain(func() { ok = n.set(r) })
	return
}

func (n *System) set(r *image.Rectangle) bool {
	if n.window == 0 {
		return r == nil
	}
	var bounds cursorCocoaRect
	if r != nil {
		if cursorCocoaWindow() != n.window {
			n.set(nil)
			return false
		}
		bounds = cursorCocoaRect{X: float64(r.Min.X), Y: float64(r.Min.Y), W: float64(r.Dx()), H: float64(r.Dy())}
	}
	n.window.Send(cursorConfineSelector, bounds)
	if r == nil {
		n.window.Send(objc.RegisterName("release"))
		n.window = 0
	}
	return true
}
