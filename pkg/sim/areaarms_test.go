package sim

import "testing"

// TestAreaArmsKeepTheCellRecordSlots: each slot keeps its arm, once.
func TestAreaArmsKeepTheCellRecordSlots(t *testing.T) {
	if want := [6]uint16{3, 7, 8, 19, 12, 17}; areaLayerSpells != want {
		t.Fatalf("layer slots %v, want %v", areaLayerSpells, want)
	}
	taken := map[int8]uint16{}
	blocking := 0
	for _, a := range areaArms {
		if a.layer < noAreaLayer || a.layer >= int8(len(areaLayerSpells)) {
			t.Fatalf("arm %d names slot %d", a.arm, a.layer)
		}
		if a.layer != noAreaLayer {
			if other, twice := taken[a.layer]; twice {
				t.Fatalf("arms %d and %d share slot %d", other, a.arm, a.layer)
			}
			taken[a.layer] = a.arm
		}
		if a.blocks {
			blocking++
			if a.layer == noAreaLayer {
				t.Fatalf("arm %d blocks without a slot", a.arm)
			}
		}
		for _, c := range append(append([]uint16(nil), a.clears...), a.yieldsTo) {
			if c != 0 && !layerSpell(c) {
				t.Fatalf("arm %d clears or yields to arm %d, which has no layer", a.arm, c)
			}
		}
		if (a.stages != 0) != (a.stage != nil) || a.draws && a.stages == 0 {
			t.Fatalf("arm %d: staged program fields disagree: %+v", a.arm, a)
		}
	}
	if len(taken) != len(areaLayerSpells) || blocking != 1 {
		t.Fatalf("%d slots taken, %d blocking", len(taken), blocking)
	}
	if layer, arm := areaBlockingLayer(); layer != 3 || arm != 19 {
		t.Fatalf("blocking slot %d arm %d, want 3 and 19", layer, arm)
	}
}

// TestAreaArmConflictsAreTheLayerRules: the layer rules written out by hand.
func TestAreaArmConflictsAreTheLayerRules(t *testing.T) {
	want := map[[2]uint16]bool{
		{2, 7}: true, {2, 8}: true, {3, 7}: true, {3, 8}: true, {7, 3}: true,
		{12, 12}: true, {12, 17}: true, {17, 12}: true, {17, 17}: true,
		{3, 3}: true, {7, 7}: true, {8, 8}: true, {19, 19}: true,
	}
	for incoming := uint16(0); incoming < 40; incoming++ {
		for old := uint16(0); old < 40; old++ {
			if got := areaLayerConflict(incoming, old); got != want[[2]uint16{incoming, old}] {
				t.Errorf("areaLayerConflict(%d, %d) = %t", incoming, old, got)
			}
		}
		clears := incoming == 2 || layerSpell(incoming)
		if areaArmClearsLayers(incoming) != clears {
			t.Errorf("arm %d clears layers: want %t", incoming, clears)
		}
		under, ok := areaArmYieldsTo(incoming)
		if ok != (incoming == 8) || ok && under != 3 {
			t.Errorf("arm %d yields to %d (%t)", incoming, under, ok)
		}
	}
	for arm, n := range map[uint16]int{4: 2, 9: 6, 21: 32, 3: 0, 7: 0} {
		if ringStageCount(arm) != n || AreaArmStaged(arm) != (n != 0) || ringStageDraws(arm) != (arm == 21) {
			t.Errorf("arm %d: %d stages, staged %t, draws %t", arm, ringStageCount(arm), AreaArmStaged(arm), ringStageDraws(arm))
		}
	}
	for arm := uint16(0); arm < 40; arm++ {
		layer, ok := AreaArmLayer(arm)
		if ok != layerSpell(arm) || ok && areaLayerSpells[layer] != arm {
			t.Errorf("AreaArmLayer(%d) = %d, %t", arm, layer, ok)
		}
	}
}
