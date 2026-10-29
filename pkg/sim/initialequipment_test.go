package sim

import "testing"

func TestInitialEquipmentDefinitionCannotReplaceALiveOrImportedActor(t *testing.T) {
	for _, imported := range []bool{false, true} {
		w := rcWorld(t, rcEnt(1, 1, 1, 1, 2, 3, 4, 5, 6, 7, false, 1))
		if imported {
			w.entities[0].ActorLoad.Present = true
		} else {
			w.tick = 1
		}
		before := w.Hash()
		definition := ActorDefinition{HP: 99, MaxHP: 99}
		if w.SetDerived(1, DerivedBlock{InitialPlacement: &definition}) || before != w.Hash() {
			t.Fatal("initial equipment definition replaced existing state", imported)
		}
	}
}
