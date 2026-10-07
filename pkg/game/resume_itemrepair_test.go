package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type resumeRepairRow struct {
	params []int32
}

type resumeRepairCollection []resumeRepairRow

func (c resumeRepairCollection) Len() int                  { return len(c) }
func (resumeRepairCollection) EntryName(int) string        { return "" }
func (c resumeRepairCollection) EntryParams(i int) []int32 { return c[i].params }
func (resumeRepairCollection) EntryStrings(int) []string   { return nil }

type resumeRepairScale struct {
	rows   int
	index  int
	factor float64
}

func (s resumeRepairScale) Len() int           { return s.rows }
func (resumeRepairScale) EntryName(int) string { return "" }
func (s resumeRepairScale) EntryDoubles(i int) []float64 {
	row := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}
	if i == s.index {
		row[2] = s.factor
	}
	return row
}

func resumeRepairTable() *mapload.Table {
	weapons := make(resumeRepairCollection, 21)
	weapons[20].params = []int32{0, 0, 207}
	magic := make(resumeRepairCollection, 22)
	magic[12].params = []int32{10}
	magic[21].params = []int32{10}
	return &mapload.Table{
		Shapes:    resumeRepairScale{rows: 4, index: 3, factor: 1},
		Materials: resumeRepairScale{rows: 9, index: 8, factor: 267.0 / 207.0},
		Weapons:   weapons,
		Magic:     magic,
	}
}

func TestResumeRepairsLegacyLinkedItemsInEveryWorldContainer(t *testing.T) {
	legacyAttack := sim.ItemInstance{Code: 0x8134, Kind: 2, Price: 207, Effects: []sim.ItemEffect{
		{Kind: 12, Operand: 0x0000fb05},
	}}
	legacyFire := sim.ItemInstance{Code: 0x8134, Kind: 2, Price: 207, Effects: []sim.ItemEffect{
		{Kind: 21, Operand: 0x0000fb05},
	}}
	var equipped [sim.EquipSlots]sim.ItemInstance
	equipped[0] = legacyFire
	saved, err := sim.NewStockedWorld(17, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{{ID: 9, X: 2, Y: 2}}, nil, sim.Relations{},
		[]sim.Sack{{X: 5, Y: 4, ItemInstances: []sim.ItemInstance{legacyAttack}}},
		[]sim.Stock{{ID: 9, ItemInstances: []sim.ItemInstance{legacyAttack}, EquippedItems: equipped}})
	if err != nil {
		t.Fatalf("NewStockedWorld(saved): %v", err)
	}
	raw, err := saved.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary(saved): %v", err)
	}
	fresh, err := sim.NewWorld(17, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 9, X: 2, Y: 2}})
	if err != nil {
		t.Fatalf("NewWorld(fresh): %v", err)
	}
	mission := &Mission{World: fresh}
	if err := resumeWorld(mission, &Snapshot{World: raw}, resumeRepairTable()); err != nil {
		t.Fatalf("resumeWorld: %v", err)
	}
	if defects := sim.InspectLegacyLinkedItems(mission.World); len(defects) != 0 {
		t.Fatalf("resumed world still has legacy effects: %+v", defects)
	}
	assertItem := func(where string, item sim.ItemInstance, kind uint8) {
		t.Helper()
		if item.Price != 6106 || len(item.Effects) != 1 ||
			item.Effects[0] != (sim.ItemEffect{Kind: kind, Operand: 5}) {
			t.Errorf("%s item = %+v, want value 6106 and repaired kind-%d +5 effect", where, item, kind)
		}
	}
	carried, _ := mission.World.CarriedItems(9)
	if len(carried) != 1 {
		t.Fatalf("carried items = %+v", carried)
	}
	assertItem("carried", carried[0], 12)
	worn, _ := mission.World.EquippedItems(9)
	assertItem("equipped", worn[0], 21)
	sacks := mission.World.Sacks()
	if len(sacks) != 1 || len(sacks[0].ItemInstances) != 1 {
		t.Fatalf("sacks = %+v", sacks)
	}
	assertItem("sack", sacks[0].ItemInstances[0], 12)

	repairedForm, err := mission.World.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary(repaired): %v", err)
	}
	var roundTrip sim.World
	if err := roundTrip.UnmarshalBinary(repairedForm); err != nil {
		t.Fatalf("UnmarshalBinary(repaired): %v", err)
	}
	if defects := sim.InspectLegacyLinkedItems(&roundTrip); len(defects) != 0 {
		t.Fatalf("repaired round trip restored defects: %+v", defects)
	}
	roundCarried, _ := roundTrip.CarriedItems(9)
	assertItem("round-trip carried", roundCarried[0], 12)
}

func TestPrepareTownRestoreRepairsPartyItemsBeforeCommit(t *testing.T) {
	legacy := sim.ItemInstance{Code: 0x8134, Kind: 2, Price: 267,
		Effects: []sim.ItemEffect{{Kind: 12, Operand: 0x0000fb05}}}
	f := &FrontEnd{InstallResources: InstallResources{Table: resumeRepairTable()}}
	candidate, err := f.prepareRestore(Snapshot{Party: []mapload.PartyMember{{
		ID: "hero", Carry: &mapload.Carry{ItemInstances: []sim.ItemInstance{legacy}},
	}}})
	if err != nil {
		t.Fatalf("prepareRestore: %v", err)
	}
	items := mapload.MemberCarriedItems(candidate.carried[0], f.Table)
	if len(items) != 1 || items[0].Price != 6106 || items[0].Effects[0].Operand != 5 {
		t.Fatalf("prepared town item = %+v, want repaired +5 and value 6106", items)
	}
}
