package ui

// A POPUP is a box the map screen draws over the map and holds the player in
// until it is dismissed, and this file holds the ONE ANSWER the whole
// front-end asks about one.
//
// Four things are a function of it and each reads it here: the dim's
// composition, the viewer's camera, the viewer's ambient animation clock, and
// the map arm's own gate over every input a popup takes. Nothing else in this
// package tests for a notice in order to decide any of them.
//
// THE POINT OF THE FILE IS THAT THERE IS ONE ANSWER AND NOT FOUR. The
// product's author has ruled that the in-game menu, when this tree gains
// one, is a second popup and must behave identically. A second popup
// therefore inherits the whole of 0077 by making popupOpen answer true while
// it is up — one more disjunct in one method — and by nothing else.
//
// A LATER POPUP OWNED BY THE FLOW rather than drawn by the viewer must still
// raise THIS answer rather than gain its own beside it. The viewer is where the
// dim, the camera and the clock live, and it is also cmd/mapview's type, so a
// predicate rooted on the flow would be one three of the four readers could not
// reach.

// popupOpen reports whether a popup is standing over this viewer's map.
//
// The standalone developer viewer never sets a font's notice, so this is
// constantly false there and every reader takes the path it took before 0077.
//
// THE SECOND DISJUNCT IS THE IN-GAME MENU, and it is the second popup this
// file was written for. The flow raises menuUp when Escape opens the panel
// and lowers it when the panel closes; nothing else in this package reads
// that field. The dim, the camera pin, the ambient clock and the map arm's
// input gate then all move with it, because all four read here.
func (v *Viewer) popupOpen() bool { return v.NoticeOpen() || v.menuUp || v.docUp }

// popupOpen reports whether a popup is standing over the front-end's open map
// screen. It is the viewer's answer plus the screen test, so it is false on
// every other screen and for a front-end holding no viewer at all.
//
// IT IS THE WHOLE OF THIS PRODUCT'S AUTHORED RULING that a popup stops the
// world and that it takes every map-screen input but its own three. The
// decode agrees on both: the original's idle handler tests one mask, and the
// branch it takes while a panel is up runs no pacer arm at all — not the
// sub-tick, not the presentation tick the ambient animation rides, not the
// map-view drain, and not the command drain, which is what makes the halt
// stricter than the engine's own pause (DLG-STOP-012, High).
//
// It is a method over noticeOpen rather than a second predicate beside it: the
// rule belongs to "a popup is open" and not to a popup KIND, so the dialogue,
// the outcome and any kind added later are covered by there being no kind test
// here at all.
func (f *flow) popupOpen() bool {
	if f.screen == ScreenSave || f.screen == ScreenMod && f.modUI.back == ScreenGameMenu {
		return f.menuBack == ScreenMap && f.viewer != nil && f.viewer.popupOpen()
	}
	if f.mapShowing() {
		return f.viewer != nil && f.viewer.popupOpen()
	}
	return f.screen == ScreenDocuments && f.docBack == ScreenMap && f.viewer != nil && f.viewer.popupOpen()
}

// mapShowing reports whether the map screen is the surface the player is
// looking at. It is the map screen itself, or the in-game menu standing OVER
// a map screen.
//
// THE TEST WIDENED RATHER THAN THE POPUP ANSWER SPLITTING. Until 0158 the menu
// replaced the surface behind it, so "the menu is up" and "the map is showing"
// were exclusive; a popup over a map is neither, and a screen test that still
// read ScreenMap alone would have left the dim, the camera and the world's own
// stop all reading false on every frame the panel stood on.
func (f *flow) mapShowing() bool {
	switch f.screen {
	case ScreenMap:
		return true
	case ScreenGameMenu:
		return f.menuBack == ScreenMap
	}
	return false
}
