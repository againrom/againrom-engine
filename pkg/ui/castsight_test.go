package ui

import (
	"testing"
)

// castSightPlane is a 60x60 client fog plane, every vertex explored, with the
// listed vertices in sight now.
func castSightPlane(visible ...[2]int) []byte {
	plane := make([]byte, 60*60)
	for i := range plane {
		plane[i] = FogExplored
	}
	for _, c := range visible {
		plane[c[1]*60+c[0]] = FogVisible
	}
	return plane
}

// castSightFixture arms one cast arm over the panel caster: "area" and
// "target" from the open book, "item" through ArmItemCast. The item sink
// records each release it is handed.
func castSightFixture(t *testing.T, arm string, gate bool) (*App, *Viewer, *int) {
	t.Helper()
	a, v := castCursorApp(t)
	v.SetCastNeedsSight(gate)
	sent := new(int)
	v.SetItemCastSink(func(uint32, int, string, uint32, int, int, bool) { *sent++ })
	switch arm {
	case "area":
		pressPanelKey(t, a, "space")
		chooseSpell(t, v, castArea)
	case "target":
		pressPanelKey(t, a, "space")
		chooseSpell(t, v, castTarget)
	case "item":
		if !v.ArmItemCast(panelCasterID, 0, castArea, "scroll") {
			t.Fatal("setup: the item cast did not arm")
		}
	default:
		t.Fatalf("setup: unknown arm %q", arm)
	}
	return a, v, sent
}

// The cursor of each cast arm over a cell whose corners are all in sight, all
// explored and dark, and all never seen. The ground cell is (10,8); the target
// arm is tested over the unit at (7,3), which the fixture owns and so is drawn
// in the dark. Gate off is the control: the build before the gate.
func TestCastCursorNeedsACornerInCurrentSight(t *testing.T) {
	cells := map[string][2]int{"area": castCells.ground, "item": castCells.ground, "target": castCells.other}
	for _, arm := range []string{"area", "target", "item"} {
		cell := cells[arm]
		corners := [][2]int{{cell[0], cell[1]}, {cell[0] + 1, cell[1]}, {cell[0], cell[1] + 1}, {cell[0] + 1, cell[1] + 1}}
		unseen := make([]byte, 60*60)
		for _, c := range []struct {
			name  string
			plane []byte
			gate  bool
			want  string
		}{
			{"seen", castSightPlane(corners...), true, "cast"},
			{"explored, dark", castSightPlane(), true, "move"},
			{"never seen", unseen, true, "move"},
			{"control: explored, dark, gate off", castSightPlane(), false, "cast"},
			{"control: never seen, gate off", unseen, false, "cast"},
		} {
			t.Run(arm+"/"+c.name, func(t *testing.T) {
				a, v, _ := castSightFixture(t, arm, c.gate)
				v.SetFog(c.plane, 60, 60)
				if got := cursorAtCell(t, a, v, cell[0], cell[1]); got != c.want {
					t.Fatalf("cursor %q, want %q", got, c.want)
				}
				x, y := cellPoint(v, cell[0], cell[1])
				if got := v.gestureCursorAt(x, y); got != c.want {
					t.Fatalf("the click is made under %q, want %q", got, c.want)
				}
			})
		}
	}
}

// One corner in sight passes the cell, whichever corner it is; a vertex that
// is no corner of the cell does not. The four cells sharing vertex (8,6)
// pass; the cells one step beyond them fail.
func TestCastCursorOneCornerInSightIsEnough(t *testing.T) {
	a, v, _ := castSightFixture(t, "area", true)
	v.SetFog(castSightPlane([2]int{8, 6}), 60, 60)
	for _, c := range []struct {
		cell [2]int
		want string
	}{
		{[2]int{7, 5}, "cast"}, {[2]int{8, 5}, "cast"}, {[2]int{7, 6}, "cast"}, {[2]int{8, 6}, "cast"},
		{[2]int{9, 7}, "move"}, {[2]int{6, 4}, "move"}, {[2]int{9, 5}, "move"}, {[2]int{7, 7}, "move"},
	} {
		if got := cursorAtCell(t, a, v, c.cell[0], c.cell[1]); got != c.want {
			t.Fatalf("cell %v with only vertex (8,6) in sight: cursor %q, want %q", c.cell, got, c.want)
		}
	}
}

// A click under the move cursor issues the ordinary move to that cell and no
// cast. A book spell stays chosen; an item cast is dropped and its sink is
// not called. The control is the same click with the gate off, which casts.
func TestCastClickOnDarkGroundMoves(t *testing.T) {
	cells := map[string][2]int{"area": castCells.ground, "item": castCells.ground, "target": castCells.other}
	for _, arm := range []string{"area", "target", "item"} {
		t.Run(arm, func(t *testing.T) {
			cell := cells[arm]
			_, v, sent := castSightFixture(t, arm, true)
			v.SetFog(castSightPlane(), 60, 60)
			x, y := cellPoint(v, cell[0], cell[1])
			ords, ok := tapAt(v, x, y)
			if !ok || len(ords) != 1 || ords[0].kind != orderKindMove || ords[0].spell != 0 ||
				ords[0].x != cell[0] || ords[0].y != cell[1] || ords[0].entity != panelCasterID {
				t.Fatalf("click on dark ground: %+v, want one move of the caster to %v", ords, cell)
			}
			if *sent != 0 {
				t.Fatal("click on dark ground released the item cast")
			}
			if arm == "item" {
				if v.itemCast != nil {
					t.Fatal("the item cast stayed armed after the move")
				}
			} else if v.selectedSpell == 0 || !v.spellModeLive() {
				t.Fatalf("the book spell was dropped by the move: spell %d live %v", v.selectedSpell, v.spellModeLive())
			}

			_, v, sent = castSightFixture(t, arm, false)
			v.SetFog(castSightPlane(), 60, 60)
			x, y = cellPoint(v, cell[0], cell[1])
			ords, _ = tapAt(v, x, y)
			if arm == "item" {
				// Gate off keeps the single-cell sack gate, which refuses a
				// dark cell without a move.
				if len(ords) != 0 || *sent != 0 || v.itemCast != nil {
					t.Fatalf("control, item: orders %+v sent %d armed %v", ords, *sent, v.itemCast != nil)
				}
				return
			}
			if len(ords) != 1 || ords[0].kind != orderKindCast {
				t.Fatalf("control, gate off: %+v, want the cast", ords)
			}
		})
	}
}

// A click on a cell in sight casts as before, for each arm.
func TestCastClickOnSeenGroundCasts(t *testing.T) {
	cells := map[string][2]int{"area": castCells.ground, "item": castCells.ground, "target": castCells.other}
	for _, arm := range []string{"area", "target", "item"} {
		t.Run(arm, func(t *testing.T) {
			cell := cells[arm]
			_, v, sent := castSightFixture(t, arm, true)
			v.SetFog(castSightPlane([2]int{cell[0] + 1, cell[1] + 1}), 60, 60)
			x, y := cellPoint(v, cell[0], cell[1])
			ords, _ := tapAt(v, x, y)
			if arm == "item" {
				if *sent != 1 || len(ords) != 0 {
					t.Fatalf("item click in sight: sent %d orders %+v", *sent, ords)
				}
				return
			}
			if len(ords) != 1 || ords[0].kind != orderKindCast {
				t.Fatalf("click in sight: %+v, want the cast", ords)
			}
		})
	}
}

// Teleport takes the sight gate of every spell (MAGIC-TELEPORT-364).
func TestTeleportAimsOnlyAtGroundInSight(t *testing.T) {
	const teleport = 26
	for _, c := range []struct {
		name string
		lit  bool
		want string
	}{{"never seen", false, "move"}, {"in sight", true, "cast"}} {
		a, v := castCursorApp(t)
		v.SetCastNeedsSight(true)
		v.SetSpellbook(panelCasterID, append(castKindBook(), SpellEntry{ID: teleport, Name: "Teleport", PointTarget: true}))
		pressPanelKey(t, a, "space")
		chooseSpell(t, v, teleport)
		cell := castCells.ground
		plane := make([]byte, 60*60)
		if c.lit {
			plane = castSightPlane([2]int{cell[0] + 1, cell[1] + 1})
		}
		v.SetFog(plane, 60, 60)
		if got := cursorAtCell(t, a, v, cell[0], cell[1]); got != c.want {
			t.Fatalf("Teleport over a cell %s: cursor %q, want %q", c.name, got, c.want)
		}
		x, y := cellPoint(v, cell[0], cell[1])
		ords, _ := tapAt(v, x, y)
		if len(ords) != 1 || (ords[0].kind == orderKindCast) != (c.want == "cast") {
			t.Fatalf("Teleport click on a cell %s: %+v, want %s", c.name, ords, c.want)
		}
	}
}
