package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A second-game placement whose Humans row is above a byte, as the knights
// of missions 73 and 86 are (R2-SESSION-297), constructs with that row: the
// binding holds the whole row, which SAVE writes (R2-ENGINE-739), and the
// LOAD relink of the bound row reads the same Humans row (R2-ENGINE-740).
func TestAPlacementAboveRow255BindsItsRow(t *testing.T) {
	humans := make(defCollection, 258)
	humans[256] = defEntry{name: "Knight", params: withServerID(composedHumanRow(9, 52, 0), 7301)}
	humans[257] = defEntry{name: "Knight2", params: withServerID(composedHumanRow(9, 52, 0), 7302)}
	table := &mapload.Table{UnitKeys: mapload.ServerUnitKeys, Units: defCollection{{}}, Humans: humans,
		Shapes: identityScale(), Materials: identityScale(), Weapons: swordAndBow()}
	unit := alm.Unit{X: 0x0c80, Y: 0x0c80, ClassID: 9, ClassSubID: 1, ServerID: 7301, Flags: 0x10}
	m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{unit}}
	world, roster, err := mapload.FromALMRoster(m, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	person := roster[sim.EntityID(0)]
	entity, _ := world.Entity(0)
	human := data.HumanState{TypeID: uint16(entity.TypeID), Fighter: true}
	bound, name, err := mapload.ConstructActorBasis(entity, person, &unit, table, 1, 1, &human, 0)
	if err != nil {
		t.Fatalf("constructor basis: %v", err)
	}
	if row := int(bound.SourceBinding.TokenRow); row != 256 || int(bound.SourceBinding.DefinitionRow()) != 256 || name != "Knight" {
		t.Fatalf("bound row %d definition row %d name %q, want 256 Knight", row, bound.SourceBinding.DefinitionRow(), name)
	}
	loaded, err := mapload.SourceActorPerson(bound, person.Name, table)
	if err != nil || loaded.Name != "Knight" {
		t.Fatalf("LOAD relink %q: %v", loaded.Name, err)
	}
}
