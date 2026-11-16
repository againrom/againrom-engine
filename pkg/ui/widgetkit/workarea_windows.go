package widgetkit

import (
	"syscall"
	"unsafe"
)

const (
	spiGetWorkArea   = 0x0030
	smCyCaption      = 4
	smCxSizeFrame    = 32
	smCySizeFrame    = 33
	smCxPaddedBorder = 92
)

var (
	workUser          = syscall.NewLazyDLL("user32.dll")
	procSystemParams  = workUser.NewProc("SystemParametersInfoW")
	procMetricsForDpi = workUser.NewProc("GetSystemMetricsForDpi")
	procMetrics       = workUser.NewProc("GetSystemMetrics")
	procSystemDpi     = workUser.NewProc("GetDpiForSystem")
)

type winRect struct{ Left, Top, Right, Bottom int32 }

func metric(index, dpi uintptr) int {
	if procMetricsForDpi.Find() == nil {
		r, _, _ := procMetricsForDpi.Call(index, dpi)
		return int(int32(r))
	}
	r, _, _ := procMetrics.Call(index)
	return int(int32(r))
}

// platformDesktop is the primary monitor's work area (the screen less the
// taskbar) and the title bar and sizing border of a resizable window, in
// device-independent pixels.
func platformDesktop() Desktop {
	var r winRect
	ok, _, _ := procSystemParams.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0)
	if ok == 0 {
		return Desktop{}
	}
	dpi := uintptr(96)
	if procSystemDpi.Find() == nil {
		if v, _, _ := procSystemDpi.Call(); v != 0 {
			dpi = v
		}
	}
	pad := metric(smCxPaddedBorder, dpi)
	frameW := 2 * (metric(smCxSizeFrame, dpi) + pad)
	frameH := metric(smCyCaption, dpi) + 2*(metric(smCySizeFrame, dpi)+pad)
	return desktopOf(int(r.Left), int(r.Top), int(r.Right-r.Left), int(r.Bottom-r.Top), frameW, frameH, float64(dpi)/96)
}
