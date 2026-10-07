package sim

import (
	"slices"
	"testing"
)

func TestCurrentItemWeightsKeepNextConstructionAndAbsence(t *testing.T) {
	for _, present := range []bool{false, true} {
		const code = 0x0701
		w := bookWorld(t, Entity{ID: 1, HP: 50, MaxHP: 100}, PlainItem(code))
		source := SourceActor{Class: 2, Fighter: true, Stats: [14]uint16{10, 20, 10, 10, 18, 0, 0, 101, 50, 100, 100, 0, 0, 50}}
		load := ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true, Source: source}, Capacity: 101, Speed: 18, Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Capacity: 101}}
		if err := w.RestoreActorLoad(1, load); err != nil {
			t.Fatal(err)
		}
		w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
		constructor := SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7, Defence: [22]byte{13}}
		weight := ItemWeight{Code: code, Weight: 7}
		if present {
			weight.Constructor = constructor
		}
		if err := w.DeclareItemWeights([]ItemWeight{weight}); err != nil {
			t.Fatal(err)
		}
		p := w.CurrentPolicy()
		cold := *w
		cold.itemWeights = []ItemWeight{{Code: code, Weight: 7, Constructor: constructor}}
		cold.itemWeights[0].Constructor.Defence[0] = 99
		if err := cold.RestoreCurrentContinuation(&p, nil, w.Actions(), nil); err != nil || cold.Hash() != w.Hash() {
			t.Fatal("current table was replaced by LOAD constructors", present, err)
		}
		left, right := w.EquipSourceCarried(1, 0), cold.EquipSourceCarried(1, 0)
		if left != present || right != left || w.Hash() != cold.Hash() {
			t.Fatal("next generated-item construction changed", present, left, right)
		}
		if present && (cold.equipment[0][6].Weight != 7 || cold.equipment[0][6].SourceEquipment.Defence[0] != 13) {
			t.Fatal("next equipment used the replacement constructor")
		}
		(*p.ItemWeights)[0].Weight++
		if cold.itemWeights[0].Weight != 7 {
			t.Fatal("capture or restore retained caller storage")
		}
	}
}

func TestCurrentItemWeightPolicyIsAtomic(t *testing.T) {
	w := mustWorld(t, 7, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1}})
	if err := w.DeclareItemWeights([]ItemWeight{{Code: 3, Weight: 9}}); err != nil {
		t.Fatal(err)
	}
	for _, rows := range [][]ItemWeight{
		{{Code: 0}},
		{{Code: 3, Weight: 1}, {Code: 3, Weight: 1}},
		{{Code: 3, Constructor: SourceEquipment{Class: 255}}},
		{{Code: 3, Constructor: SourceEquipment{Class: SourceWeapon, EffectsUnsupported: true}}},
	} {
		p := w.CurrentPolicy()
		p.ItemWeights = &rows
		before := w.Hash()
		if err := w.RestoreCurrentContinuation(&p, nil, w.Actions(), nil); err == nil || w.Hash() != before {
			t.Fatal("malformed table was accepted or changed the World")
		}
	}
	p := w.CurrentPolicy()
	empty := []ItemWeight{}
	p.ItemWeights = &empty
	if err := w.RestoreCurrentContinuation(&p, nil, w.Actions(), nil); err != nil || !slices.Equal(w.ItemWeights(), empty) {
		t.Fatal("explicit absent table was repopulated", err)
	}
}
