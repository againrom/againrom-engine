package sim

import (
	"strings"
	"testing"
)

func TestPoison1164SignedCloudCrossesFallenAndSaves(t *testing.T) {
	target := spEnt(1, 10, 10)
	target.HP, target.MaxHP, target.Protection[1], target.Defence = 10, 100, 150, 40
	w := areaLifecycleWorld(t, target)
	landAreaForTest(t, w, 8, 10, 10, ^EntityID(0), 30)
	for range 15 {
		Step(w, nil)
	}
	Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 17}})
	if e := w.entities[0]; e.HP != -1 || e.Decay != DecayFallen || e.Defence != 20 || w.attached[0].Remaining != 127 || w.attached[0].Magnitude != -8 {
		t.Fatal("ordinary damage after first cloud pulse", e, w.attached)
	}
	cold := cold1164(t, w)
	for range 8 {
		Step(w, nil)
		Step(cold, nil)
	}
	for _, current := range []*World{w, cold} {
		e := current.entities[0]
		if e.HP != 2 || e.Decay != DecayNone || e.Dwell != 0 || e.Defence != 40 || current.attached[0].Remaining != 119 {
			t.Fatalf("signed -3 damage: HP%d decay%d dwell%d defence%d counter%d, want2/0/0/40/119", e.HP, e.Decay, e.Dwell, e.Defence, current.attached[0].Remaining)
		}
	}
	if w.Hash() != cold.Hash() {
		t.Fatal("fallen-cut continuation differs")
	}
	cold = cold1164(t, w)
	for _, current := range []*World{w, cold} {
		Step(current, []Command{{Kind: KindDamage, Entity: 1, X: 1}})
		for range 7 {
			Step(current, nil)
		}
		if e := current.entities[0]; e.HP != 4 || e.Defence != 40 || e.Decay != DecayNone {
			t.Fatal("next input/pulse restored defence more than once", e)
		}
	}
	if w.Hash() != cold.Hash() {
		t.Fatal("healed-cut continuation differs")
	}
	cold1164(t, w)
}

func TestAreas1164MixedCurrentOwnersRejectedAtomically(t *testing.T) {
	for _, armed := range []bool{false, true} {
		w := retainedWallCells1162(t, armed)
		w.effects = []cellEffect{{Key: 0x1010, Spell: 19, Remaining: 17, Mode: AreaModeCloud, Cells: []uint16{0x1010}}}
		if _, err := w.MarshalBinary(); err == nil || !strings.Contains(err.Error(), "duplicate current area layer owner") {
			t.Fatal("SAVE admitted mixed current owners", armed, err)
		}
		receiver := retainedWallCells1162(t, true)
		before := receiver.Hash()
		if err := receiver.UnmarshalBinary(w.encode()); err == nil || receiver.Hash() != before {
			t.Fatal("hostile form92 admission was not atomic", armed, err)
		}
	}
}
