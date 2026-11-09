package mapload

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

func unplacedCreatureRow(name string, body, typeID, face int32) ghostResistanceEntry {
	params := make([]int32, 41)
	for i := range params {
		params[i] = -1
	}
	params[0], params[29], params[30] = body, typeID, face
	return ghostResistanceEntry{name: name, params: params}
}

// TestConstructActorBasisUnplacedCreatureRow pins DIV-4748 and its controls.
func TestConstructActorBasisUnplacedCreatureRow(t *testing.T) {
	table := &Table{Units: ghostResistanceCollection{
		{},
		unplacedCreatureRow(ghostRowName, 11, 61, 3),
		unplacedCreatureRow("F_Skeleton.2", 9, 92, 2),
		unplacedCreatureRow("F_Skeleton.4", 47, 92, 1),
		unplacedCreatureRow("Faceless", 8, 200, 2),
	}}
	for _, tc := range []struct {
		name      string
		typeID    int32
		placement *alm.Unit
		row       uint16
		face      uint8
		body      uint16
		unit      string
	}{
		{name: "no placement takes the face-1 row", typeID: 92, row: 3, face: 1, body: 47, unit: "F_Skeleton.4"},
		{name: "placed creature keeps its row", typeID: 92, placement: &alm.Unit{ClassID: 92, ClassSubID: 2}, row: 2, face: 2, body: 9, unit: "F_Skeleton.2"},
		{name: "Ghost keeps its row", typeID: 61, row: 1, face: 3, body: 11, unit: ghostRowName},
		{name: "no face-1 row keeps no row", typeID: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := sim.Entity{ID: 5, TypeID: tc.typeID, HP: 10, MaxHP: 10}
			got, name, err := ConstructActorBasis(e, PartyMember{}, tc.placement, table, 1, 1, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			b := got.SourceBinding
			if b.TokenRow != tc.row || b.Face != tc.face || got.ActorLoad.Source.Stats[0] != tc.body || name != tc.unit {
				t.Fatalf("row/face/body/name = %d/%d/%d/%q, want %d/%d/%d/%q", b.TokenRow, b.Face, got.ActorLoad.Source.Stats[0], name, tc.row, tc.face, tc.body, tc.unit)
			}
		})
	}
}
