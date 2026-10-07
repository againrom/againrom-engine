package mapload

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func TestSourceEquipment1110BindsSavedRowNotAppearanceOrOwnKind(t *testing.T) {
	rows := make(ghostResistanceCollection, 38)
	rows[1].params = []int32{0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 3, 4, 1, 0}
	rows[37].params = []int32{0, 0, 0, 0, 0, 12, 0, 0, 0, 0, 0, 0, -1, 11, 2, 3}
	item := sim.ItemInstance{Code: 0x0101, SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 37, OwnKind: 3}}
	bound := BindSourceItemDefinition(item, &Table{Weapons: rows})
	want := item
	want.SourceEquipment.Definition = sim.SourceWeaponDefinition{Present: true, AttackType: 12, Hands: 2, Charge: -1, Relax: 11, Suitable: 3}
	if !reflect.DeepEqual(bound, want) {
		t.Fatal("source definition selected appearance/own-kind", bound)
	}
	rows[37].params[5] = 1
	if bound.SourceEquipment.Definition.AttackType != 12 {
		t.Fatal("binding aliases table")
	}
	if got := BindSourceItemDefinition(item, nil); !reflect.DeepEqual(got, item) {
		t.Fatal("missing table invented binding")
	}
	rows[37].params = rows[37].params[:5]
	if got := BindSourceItemDefinition(item, &Table{Weapons: rows}); got.SourceEquipment.Definition.Present {
		t.Fatal("short row invented binding")
	}
}
