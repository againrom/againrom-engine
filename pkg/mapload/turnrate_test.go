package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestPlacedHumanTurnRateUsesDerivedSpeed(t *testing.T) {
	table := &mapload.Table{Humans: defCollection{{}, {name: "person", params: defRow(map[int]int32{
		0: 20, 1: 20, 2: 20, 3: 20, 4: 30, 6: 10, 7: 99, 16: 7, 17: 1,
	})}}}
	m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{{X: 20 << 8, Y: 20 << 8, ClassID: 7}}}
	w, err := mapload.FromALMWith(m, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := w.Entity(0)
	if !ok || !e.Humanoid || e.RotationSpeed != 16 {
		t.Fatalf("placed Human turn rate = %d, humanoid = %v, present = %v; want 16", e.RotationSpeed, e.Humanoid, ok)
	}
}

func TestNativePartyTurnRateUsesDerivedSpeed(t *testing.T) {
	for _, hired := range []bool{false, true} {
		p := mapload.PartyMember{Hero: data.Hero{Body: 20, Reaction: 45, Mind: 20, Spirit: 20}, HiredRotationSpeed: 99}
		if hired {
			p.MercenaryType = 3
		}
		for _, spawn := range []func(mapload.PartyMember) (data.Derived, int32, int32){
			mapload.PartySpawn,
			func(p mapload.PartyMember) (data.Derived, int32, int32) { return mapload.PartySpawnWithTable(p, nil) },
		} {
			d, _, _ := spawn(p)
			if d.Speed != 21 || d.RotationSpeed != 21 {
				t.Errorf("hired = %v: speed/turn rate = %d/%d, want 21/21", hired, d.Speed, d.RotationSpeed)
			}
		}
	}
}

func TestPlacedCreatureKeepsDefinitionTurnRate(t *testing.T) {
	table := &mapload.Table{Units: defCollection{{}, {name: "creature", params: defRow(map[int]int32{
		8: 10, 9: 21, slotUnitType: 64, slotUnitFace: 1,
	})}}}
	m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{{X: 20 << 8, Y: 20 << 8, ClassID: 64, ClassSubID: 1}}}
	w, err := mapload.FromALMWith(m, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := w.Entity(sim.EntityID(0))
	if !ok || e.Humanoid || e.RotationSpeed != 21 {
		t.Fatalf("placed creature turn rate = %d, humanoid = %v, present = %v; want 21", e.RotationSpeed, e.Humanoid, ok)
	}
}
