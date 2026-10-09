package ui

import (
	"image"
	"testing"
)

// MENU-077: the base snaps an argument rectangle to 96k+8 by 64k+104 and
// centres it on the 640x480 frame.
func TestDialogBaseSnapsAndCentresTheThreeArgumentRectangles(t *testing.T) {
	for _, tc := range []struct {
		name         string
		w, h         int
		wantW, wantH int
		origin       image.Point
	}{
		{"game options", 560, 480, 488, 424, image.Pt(76, 28)},
		{"sound options", 540, 420, 488, 360, image.Pt(76, 60)},
		{"cutscene list", 440, 420, 392, 360, image.Pt(124, 60)},
	} {
		g := newDialogGeometry(tc.w, tc.h)
		if g.Dx() != tc.wantW || g.Dy() != tc.wantH || g.Min != tc.origin {
			t.Errorf("%s: %v, want %dx%d at %v", tc.name, g.Rectangle, tc.wantW, tc.wantH, tc.origin)
		}
		if body := g.Body(); body.Dx() != tc.wantW-8 || body.Dy() != tc.wantH-8 {
			t.Errorf("%s: body %v is not the snapped size less the 8 pixel shadow", tc.name, body)
		}
	}
	if gameOptionsDialog != newDialogGeometry(560, 480) || soundOptionsDialog != newDialogGeometry(540, 420) {
		t.Fatal("the options dialogs are not built from their argument rectangles")
	}
	// Truncation toward zero: a size just under the next tile keeps the lower one.
	if w, h := snapDialogSize(8+96*3-1, 104+64*2-1); w != 8+96*2 || h != 104+64 {
		t.Fatalf("snap of an off-tile size = %dx%d", w, h)
	}
}

// Every control lies inside its frame body; Game Options buttons use its width.
func TestOptionsDialogControlsLieInsideTheirFrames(t *testing.T) {
	for _, action := range []gameMenuAction{gameMenuSpeedDown, gameMenuDayNight, gameMenuSmoothing, gameMenuShadows,
		gameMenuLighting, gameMenuAnimation, gameMenuTooltipDelay, gameMenuFormation, gameMenuHealth, gameMenuDamage,
		gameMenuToggleTips, gameMenuAutoHealing, gameMenuPathfinding, gameMenuRetreat, gameMenuTimedAutosave,
		gameMenuAutosaveMinutes, gameMenuPageReturn, gameMenuOptionsCancel} {
		if r := gameOptionRect(action); r.Empty() || !r.In(gameOptionsDialog.Body()) {
			t.Errorf("game option %d rectangle %v outside %v", action, r, gameOptionsDialog.Body())
		}
	}
	ok, cancel := gameOptionRect(gameMenuPageReturn), gameOptionRect(gameMenuOptionsCancel)
	g := gameOptionsDialog
	if ok != g.Rect(68, 372, 205, 396) || cancel != g.Rect(274, 372, 411, 396) {
		t.Errorf("buttons %v %v, want x 68..205 and 274..411 at y 372..396", ok, cancel)
	}
	for _, action := range []gameMenuAction{gameMenuMusicVolume, gameMenuEffectsVolume, gameMenuSpeechVolume,
		gameMenuMusicTracks, gameMenuMusicUp, gameMenuMusicDown, gameMenuMusicScroll, gameMenuMusicRandom,
		gameMenuAcknowledgments, gameMenuMusicPlay, gameMenuMusicStop, gameMenuPageReturn, gameMenuToggleSound,
		gameMenuVolumeDown, gameMenuVolumeUp, gameMenuTestSound} {
		if r := soundOptionRect(action); r.Empty() || !r.In(soundOptionsDialog.Body()) {
			t.Errorf("sound option %d rectangle %v outside %v", action, r, soundOptionsDialog.Body())
		}
	}
}
