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
	if gameOptionsDialog != newTopDialogGeometry(560, 544) || soundOptionsDialog != newDialogGeometry(540, 420) {
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
		gameMenuAutosaveMinutes, gameMenuLootDots, gameMenuPageReturn, gameMenuOptionsCancel} {
		if r := gameOptionRect(action); r.Empty() || !r.In(gameOptionsDialog.Body()) {
			t.Errorf("game option %d rectangle %v outside %v", action, r, gameOptionsDialog.Body())
		}
	}
	ok, cancel := gameOptionRect(gameMenuPageReturn), gameOptionRect(gameMenuOptionsCancel)
	g := gameOptionsDialog
	if ok != g.Rect(68, 436, 205, 460) || cancel != g.Rect(274, 436, 411, 460) {
		t.Errorf("buttons %v %v, want x 68..205 and 274..411 at y 436..460", ok, cancel)
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

// The Game Options frame is one tile row taller than the original's for the
// engine's rows; every original control, the speed caption and the frame keep
// the offsets they had from the frame's top-left corner. The engine's timed
// autosave interval and loot-dot rows stand under the autosave toggle, and the
// automatic-casting button in the right column.
func TestGameOptionsPageKeepsEveryOriginalControlOffset(t *testing.T) {
	g := gameOptionsDialog
	if g.Min != image.Pt(76, 0) || g.Dx() != 488 || g.Dy() != 488 || g.Body().Dy() != 480 {
		t.Fatalf("frame %v, want 488x488 at (76,0)", g.Rectangle)
	}
	w := g.Body().Dx()
	for _, tc := range []struct {
		action gameMenuAction
		want   image.Rectangle
	}{
		{gameMenuSpeedDown, image.Rect(24, 76, 216, 100)},
		{gameMenuSpeedUp, image.Rect(24, 76, 216, 100)},
		{gameMenuDayNight, image.Rect(24, 104, 234, 128)},
		{gameMenuSmoothing, image.Rect(24, 128, 234, 152)},
		{gameMenuShadows, image.Rect(24, 152, 234, 176)},
		{gameMenuLighting, image.Rect(24, 176, 234, 200)},
		{gameMenuAnimation, image.Rect(24, 200, 234, 224)},
		{gameMenuTooltipDelay, image.Rect(24, 228, 234, 248)},
		{gameMenuFormation, image.Rect(24, 248, 216, 338)},
		{gameMenuHealth, image.Rect(240, 48, 456, 72)},
		{gameMenuDamage, image.Rect(240, 76, 456, 100)},
		{gameMenuToggleTips, image.Rect(240, 104, 456, 128)},
		{gameMenuAutoHealing, image.Rect(240, 132, 432, 224)},
		{gameMenuPathfinding, image.Rect(240, 224, 456, 248)},
		{gameMenuRetreat, image.Rect(240, 248, 408, 338)},
		{gameMenuTimedAutosave, image.Rect(24, 340, 234, 364)},
		{gameMenuAutosaveMinutes, image.Rect(24, 364, 234, 388)},
		{gameMenuAutoCast, image.Rect(240, 340, 456, 364)},
		{gameMenuPageReturn, image.Rect(w/7, 372+64, 3*w/7, 396+64)},
		{gameMenuOptionsCancel, image.Rect(4*w/7, 372+64, 6*w/7, 396+64)},
	} {
		if got := gameOptionRect(tc.action).Sub(g.Min); got != tc.want {
			t.Errorf("action %d at %v from the frame corner, want %v", tc.action, got, tc.want)
		}
	}
	if got := gameOptionSpeedLabel().Sub(g.Min); got != image.Rect(24, 48, 216, 72) {
		t.Errorf("speed caption at %v", got)
	}
	if got := gameOptionRect(gameMenuLootDots).Sub(g.Min); got != image.Rect(24, 388, 234, 412) {
		t.Errorf("loot dots row at %v", got)
	}
}
