package ui

// The two debug blow keys, driven through the SHIPPED front-end path: a whole
// App on the map screen, a real gesture to select with, and the seam the loader
// handed over as the only place a blow can be observed.
//
// Nothing here calls the seam directly.

import (
	"image"
	"reflect"
	"slices"
	"testing"
	"time"
)

// affAt is the frozen clock every frame below is stamped with. The blow keys
// read no clock and neither does the seam, so the instant only has to be stable.
var affAt = time.Unix(1_700_000_000, 0)

// The units the cases select and hit. Ids are SPARSE and given out of ascending
// order in the snapshot, so a walk that emitted slice positions or snapshot
// order rather than the selection's own would not produce the lists below.
const (
	affLoID, affLoCol, affLoRow = 4, 3, 3
	affHiID, affHiCol, affHiRow = 11, 5, 3
	affDeadID                   = 7
	affEmptyCol, affEmptyRow    = 1, 1
)

// affEntities is the snapshot: two living units on cells of their own and a
// corpse between them in id order, on a third cell. The corpse's id falls
// BETWEEN the two living ones, so a list that emitted it would be caught by the
// order as well as by the length.
func affEntities() []MapEntity {
	return []MapEntity{
		{ID: affLoID, Cell: image.Pt(affLoCol, affLoRow), Life: LifeAlive, HP: 100, MaxHP: 100},
		{ID: affDeadID, Cell: image.Pt(4, 5), Life: LifeDead, HP: -10, MaxHP: 100},
		{ID: affHiID, Cell: image.Pt(affHiCol, affHiRow), Life: LifeDowned, HP: 0, MaxHP: 100},
	}
}

// affOnMap parks a whole App on the map screen over a loader that hands out a
// recording seam, with the camera at the origin and the snapshot above in place.
func affOnMap(t *testing.T) (*App, *Viewer, *mapSeam) {
	t.Helper()
	seams := &[]*mapSeam{}
	a := newTestApp(t, appRows(3), seamLoader(t, seams))
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, affAt)
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	v.SetEntities(affEntities())
	return a, v, (*seams)[0]
}

// affFrame is one tick of front-end input at a window position, with the
// viewer's own cursor set to the same place — which is what readAppInput does.
func affFrame(x, y int) appInput {
	return appInput{Viewer: Input{CursorX: x, CursorY: y}, CursorX: x, CursorY: y}
}

// affSelect taps the cell (col,row) and asserts what the tap selected, so every
// case below states the set its key press is read against rather than assuming
// one.
func affSelect(t *testing.T, a *App, v *Viewer, col, row int, want selection) {
	t.Helper()
	x, y := cellPoint(v, col, row)
	in := affFrame(x, y)
	in.PrimaryPressed, in.PrimaryReleased = true, true
	a.step(in, affAt)
	if !slices.Equal(v.sel, want) {
		t.Fatalf("setup: the tap on cell (%d,%d) left %v, want %v", col, row, v.sel, want)
	}
}

// TestTheTwoKeysIssueOneBlowPerMarkedUnitAscending is AC-11's K and L half.
//
// The selection is written whole rather than tapped, because a tap yields a set
// of one and this is about a GROUP: the two living units and the corpse between
// them, which is the state a selection reaches when a member dies under it.
func TestTheTwoKeysIssueOneBlowPerMarkedUnitAscending(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   func(appInput) appInput
		want []struck
	}{
		{"K issues one kill each", func(in appInput) appInput { in.Kill = true; return in },
			[]struck{{affLoID, true}, {affHiID, true}}},
		{"L issues one chip each", func(in appInput) appInput { in.Chip = true; return in },
			[]struck{{affLoID, false}, {affHiID, false}}},
		// Both in one frame: each key's own emission is ascending, and the two
		// passes name the same units — the kill pass does not shorten the chip's.
		{"both in one frame", func(in appInput) appInput { in.Kill, in.Chip = true, true; return in },
			[]struck{{affLoID, true}, {affHiID, true}, {affLoID, false}, {affHiID, false}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v, s := affOnMap(t)
			before := selection{affLoID, affDeadID, affHiID}
			v.sel = append(selection(nil), before...)

			x, y := cellPoint(v, affEmptyCol, affEmptyRow)
			a.step(tc.in(affFrame(x, y)), affAt)

			if !reflect.DeepEqual(s.blows, tc.want) {
				t.Errorf("the seam received %v, want %v", s.blows, tc.want)
			}
			// NEITHER KEY CHANGES THE SELECTION, and the set is compared WHOLE —
			// the dead id included, which is what says a blow is not a fifth way
			// to replace it.
			if !slices.Equal(v.sel, before) {
				t.Errorf("the selection is %v after the key, want %v", v.sel, before)
			}
			// And no order went out with them: a key press is not a click.
			if len(s.orders) != 0 {
				t.Errorf("the key also issued the orders %v", s.orders)
			}
		})
	}
}

// TestAKeyPressedWithNothingMarkedIssuesNothing is the empty arm, and it is
// two cases rather than one: nothing selected at all, and a selection every
// member of which the snapshot reports dead.
func TestAKeyPressedWithNothingMarkedIssuesNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		sel  selection
	}{
		{"nothing selected", nil},
		{"a selection holding only a dead id", selection{affDeadID}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v, s := affOnMap(t)
			v.sel = tc.sel
			x, y := cellPoint(v, affEmptyCol, affEmptyRow)
			in := affFrame(x, y)
			in.Kill, in.Chip = true, true
			a.step(in, affAt)
			if len(s.blows) != 0 {
				t.Errorf("the seam received %v, want nothing", s.blows)
			}
		})
	}
}

func TestTheBlowKeysReachNothingOnAnyOtherScreen(t *testing.T) {
	a, v, s := affOnMap(t)
	v.sel = selection{affLoID, affHiID}

	// Esc leaves the map screen, which drops the whole seam with the viewer.
	a.step(appInput{Escape: true}, affAt)
	leaveViaMenu(a.flow) //
	if a.Screen() != ScreenPicker {
		t.Fatalf("setup: screen = %v after Esc, want ScreenPicker", a.Screen())
	}
	for _, screen := range []Screen{ScreenPicker, ScreenMenu} {
		a.flow.screen = screen
		in := appInput{Kill: true, Chip: true}
		a.step(in, affAt)
		if len(s.blows) != 0 {
			t.Errorf("on the %v screen the keys issued %v", screen, s.blows)
		}
	}
}

// TestTheBlowKeysAreHarmlessWithNothingUnderTheMap is the other half of "no
// world": a map screen whose loader put nothing under it holds no seam at all,
// and a key press there must be nothing rather than a nil call.
func TestTheBlowKeysAreHarmlessWithNothingUnderTheMap(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, affAt)
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	if a.flow.affect != nil {
		t.Fatalf("setup: the bare loader handed back a blow seam")
	}
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities(affEntities())
	v.sel = selection{affLoID, affHiID}

	in := appInput{Kill: true, Chip: true}
	a.step(in, affAt) // must not panic
	if !slices.Equal(v.sel, selection{affLoID, affHiID}) {
		t.Errorf("the keys changed the selection to %v with no world under the map", v.sel)
	}
}

func TestAKeyIsReadOnItsPressEdgeAndNotWhileHeld(t *testing.T) {
	a, v, s := affOnMap(t)
	v.sel = selection{affLoID}

	x, y := cellPoint(v, affEmptyCol, affEmptyRow)
	press := affFrame(x, y)
	press.Kill = true

	a.step(press, affAt)
	if len(s.blows) != 1 {
		t.Fatalf("the press issued %d blow(s), want 1", len(s.blows))
	}
	for k := 0; k < 3; k++ {
		a.step(affFrame(x, y), affAt)
	}
	if len(s.blows) != 1 {
		t.Errorf("%d blow(s) after three frames with the key not pressed, want the 1 the edge issued",
			len(s.blows))
	}
	a.step(press, affAt)
	if len(s.blows) != 2 {
		t.Errorf("%d blow(s) after a second press edge, want 2", len(s.blows))
	}
}

// TestAUnitSelectedByThisFrameIsHitByThisFrameIsKey fixes the ORDER of the two
// reads on the map arm: the gesture is resolved first, so a key pressed in the
// same frame as the tap that selected a unit hits that unit.
//
// The reverse order is the plausible mistake and it is invisible in a window —
// one frame is 16 ms — so it is stated here where it is decidable.
func TestAUnitSelectedByThisFrameIsHitByThisFrameIsKey(t *testing.T) {
	a, v, s := affOnMap(t)
	if len(v.sel) != 0 {
		t.Fatalf("setup: the map screen opened holding the selection %v", v.sel)
	}

	x, y := cellPoint(v, affLoCol, affLoRow)
	in := affFrame(x, y)
	in.PrimaryPressed, in.PrimaryReleased = true, true
	in.Kill = true
	a.step(in, affAt)

	if want := (selection{affLoID}); !slices.Equal(v.sel, want) {
		t.Fatalf("the tap left %v, want %v", v.sel, want)
	}
	if want := []struck{{affLoID, true}}; !reflect.DeepEqual(s.blows, want) {
		t.Errorf("the seam received %v, want %v — the keys are read after the gesture", s.blows, want)
	}
}

// TestATapStillSelectsWithoutAKeyBesideIt is the control for the case above: the
// same tap with no key pressed issues nothing, so the blow there is the key's
// and not something a tap does.
func TestATapStillSelectsWithoutAKeyBesideIt(t *testing.T) {
	a, v, s := affOnMap(t)
	affSelect(t, a, v, affLoCol, affLoRow, selection{affLoID})
	if len(s.blows) != 0 {
		t.Errorf("a tap on its own issued %v", s.blows)
	}
}
