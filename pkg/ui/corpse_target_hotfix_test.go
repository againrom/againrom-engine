package ui

import (
	"image"
	"testing"
)

func TestMapTargetHitKeepsFinishingBodiesAndSkipsTheMinusTenCorpse(t *testing.T) {
	placed := func(MapEntity) (screenRect, bool) {
		return screenRect{X: 0, Y: 0, W: 10, H: 10}, true
	}
	ents := []MapEntity{
		{ID: 2, Life: LifeDead, HP: -9},
		{ID: 1, Life: LifeDead, HP: -10, Untargetable: true},
	}
	if id, ok := targetAt(ents, placed, 5, 5, false); !ok || id != 2 {
		t.Fatalf("ordinary hit returned %d/%v, want finishable body 2", id, ok)
	}
	if id, ok := targetAt(ents[1:], placed, 5, 5, false); ok {
		t.Fatalf("ordinary hit named -10 corpse %d", id)
	}
	if id, ok := targetAt(ents, placed, 5, 5, true); !ok || id != 1 {
		t.Fatalf("Control Spirit hit returned %d/%v, want corpse 1", id, ok)
	}
}

func TestMinimapAttackSkipsUntargetableTopEntity(t *testing.T) {
	cell := image.Pt(8, 9)
	v := &Viewer{localOwner: 1, entities: []MapEntity{
		{ID: 2, Owner: 1, Cell: cell, HP: -9},
		{ID: 3, Owner: 1, Cell: cell, HP: -10, Untargetable: true},
	}}
	if id, ok := v.topEntityAtCell(cell); !ok || id != 2 {
		t.Fatalf("minimap target returned %d/%v, want finishable entity 2 below the corpse", id, ok)
	}
	v.entities = v.entities[1:]
	if id, ok := v.topEntityAtCell(cell); ok {
		t.Fatalf("minimap named untargetable corpse %d", id)
	}
}

func TestAttackCursorCanFinishAMinusNineBody(t *testing.T) {
	a, v, seam := atOnMap(t)
	ents := atEntities()
	for i := range ents {
		if ents[i].ID == atDeadID {
			ents[i].HP, ents[i].Untargetable = -9, false
		}
	}
	v.SetEntities(ents)
	v.sel = selection{atLoID}
	atArm(a, v)
	atPress(a, v, atDeadCol, atDeadRow)
	if len(seam.attacks) != 1 || seam.attacks[0].victim != atDeadID {
		t.Fatalf("attack cursor issued %+v, want one finish order on %d", seam.attacks, atDeadID)
	}
}

func TestFinishableBodyGetsAttackHoverAndArmedMarker(t *testing.T) {
	a, v, _ := atOnMap(t)
	ents := atEntities()
	for i := range ents {
		if ents[i].ID == atDeadID {
			ents[i].HP, ents[i].Untargetable, ents[i].Hostile = -9, false, true
		}
	}
	v.SetEntities(ents)
	v.sel = selection{atLoID}
	x, y := cellPoint(v, atDeadCol, atDeadRow)
	v.cursorX, v.cursorY, v.hasCursor = x, y, true
	if name, ok := v.missionHoverCursor(); !ok || name != "attack" {
		t.Fatalf("HP -9 hostile hover = %q/%v, want attack", name, ok)
	}

	a.step(afHeld(x, y), atAt)
	got, ok := v.attackTargetRect()
	if !ok {
		t.Fatal("armed attack drew no marker over HP -9 body")
	}
	var want screenRect
	for _, e := range ents {
		if e.ID == atDeadID {
			want, _ = v.entityPickRect(e)
		}
	}
	if got != want {
		t.Fatalf("HP -9 marker = %+v, want target rectangle %+v", got, want)
	}

	ents[1].HP, ents[1].Untargetable = -10, true
	v.SetEntities(ents)
	// Lower the held attack mode before asking the ordinary-hover cascade; an
	// armed mode deliberately names its own cursor regardless of the hover.
	a.step(atFrame(x, y), atAt)
	if name, ok := v.missionHoverCursor(); !ok || name != "move" {
		t.Fatalf("HP -10 corpse hover = %q/%v, want move", name, ok)
	}
	a.step(afHeld(x, y), atAt)
	if _, ok := v.attackTargetRect(); ok {
		t.Fatal("armed attack drew a marker over HP -10 corpse")
	}
}
