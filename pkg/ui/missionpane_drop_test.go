package ui

import (
	"image"
	"testing"
)

// paneDropModes are the pane's two presentation modes. A pack item released
// on the pane is worn in both: TOWN-348 tests the held item before any
// rectangle and the mode is read only when painting.
var paneDropModes = []struct {
	name       string
	statistics bool
}{{"figure", false}, {"statistics", true}}

func openPaneDropAt(t *testing.T, statistics bool) (*Viewer, image.Rectangle) {
	t.Helper()
	v, _ := openDollAt(t)
	if statistics {
		v.toggleHudPanel(hudPanelDoll)
	}
	if v.characterPaneStatistics() != statistics {
		t.Fatalf("setup: statistics mode = %v, want %v", v.characterPaneStatistics(), statistics)
	}
	pane, ok := v.characterPaneRect()
	if !ok {
		t.Fatal("setup: the frame has no character pane")
	}
	return v, pane
}

func armPackDrag(t *testing.T, v *Viewer, cell int) {
	t.Helper()
	cx, cy := packCellCenter(t, v, cell)
	dollPress(v, cx, cy)
	dollMove(v, cx+3*TapSlop, cy)
	if !v.dragActive || v.dragCandKind != dragFromPack {
		t.Fatalf("setup: dragActive=%v dragCandKind=%v, want an armed pack drag", v.dragActive, v.dragCandKind)
	}
}

func TestDragPackCellOntoThePaneEquipsInEitherMode(t *testing.T) {
	for _, mode := range paneDropModes {
		t.Run(mode.name, func(t *testing.T) {
			v, pane := openPaneDropAt(t, mode.statistics)
			armPackDrag(t, v, 1)
			dollRelease(v, (pane.Min.X+pane.Max.X)/2, (pane.Min.Y+pane.Max.Y)/2)

			idx, ok := v.TakeInventoryEquip()
			if !ok || idx != 1 {
				t.Fatalf("TakeInventoryEquip = (%d,%v), want (1,true)", idx, ok)
			}
			if worn, dropIdx, x, y, ok := v.TakeInventoryDrop(); ok {
				t.Errorf("the release also raised a ground drop (worn=%v idx=%d x=%d y=%d)", worn, dropIdx, x, y)
			}
			if v.dragActive || v.dragCandKind != dragNone {
				t.Errorf("dragActive=%v dragCandKind=%v after the release, want the gesture cleared", v.dragActive, v.dragCandKind)
			}
		})
	}
}

func TestDragPackCellOntoAPaneCornerEquipsAndPressesNoCorner(t *testing.T) {
	corners := []struct {
		name   string
		corner CharacterPaneCorner
	}{{"backpack", CharacterPaneBackpack}, {"book", CharacterPaneBook}, {"mode", CharacterPaneMode},
		{"previous", CharacterPanePrev}, {"next", CharacterPaneNext}, {"menu", CharacterPaneMenu}}
	for _, mode := range paneDropModes {
		for _, c := range corners {
			t.Run(mode.name+"/"+c.name, func(t *testing.T) {
				v, pane := openPaneDropAt(t, mode.statistics)
				switches := v.hudHidden
				armPackDrag(t, v, 2)
				at := CharacterPaneCornerRect(pane, c.corner).Min.Add(image.Pt(1, 1))
				dollRelease(v, at.X, at.Y)

				idx, ok := v.TakeInventoryEquip()
				if !ok || idx != 2 {
					t.Fatalf("TakeInventoryEquip = (%d,%v), want (2,true)", idx, ok)
				}
				if v.hudHidden != switches {
					t.Errorf("the release pressed the corner: switches %v, want %v", v.hudHidden, switches)
				}
				if v.TakeCharacterPaneMenu() {
					t.Error("the release raised the menu request")
				}
			})
		}
	}
}

func TestDragPackCellOffThePaneInStatisticsModeStillDropsToTheGround(t *testing.T) {
	v, _ := openPaneDropAt(t, true)
	armPackDrag(t, v, 1)
	dollRelease(v, 0, 0)

	worn, idx, _, _, ok := v.TakeInventoryDrop()
	if !ok || worn || idx != 1 {
		t.Fatalf("TakeInventoryDrop = (worn=%v,idx=%d,ok=%v), want (worn=false,idx=1,ok=true)", worn, idx, ok)
	}
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Errorf("a release off the pane also raised TakeInventoryEquip(%d)", idx)
	}
}
