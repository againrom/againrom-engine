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

// MAGIC-463: stage k on the (1 + 3k)th area update.
func TestFirstStagedAreaFirstStageWaitsForTheNextAreaPass(t *testing.T) {
	sacrifice := acidStreamStartRow()
	sacrifice.ID = 4
	for _, row := range []SpellRule{acidStreamStartRow(), sacrifice} {
		w, err := NewSpelledWorld(7, Bounds{Width: 40, Height: 40}, ModeCanonical, nil,
			[]Entity{blizzardCaster(20, 12)}, nil, []SpellRule{row})
		if err != nil {
			t.Fatal(err)
		}
		if !w.landArea(w.spells[0], 50, 1, true, 20, 12, 20, 20, nil) {
			t.Fatalf("spell %d: no area", row.ID)
		}
		if len(w.effects[0].Cells) != 0 {
			t.Fatalf("spell %d: insertion painted a stage", row.ID)
		}
		w.decayCellEffects(nil)
		if len(w.effects) != 1 || len(w.effects[0].Cells) == 0 {
			t.Fatalf("spell %d: first area update painted no stage", row.ID)
		}
		first := w.effects[0].Cells
		w.decayCellEffects(nil)
		w.decayCellEffects(nil)
		if len(w.effects) != 1 || len(w.effects[0].Cells) != len(first) {
			t.Fatalf("spell %d: stage changed before update 4", row.ID)
		}
		w.decayCellEffects(nil)
		live := ringStageCount(w.spellArm(row.ID)) > 2
		if live != (len(w.effects) == 1) || live && w.effects[0].Phase != 4 {
			t.Fatalf("spell %d: update 4: %+v", row.ID, w.effects)
		}
	}
	meteor := acidStreamStartRow()
	meteor.ID = 21
	if got := areaLife(meteor, 50); got != ringLife(21)+1 {
		t.Fatalf("Meteor Storm lives %d updates, want %d", got, ringLife(21)+1)
	}
}
