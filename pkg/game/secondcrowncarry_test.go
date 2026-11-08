package game

import (
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The crown parts and the assembled Gold Crown of missions 70 and 80.
const (
	secondCrownOne      uint16 = 0x0e4f
	secondCrownTwelve   uint16 = 0x0e50
	secondCrownThirteen uint16 = 0x0e51
	secondGoldCrown     uint16 = 0x46c6
)

// secondCrownUnit is the Gold Crown as mission 70's storage unit holds it:
// its map's type-9 record gives it five effects.
func secondCrownUnit() sim.ItemInstance {
	return sim.ItemInstance{Code: secondGoldCrown, Kind: 1, Price: 9999999, Effects: []sim.ItemEffect{
		{Kind: 25, Operand: 70}, {Kind: 23, Operand: 70}, {Kind: 2, Operand: 10}, {Kind: 16, Operand: 10}, {Kind: 1, Operand: 1000000},
	}}
}

// secondCrownWhere is where f's party member holds the Gold Crown, with the
// unit: "bag" for the hero's bag, "worn", "shown" for a pack unit the shop's
// strip shows, and "pack" for one no town page shows. It fails unless he
// holds exactly one.
func secondCrownWhere(t *testing.T, stage string, f *FrontEnd, member int) (string, sim.ItemInstance) {
	t.Helper()
	worn := mapload.MemberItemEquipment(f.Carried[member], f.Table)
	var bag []sim.ItemInstance
	if member == secondHeroIndex(f.Carried) {
		bag = *f.Town.second.heroBag()
	}
	where, unit, n := "", sim.ItemInstance{}, 0
	for _, place := range []struct {
		name  string
		items []sim.ItemInstance
	}{{"bag", bag}, {"pack", mapload.MemberCarriedItems(f.Carried[member], f.Table)}, {"worn", worn[:]}} {
		for _, it := range place.items {
			if it.Code == secondGoldCrown {
				where, unit, n = place.name, it, n+1
			}
		}
	}
	if n != 1 {
		t.Fatalf("%s: member %d holds %d Gold Crowns", stage, member, n)
	}
	if where == "pack" && slices.ContainsFunc(f.memberUnits(member, f.Table), func(it sim.ItemInstance) bool { return it.Code == secondGoldCrown }) {
		where = "shown"
	}
	return where, unit
}

// secondCrownPour runs, on a copy of the mission's world, what the second
// of missions 70 and 80 does: a storage unit holding the crown leaves the
// map (instant 16), pours it to the party member who completes the quest
// (28), and every unit on the map loses crown1 and crown13 (38). It returns
// the world after the pour.
func secondCrownPour(t *testing.T, mw *mapWorld, member int) *sim.World {
	t.Helper()
	w, hero := mw.world, mw.mission.ids[member]
	e, _ := w.Entity(hero)
	storage, err := w.CheatSummon(sim.Entity{HP: 20, MaxHP: 20, Reach: 1, TokenSize: 1, Owner: e.Owner}, []sim.ItemStack{sim.StackItem(secondCrownUnit(), 1)}, [sim.EquipSlots]sim.ItemInstance{}, e.X+2, e.Y)
	if err != nil {
		t.Fatal(err)
	}
	instants := []sim.ScriptInstant{
		{Op: sim.ScriptInstantTakeOffMap, Unit: storage, HasUnit: true},
		{Op: sim.ScriptInstantGiveAll, Unit: storage, HasUnit: true, Unit2: hero, HasUnit2: true},
		{Op: sim.ScriptInstantTakeItemFromMap, Item: secondCrownOne, HasItem: true},
		{Op: sim.ScriptInstantTakeItemFromMap, Item: secondCrownThirteen, HasItem: true},
	}
	program, err := sim.NewROM2Script(nil, instants, []sim.ScriptTrigger{{Instants: [4]int32{0, 1, 2, 3}, Once: true}})
	if err != nil {
		t.Fatal(err)
	}
	poured, err := sim.NewControlledScriptWorld(w, program)
	if err != nil {
		t.Fatal(err)
	}
	for range 40 {
		sim.Step(poured, nil)
	}
	pack, _ := poured.CarriedItems(hero)
	if i := slices.IndexFunc(pack, func(it sim.ItemInstance) bool { return it.Code == secondGoldCrown }); i < 0 || !slices.Equal(pack[i].Effects, secondCrownUnit().Effects) {
		t.Fatalf("after the pour the hero's pack is %v", bagLabels(pack))
	}
	if slices.ContainsFunc(pack, func(it sim.ItemInstance) bool { return it.Code == secondCrownOne || it.Code == secondCrownThirteen }) {
		t.Fatalf("instant 38 left a part in the hero's pack %v", bagLabels(pack))
	}
	if left, _ := poured.CarriedItems(storage); len(left) != 0 {
		t.Fatalf("the storage unit keeps %v", bagLabels(left))
	}
	return poured
}

// The Gold Crown a script pours into the hero's pack comes home: the
// victory carries it, the arrival at town 2 puts it in the bag with its five
// effects, where the shop shows it, and a town SAVE and its cold LOAD keep
// it there. Put on before the victory it stays worn. Poured to a companion
// it stays in his pack, where the shop's strip shows it. Fails with a unit
// of more than three effects having no bag form, so that no town page shows
// it.
func TestSecondGoldCrownComesHomeFromTheScript(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		member     int
	}{{"pack", "bag", 0}, {"worn", "worn", 0}, {"companion", "shown", 1}} {
		t.Run(tc.name, func(t *testing.T) {
			f, app, screen, _ := secondBagFront(t, nil)
			parts := []sim.ItemInstance{{Code: secondCrownOne, Price: -1}, {Code: secondCrownThirteen, Price: -1}}
			hero, ok := memberWithPack(f.Carried[0], parts, f.Table, true)
			if !ok {
				t.Fatal("the hero's pack refused the parts")
			}
			f.Carried[0] = hero
			secondBagEnter(t, f, app, screen)
			w := secondCrownPour(t, f.live, tc.member)
			if tc.name == "worn" {
				id := f.live.mission.ids[0]
				stacks, _ := w.CarriedStacks(id)
				i := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == secondGoldCrown })
				sim.Step(w, []sim.Command{sim.Equip(id, sim.ItemSlot(i), sim.EquipSlot(secondGoldCrown>>8&0xf))})
				for range 8 {
					sim.Step(w, nil)
				}
				if eq, _ := w.EquippedItems(id); !slices.ContainsFunc(eq[:], func(it sim.ItemInstance) bool { return it.Code == secondGoldCrown }) {
					t.Fatal("the hero did not put the crown on")
				}
			}
			secondBagWinOver(t, f, 10, w)
			if where, _ := secondCrownWhere(t, "after the victory", f, tc.member); where == "bag" {
				t.Fatal("the victory put the crown in the bag on the map")
			}
			c := f.Town.second
			if !c.has(partyPlace{kind: placeTown, id: 2}) {
				c.available = append(c.available, partyPlace{kind: placeTown, id: 2})
			}
			if at := screen.enterLocation(partyPlace{kind: placeTown, id: 2}); at.Msg != "" {
				t.Fatalf("the arrival at town 2 reports %q", at.Msg)
			}
			where, unit := secondCrownWhere(t, "at town 2", f, tc.member)
			if where != tc.want || !slices.Equal(unit.Effects, secondCrownUnit().Effects) || unit.Price != secondCrownUnit().Price {
				t.Fatalf("at town 2 the crown is in the %s as %v, want the %s as %v", where, bagLabels([]sim.ItemInstance{unit}), tc.want, bagLabels([]sim.ItemInstance{secondCrownUnit()}))
			}
			s, label, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := f.ExportCurrentSave(s, label)
			if err != nil {
				t.Fatal(err)
			}
			cold, _, _, _ := secondBagFront(t, nil)
			if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
				t.Fatal("cold town LOAD", town, err)
			}
			if got, unit := secondCrownWhere(t, "after the town SAV's cold LOAD", cold, tc.member); got != where || !slices.Equal(unit.Effects, secondCrownUnit().Effects) {
				t.Fatalf("the cold LOAD holds the crown in the %s as %v, the town in the %s", got, bagLabels([]sim.ItemInstance{unit}), where)
			}
		})
	}
}
