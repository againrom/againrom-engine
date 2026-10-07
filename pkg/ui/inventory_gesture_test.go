package ui

import (
	"image"
	"image/color"
	"testing"
	"time"
)

// TestDraggingTheSecondPressOfADoubleClickActsOnOnlyThePickedPackItem is the
// combined gesture the separate double-click and drag cases did not cover: a
// first tap, then a matching second press which stays down and crosses
// TapSlop. The App tick seam drains the same one-shot requests pkg/game drains
// before command() sees the current frame, and removes the acted-on element so
// the adjacent blue item shifts into index 0. That makes an early equip and a
// later index-based drop two distinguishable actions rather than two counters.
func TestDraggingTheSecondPressOfADoubleClickActsOnOnlyThePickedPackItem(t *testing.T) {
	a := inventoryTestApp(t)
	v := a.flow.viewer
	layoutViewport(v, MenuWindowW, MenuWindowH)
	v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}})
	v.sel = selection{5}
	red := solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff})
	blue := solidPic(8, 8, color.RGBA{B: 0xff, A: 0xff})
	subject := packOf(5, red, blue)
	v.SetInventorySubject(subject)

	var equipped, dropped []*image.RGBA
	a.flow.tick = func() {
		if idx, ok := v.TakeInventoryEquip(); ok {
			if idx < 0 || idx >= len(subject.Pack) {
				t.Fatalf("equip request index %d outside pack of %d", idx, len(subject.Pack))
			}
			equipped = append(equipped, subject.Pack[idx])
			subject.Pack = append(subject.Pack[:idx], subject.Pack[idx+1:]...)
			v.SetInventorySubject(subject)
		}
		if worn, idx, _, _, ok := v.TakeInventoryDrop(); ok {
			if worn {
				t.Fatal("pack gesture raised a worn-item drop")
			}
			if idx < 0 || idx >= len(subject.Pack) {
				t.Fatalf("drop request index %d outside pack of %d", idx, len(subject.Pack))
			}
			dropped = append(dropped, subject.Pack[idx])
			subject.Pack = append(subject.Pack[:idx], subject.Pack[idx+1:]...)
			v.SetInventorySubject(subject)
		}
	}

	cx, cy := packCellCenter(t, v, 0)
	now := time.Unix(1_700_000_000, 0)
	press := func(x, y int) {
		a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true,
			Viewer: Input{CursorX: x, CursorY: y, PrimaryDown: true}}, now)
	}
	move := func(x, y int) {
		a.step(appInput{CursorX: x, CursorY: y,
			Viewer: Input{CursorX: x, CursorY: y, PrimaryDown: true}}, now)
	}
	release := func(x, y int) {
		a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true,
			Viewer: Input{CursorX: x, CursorY: y}}, now)
	}

	// First click, then the second press of the double-click, held down.
	press(cx, cy)
	release(cx, cy)
	press(cx, cy)
	move(cx-3*TapSlop, cy-3*TapSlop)
	if len(equipped) != 0 || len(subject.Pack) != 2 {
		t.Errorf("before release: equipped=%d pack=%d, want 0 and 2: a drag candidate must not mutate the world", len(equipped), len(subject.Pack))
	}

	// Release over map ground, then give the production tick seam one frame to
	// drain the one request command() raised on release.
	release(5, 5)
	a.HeadlessStep()
	if len(equipped) != 0 {
		t.Errorf("equipped %d item(s), want none: the drag must win over the pending double-click", len(equipped))
	}
	if len(dropped) != 1 || dropped[0] != red {
		got := "none"
		if len(dropped) > 0 && dropped[0] == blue {
			got = "adjacent blue item after index drift"
		}
		t.Errorf("dropped=%s (count %d), want the originally picked red item exactly once", got, len(dropped))
	}
	if len(subject.Pack) != 1 || subject.Pack[0] != blue {
		t.Errorf("remaining pack = %d item(s), want the untouched adjacent blue item", len(subject.Pack))
	}
}
