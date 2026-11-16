package widgetkit

// WorkArea is the part of a monitor a window may cover, in device-independent
// pixels from the monitor's top-left corner: the monitor less the taskbar or
// dock. W == 0 means the platform reported none.
type WorkArea struct{ X, Y, W, H int }

// Desktop is what the window system reports about the screen: the work area
// and the extra width and height a title bar and border add to a client area.
type Desktop struct {
	Work           WorkArea
	FrameW, FrameH int
}

// desktopOf converts the platform's pixel report into device-independent
// pixels. scale is the monitor's device scale factor.
func desktopOf(workX, workY, workW, workH, frameW, frameH int, scale float64) Desktop {
	if scale <= 0 {
		scale = 1
	}
	dip := func(v int) int { return int(float64(v)/scale + 0.5) }
	up := func(v int) int {
		n := int(float64(v) / scale)
		if float64(n)*scale < float64(v) {
			n++
		}
		return n
	}
	return Desktop{
		Work:   WorkArea{dip(workX), dip(workY), dip(workW), dip(workH)},
		FrameW: up(frameW), FrameH: up(frameH),
	}
}

// fitClient is the client size, at most w x h, whose decorated outer window
// lies inside the desktop's work area. A desktop that reports no work area
// leaves the size as asked.
func fitClient(w, h int, d Desktop) (int, int) {
	if d.Work.W <= 0 || d.Work.H <= 0 {
		return w, h
	}
	return max(min(w, d.Work.W-d.FrameW), 1), max(min(h, d.Work.H-d.FrameH), 1)
}

// centreClient is the client's top-left corner that centres the decorated
// outer window w x h in the work area. The window system places the client
// area, so the title bar and top border lie above the returned y.
func centreClient(w, h int, d Desktop) (int, int) {
	side := d.FrameW / 2
	x := d.Work.X + (d.Work.W-(w+d.FrameW))/2 + side
	y := d.Work.Y + (d.Work.H-(h+d.FrameH))/2 + d.FrameH - side
	return max(x, d.Work.X+side), max(y, d.Work.Y+d.FrameH-side)
}
