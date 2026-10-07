package sav

import (
	"encoding/json"
	"testing"
)

func TestPendingCommandRemapKeepsOperandDomains(t *testing.T) {
	raw := json.RawMessage(`{"Commands":[{"Command":{"Kind":4,"Entity":0,"X":77,"Y":12,"Group":9,"Spell":5},"Issuer":{"ID":0,"Object":1,"Missing":false},"Target":{"ID":77,"Structure":true,"Object":2,"Missing":false}}],"Commanded":[{"ID":0,"Object":1,"Missing":false}]}`)
	for _, permutation := range [][]uint16{{0, 2, 1}, {0, 0, 1}, {0, 1, 2}} {
		result, err := remapPendingCommandObjects(raw, permutation)
		if err != nil {
			t.Fatal(err)
		}
		var q struct {
			Commands []struct {
				Command        struct{ Kind, Entity, X, Y, Group, Spell uint32 }
				Issuer, Target struct {
					ID, Object         uint32
					Missing, Structure bool
				}
			}
			Commanded []struct {
				ID, Object uint32
				Missing    bool
			}
		}
		if err := json.Unmarshal(result, &q); err != nil {
			t.Fatal(err)
		}
		p := q.Commands[0]
		if p.Issuer.ID != 0 || p.Issuer.Object != uint32(permutation[1]) || p.Issuer.Missing != (permutation[1] == 0) || p.Target.ID != 77 || !p.Target.Structure || p.Target.Object != uint32(permutation[2]) || p.Command.X != 77 || p.Command.Y != 12 || p.Command.Group != 9 || p.Command.Spell != 5 || q.Commanded[0].Object != uint32(permutation[1]) {
			t.Fatalf("endpoint relocation changed operand domains: %+v", q)
		}
	}
	if _, err := remapPendingCommandObjects(raw, []uint16{0, 1}); err == nil {
		t.Fatal("out-of-range endpoint was accepted")
	}
}
