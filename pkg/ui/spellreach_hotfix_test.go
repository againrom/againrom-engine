package ui

import (
	"image"
	"testing"
)

func TestBookPointSpellKeepsClickedGroundUnderCreature(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities([]MapEntity{
		{ID: 5, Cell: image.Pt(3, 3), Life: LifeAlive},
		{ID: 9, Cell: image.Pt(7, 3), Life: LifeAlive},
	})
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	v.sel = selection{5}
	for _, point := range []bool{true, false} {
		book := []SpellEntry{{ID: 2, Name: "Fire Ball", PointTarget: point}}
		v.cancelMapCommand()
		if !v.hudShown(hudPanelBook) {
			v.toggleHudPanel(hudPanelBook)
		}
		v.sel = selection{5}
		v.SetSpellbook(5, book)
		bx, by := sbEntryPoint(t, v, book, 0)
		sbClick(v, bx, by)
		x, y := cellPoint(v, 7, 3)
		col, row, inside := v.groundCellAt(float64(x), float64(y))
		orders, ok := tapAt(v, x, y)
		if !ok || len(orders) != 1 || orders[0].kind != orderKindCast || orders[0].spell != 2 {
			t.Fatalf("point=%t: orders=%+v selected=%d armed=%t cursor=%s ground=%d,%d,%t", point, orders, v.selectedSpell, v.spellArmed, v.gestureCursorAt(x, y), col, row, inside)
		}
		o := orders[0]
		if point && (!o.cell || o.victim != 0 || o.x != 7 || o.y != 3) {
			t.Fatalf("point spell attached to creature: %+v", o)
		}
		if !point && (o.cell || o.victim != 9) {
			t.Fatalf("unit spell lost creature target: %+v", o)
		}
	}
}

func TestPointSpellClickCastsWhateverTheFogOverTheCell(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(3, 3), Life: LifeAlive}})
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	v.sel = selection{5}
	for _, spell := range []uint32{2, 26} {
		v.SetSpellbook(5, []SpellEntry{{ID: spell, PointTarget: true}})
		v.selectedSpell = spell
		v.armSpell()
		for corner := -2; corner < 4; corner++ {
			plane := make([]byte, 16*16)
			if corner != -2 {
				for i := range plane {
					plane[i] = FogExplored
				}
			}
			if corner >= 0 {
				p := [4]image.Point{{7, 3}, {8, 3}, {7, 4}, {8, 4}}[corner]
				plane[p.Y*16+p.X] = FogVisible
			}
			v.SetFog(plane, 16, 16)
			x, y := cellPoint(v, 7, 3)
			orders, ok := tapAt(v, x, y)
			if !ok || len(orders) != 1 {
				t.Fatalf("spell=%d corner=%d: orders=%+v ok=%t", spell, corner, orders, ok)
			}
			if orders[0].kind != orderKindCast || !orders[0].cell {
				t.Fatalf("spell=%d corner=%d: got %+v want a cast at the cell", spell, corner, orders[0])
			}
			if v.selectedSpell != spell || !v.spellArmed {
				t.Fatal("A cast cleared the armed spell")
			}
		}
	}
}
