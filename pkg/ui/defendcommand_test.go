package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"
)

func TestPlayerDefendDPanelMapAndCancel1087(t *testing.T) {
	if expr := bindingSource(t)["Defend"]; !strings.Contains(expr, "inpututil.IsKeyJustPressed(ebiten.KeyD)") {
		t.Fatalf("D binding = %s", expr)
	}
	for _, panel := range []bool{false, true} {
		a, v, _ := atOnMap(t)
		v.sel = selection{atLoID, atHiID}
		var got [][2]uint32
		v.SetDefendSink(func(id, subject uint32) { got = append(got, [2]uint32{id, subject}) })
		// A newly selected Defend must replace 1090's unissued scroll aim;
		// otherwise map release would consume that item instead of defending.
		itemCasts := 0
		v.SetItemCastSink(func(uint32, int, string, uint32, int, int, bool) { itemCasts++ })
		v.itemCast = &itemCastSelection{owner: atLoID, spell: 1}
		if panel {
			v.pressCommandPanelCell(commandCellDefend, true)
		} else {
			if err := a.HeadlessKey("d"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessKey("d"); err != nil {
				t.Fatal(err)
			}
		}
		if v.missionMode() != 4 || v.itemCast != nil {
			t.Fatalf("D/panel mode=%d, want4", v.missionMode())
		}
		if c, ok := commandPanelSelected(v); !ok || c != 3 {
			t.Fatalf("overlay=%d/%v", c, ok)
		}
		atPress(a, v, atFoeCol, atFoeRow)
		if itemCasts != 0 {
			t.Fatal("Defend released the superseded scroll aim")
		}
		if want := [][2]uint32{{atLoID, atFoeID}, {atHiID, atFoeID}}; !reflect.DeepEqual(got, want) {
			t.Fatalf("Defend orders=%v want%v", got, want)
		}
		if v.missionMode() != 0 {
			t.Fatal("acting release did not spend Defend")
		}
		v.armDefend()
		atRightClick(a, v, atEmptyCol, atEmptyRow)
		if v.missionMode() != 0 || len(v.sel) != 2 || len(got) != 2 {
			t.Fatal("right click did not cancel only armed mode")
		}
		v.armDefend()
		atPress(a, v, atEmptyCol, atEmptyRow)
		if len(got) != 2 || v.missionMode() != 0 {
			t.Fatal("empty ground issued or retained Defend")
		}
		v.armDefend()
		a.step(appInput{Move: true}, atAt)
		if v.missionMode() != modeMove {
			t.Fatal("Move did not replace armed Defend")
		}
		v.armDefend()
		a.step(appInput{Unfocused: true, Defend: true}, atAt)
		if v.missionMode() != modeNone {
			t.Fatal("focus loss retained/rearmed Defend")
		}
	}
}

func TestPlayerDefendOwnershipModalAndMinimap1087(t *testing.T) {
	a, v := mkOnMap(t)
	for _, sel := range []selection{nil, {7}} {
		v.sel = sel
		a.step(appInput{Defend: true}, atAt)
		if v.missionMode() != 0 {
			t.Fatal("empty/foreign selection armed Defend")
		}
	}
	v.sel = selection{4}
	v.SetFont(panelFont())
	v.SetNotice("modal", NoticeDialogue)
	a.step(appInput{Defend: true}, atAt)
	if v.missionMode() != 0 {
		t.Fatal("modal armed Defend")
	}
	v.ClearNotice()
	// Read-only cell mapping, real down/drag/up delivery, no direct order call.
	var got [][2]uint32
	v.SetDefendSink(func(id, subject uint32) { got = append(got, [2]uint32{id, subject}) })
	v.armDefend()
	x, y := minimapPixelFor(t, v, image.Pt(4, 3))
	down := atFrame(x, y)
	down.PrimaryPressed = true
	down.Viewer.PrimaryDown = true
	a.step(down, atAt)
	if len(got) != 0 {
		t.Fatalf("minimap down = %v", got)
	}
	x, y = minimapPixelFor(t, v, image.Pt(5, 3))
	move := atFrame(x, y)
	move.Viewer.PrimaryDown = true
	a.step(move, atAt)
	if len(got) != 0 {
		t.Fatalf("minimap drag = %v", got)
	}
	up := atFrame(x, y)
	up.PrimaryReleased = true
	a.step(up, atAt)
	if v.missionMode() != modeDefend || len(got) != 0 {
		t.Fatal("minimap up reissued or retained mode")
	}
	v.armDefend()
	x, y = minimapPixelFor(t, v, image.Pt(2, 2))
	down = atFrame(x, y)
	down.PrimaryPressed = true
	down.Viewer.PrimaryDown = true
	a.step(down, atAt)
	if len(got) != 0 {
		t.Fatal("empty minimap cell issued Defend")
	}
}

func TestPlayerDefendMinimapReleaseOnly1087(t *testing.T) {
	for _, tc := range []struct {
		name          string
		held, both    bool
		releaseColumn int
	}{
		{"stationary-release", false, false, 4},
		{"release-new-position", false, false, 5},
		{"held-drag-then-release", true, false, 4},
		{"both-buttons-drag-then-release", true, true, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v := mkOnMap(t)
			v.sel = selection{4}
			var got [][2]uint32
			v.SetDefendSink(func(id, subject uint32) { got = append(got, [2]uint32{id, subject}) })
			v.armDefend()
			x, y := minimapPixelFor(t, v, image.Pt(4, 3))
			down := atFrame(x, y)
			down.PrimaryPressed, down.Viewer.PrimaryDown = true, true
			a.step(down, atAt)
			var want [][2]uint32
			if tc.held {
				x, y = minimapPixelFor(t, v, image.Pt(5, 3))
				move := atFrame(x, y)
				move.Viewer.PrimaryDown, move.Viewer.SecondaryDown = true, tc.both
				a.step(move, atAt)
			}
			x, y = minimapPixelFor(t, v, image.Pt(tc.releaseColumn, 3))
			up := atFrame(x, y)
			up.PrimaryReleased = true
			a.step(up, atAt)
			if !reflect.DeepEqual(got, want) || v.missionMode() != modeDefend || v.minimapGrab || v.held {
				t.Fatalf("release: orders=%v want=%v mode=%d grab=%v held=%v", got, want, v.missionMode(), v.minimapGrab, v.held)
			}
		})
	}
}
