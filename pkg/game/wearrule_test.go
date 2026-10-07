package game

import (
	"os"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/databin"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// 0162-wear-rule: the two sites this build enforces the wear rule at, and the
// one it deliberately does not.
//
// THE FIXTURE'S ROWS STATE THEIR OWN sutableFor CELL, which is what the tests
// here turn on: gaArmorRowSuit writes an Armors row at a named suitability, so a
// refusal below is the ROW's answer and not the absence of a column.

const (
	// wrFighterOnly, wrMageOnly and wrEither are the three sutableFor values a
	// shipped row carries beside 0. Bit 0 is the fighter and bit 1 the mage.
	wrFighterOnly = int32(1)
	wrMageOnly    = int32(2)
	wrEither      = int32(3)
)

// wrTable is one Armors collection of three rows, one per suitability, all at
// slot 12 so the code's field B and the row's Slot column agree.
func wrTable(t *testing.T) *mapload.Table {
	t.Helper()
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinShapes:    {gaIdentityRow("Plain")},
			synth.DataBinMaterials: {gaIdentityRow("Steel")},
			synth.DataBinWeapons:   {eqWeaponRow("Sword", data.SkillBlade, 10, 20, 5, 3, 2, 7, 4)},
			synth.DataBinArmors: {
				gaArmorRowSuit("Plate Boots", 12, 7, 3, wrFighterOnly),
				gaArmorRowSuit("Shoes", 12, 1, 0, wrMageOnly),
				gaArmorRowSuit("Ring", 12, 2, 1, wrEither),
			},
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("databin.Parse: %v", err)
	}
	return &mapload.Table{
		Shapes: f.Collection(databin.Shapes), Materials: f.Collection(databin.Materials),
		Weapons: f.Collection(databin.Weapons), Armors: f.Collection(databin.Armors),
	}
}

// The three codes wrTable's three Armors rows answer to: material 0, field B
// 12 (the row's own slot), shape 0, row 1, 2 and 3.
const (
	wrFighterCode = uint16(0)<<12 | uint16(12)<<8 | uint16(0)<<5 | uint16(1)
	wrMageCode    = uint16(0)<<12 | uint16(12)<<8 | uint16(0)<<5 | uint16(2)
	wrEitherCode  = uint16(0)<<12 | uint16(12)<<8 | uint16(0)<<5 | uint16(3)
)

// wrMission is equipMission with the party member's own Mage flag stated —
// the character side of the rule, which equipMission leaves false.
func wrMission(t *testing.T, mage bool, table *mapload.Table, carried ...uint16) (*mapWorld, *sim.World) {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: carried}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{Hero: eqHero(), Mage: mage}},
		Start: mapload.Start{IDs: []sim.EntityID{7}}}
	return openMission(ms, table, nil, v, missionSource{}, nil, nil), w
}

// TestAMageIsRefusedAFighterOnlyItemAndNothingMoves is 0162 spec AC-3, SC-4.
// The refusal is a bare return: no command, and the container is what it was.
func TestAMageIsRefusedAFighterOnlyItemAndNothingMoves(t *testing.T) {
	table := wrTable(t)
	mw, w := wrMission(t, true, table, wrFighterCode)

	before, _ := w.Carried(7)
	mw.enqueueEquip(0)
	if len(mw.pending) != 0 {
		t.Fatalf("pending = %v, want no command — a mage may not equip a fighter-only item", mw.pending)
	}

	mw.tick()
	after, _ := w.Carried(7)
	if len(after) != len(before) || (len(after) > 0 && after[0] != before[0]) {
		t.Errorf("Carried(7) = %v after the refused equip, want %v unchanged", after, before)
	}
	if eq, _ := w.Equipped(7); eq != ([12]uint16{}) {
		t.Errorf("Equipped(7) = %v, want every slot empty — the refusal must move nothing", eq)
	}
}

// TestAFighterIsRefusedAMageOnlyItem is AC-3's mirror: the rule refuses in both
// directions, so a polarity swap cannot pass both this test and the one above.
func TestAFighterIsRefusedAMageOnlyItem(t *testing.T) {
	table := wrTable(t)
	mw, _ := wrMission(t, false, table, wrMageCode)

	mw.enqueueEquip(0)
	if len(mw.pending) != 0 {
		t.Fatalf("pending = %v, want no command — a fighter may not equip a mage-only item", mw.pending)
	}
}

// TestASuitableItemStillEquips is 0162 spec AC-4, SC-4: every permitted
// combination reaches the same command it reached before the rule existed.
func TestASuitableItemStillEquips(t *testing.T) {
	for _, c := range []struct {
		what string
		mage bool
		code uint16
	}{
		{"mage, mage-only item", true, wrMageCode},
		{"fighter, fighter-only item", false, wrFighterCode},
		{"mage, item usable by either", true, wrEitherCode},
		{"fighter, item usable by either", false, wrEitherCode},
	} {
		table := wrTable(t)
		mw, _ := wrMission(t, c.mage, table, c.code)

		mw.enqueueEquip(0)
		want := sim.Command{Kind: sim.KindEquip, Entity: 7, X: 0, Y: 12}
		if len(mw.pending) != 1 || mw.pending[0] != want {
			t.Errorf("%s: pending = %v, want exactly %+v", c.what, mw.pending, want)
		}
	}
}

func TestTheSimulationAppliesNoClassRule(t *testing.T) {
	_, w := wrMission(t, true, wrTable(t), wrFighterCode)

	sim.Step(w, []sim.Command{{Kind: sim.KindEquip, Entity: 7, X: 0, Y: 12}})

	eq, ok := w.Equipped(7)
	if !ok || eq[11] != wrFighterCode {
		t.Errorf("Equipped(7)[11] = 0x%x, want 0x%x — pkg/sim must equip what it is told",
			eq[11], wrFighterCode)
	}
}

// TestAnUnreadableRowDoesNotRefuseAnEquip is 0162 spec FR-2a, plan R-2. A nil
// table already refuses at EquipTarget, so the case that discriminates is a
// TABLE THAT RESOLVES THE SLOT while the wear rule reads no column: a magic
// item's class, which reaches none of the three collections.
func TestAnUnreadableRowDoesNotRefuseAnEquip(t *testing.T) {
	table := wrTable(t)
	mw, _ := wrMission(t, true, table, wrMageCode)

	magic := data.ComposeItemCode(0, data.ItemClassCarried, 0, 1)
	if !mw.wearAllows(magic) {
		t.Error("an item class the three collections do not carry was refused; " +
			"an unread column is not a refusal")
	}
	if !mw.wearAllows(data.ComposeItemCode(0, 12, 0, 30)) {
		t.Error("an armour row past the end of the collection was refused")
	}
}

// wrShopTable is shopTable with every row of the armour shelf — armours and
// shields alike — stated fighter-only, so whichever of the two the generator
// draws into a cell answers the same way and the test is not reading the
// generator's own draw order.
func wrShopTable() *mapload.Table {
	table := shopTable()
	table.Armors = shopCollection{
		{}, {name: "helm", price: 25, slot: 5, suit: wrFighterOnly,
			masks: [data.ShopShapes]uint16{1<<2 | 1<<3}},
	}
	table.Shields = shopCollection{
		{}, {name: "targe", price: 15, slot: 2, suit: wrFighterOnly,
			masks: [data.ShopShapes]uint16{1 << 4}},
	}
	return table
}

// TestTheShopGreysACellTheShownMemberCannotUse is 0162 spec AC-5, SC-5, over
// all three grids and both classes of shown member. The unusable cell is still
// occupied, which is what keeps it buyable, sellable and clickable.
func TestTheShopGreysACellTheShownMemberCannotUse(t *testing.T) {
	table := wrShopTable()

	for _, mage := range []bool{false, true} {
		f, s := shopRoom(t, nil)
		f.Table = table
		f.Carried[0].Mage = mage
		f.Shop = NewShop(1000)
		f.Shop.Generate(f.Table, 11)
		click(s, ui.ShopControlShelfPick, roomArmour)

		cell := s.ShopScreen().Shelf[0]
		if !cell.Occupied() {
			t.Fatalf("mage=%v: the first armour cell is empty, nothing to test", mage)
		}
		if mage {
			if cell.Back != ui.ShopBackUnusable {
				t.Errorf("mage=%v: background = %v, want the unusable one for a fighter-only helm",
					mage, cell.Back)
			}
			continue
		}
		if cell.Back == ui.ShopBackUnusable {
			t.Errorf("mage=%v: background = %v, want a usable one for a fighter-only helm",
				mage, cell.Back)
		}
	}
}

func TestAnUnusableShelfItemIsNeverDrawnAffordable(t *testing.T) {
	table := wrShopTable()
	f, s := shopRoom(t, nil)
	f.Table = table
	f.Carried[0].Mage = true
	f.Shop = NewShop(1000)
	f.Shop.Generate(f.Table, 11)
	click(s, ui.ShopControlShelfPick, roomArmour)

	f.Town.gold = 1000000
	if got := s.ShopScreen().Shelf[0].Back; got != ui.ShopBackUnusable {
		t.Errorf("background = %v with a full purse, want the unusable one", got)
	}
}

// TestTheShippedCensusMatchesTheWearRule is 0162 spec AC-2, SC-2. It reads the
// three equipment collections of a lawful install and asserts the partition
// and four named rows. It is skipped without an install, on golden rule 2.
func TestTheShippedCensusMatchesTheWearRule(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the shipped census needs a lawful install")
	}
	f, err := NewFrontEnd(root)
	if err != nil {
		t.Fatalf("NewFrontEnd(%q): %v", root, err)
	}

	type split struct{ fighter, mage, both, neither int }
	for _, c := range []struct {
		name string
		coll data.Collection
		want split
	}{
		{"Armors", f.Table.Armors, split{fighter: 19, mage: 9, both: 2}},
		{"Shields", f.Table.Shields, split{fighter: 9}},
		{"Weapons", f.Table.Weapons, split{fighter: 19, mage: 2, both: 3, neither: 3}},
	} {
		if c.coll == nil {
			t.Fatalf("%s: the install supplied no collection", c.name)
		}
		var got split
		// Row 0 is the reserved empty entry every one-based collection
		// carries; it is not a shipped item and is not counted.
		for i := 1; i < c.coll.Len(); i++ {
			s, ok := data.SuitabilityFromParams(c.coll.EntryParams(i))
			if !ok {
				t.Fatalf("%s row %d (%q): the row was not read", c.name, i, c.coll.EntryName(i))
			}
			switch {
			case s.Fighter && s.Mage:
				got.both++
			case s.Fighter:
				got.fighter++
			case s.Mage:
				got.mage++
			default:
				got.neither++
			}
		}
		if got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}

	for _, c := range []struct {
		coll data.Collection
		name string
		want data.Suitability
	}{
		{f.Table.Armors, "Plate Cuirass", data.Suitability{Fighter: true}},
		{f.Table.Armors, "Robe", data.Suitability{Mage: true}},
		{f.Table.Armors, "Ring", data.Suitability{Fighter: true, Mage: true}},
		{f.Table.Weapons, "Sonic Beam", data.Suitability{}},
	} {
		row := -1
		for i := 0; i < c.coll.Len(); i++ {
			if c.coll.EntryName(i) == c.name {
				row = i
				break
			}
		}
		if row < 0 {
			t.Fatalf("the install carries no row named %q", c.name)
		}
		got, _ := data.SuitabilityFromParams(c.coll.EntryParams(row))
		if got != c.want {
			t.Errorf("%q: %+v, want %+v", c.name, got, c.want)
		}
	}
}
