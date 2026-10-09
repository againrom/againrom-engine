package game

import (
	"image"
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
		}}, sim.Bounds{Width: 32, Height: 32})
		switch id {
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

// TestLiveWallOfFireStampsTheLightGridAndRestoresOnUpdate: each retained wall
// cell is a radius-1 point stamp at its flicker level 0 or 12
// (MAGIC-UNITLIGHT-057, MAGIC-271), not a cell-bit plane entry.
func TestLiveWallOfFireStampsTheLightGridAndRestoresOnUpdate(t *testing.T) {
	w := liveWallLightingWorld(t)
	mw := &mapWorld{world: w, scene: 0}
	if got := mw.spellLighting(); len(got) != 0 {
		t.Fatalf("Wall of Fire wrote %d cell-bit plane cells, want none", len(got))
	}
	stamps := mw.objectLightStamps(nil)
	cells := 0
	for _, e := range w.CellEffects() {
		cells += len(e.Cells)
	}
	if len(stamps) != 12*cells {
		t.Fatalf("%d wall cells wrote %d stamps, want 12 each", cells, len(stamps))
	}
	grid := map[image.Point]ui.LightStamp{}
	for _, s := range stamps {
		if !s.Point {
			t.Fatalf("wall stamp %+v is not a point-helper stamp", s)
		}
		grid[s.Vertex] = s
	}
	// Source (24,18) at scene 0: (0/2 + 24*18)/5 = 86, even, level 0. Its
	// radius-1 footprint holds vertex (23,18) and (24,17).
	for _, v := range []image.Point{{24, 18}, {25, 19}, {23, 18}, {24, 17}} {
		if _, ok := grid[v]; !ok {
			t.Errorf("vertex %v of source (24,18) is unstamped", v)
		}
	}
	if got := wallFireLightLevel(0, image.Pt(24, 18)); got != 0 {
		t.Errorf("scene 0 source (24,18) level %d, want 0", got)
	}
	if got := wallFireLightLevel(10, image.Pt(24, 18)); got != 12 {
		t.Errorf("scene 10 source (24,18) level %d, want 12", got)
	}

	sim.Step(w, []sim.Command{{Kind: sim.KindCastAt, Entity: 1, X: 24, Y: 20, Spell: 3}})
	for i := 0; i < 32 && len(w.CellEffects()) < 2; i++ {
		sim.Step(w, nil)
	}
	stacked := map[image.Point]uint8{}
	for _, s := range mw.objectLightStamps(nil) {
		stacked[s.Vertex] = s.Level
	}
	for v, s := range grid {
		if stacked[v] != s.Level {
			t.Fatalf("a stacked identical wall changed vertex %v from %d to %d", v, s.Level, stacked[v])
		}
	}

	for i := 0; i < 300 && len(w.CellEffects()) > 0; i++ {
		sim.Step(w, nil)
	}
	if got := mw.objectLightStamps(nil); len(got) != 0 {
		t.Fatalf("expired walls left %d stamps", len(got))
	}
}
