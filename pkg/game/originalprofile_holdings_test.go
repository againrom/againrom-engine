package game

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func profileHoldings1107() *holdingFixture {
	h := &holdingFixture{
		weapon: &holdingFixtureItem{class: "Weapon", code: 0x0101, count: 1, kind: 2, price: -201},
		items: []*holdingFixtureItem{{class: "Item", code: 0x0e06, count: 3, kind: 3, price: -203,
			effects: []sim.ItemEffect{{Kind: 12, Operand: 3}, {Kind: 12, Operand: 2}, {Kind: 12, Operand: 3}}}},
	}
	h.worn[6] = &holdingFixtureItem{class: "Armor", code: 0x0701, count: 1, kind: 1, price: -202}
	return h
}

// The signed-health cases can move the imported items from the actor to a
// native corpse sack. Count exact canonical values across those two owners,
// without claiming that the original post-pool callback performs that move.
func assertProfileHoldings1107(t *testing.T, w *sim.World, id sim.EntityID) {
	t.Helper()
	want := []sim.ItemStack{
		sim.StackItem(sim.ItemInstance{Code: 0x0101, Kind: 2, Price: -201, WeightPresent: true, SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 1}}, 1),
		sim.StackItem(sim.ItemInstance{Code: 0x0701, Kind: 1, Price: -202, WeightPresent: true, SourceEquipment: sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 1, OwnKind: 7}}, 1),
		sim.StackItem(sim.ItemInstance{Code: 0x0e06, Kind: 3, Price: -203, WeightPresent: true,
			Effects: []sim.ItemEffect{{Kind: 12, Operand: 3}, {Kind: 12, Operand: 2}, {Kind: 12, Operand: 3}}}, 3),
	}
	counts := make([]uint32, len(want))
	add := func(item sim.ItemInstance, count uint32) {
		for i, expected := range want {
			if item.Code == expected.Code {
				if !reflect.DeepEqual(item, expected.Instance()) {
					t.Fatalf("imported item changed: %+v want %+v", item, expected.Instance())
				}
				counts[i] += count
			}
		}
	}
	carried, _ := w.CarriedStacks(id)
	for _, stack := range carried {
		add(stack.Instance(), stack.Count)
	}
	worn, _ := w.EquippedItems(id)
	for _, item := range worn {
		add(item, 1)
	}
	for _, sack := range w.Sacks() {
		for _, item := range sack.ItemInstances {
			add(item, 1)
		}
	}
	for i, expected := range want {
		if counts[i] != expected.Count {
			t.Fatalf("imported item %#x count %d want %d", expected.Code, counts[i], expected.Count)
		}
	}
}

func TestOriginalProfile1107HoldingsRearmBooksAndOverlayCompose(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 1, MapUnitID: 91, X: 5, Y: 6, HP: 20, MaxHP: 20,
			SecondBase: 13, SecondSpread: 7, KnownSpells: 1 << 6}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 1, Items: []uint16{0x0e07}}})
	if err != nil {
		t.Fatal(err)
	}
	tails := []sim.CellTail{{X: 8, Y: 9, Bytes: [6]byte{13, 1, 19, 61, 21, 63}}}
	if err := w.ImportOriginalCellTails(tails); err != nil {
		t.Fatal(err)
	}
	member := mapload.PartyMember{Hero: data.Hero{Body: 30, Reaction: 20, Mind: 10, Spirit: 5}}
	ms := &Mission{Map: &alm.Map{Width: 40, Height: 40}, World: w,
		Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{1: member}}}
	holdings := []sav.ActorHoldings{{MapUnitID: 91, Cell: 0x0605, HP: 7,
		Items: []sav.Piece{{Class: "Item", Code: 0x0e06, Stack: 3, Kind: 3, Price: -203}}}}
	if err := restoreOriginalActorStock(ms, holdings, nil, &OriginalSaveResume{}); err != nil {
		t.Fatal(err)
	}
	if e := poolEntity(t, w, 91); e.MaxHP == 20 || e.MaxHP == 31 || e.ToHit == 2000 {
		t.Fatal("fixture did not exercise a different native Rearm sheet")
	}
	pools := []sav.ActorPools{{MapUnitID: 91, Cell: 0x0605, HP: 7, MaxHP: 31, Mana: 5, MaxMana: 23}}
	books := []sav.ActorSpellbook{{MapUnitID: 91, RuntimeID: 1, Cell: 0x0605, HP: 7, HasSpellbook: true,
		SpellCount: 2, Spells: []sav.SavedSpell{{Slot: 1, ID: 1, Range: 7, ManaCost: 3}}}}
	profiles := []sav.ActorCurrent{{ActorPools: pools[0], Reaction: -3, Mind: -32767, Spirit: 32766,
		ToHit: 2000, DamageBase: 40, Active: 1, SecondBase: 20, ElementalBase: 8, ElementalKind: 1,
		Defence: -2, Absorption: 7, Protection: [5]int16{10, 20, 30, 40, -5}, Resistance: [5]uint8{1, 2, 3, 128, 255},
		HealthPeriod: 100, ManaPeriod: 100, HealthRegeneration: 50, ManaRegeneration: 50, HealthHundredths: 77, ManaHundredths: 51}}
	var report OriginalSaveResume
	if err := restoreOriginalActors(ms, holdings, pools, books, nil, &report, profiles); err != nil {
		t.Fatal(err)
	}
	e := poolEntity(t, w, 91)
	assertCurrent1107(t, e)
	assertPools(t, e, [4]int32{7, 31, 5, 23})
	pack, _ := w.CarriedStacks(e.ID)
	if e.Book.State != sim.BookPresent || e.Book.Slots[0] != (sim.BookSpell{Range: 7, Defensive: 0, ManaCost: 3}) || e.KnownSpells != 1<<1 ||
		len(pack) != 1 || !sim.StackStateEqual(pack[0], sim.StackItem(sim.ItemInstance{Code: 0x0e06, Kind: 3, Price: -203, WeightPresent: true}, 3)) ||
		!reflect.DeepEqual(w.CellTails(), tails) || w.Tick() != 0 || report.Stocked != 1 || report.Books.Restored != 1 || report.ProfilesRestored != 1 || report.PoolsRestored != 1 {
		t.Fatalf("composed handoff changed stock/book/overlay/tick: %+v %+v %+v", e, pack, report)
	}
}
