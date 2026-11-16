package widgetkit

import "testing"

// fakeScreen models the window system's rules that matter for the fit: a
// decoration adds the frame around the client area, and a size request is
// ignored while the window is maximized.
type fakeScreen struct {
	d          Desktop
	cw, ch     int
	decorated  bool
	maximized  bool
	cx, cy     int
	worstOuter [2]int
}

func (s *fakeScreen) outer() (int, int) {
	if s.decorated {
		return s.cw + s.d.FrameW, s.ch + s.d.FrameH
	}
	return s.cw, s.ch
}

func (s *fakeScreen) note() {
	if !s.decorated {
		return
	}
	w, h := s.outer()
	s.worstOuter[0], s.worstOuter[1] = max(s.worstOuter[0], w), max(s.worstOuter[1], h)
}

func (s *fakeScreen) calls() WindowCalls {
	return WindowCalls{
		Desktop:   func() Desktop { return s.d },
		Maximized: func() bool { return s.maximized },
		Restore:   func() { s.maximized = false },
		SetSize: func(w, h int) {
			if !s.maximized {
				s.cw, s.ch = w, h
			}
			s.note()
		},
		Decorate:   func(v bool) { s.decorated = v; s.note() },
		Place:      func(x, y int) { s.cx, s.cy = x, y },
		Limits:     func(int, int) {},
		Resizing:   func(bool) {},
		Fullscreen: func(bool) {},
	}
}

func desk(w, h int) Desktop { return Desktop{Work: WorkArea{W: w, H: h}, FrameW: 16, FrameH: 39} }

func TestSwitchFromWholeScreenKeepsWindowInsideWorkArea(t *testing.T) {
	for _, maximized := range []bool{false, true} {
		s := &fakeScreen{d: desk(2560, 1392), cw: 2560, ch: 1440, maximized: maximized}
		applyWindow(s.calls(), WindowRow{Resizable: true, MonitorAware: true},
			WindowForm{W: 1280, H: 960, Decorated: true, Place: true, X: 640, Y: 240})
		w, h := s.outer()
		if !s.decorated || w > 2560 || h > 1392 || s.worstOuter[0] > 2560 || s.worstOuter[1] > 1392 {
			t.Errorf("maximized=%t: outer %dx%d (worst %v) decorated=%t, want inside 2560x1392", maximized, w, h, s.worstOuter, s.decorated)
		}
		if s.cw != 1280 || s.ch != 960 {
			t.Errorf("maximized=%t: client %dx%d, want 1280x960", maximized, s.cw, s.ch)
		}
		if s.cy-(39-8) < 0 || s.cy+s.ch+8 > 1392 {
			t.Errorf("maximized=%t: client at y=%d leaves the title bar or bottom edge off the work area", maximized, s.cy)
		}
	}
}

func TestFitClientStaysInsideWorkArea(t *testing.T) {
	for _, tc := range []struct {
		work         [2]int
		w, h         int
		wantW, wantH int
	}{
		{[2]int{2560, 1392}, 1280, 960, 1280, 960},
		{[2]int{2560, 1392}, 1280, 800, 1280, 800},
		{[2]int{1920, 1040}, 1280, 960, 1280, 960},
		{[2]int{1920, 1040}, 1280, 1100, 1280, 1001},
		{[2]int{1366, 728}, 1280, 960, 1280, 689},
		{[2]int{1366, 728}, 760, 760, 760, 689},
		{[2]int{1280, 680}, 1280, 960, 1264, 641},
		{[2]int{0, 0}, 1280, 960, 1280, 960},
	} {
		d := desk(tc.work[0], tc.work[1])
		w, h := fitClient(tc.w, tc.h, d)
		if w != tc.wantW || h != tc.wantH {
			t.Errorf("work %v, %dx%d: got %dx%d, want %dx%d", tc.work, tc.w, tc.h, w, h, tc.wantW, tc.wantH)
		}
		if tc.work[0] > 0 && (w+d.FrameW > tc.work[0] || h+d.FrameH > tc.work[1]) {
			t.Errorf("work %v: outer %dx%d exceeds it", tc.work, w+d.FrameW, h+d.FrameH)
		}
	}
}

func TestWholeScreenFormIsNotFitted(t *testing.T) {
	s := &fakeScreen{d: desk(2560, 1392), cw: 1280, ch: 960, decorated: true}
	applyWindow(s.calls(), WindowRow{Resizable: true, MonitorAware: true}, WindowForm{W: 2560, H: 1440, Place: true})
	if s.decorated || s.cw != 2560 || s.ch != 1440 || s.cx != 0 || s.cy != 0 {
		t.Errorf("whole screen: decorated=%t client %dx%d at %d,%d", s.decorated, s.cw, s.ch, s.cx, s.cy)
	}
}
