package ui

import (
	"image"
	"reflect"
	"testing"
)

func TestFinishableBodyCanBeClickedSelectedAndOrderedWhileTerminalCorpseCannot(t *testing.T) {
	placed := func(MapEntity) (screenRect, bool) {
		return screenRect{X: 0, Y: 0, W: 10, H: 10}, true
	}
	fallen := MapEntity{ID: 7, Cell: image.Pt(2, 2), Life: LifeDead, HP: -9, MaxHP: 20, Selectable: true}
	terminal := MapEntity{ID: 8, Cell: image.Pt(3, 2), Life: LifeDead, HP: -10, MaxHP: 20}

	if id, ok := topAt([]MapEntity{fallen}, placed, 5, 5); !ok || id != fallen.ID {
		t.Fatalf("finishable body click = %d/%v, want %d/true", id, ok, fallen.ID)
	}
	if _, ok := topAt([]MapEntity{terminal}, placed, 5, 5); ok {
		t.Fatal("terminal corpse remained clickable")
	}
	if got := presentSelected(selection{fallen.ID}, []MapEntity{fallen}); !reflect.DeepEqual(got, []MapEntity{fallen}) {
		t.Fatalf("selected finishable body filtered to %+v", got)
	}

	ground := func(float64, float64) (int, int, bool) { return 6, 6, true }
	_, _, orders, ok := decide(selection{fallen.ID}, []MapEntity{fallen}, ground, placed, 1,
		gesture{tap: true, x: 5, y: 5, cursor: "move"})
	want := []order{{kind: orderKindMove, entity: fallen.ID, x: 6, y: 6}}
	if !ok || !reflect.DeepEqual(orders, want) {
		t.Fatalf("selected finishable body orders = %+v/%v, want %+v/true", orders, ok, want)
	}
}

func TestDamageJoltMovesOnlyTheSpriteInsideAStablePickFootprint(t *testing.T) {
	v := overlayViewer(t, 100, 80, 400, 300)
	art := entityArtA()
	base := withFrame(image.Pt(2, 2), art)
	base.ID, base.Life = 7, LifeDead
	hit := base
	hit.DamageJolt = image.Pt(2, 0)

	v.SetEntities([]MapEntity{base})
	baseSprites, _, _ := v.entityLayer()
	basePick, _ := v.entityPickRect(base)
	v.SetEntities([]MapEntity{hit})
	hitSprites, _, _ := v.entityLayer()
	hitPick, _ := v.entityPickRect(hit)

	if len(baseSprites) != 1 || len(hitSprites) != 1 {
		t.Fatalf("sprite counts = %d/%d, want 1/1", len(baseSprites), len(hitSprites))
	}
	if got := hitSprites[0].TopLeft.Sub(baseSprites[0].TopLeft); got != hit.DamageJolt {
		t.Fatalf("sprite jolt = %v, want %v", got, hit.DamageJolt)
	}
	if hitPick != basePick {
		t.Fatalf("damage jolt moved pick rect from %+v to %+v", basePick, hitPick)
	}
}

func TestSelectAllKeepsOwnedFinishableBodiesButNotTerminalCorpses(t *testing.T) {
	v := commandViewer(t)
	v.SetLocalOwner(1)
	v.SetEntities([]MapEntity{
		{ID: 4, Life: LifeAlive, Owner: 1},
		{ID: 7, Life: LifeDead, HP: -9, MaxHP: 20, Owner: 1, Selectable: true},
		{ID: 8, Life: LifeDead, HP: -10, MaxHP: 20, Owner: 1},
		{ID: 9, Life: LifeDead, HP: -9, MaxHP: 20, Owner: 2, Selectable: true},
	})

	v.selectAllOwnedUnits()
	if want := (selection{4, 7}); !reflect.DeepEqual(v.sel, want) {
		t.Fatalf("select-all result = %v, want owned selectable actors %v", v.sel, want)
	}
}
