package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

func TestSelectedUnitsUseOverheadStatusWithoutDiagnosticRims(t *testing.T) {
	v := commandViewer(t)
	v.SetEntities([]MapEntity{
		{ID: 5, Cell: image.Pt(2, 2), HP: 80, MaxHP: 100, Mana: 20, MaxMana: 40},
		{ID: 9, Cell: image.Pt(4, 2), HP: 60, MaxHP: 100, Mana: 10, MaxMana: 40},
	})
	v.sel = selection{5}
	v.healthBarsHidden = true
	for _, pass := range v.overlayPasses() {
		if pass.Color == terrain.SelectionMarkerColor {
			t.Fatal("ordinary selection draws a white cell rim")
		}
	}
	health, _ := v.healthBarScreenRects()
	mana, _ := v.manaBarScreenRects()
	if len(health) != 1 || len(mana) != 1 {
		t.Fatalf("selected status: health%d mana%d", len(health), len(mana))
	}
	v.sel = selection{5, 9}
	health, _ = v.healthBarScreenRects()
	mana, _ = v.manaBarScreenRects()
	if len(health) != 2 || len(mana) != 2 {
		t.Fatal("group selection omitted a member's status")
	}
	v.sel = nil
	health, _ = v.healthBarScreenRects()
	mana, _ = v.manaBarScreenRects()
	if len(health) != 0 || len(mana) != 0 {
		t.Fatal("hidden unselected status remained")
	}
}

func TestWaterStopsOutsideCurrentSightAndResumesWhenVisible(t *testing.T) {
	v, err := NewViewer("water fog", grid(4, 4), &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	v.SetAnimated(true)
	const word = uint16(8<<6 | 1<<4)
	const col, row = 2, 3
	for _, fog := range []byte{FogUnseen, FogExplored, FogVisible, FogExplored, FogVisible} {
		plane := make([]byte, 16)
		plane[row*4+col] = fog
		v.SetFog(plane, 4, 4)
		want := terrain.Resolve(word)
		if fog == FogVisible {
			want = terrain.ResolveAnimated(word, col, row, v.AnimationCounter())
		}
		if got := v.resolveCell(word, col, row); got != want {
			t.Fatalf("fog%d: frame%+v, want%+v", fog, got, want)
		}
	}
	if terrain.Resolve(word) == terrain.ResolveAnimated(word, col, row, v.AnimationCounter()) {
		t.Fatal("fixture cannot distinguish moving water")
	}
}
