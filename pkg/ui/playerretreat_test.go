package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"
)

func TestPlayerRetreat1089RAndPanelIssueImmediatelyAndSpendArmedMode(t *testing.T) {
	if expr := bindingSource(t)["PlayerRetreat"]; !strings.Contains(expr, "inpututil.IsKeyJustPressed(ebiten.KeyR)") {
		t.Fatal("R binding", expr)
	}
	for _, panel := range []bool{false, true} {
		a, v, _ := atOnMap(t)
		v.sel = selection{atLoID, atHiID}
		var got []uint32
		var presses [][]uint32
		v.SetPlayerRetreatSink(func(ids []uint32) {
			got = append(got, ids...)
			presses = append(presses, append([]uint32(nil), ids...))
		})
		v.armAttack()
		if panel {
			v.pressCommandPanelCell(commandCellRetreat, true)
		} else if err := a.HeadlessKey("r"); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []uint32{atLoID, atHiID}) || len(presses) != 1 {
			t.Fatal("Retreat not issued immediately", got)
		}
		if v.armed || v.aimed != commandNone || v.selectedSpell != 0 {
			t.Fatal("Retreat left an armed cursor")
		}
		if _, selected := commandPanelSelected(v); selected {
			t.Fatal("immediate command left overlay")
		}
		if commandPanelCellSkipped(commandCellRetreat) {
			t.Fatal("Retreat panel disabled")
		}
		// Each new key event reissues; a plain frame does not.
		a.step(appInput{}, atAt)
		if len(got) != 2 {
			t.Fatal("idle frame repeated Retreat")
		}
		a.step(appInput{PlayerRetreat: true}, atAt)
		if len(got) != 4 || len(presses) != 2 || !reflect.DeepEqual(presses[0], presses[1]) {
			t.Fatal("repeated key did not reissue")
		}
	}
}

func TestPlayerRetreat1089OwnershipModalFocusItemAndThresholdSeparation(t *testing.T) {
	a, v := mkOnMap(t)
	calls, auto := 0, 0
	v.SetPlayerRetreatSink(func([]uint32) { calls++ })
	v.SetRetreatSink(func() { auto++ })
	for _, sel := range []selection{nil, {7}} {
		v.sel = sel
		a.step(appInput{PlayerRetreat: true}, atAt)
		v.pressCommandPanelCell(commandCellRetreat, true)
	}
	v.sel = selection{4}
	v.SetFont(panelFont())
	v.SetNotice("modal", NoticeDialogue)
	a.step(appInput{PlayerRetreat: true}, atAt)
	v.ClearNotice()
	a.step(appInput{Unfocused: true, PlayerRetreat: true}, atAt)
	v.pressCommandPanelCell(commandCellRetreat, true)
	a.step(appInput{}, atAt)
	v.dragCandKind, v.dragActive = dragFromPack, true
	v.issuePlayerRetreat()
	v.pressCommandPanelCell(commandCellRetreat, true)
	v.dragCandKind, v.dragActive = dragNone, false
	if calls != 0 {
		t.Fatal("inadmissible Retreat emitted", calls)
	}
	a.step(appInput{Retreat: true}, atAt)
	if calls != 0 || auto != 1 {
		t.Fatal("Ctrl+W issued explicit Retreat")
	}
	a.step(appInput{PlayerRetreat: true}, atAt)
	if calls != 1 || auto != 1 {
		t.Fatal("R changed automatic withdrawal preference")
	}
	a.flow.screen = ScreenTown
	a.step(appInput{PlayerRetreat: true}, atAt)
	if calls != 1 {
		t.Fatal("town issued Retreat")
	}
}

func TestPlayerRetreat1089PanelPhysicalDownDragRelease(t *testing.T) {
	a, v := mkOnMap(t)
	v.sel = selection{4}
	calls := 0
	v.SetPlayerRetreatSink(func([]uint32) { calls++ })
	bar, ok := v.commandPanelBar()
	if !ok {
		t.Fatal("no panel")
	}
	r := commandCellRects(bar)[7]
	p := r.Min.Add(image.Pt(4, 4))
	down := atFrame(p.X, p.Y)
	down.PrimaryPressed, down.Viewer.PrimaryDown = true, true
	a.step(down, atAt)
	if calls != 1 {
		t.Fatal("physical down", calls)
	}
	held := atFrame(p.X, p.Y)
	held.Viewer.PrimaryDown = true
	a.step(held, atAt)
	if calls != 1 {
		t.Fatal("held panel repeated", calls)
	}
	up := atFrame(p.X, p.Y)
	up.PrimaryReleased = true
	a.step(up, atAt)
	if calls != 1 {
		t.Fatal("release repeated", calls)
	}
	a.step(down, atAt)
	if calls != 2 {
		t.Fatal("fresh down did not reissue", calls)
	}
}

func TestPlayerRetreat1089SpendsUnissuedScrollAim(t *testing.T) {
	for _, panel := range []bool{false, true} {
		a, v, _ := atOnMap(t)
		v.sel = selection{atLoID}
		retreats, casts := 0, 0
		v.SetPlayerRetreatSink(func([]uint32) { retreats++ })
		v.SetItemCastSink(func(uint32, int, string, uint32, int, int, bool) { casts++ })
		if !v.ArmItemCast(atLoID, 0, 1, "scroll") {
			t.Fatal("fixture could not arm scroll")
		}
		if panel {
			v.pressCommandPanelCell(commandCellRetreat, true)
		} else if err := a.HeadlessKey("r"); err != nil {
			t.Fatal(err)
		}
		if retreats != 1 || v.itemCast != nil {
			t.Fatalf("Retreat retained scroll aim: retreats=%d aim=%+v", retreats, v.itemCast)
		}
		if cell, selected := commandPanelSelected(v); selected {
			t.Fatalf("Retreat left command cell %d selected", cell)
		}
		atPress(a, v, atFoeCol, atFoeRow)
		if casts != 0 {
			t.Fatal("map release cast the superseded scroll")
		}
	}
}
