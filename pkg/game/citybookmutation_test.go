package game

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func cityBookSlot(t *testing.T, g *cityObjectTopology, partyID string, slot int) sim.SavedObjectID {
	t.Helper()
	for _, book := range g.Books {
		if string(book.PartyID) == partyID {
			return book.Slots[slot]
		}
	}
	t.Fatal("missing book", partyID)
	return 0
}

func cityBookFixture(t *testing.T, oldRange uint8, distinct, second, weapon bool) (*FrontEnd, func([]byte) *FrontEnd) {
	t.Helper()
	source := mageCity1101(t, true, true)
	file, err := sav.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	p, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	d := p.Data()
	var firstSpell uint16
	var units []*sav.CityUnitData
	for i := range d.Objects {
		if d.Objects[i].Spell != nil {
			firstSpell = uint16(i + 1)
			d.Objects[i].Spell.Fields[1] = oldRange
		}
		if d.Objects[i].Unit != nil {
			units = append(units, d.Objects[i].Unit)
		}
	}
	if firstSpell == 0 || len(units) != 2 {
		t.Fatal("fixture has no shared Spell or two Humans")
	}
	if distinct {
		fields := append([]byte(nil), d.Objects[firstSpell-1].Spell.Fields...)
		fields[5]++
		d.Objects = append(d.Objects, sav.CityObjectData{Class: "Spell", Spell: &sav.CitySpellData{Fields: fields}})
		for _, unit := range units {
			if unit.Name == "Leader" {
				unit.Spells[0] = uint16(len(d.Objects))
			}
		}
	}
	if second {
		d.Objects = append(d.Objects, sav.CityObjectData{Class: "Spell", Spell: &sav.CitySpellData{Fields: []byte{2, 99, 3, 188, 1, 0, 0x60, 0, 0}}})
		for _, u := range units {
			u.SpellbookCount = 3
			u.Spells = append(u.Spells, uint16(len(d.Objects)))
		}
	}
	if weapon {
		token, fields, derived := make([]byte, 37), make([]byte, 12), make([]byte, 47)
		binary.LittleEndian.PutUint32(token[29:], 0x7000)
		binary.LittleEndian.PutUint16(fields, 0x101)
		binary.LittleEndian.PutUint16(fields[2:], 1)
		fields[4], derived[0] = 2, 1
		d.Objects = append(d.Objects, sav.CityObjectData{Class: "Weapon", Item: &sav.CityItemData{Token: token, Fields: fields, Derived: derived, WeaponExtra: firstSpell}})
		for _, u := range units {
			u.Reference74 = uint16(len(d.Objects))
		}
	}
	p, err = sav.CityFromData(d)
	if err != nil {
		t.Fatal(err)
	}
	update := sav.CityUpdate{Money: 20000}
	for _, actor := range p.Roster() {
		update.Characters = append(update.Characters, originalCityBaselineUpdate(actor))
	}
	source, err = p.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	resources := mageFront1101(t, source).InstallResources
	rows := resources.Table.Spells.(dbCollection)
	params := make([]int32, 22)
	params[1], params[2], params[4], params[6], params[8], params[21] = 17, 1, 1, 8, 1, 77
	resources.Table.Spells = append(rows, dbEntry{name: "second spell", params: params})
	// The shelf's book code is the installed Book_Fire row, where the shipped
	// table holds it.
	items := make(dbCollection, 0x16)
	items[0x15].name = "Book_Fire"
	resources.Table.MagicItems = items
	load := func(raw []byte) *FrontEnd {
		f := &FrontEnd{InstallResources: resources}
		if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
			t.Fatal("city book LOAD", town, err)
		}
		return f
	}
	return load(source), load
}

func TestCityBookTrainingUpdatesEveryChangedSlotAndPreservesWeapon(t *testing.T) {
	f, load := cityBookFixture(t, 99, false, true, true)
	graph := f.Town.cityObjects.Clone()
	before := mapload.CloneParty(f.Carried)
	if msg := train1099(t, f, "hero", 5); !strings.Contains(msg, "trained") {
		t.Fatal(msg)
	}
	hero := trainingPartyMember(t, f, "hero")
	if hero.Book.Slots[0] != (sim.BookSpell{Range: 6, Defensive: 2, ManaCost: 65535}) || hero.Book.Slots[1] != (sim.BookSpell{Range: 9, Defensive: 3, ManaCost: 444}) {
		t.Fatal("derive did not refresh both current slots", hero.Book)
	}
	for slot := 0; slot < 2; slot++ {
		if cityBookSlot(t, f.Town.cityObjects, "hero", slot) != graph.NextID+sim.SavedObjectID(slot) {
			t.Fatal("changed slot did not receive its own new node", slot)
		}
	}
	if f.Town.cityObjects.NextID != graph.NextID+2 || !reflect.DeepEqual(f.Town.cityObjects.Items, graph.Items) || !reflect.DeepEqual(f.Town.cityObjects.Roots, graph.Roots) {
		t.Fatal("book training changed item edges or allocation floor")
	}
	for _, member := range before {
		got := trainingPartyMember(t, f, member.ID)
		if !reflect.DeepEqual(mapload.MemberItemEquipment(got, f.Table), mapload.MemberItemEquipment(member, f.Table)) || member.ID != "hero" && got.Book != member.Book {
			t.Fatal("training changed a surviving weapon or another book")
		}
	}
	fresh := load(cityProjectionSave(t, f))
	// The current graph carries native record supplements only when they were
	// loaded from a document. A newly trained Spell gets its native `This`
	// handle when the SAV producer constructs the record, so that supplement
	// appears after cold LOAD and is not a runtime graph mutation. Compare the
	// identity/location graph and every supplement that already existed before
	// SAVE; the current party remains authoritative for the new Spell values.
	if !reflect.DeepEqual(cityObjectIdentityTopology(fresh.Town.cityObjects), cityObjectIdentityTopology(f.Town.cityObjects)) ||
		!reflect.DeepEqual(fresh.Town.cityObjects.ItemRecords, f.Town.cityObjects.ItemRecords) ||
		!reflect.DeepEqual(fresh.Town.cityObjects.EffectRecords, f.Town.cityObjects.EffectRecords) ||
		trainingPartyMember(t, fresh, "hero").Book != hero.Book {
		t.Fatal("cold LOAD lost derived book values or identities")
	}
	for id, want := range f.Town.cityObjects.SpellRecords {
		got, ok := fresh.Town.cityObjects.SpellRecords[id]
		if !ok || !reflect.DeepEqual(got, want) {
			t.Fatal("cold LOAD changed an existing Spell supplement", id, got, want)
		}
	}
	bookValues := cityMemberBook(hero, f.Table)
	generated := make(map[sim.SavedObjectID]sim.SourceItemSpell, 2)
	for slot := 0; slot < 2; slot++ {
		id := cityBookSlot(t, f.Town.cityObjects, "hero", slot)
		if _, existed := graph.SpellRecords[id]; existed {
			t.Fatal("changed slot reused an existing Spell record", slot, id)
		}
		generated[id] = bookValues[slot]
	}
	for id, want := range generated {
		got, ok := fresh.Town.cityObjects.SpellRecords[id]
		if !ok || got.This == 0 || got.Value != want {
			t.Fatal("cold LOAD lost generated Spell supplement", id, got, want)
		}
	}
	second := load(cityProjectionSave(t, fresh))
	for id, want := range generated {
		first := fresh.Town.cityObjects.SpellRecords[id]
		got, ok := second.Town.cityObjects.SpellRecords[id]
		if !ok || got.This != first.This || got.Value != want {
			t.Fatal("second SAVE/LOAD changed generated Spell supplement", id, got, first, want)
		}
	}
}

func TestCityBookUnchangedWeaponAliasAndEqualDistinctBooksSurviveTraining(t *testing.T) {
	for _, distinct := range []bool{false, true} {
		t.Run(map[bool]string{false: "shared weapon and book", true: "equal distinct books"}[distinct], func(t *testing.T) {
			f, load := cityBookFixture(t, 6, distinct, false, true)
			graph := f.Town.cityObjects.Clone()
			heroID := cityBookSlot(t, graph, "hero", 0)
			if len(graph.Items) != 1 || graph.Items[0].Spell == 0 || (graph.Items[0].Spell != heroID) != distinct {
				t.Fatal("fixture did not distinguish shared and equal distinct Spell nodes")
			}
			beforeParty := mapload.CloneParty(f.Carried)
			before := trainingPartyMember(t, f, "hero").Book
			if msg := train1099(t, f, "hero", 5); !strings.Contains(msg, "trained") {
				t.Fatal(msg)
			}
			if !reflect.DeepEqual(graph, f.Town.cityObjects) || before != trainingPartyMember(t, f, "hero").Book {
				t.Fatal("unchanged scalar slot changed identity or parameters")
			}
			fresh := load(cityProjectionSave(t, f))
			if !reflect.DeepEqual(graph, fresh.Town.cityObjects) {
				t.Fatal("SAVE/LOAD merged equal nodes or split unchanged aliases")
			}
			for _, member := range beforeParty {
				if trainingPartyMember(t, fresh, member.ID).Book != member.Book {
					t.Fatal("cold LOAD changed an unchanged book's parameters")
				}
			}
		})
	}
}

func TestCityBookLearningCommitsPurseAndFreshSlotThroughSAV(t *testing.T) {
	for _, table := range []bool{false, true} {
		t.Run(map[bool]string{false: "shelf", true: "table"}[table], func(t *testing.T) {
			f, load := cityBookFixture(t, 6, false, false, true)
			f.Shop = NewShop(1000)
			f.Shop.shelves[ShelfBooks] = shopBookPool(f.Table, 1000)
			screen := f.TownScreen().(*townScreen)
			screen.shopChosen = roomBooks
			for i, member := range f.Carried {
				if member.ID == "hero" {
					screen.shopMember = i
				}
			}
			graph := f.Town.cityObjects.Clone()
			var action ui.TownAction
			if table {
				if !f.Shop.TakeFromShelf(ShelfBooks, 0, 1) {
					t.Fatal("could not stage merchant book")
				}
				action = screen.shopEquipFromTable(0)
			} else {
				action = screen.shopEquipFromShelf(0)
			}
			if !strings.Contains(action.Msg, "bought and learned") || f.Town.Gold() != 19923 {
				t.Fatal("book purchase failed or charged wrong amount", action, f.Town.Gold())
			}
			book := trainingPartyMember(t, f, "hero").Book
			if book.Slots[1] != (sim.BookSpell{Range: 9, Defensive: 0, ManaCost: 17}) || cityBookSlot(t, f.Town.cityObjects, "hero", 1) != graph.NextID+2 || f.Town.cityObjects.NextID != graph.NextID+3 || cityBookSlot(t, f.Town.cityObjects, "hero", 0) != cityBookSlot(t, graph, "hero", 0) {
				t.Fatal("learning did not preserve old alias and create one new slot", book)
			}
			if len(f.Shop.Table()) != 0 || shelfUnits(f.Shop, ShelfBooks) != 0 {
				t.Fatal("purchased book did not leave its exact shop location")
			}
			fresh := load(cityProjectionSave(t, f))
			if !reflect.DeepEqual(fresh.Town.cityObjects.Books, f.Town.cityObjects.Books) || fresh.Town.cityObjects.NextID != f.Town.cityObjects.NextID || trainingPartyMember(t, fresh, "hero").Book != book || fresh.Town.Gold() != 19923 {
				t.Fatal("book purchase did not survive cold LOAD")
			}
		})
	}
}

func TestCityBookLearningThenTrainingUsesCurrentBookWithoutReload(t *testing.T) {
	f, load := cityBookFixture(t, 6, false, false, true)
	f.Shop = NewShop(1000)
	f.Shop.shelves[ShelfBooks] = shopBookPool(f.Table, 1000)
	screen := f.TownScreen().(*townScreen)
	screen.shopChosen = roomBooks
	for i, member := range f.Carried {
		if member.ID == "hero" {
			screen.shopMember = i
		}
	}
	if action := screen.shopEquipFromShelf(0); !strings.Contains(action.Msg, "learned") {
		t.Fatal(action)
	}
	learned := trainingPartyMember(t, f, "hero")
	graph := f.Town.cityObjects.Clone()
	if msg := train1099(t, f, "hero", 5); !strings.Contains(msg, "to 33 for 3172") {
		t.Fatal("same-city learning blocked valid training", msg)
	}
	trained := trainingPartyMember(t, f, "hero")
	if f.Town.Gold() != 16751 || trained.KnownSpells != learned.KnownSpells || trained.Book != learned.Book || !reflect.DeepEqual(f.Town.cityObjects, graph) {
		t.Fatal("training replaced current learned fields, identity, or price")
	}
	fresh := load(cityProjectionSave(t, f))
	if got := trainingPartyMember(t, fresh, "hero"); got.Book != trained.Book || got.Hero != trained.Hero || fresh.Town.Gold() != 16751 {
		t.Fatal("learning then training did not survive cold LOAD")
	}
}

func TestCityBookAllocationFailureLeavesTrainingAndShopUntouched(t *testing.T) {
	for _, action := range []string{"training first", "training second", "shelf", "table", "pack"} {
		t.Run(action, func(t *testing.T) {
			f, _ := cityBookFixture(t, 99, false, true, true)
			f.Town.cityObjects.NextID = ^sim.SavedObjectID(0)
			if action == "training second" {
				f.Town.cityObjects.NextID--
			}
			f.Shop = NewShop(1000)
			f.Shop.shelves[ShelfBooks] = shopBookPool(f.Table, 1000)
			screen := f.TownScreen().(*townScreen)
			screen.room, screen.shopChosen = roomSchool, roomBooks
			for i, member := range f.Carried {
				if member.ID == "hero" {
					screen.shopMember = i
				}
			}
			if action == "table" && !f.Shop.TakeFromShelf(ShelfBooks, 0, 1) {
				t.Fatal("could not stage merchant book")
			}
			if action == "pack" {
				book := f.Shop.Shelf(ShelfBooks)[0].Instance()
				f.Carried[screen.shopMember].Carry.ItemInstances = []sim.ItemInstance{book}
				f.Carried[screen.shopMember].Carry.Items = []uint16{book.Code}
				f.Carried[screen.shopMember].Carry.OrderedStacks = []sim.ItemStack{sim.StackItem(book, 1)}
				f.Town.cityObjects.NextID = 100
				p, err := newCityObjectProjection(f.Town.cityObjects, f.Carried, f.Table)
				if err != nil {
					t.Fatal(err)
				}
				f.Town.cityObjects = p.graph
				f.Town.cityObjects.NextID = ^sim.SavedObjectID(0)
			}
			party, graph := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone()
			gold, history, stock, tray := f.Town.Gold(), f.originalCity.snapshot(), f.Shop.Shelf(ShelfBooks), f.Shop.Table()
			diamond := schoolDiamondState(screen)
			var msg string
			switch action {
			case "shelf":
				msg = screen.shopEquipFromShelf(0).Msg
			case "table":
				msg = screen.shopEquipFromTable(0).Msg
			case "pack":
				msg = screen.shopEquipFromPack(1).Msg
			default:
				msg = screen.trainHeroSkill(1)
			}
			if !strings.Contains(msg, "exhaust") {
				t.Fatal("allocation failure was not reported", msg)
			}
			if !reflect.DeepEqual(f.Carried, party) || !reflect.DeepEqual(f.Town.cityObjects, graph) || f.Town.Gold() != gold || !reflect.DeepEqual(f.originalCity.snapshot(), history) || !reflect.DeepEqual(f.Shop.Shelf(ShelfBooks), stock) || !reflect.DeepEqual(f.Shop.Table(), tray) || schoolDiamondState(screen) != diamond {
				t.Fatal("failed book mutation leaked values, edges, gold, stock, history or animation")
			}
		})
	}
}

func TestCityBookChangesRejectStaleSlotsAndMalformedCandidates(t *testing.T) {
	for _, kind := range []string{"missing party", "changed party", "missing slot", "unexpected slot", "bad book", "bad membership", "bad floor"} {
		t.Run(kind, func(t *testing.T) {
			f, _ := cityBookFixture(t, 99, false, false, true)
			g := f.Town.cityObjects
			before := trainingPartyMember(t, f, "hero")
			next := mapload.CloneParty([]mapload.PartyMember{before})[0]
			next.Book.Slots[0].Range = 6
			switch kind {
			case "missing party":
				before.ID, next.ID = "missing", "missing"
			case "changed party":
				next.ID = "different"
			case "missing slot":
				for i := range g.Books {
					if string(g.Books[i].PartyID) == "hero" {
						g.Books[i].Slots[0] = 0
					}
				}
			case "unexpected slot":
				before.KnownSpells, before.Book.Slots[0] = 0, sim.BookSpell{}
			case "bad book":
				next.Book.State = 255
			case "bad membership":
				next.KnownSpells = 0
			case "bad floor":
				g.NextID = 0
			}
			oldGraph := g.Clone()
			oldParty := mapload.CloneParty([]mapload.PartyMember{before, next})
			if candidate, err := prepareCityBookChanges(g, before, next, f.Table); err == nil || candidate != nil {
				t.Fatal("malformed book candidate admitted", kind, err)
			}
			if !reflect.DeepEqual(g, oldGraph) || !reflect.DeepEqual([]mapload.PartyMember{before, next}, oldParty) {
				t.Fatal("malformed input changed graph or values")
			}
		})
	}
}

func TestCityBookRemovalKeepsSurvivingWeaponAndOtherBook(t *testing.T) {
	f, _ := cityBookFixture(t, 6, false, false, true)
	g := f.Town.cityObjects.Clone()
	before := trainingPartyMember(t, f, "hero")
	next := mapload.CloneParty([]mapload.PartyMember{before})[0]
	next.KnownSpells, next.Book.Slots = 0, [28]sim.BookSpell{}
	candidate, err := prepareCityBookChanges(g, before, next, f.Table)
	if err != nil {
		t.Fatal(err)
	}
	want := g.Clone()
	for i := range want.Books {
		if string(want.Books[i].PartyID) == "hero" {
			want.Books[i].Slots[0] = 0
		}
	}
	if !reflect.DeepEqual(candidate, want) || cityBookSlot(t, g, "hero", 0) == 0 {
		t.Fatal("removal pruned a shared Spell or mutated input")
	}
}

func TestCityBookNativeTrainingUsesCurrentGraphAndKeepsMemberPointer(t *testing.T) {
	f := shellFrontEnd()
	f.Table.Spells = dbCollection{{}, {name: "spell", params: []int32{0, 7, 1, 0, 1, 0, 5, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0}}}
	m := &f.Carried[0]
	m.ID, m.Mage, m.SpellbookRestored, m.SpellbookPresent = "hero", true, true, true
	m.Hero.Skill[1], m.Hero.Mind = 32, 30
	m.Book.State, m.KnownSpells = sim.BookPresent, 1<<1
	m.Book.Slots[0] = sim.BookSpell{Range: 99, Defensive: 2, ManaCost: 65535}
	other := mapload.CloneParty([]mapload.PartyMember{*m})[0]
	other.ID, other.Name = "other", "Other"
	f.Carried = append(f.Carried, other)
	m = &f.Carried[0]
	f.Town.gold = 20000
	f.Town.cityObjects = &cityObjectTopology{Version: cityObjectTopologyVersion, NextID: 2,
		Spells: []sim.SavedObjectID{1}, Roots: []cityPartyObjectRoots{{PartyID: []byte("hero")}, {PartyID: []byte("other")}},
		Books: []cityBookTopology{{PartyID: []byte("hero"), Slots: [28]sim.SavedObjectID{1}}, {PartyID: []byte("other"), Slots: [28]sim.SavedObjectID{1}}}}
	if msg := f.townUI.trainHeroSkill(1); !strings.HasPrefix(msg, "trained ") {
		t.Fatal(msg)
	}
	if m.Hero.Skill[1] != 33 || m.Book.Slots[0] != (sim.BookSpell{Range: 6, Defensive: 2, ManaCost: 65535}) || f.Town.Gold() != 15778 {
		t.Fatal("native training changed price or lost the current member pointer", m.Hero, m.Book, f.Town.Gold())
	}
	if cityBookSlot(t, f.Town.cityObjects, "hero", 0) != 2 || cityBookSlot(t, f.Town.cityObjects, "other", 0) != 1 || f.Carried[1].Book.Slots[0] != other.Book.Slots[0] {
		t.Fatal("native training did not independently update the selected book")
	}
}

func cityBookEquipFixture(t *testing.T, route string) (*FrontEnd, *townScreen, int, sim.ItemInstance) {
	t.Helper()
	f, screen := shopRoom(t, nil)
	f.Table.Spells = dbCollection{{}, {name: "spell", params: []int32{0, 7, 1, 0, 1, 0, 5, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0}}}
	m := &f.Carried[0]
	m.ID, m.Mage, m.SpellbookRestored, m.SpellbookPresent = "hero", true, true, true
	m.Hero.Skill[1], m.Hero.Mind = 32, 30
	m.Book.State, m.KnownSpells = sim.BookPresent, 1<<1
	m.Book.Slots[0] = sim.BookSpell{Range: 99, Defensive: 2, ManaCost: 65535}
	other := mapload.CloneParty([]mapload.PartyMember{*m})[0]
	other.ID = "other"
	f.Carried = append(f.Carried, other)
	item := f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)].Instance()
	slot, ok := EquipTarget(data.ItemCode(item.Code), f.Table)
	if !ok {
		t.Fatal("fixture item has no worn slot")
	}
	screen.shopChosen = roomArmour
	switch route {
	case "pack":
		setShopIdentityPack(f, item)
	case "shelf":
		f.Shop.shelves[ShelfArmour] = []ShopItem{shopItemFromInstance(item, 1)}
	case "table":
		f.Shop.table = []ShopPlace{{ShopItem: shopItemFromInstance(item, 1)}}
	case "unequip", "unequip table":
		f.Carried[0].Carry.EquippedItems[slot-1] = item
		f.Carried[0].Carry.Equipped[slot-1] = item.Code
	}
	p, err := newCityObjectProjection(nil, f.Carried, f.Table)
	if err != nil {
		t.Fatal(err)
	}
	f.Town.cityObjects = p.graph
	id := cityBookSlot(t, p.graph, "hero", 0)
	for i := range p.graph.Books {
		if string(p.graph.Books[i].PartyID) == "other" {
			p.graph.Books[i].Slots[0] = id
		}
	}
	return f, screen, slot, item
}

func TestCityBookEquipmentHooksCommitOrRollbackTogether(t *testing.T) {
	for _, route := range []string{"pack", "shelf", "table", "direct", "unequip", "unequip table"} {
		for _, exhausted := range []bool{false, true} {
			t.Run(route+map[bool]string{false: " success", true: " overflow"}[exhausted], func(t *testing.T) {
				f, screen, slot, item := cityBookEquipFixture(t, route)
				if exhausted {
					f.Town.cityObjects.NextID = ^sim.SavedObjectID(0)
				}
				before, graph := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone()
				gold, stock, tray := f.Town.Gold(), f.Shop.Shelf(ShelfArmour), f.Shop.Table()
				var msg string
				switch route {
				case "pack":
					msg = screen.shopEquipFromPack(1).Msg
				case "shelf":
					msg = screen.shopEquipFromShelf(0).Msg
				case "table":
					msg = screen.shopEquipFromTable(0).Msg
				case "direct":
					screen.shopWearItemInto(slot, item)
				case "unequip":
					msg = screen.shopUnequipDoll(slot).Msg
				case "unequip table":
					msg = screen.shopUnequipToTable(slot).Msg
				}
				if exhausted {
					if route != "direct" && !strings.Contains(msg, "exhaust") {
						t.Fatal("equipment book failure not reported", msg)
					}
					if !reflect.DeepEqual(f.Carried, before) || !reflect.DeepEqual(f.Town.cityObjects, graph) || f.Town.Gold() != gold || !reflect.DeepEqual(f.Shop.Shelf(ShelfArmour), stock) || !reflect.DeepEqual(f.Shop.Table(), tray) {
						t.Fatal("failed equipment book update leaked party, graph, gold, stock or table")
					}
					return
				}
				wantBook := graph.NextID
				if route == "shelf" || route == "table" || route == "direct" {
					wantBook++
				}
				if f.Carried[0].Book.Slots[0] != (sim.BookSpell{Range: 6, Defensive: 2, ManaCost: 65535}) || f.Carried[1].Book != before[1].Book || cityBookSlot(t, f.Town.cityObjects, "hero", 0) != wantBook || cityBookSlot(t, f.Town.cityObjects, "other", 0) != cityBookSlot(t, graph, "other", 0) {
					t.Fatal("equipment change did not independently refresh the current book", msg)
				}
				wantGold := gold
				if route == "shelf" || route == "table" {
					wantGold -= int(item.Price)
				}
				if f.Town.Gold() != wantGold {
					t.Fatal("equipment book hook changed transaction cost")
				}
			})
		}
	}
}
