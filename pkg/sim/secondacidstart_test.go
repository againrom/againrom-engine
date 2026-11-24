package sim

import "testing"

func acidStreamStartRow() SpellRule {
	return SpellRule{ID: 9, School: 2, ManaCost: 100, MaxRange: 10, Area: true,
		Distribution: distributionStaged, Radius: 4, AreaDuration: 8, DamageMin: 10, DamageMax: 30, Damaging: true}
}

// The second game's Acid Stream shares Blizzard's stage body: insertion
// paints nothing and the next area update paints stage zero, then every third.
func TestSecondAcidStreamFirstStageWaitsForTheNextAreaPass(t *testing.T) {
	w := secondWorld(t, 7, Bounds{Width: 40, Height: 40}, []SpellRule{secondRow(acidStreamStartRow())}, blizzardCaster(20, 12))
	if !w.landArea(w.spells[0], 50, 1, true, 20, 12, 20, 20, nil) {
		t.Fatal("no area")
	}
	if len(w.effects[0].Cells) != 0 {
		t.Fatal("insertion painted a stage")
	}
	w.decayCellEffects(nil)
	if len(w.effects) != 1 || len(w.effects[0].Cells) == 0 {
		t.Fatal("first area update painted no stage")
	}
	first := w.effects[0].Cells
	w.decayCellEffects(nil)
	w.decayCellEffects(nil)
	if len(w.effects) != 1 || len(w.effects[0].Cells) != len(first) {
		t.Fatal("stage changed before update 4")
	}
	w.decayCellEffects(nil)
	if len(w.effects) != 1 || w.effects[0].Phase != 4 {
		t.Fatalf("update 4: %+v", w.effects)
	}
}

// The first game's Acid Stream paints stage zero at insertion.
func TestFirstAcidStreamPaintsStageZeroAtInsertion(t *testing.T) {
	w, err := NewSpelledWorld(7, Bounds{Width: 40, Height: 40}, ModeCanonical, nil,
		[]Entity{blizzardCaster(20, 12)}, nil, []SpellRule{acidStreamStartRow()})
	if err != nil {
		t.Fatal(err)
	}
	if !w.landArea(w.spells[0], 50, 1, true, 20, 12, 20, 20, nil) || len(w.effects[0].Cells) == 0 {
		t.Fatal("no stage at insertion")
	}
}
