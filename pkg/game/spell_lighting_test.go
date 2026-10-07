package game

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func spellLightingWorld(t *testing.T, id uint16) *sim.World {
	t.Helper()
	// One release spends the complete pool, so the retained order cannot refresh
	// the short-lived lighting record while this presentation witness observes it.
	caster := sim.Entity{ID: 1, X: 10, Y: 10, HP: 100, MaxHP: 100, Owner: sim.SelfSlot,
		Mind: 20, Mana: 1, MaxMana: 1, KnownSpells: 1 << id, ScanRange: 20,
		AttackCharge: 1, AttackRelax: 0}
	rule := sim.SpellRule{ID: id, ManaCost: 1, School: 3, MaxRange: 10, Area: true,
		Distribution: 3, Radius: 2, AreaDuration: 1}
	w, err := sim.NewSpelledWorld(0x11a7+uint64(id), sim.Bounds{Width: 32, Height: 32},
		sim.ModeCanonical, nil, []sim.Entity{caster}, nil, []sim.SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindCastAt, Entity: 1, X: 12, Y: 10, Spell: id}})
	for i := 0; i < 32 && len(w.CellEffects()) == 0; i++ {
		sim.Step(w, nil)
	}
	if len(w.CellEffects()) == 0 {
		t.Fatalf("spell %d produced no live area record", id)
	}
	return w
}

func liveWallLightingWorld(t *testing.T) *sim.World {
	t.Helper()
	// The stacking witness below needs exactly two releases and no third refresh.
	caster := sim.Entity{ID: 1, X: 20, Y: 20, HP: 100, MaxHP: 100, Owner: sim.SelfSlot,
		Mind: 20, Mana: 2, MaxMana: 2, KnownSpells: 1 << 3, ScanRange: 20,
		AttackCharge: 1, AttackRelax: 0}
	rule := sim.SpellRule{ID: 3, ManaCost: 1, School: 1, MaxRange: 30, Area: true,
		Distribution: 4, Radius: 2, AreaDuration: 15}
	w, err := sim.NewSpelledWorld(0xf17e, sim.Bounds{Width: 64, Height: 64},
		sim.ModeCanonical, nil, []sim.Entity{caster}, nil, []sim.SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindCastAt, Entity: 1, X: 24, Y: 20, Spell: 3}})
	for i := 0; i < 32 && len(w.CellEffects()) == 0; i++ {
		sim.Step(w, nil)
	}
	if len(w.CellEffects()) != 1 {
		t.Fatalf("Wall of Fire produced %d live records, want 1", len(w.CellEffects()))
	}
	return w
}

func spellLightCellMap(cells []ui.SpellLightCell) map[image.Point]ui.SpellLightCell {
	out := make(map[image.Point]ui.SpellLightCell, len(cells))
	for _, cell := range cells {
		out[cell.Cell] = cell
	}
	return out
}

func TestLiveLightAndDarknessRecordsBuildTheirAbsoluteBrightnessPlanes(t *testing.T) {
	for _, tc := range []struct {
		id              uint16
		terrain, sprite float32
	}{
		{lightSpellID, 3, 2},
		{darknessSpellID, 0.5, 0.5},
	} {
		w := spellLightingWorld(t, tc.id)
		mw := &mapWorld{world: w}
		got := mw.spellLighting()
		if len(got) < 5 {
			t.Fatalf("spell %d lights %d cells, want its circular area", tc.id, len(got))
		}
		foundCentre := false
		for _, c := range got {
			if c.Terrain != tc.terrain || c.Sprite != tc.sprite {
				t.Fatalf("spell %d cell %v has terrain/sprite %v/%v, want %v/%v",
					tc.id, c.Cell, c.Terrain, c.Sprite, tc.terrain, tc.sprite)
			}
			if c.Cell == (image.Point{X: 12, Y: 10}) {
				foundCentre = true
			}
		}
		if !foundCentre {
			t.Fatalf("spell %d brightness plane omitted its aimed centre", tc.id)
		}
		for i := 0; i < 64 && len(w.CellEffects()) > 0; i++ {
			sim.Step(w, nil)
		}
		if got := mw.spellLighting(); len(got) != 0 {
			t.Fatalf("expired spell %d left %d brightness cells", tc.id, len(got))
		}
	}
}

func TestUnitLightSpellKindPopulationIsClosed(t *testing.T) {
	for id := uint16(1); id <= 28; id++ {
		got := spellLightingCells([]sim.CellEffect{{
			Spell: id, Mode: sim.AreaModeCloud, Cells: [][2]int32{{10, 10}},
		}}, sim.Bounds{Width: 32, Height: 32}, 0)
		switch id {
		case 3:
			if len(got) != 9 {
				t.Errorf("spell %d produced %d unit-light cells, want the radius-one square's 9", id, len(got))
			}
		case lightSpellID:
			if len(got) != 1 || got[0].Terrain != 3 || got[0].Sprite != 2 {
				t.Errorf("Light plane = %+v, want one terrain/sprite 3/2 cell", got)
			}
		case darknessSpellID:
			if len(got) != 1 || got[0].Terrain != 0.5 || got[0].Sprite != 0.5 {
				t.Errorf("Darkness plane = %+v, want one terrain/sprite 0.5/0.5 cell", got)
			}
		default:
			if len(got) != 0 {
				t.Errorf("non-light spell %d produced %+v", id, got)
			}
		}
	}
}

func TestLiveWallOfFireLightsActorsAroundItsCellsAndRestoresOnUpdate(t *testing.T) {
	w := liveWallLightingWorld(t)
	mw := &mapWorld{world: w, scene: 0}
	oneWall := mw.spellLighting()
	lit := spellLightCellMap(oneWall)

	// The east-facing wall is the hand-decoded 2x5 block at x=24..25,
	// y=18..22. Source (24,18) is bright at scene 0 because
	// (0/2 + 24*18)/5 = 86, whose low bit is zero. Its complete radius-one
	// square must therefore reach actor cells (23..25,17..19).
	for y := 17; y <= 19; y++ {
		for x := 23; x <= 25; x++ {
			cell, ok := lit[image.Pt(x, y)]
			if !ok || cell.Terrain != 0 || cell.Sprite != 2 {
				t.Errorf("wall actor-light cell (%d,%d) = %+v, present=%v; want sprite-only gain 2", x, y, cell, ok)
			}
		}
	}
	if _, ok := lit[image.Pt(10, 10)]; ok {
		t.Error("a cell outside every radius-one wall splat received actor lighting")
	}

	// Owner-directed wall stacking remains canonical gameplay state, but two
	// identical sources do not double a presentation gain: the decoded light
	// grid retains one value per cell and overlapping splats take the brighter.
	sim.Step(w, []sim.Command{{Kind: sim.KindCastAt, Entity: 1, X: 24, Y: 20, Spell: 3}})
	for i := 0; i < 32 && len(w.CellEffects()) < 2; i++ {
		sim.Step(w, nil)
	}
	if len(w.CellEffects()) != 2 {
		t.Fatalf("the recast retained %d walls, want two overlapping records", len(w.CellEffects()))
	}
	if got := mw.spellLighting(); !reflect.DeepEqual(got, oneWall) {
		t.Fatalf("an identical stacked wall changed the derived actor-light plane\none:   %+v\nstack: %+v", oneWall, got)
	}

	// Ten scene ticks flip the isolated northwest corner source to decoded
	// level 12: (10/2 + 24*18)/5 = 87. Level 12 cannot darken the normal grid,
	// so the corner actor cell returns to ordinary lighting.
	mw.scene = 10
	if _, ok := spellLightCellMap(mw.spellLighting())[image.Pt(23, 17)]; ok {
		t.Error("the wall's level-12 half left a brightness override on its isolated corner")
	}

	for i := 0; i < 300 && len(w.CellEffects()) > 0; i++ {
		sim.Step(w, nil)
	}
	mw.scene = 320
	if got := mw.spellLighting(); len(got) != 0 {
		t.Fatalf("expired walls left %d actor-light cells: %+v", len(got), got)
	}
}
