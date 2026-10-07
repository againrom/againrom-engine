package ui

// The arming key and the press it arms, driven through the SHIPPED front-end
// path: a whole App on the map screen, real gestures, and the seams the loader
// handed over as the only place an order can be observed.
//
// Nothing here calls a seam directly. What 0075 fixes is which press makes which
// order and what gates the key, and both are properties of App.step — a test that
// reached past it would witness the closure and not the press.

import (
	"image"
	"reflect"
	"slices"
	"testing"
	"time"
)

// atAt is the frozen clock every frame below is stamped with. Nothing on this
// path reads a clock; the instant only has to be stable.
var atAt = time.Unix(1_700_000_000, 0)

// The units the cases select and aim at. Ids are SPARSE and out of ascending
// order in the snapshot, so a walk emitting slice positions or snapshot order
// rather than the selection's own would not produce the lists below.
const (
	atLoID, atLoCol, atLoRow = 4, 3, 3
	atHiID, atHiCol, atHiRow = 11, 5, 3
	atFoeID, atFoeCol        = 9, 7
	atFoeRow                 = 3
	atDeadID, atDeadCol      = 7, 4
	atDeadRow                = 5
	atEmptyCol, atEmptyRow   = 1, 1
)

// atOwner is a roster slot no entity below carries unless the case says so, so a
// gate that compared against the wrong operand would answer differently.
const atOwner uint32 = 3

// atEntities is the snapshot: two units to order, one to aim at, and a corpse.
func atEntities() []MapEntity {
	return []MapEntity{
		{ID: atLoID, Cell: image.Pt(atLoCol, atLoRow), Life: LifeAlive, HP: 100, MaxHP: 100},
		{ID: atDeadID, Cell: image.Pt(atDeadCol, atDeadRow), Life: LifeDead, HP: -10, MaxHP: 100, Untargetable: true},
		{ID: atFoeID, Cell: image.Pt(atFoeCol, atFoeRow), Life: LifeAlive, HP: 100, MaxHP: 100},
		{ID: atHiID, Cell: image.Pt(atHiCol, atHiRow), Life: LifeAlive, HP: 100, MaxHP: 100},
	}
}

// atOnMap parks a whole App on the map screen over a loader that hands out a
// recording seam, with the camera at the origin and the snapshot in place.
func atOnMap(t *testing.T) (*App, *Viewer, *mapSeam) {
	t.Helper()
	seams := &[]*mapSeam{}
	a := newTestApp(t, appRows(3), seamLoader(t, seams))
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, atAt)
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	v.SetEntities(atEntities())
	return a, v, (*seams)[0]
}

// atFrame is one tick of front-end input at a window position, with the viewer's
// own cursor set to the same place — which is what readAppInput does.
func atFrame(x, y int) appInput {
	return appInput{Viewer: Input{CursorX: x, CursorY: y}, CursorX: x, CursorY: y}
}

// atArm presses the arming key with the cursor over empty ground, so the key is
// the only thing the frame carries.
func atArm(a *App, v *Viewer) {
	x, y := cellPoint(v, atEmptyCol, atEmptyRow)
	in := atFrame(x, y)
	in.Attack = true
	a.step(in, atAt)
}

func atPress(a *App, v *Viewer, col, row int) {
	x, y := cellPoint(v, col, row)
	atTapPoint(a, x, y)
}

// atTapPoint is atPress at a window position rather than a cell.
func atTapPoint(a *App, x, y int) {
	down := atFrame(x, y)
	down.PrimaryPressed = true
	down.Viewer.PrimaryDown = true
	a.step(down, atAt)
	up := atFrame(x, y)
	up.PrimaryReleased = true
	a.step(up, atAt)
}

// atRightClick is the secondary button's own release with no preceding move,
// which is `AI-INPUT-127`'s CLICK: it cancels an armed mode, or deselects all
// when no mode is armed. It never orders.
func atRightClick(a *App, v *Viewer, col, row int) {
	x, y := cellPoint(v, col, row)
	in := atFrame(x, y)
	in.SecondaryReleased = true
	a.step(in, atAt)
}

// atDrag is a marquee across the closed cell range, corner to corner: press,
// one moving frame, release. A marquee is always a selection and never an
// order, whatever cursor it was released under (`AI-SELECT-122`).
func atDrag(a *App, v *Viewer, c0, r0, c1, r1 int) {
	x0, y0, x1, y1 := cellSpan(v, c0, r0, c1, r1)
	down := atFrame(x0, y0)
	down.PrimaryPressed = true
	down.Viewer.PrimaryDown = true
	a.step(down, atAt)
	mid := atFrame(x1, y1)
	mid.Viewer.PrimaryDown = true
	a.step(mid, atAt)
	up := atFrame(x1, y1)
	up.PrimaryReleased = true
	a.step(up, atAt)
}

// TestArmingIsGatedOnOwnershipAndNotOnClass is AC-1 and AC-1a.
//
// THE LAST CASE IS THE ONE THAT MATTERS. A clause that once said the selection's
// capability MASK gates this was retracted when the routine behind it was read:
// it tests no unit class anywhere. So the same two selections are asked with
// their classes made to differ in every way this seam can carry — name and art —
// and the answer must not move.
func TestArmingIsGatedOnOwnershipAndNotOnClass(t *testing.T) {
	t.Run("no local participant leaves the gate open", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		v.sel = selection{atLoID}
		atArm(a, v)
		if !v.AttackArmed() {
			t.Error("a selection with a present unit did not arm with no participant established")
		}
	})

	t.Run("an empty selection arms nothing", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		atArm(a, v)
		if v.AttackArmed() {
			t.Error("an empty selection armed")
		}
	})

	t.Run("a selection the snapshot has killed arms nothing", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		v.sel = selection{atDeadID}
		atArm(a, v)
		if v.AttackArmed() {
			t.Error("a selection of one corpse armed")
		}
	})

	t.Run("a second press lowers it", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		v.sel = selection{atLoID}
		atArm(a, v)
		atArm(a, v)
		if v.AttackArmed() {
			t.Error("the key did not lower a mode it had raised")
		}
	})

	t.Run("the primary present id's owner decides", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			loOwner    uint32
			hiOwner    uint32
			wantArmed  bool
			whatItSays string
		}{
			// The selection is {atLoID, atHiID} and atLoID is the lower, so it
			// is the primary: the case that differs only in WHICH of the two
			// carries the participant's slot is what says the primary is read.
			{"the primary is the participant's", atOwner, 0, true, "armed"},
			{"only a later member is the participant's", 0, atOwner, false, "did not arm"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				a, v, _ := atOnMap(t)
				ents := atEntities()
				for i := range ents {
					switch ents[i].ID {
					case atLoID:
						ents[i].Owner = tc.loOwner
					case atHiID:
						ents[i].Owner = tc.hiOwner
					}
				}
				v.SetEntities(ents)
				v.SetLocalOwner(atOwner)
				v.sel = selection{atLoID, atHiID}
				atArm(a, v)
				if v.AttackArmed() != tc.wantArmed {
					t.Errorf("armed = %v, want %v (%s)", v.AttackArmed(), tc.wantArmed, tc.whatItSays)
				}
			})
		}
	})

	// The gate reads an owner and nothing else about a unit. Two selections that
	// differ in every class-shaped field this seam carries, and agree on the
	// owner, must answer the same — which is what stops a class test being added
	// back by somebody who remembers the retracted clause and not its retraction.
	t.Run("class, name and art do not enter", func(t *testing.T) {
		for _, name := range []string{"", "Goblin", "Catapult"} {
			a, v, _ := atOnMap(t)
			ents := atEntities()
			for i := range ents {
				ents[i].Name = name
				ents[i].Owner = atOwner
			}
			v.SetEntities(ents)
			v.SetLocalOwner(atOwner)
			v.sel = selection{atLoID}
			atArm(a, v)
			if !v.AttackArmed() {
				t.Errorf("name %q: the gate refused a unit its owner admits", name)
			}
		}
	})
}

// TestAnArmedPressMakesOneOfTwoOrders is AC-2, AC-3 and AC-5 — the decoded
// property this whole story turns on: ONE press under an armed attack makes an
// attack order when a unit is under it and a plain move to the cell when none is.
func TestAnArmedPressMakesOneOfTwoOrders(t *testing.T) {
	t.Run("over a unit it attacks, and orders no move", func(t *testing.T) {
		a, v, s := atOnMap(t)
		v.sel = selection{atLoID, atDeadID, atHiID}
		atArm(a, v)
		atPress(a, v, atFoeCol, atFoeRow)

		want := []aimed{{atLoID, atFoeID, 0}, {atHiID, atFoeID, 0}}
		if !reflect.DeepEqual(s.attacks, want) {
			t.Errorf("the attack seam received %v, want %v", s.attacks, want)
		}
		if len(s.orders) != 0 {
			t.Errorf("the move seam received %v, want nothing", s.orders)
		}
	})

	t.Run("over empty ground it moves, exactly as an unarmed press does", func(t *testing.T) {
		armed, v1, s1 := atOnMap(t)
		v1.sel = selection{atLoID, atHiID}
		atArm(armed, v1)
		atPress(armed, v1, atEmptyCol, atEmptyRow)

		plain, v2, s2 := atOnMap(t)
		v2.sel = selection{atLoID, atHiID}
		atPress(plain, v2, atEmptyCol, atEmptyRow)

		if !reflect.DeepEqual(s1.orders, s2.orders) {
			t.Errorf("armed over ground gave %v; unarmed gave %v — they must be the same press",
				s1.orders, s2.orders)
		}
		if len(s1.orders) == 0 {
			t.Error("neither press ordered anything; the fixture proves nothing")
		}
		if len(s1.attacks) != 0 {
			t.Errorf("a press over empty ground attacked %v", s1.attacks)
		}
	})

	// AN UNARMED TAP OVER A NON-HOSTILE UNIT SELECTS IT (`AI-CURSOR-226` arm
	// 4). Until this story the same gesture ordered a move onto that unit's
	// cell, the secondary button ordering ground and never picking. Nothing
	// reaches either seam.
	t.Run("an unarmed tap over a non-hostile unit selects it and orders nothing", func(t *testing.T) {
		a, v, s := atOnMap(t)
		v.sel = selection{atLoID}
		atPress(a, v, atFoeCol, atFoeRow)
		if len(s.attacks) != 0 {
			t.Errorf("an unarmed tap attacked %v", s.attacks)
		}
		if len(s.orders) != 0 {
			t.Errorf("an unarmed tap over a unit ordered %v, want nothing", s.orders)
		}
		if want := (selection{atFoeID}); !slices.Equal(v.sel, want) {
			t.Errorf("the tap left %v selected, want %v", v.sel, want)
		}
	})

	t.Run("an unarmed tap over a hostile unit attacks it", func(t *testing.T) {
		a, v, s := atOnMap(t)
		ents := atEntities()
		for i := range ents {
			if ents[i].ID == atFoeID {
				ents[i].Hostile = true
			}
		}
		v.SetEntities(ents)
		v.sel = selection{atLoID}
		if v.AttackArmed() {
			t.Fatal("setup: the attack mode is armed; this case must run with it down")
		}
		atPress(a, v, atFoeCol, atFoeRow)
		want := []aimed{{atLoID, atFoeID, 0}}
		if !reflect.DeepEqual(s.attacks, want) {
			t.Errorf("the attack seam received %v, want %v", s.attacks, want)
		}
		if len(s.orders) != 0 {
			t.Errorf("the move seam received %v, want nothing", s.orders)
		}
	})
}

func TestTheArmIsSpentByOnePress(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(a *App, v *Viewer)
	}{
		{"a press that attacked", func(a *App, v *Viewer) { atPress(a, v, atFoeCol, atFoeRow) }},
		{"a press that moved", func(a *App, v *Viewer) { atPress(a, v, atEmptyCol, atEmptyRow) }},
		{"a tap outside the extent", func(a *App, v *Viewer) {
			atTapPoint(a, outsideAfter[0], outsideAfter[1])
		}},
		{"a right click, which is the cancel", func(a *App, v *Viewer) {
			// `AI-INPUT-127`: right up with no marked drag cancels an armed
			// mode. It is a second door to the same lowering and not a
			// different rule.
			atRightClick(a, v, atEmptyCol, atEmptyRow)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v, _ := atOnMap(t)
			v.sel = selection{atLoID}
			atArm(a, v)
			if !v.AttackArmed() {
				t.Fatal("setup: the mode did not arm")
			}
			tc.run(a, v)
			if v.AttackArmed() {
				t.Error("the mode survived the gesture")
			}
		})
	}

	// A MARQUEE LEAVES IT UP, and it is the marquee rather than the tap that
	// does so now. A rectangle goes to the selection routine and never reaches
	// the cursor-to-order dispatch (`AI-SELECT-122`, `AI-CLICK-050`), so it is
	// not the press the mode was armed for; a TAP does reach the dispatch and
	// does spend it, which the cases above assert.
	t.Run("a marquee leaves it up and the next tap orders the new selection", func(t *testing.T) {
		a, v, s := atOnMap(t)
		v.sel = selection{atLoID}
		atArm(a, v)

		atDrag(a, v, atHiCol, atHiRow, atHiCol, atHiRow)
		if !v.AttackArmed() {
			t.Fatal("a marquee spent the mode")
		}
		if !slices.Equal(v.sel, selection{atHiID}) {
			t.Fatalf("the marquee left %v, want the unit it covered", v.sel)
		}

		atPress(a, v, atFoeCol, atFoeRow)
		want := []aimed{{atHiID, atFoeID, 0}}
		if !reflect.DeepEqual(s.attacks, want) {
			t.Errorf("the seam received %v, want %v — the tap orders the selection it FOUND", s.attacks, want)
		}
	})
}

// TestTheVictimIsTheOrdinaryHitTest is AC-6: the same test and the same lowest-id
// tie rule a selecting tap uses, and a corpse is not a candidate.
func TestTheVictimIsTheOrdinaryHitTest(t *testing.T) {
	t.Run("two rectangles on one point yield the lower id", func(t *testing.T) {
		a, v, s := atOnMap(t)
		// Two live units stacked on ONE cell, given in descending id so nothing
		// answering with slice position could agree.
		v.SetEntities([]MapEntity{
			{ID: 30, Cell: image.Pt(atFoeCol, atFoeRow), Life: LifeAlive, HP: 100, MaxHP: 100},
			{ID: 20, Cell: image.Pt(atFoeCol, atFoeRow), Life: LifeAlive, HP: 100, MaxHP: 100},
			{ID: atLoID, Cell: image.Pt(atLoCol, atLoRow), Life: LifeAlive, HP: 100, MaxHP: 100},
		})
		v.sel = selection{atLoID}
		atArm(a, v)
		atPress(a, v, atFoeCol, atFoeRow)
		want := []aimed{{atLoID, 20, 0}}
		if !reflect.DeepEqual(s.attacks, want) {
			t.Errorf("the seam received %v, want %v", s.attacks, want)
		}
	})

	t.Run("a corpse is not a victim and the press falls through to the move", func(t *testing.T) {
		a, v, s := atOnMap(t)
		v.sel = selection{atLoID}
		atArm(a, v)
		atPress(a, v, atDeadCol, atDeadRow)
		if len(s.attacks) != 0 {
			t.Errorf("a corpse was attacked: %v", s.attacks)
		}
		if len(s.orders) != 1 {
			t.Errorf("the move seam received %v, want the one order the fall-through makes", s.orders)
		}
	})
}

func TestTheFanOutIsUnbounded(t *testing.T) {
	const n = 300
	ents := []MapEntity{{ID: atFoeID, Cell: image.Pt(atFoeCol, atFoeRow), Life: LifeAlive, HP: 100, MaxHP: 100}}
	sel := make(selection, 0, n)
	for i := 0; i < n; i++ {
		// Ids well clear of the victim's, on one far cell so none of them is
		// under the press.
		id := uint32(1000 + i)
		ents = append(ents, MapEntity{ID: id, Cell: image.Pt(20, 20), Life: LifeAlive, HP: 100, MaxHP: 100})
		sel = append(sel, id)
	}
	a, v, s := atOnMap(t)
	v.SetEntities(ents)
	v.sel = sel
	atArm(a, v)
	atPress(a, v, atFoeCol, atFoeRow)

	if len(s.attacks) != n {
		t.Fatalf("the seam received %d order(s) for %d selected units", len(s.attacks), n)
	}
	for i, got := range s.attacks {
		if got.entity != sel[i] || got.victim != atFoeID {
			t.Fatalf("order %d is %+v, want entity %d onto %d — ascending id, none dropped",
				i, got, sel[i], atFoeID)
		}
	}
}

// TestLeavingTheMapScreenDisarms is AC-13. The mode lives on the viewer, which is
// dropped when the map screen is left, so this is a property of where it lives
// rather than of a statement somebody remembered to write.
func TestLeavingTheMapScreenDisarms(t *testing.T) {
	a, v, _ := atOnMap(t)
	v.sel = selection{atLoID}
	atArm(a, v)
	if !v.AttackArmed() {
		t.Fatal("setup: the mode did not arm")
	}
	a.step(appInput{Escape: true}, atAt)
	leaveViaMenu(a.flow) //
	if a.Screen() == ScreenMap {
		t.Fatal("setup: Esc did not leave the map screen")
	}
	if a.flow.viewer != nil {
		t.Fatal("the flow still holds a viewer after leaving the map")
	}
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, atAt)
	if a.Screen() != ScreenMap {
		t.Fatalf("re-entry: screen = %v, want ScreenMap", a.Screen())
	}
	if a.flow.viewer.AttackArmed() {
		t.Error("a freshly opened map screen is armed")
	}
}
