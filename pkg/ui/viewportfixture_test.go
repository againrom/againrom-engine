package ui

import "againrom/pkg/render/frame"

// layoutViewport lays v out so its MAP VIEWPORT is exactly viewW x viewH frame
// pixels, at a placement whose scale is 1 and whose origin is (0,0).
func layoutViewport(v *Viewer, viewW, viewH int) (int, int) {
	v.frameW, v.frameH = viewW+MissionPanelW, viewH
	if v.frameW > 0 && v.frameH > 0 {
		v.place = frame.Fit(v.frameW, v.frameH, v.frameW, v.frameH)
		v.cam.ViewW, v.cam.ViewH = viewW, viewH
		v.cam.Clamp()
		v.applyStartView()
	}
	return v.frameW, v.frameH
}

const wornBoxFixtureH = MenuWindowH + 128

// mapAtScaleOne gives an App's open map screen a mission frame the size of the
// app's own window, so the frame is placed at scale 1 with its origin at (0,0)
// and a fixture's frame coordinates are window coordinates again.
//
// IT IS A FIXTURE CONVENIENCE AND NOT A CLAIM ABOUT THE BUILD. The shipped
// mission frame is at least MissionFrameW x MissionFrameH, grows horizontally
// with a wider window and is placed at whatever scale that window gives it.
// What the fixtures calling this are about is
// which surface a press lands on, not what the placement does to it; the
// placement itself is under test in hitscale_test.go, at scales above and below
// 1:1 and with a letterbox on each axis in turn.
func mapAtScaleOne(a *App) {
	v := a.flow.viewer
	if v == nil {
		return
	}
	winW, winH := a.winW+MissionPanelW, a.winH
	a.winW, a.winH = winW, winH
	a.place = frame.Fit(frame.W, frame.H, winW, winH)
	layoutViewport(v, winW-MissionPanelW, winH)
}
