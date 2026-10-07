package game

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestReleaseEnchantedItemProducerPopulation walks every shipped producer this
// story consumes. It runs once per lawful root and prints the population and
// the retained blind spots instead of treating selected examples as a census.
func TestReleaseEnchantedItemProducerPopulation(t *testing.T) {
	f := releaseFront(t)
	cells, braced, effects, rejected := releaseEquipmentCellPopulation(t, f.Table)
	if cells != 2386 || braced != 260 || effects != 266 || rejected != 1 {
		t.Fatalf("equipment population = cells %d, braced %d, effects %d, rejected %d; want 2386/260/266/1",
			cells, braced, effects, rejected)
	}

	missions := releaseMissionPopulation(t, f)
	if len(missions) != 28 {
		t.Fatalf("campaign map population = %d, want 28: %v", len(missions), missions)
	}
	var records, elements, type9, links, linkedRecords int
	var castEligible, castSuppressed, unitItemCasts, dragons int
	brigand, scrakan, treasure, shortBow, fireRing := false, false, false, false, false
	var shortBowValue, fireRingValue int32
	flame, err := data.ResolveWeapon("Flame Thrower", f.Table.Shapes, f.Table.Materials, f.Table.Weapons)
	if err != nil {
		t.Fatalf("resolve innate Flame Thrower: %v", err)
	}
	for _, mission := range missions {
		m := releaseMissionMap(t, f, mission)
		loot, err := m.Loot()
		if err != nil {
			t.Fatalf("mission %d loot: %v", mission, err)
		}
		records += len(loot.Records)
		type9 += len(m.Enchantments)
		linked := make(map[uint32]bool)

		ms, err := StartMissionFrom(m, fmt.Sprintf("mission %d", mission), mission,
			f.Table, mapload.DifficultyNormal, nil)
		if err != nil {
			t.Fatalf("StartMission(%d): %v", mission, err)
		}
		entities := ms.World.Entities()
		var dragonsToKill []sim.EntityID
		var scrakanToKill *sim.Entity
		var scrakanStaff uint16
		for i, u := range m.Units {
			resolution := mapload.Resolve(u, f.Table)
			if !resolution.Found() {
				continue
			}
			entity := entities[i]
			var collection data.Collection
			if resolution.Arm == mapload.ArmUnits {
				collection = f.Table.Units
			} else {
				collection = f.Table.Humans
			}
			name := collection.EntryName(resolution.Index)
			worn, _ := ms.World.EquippedItems(entity.ID)
			if resolution.Arm != mapload.ArmUnits {
				if spell, power, ok := worn[0].CastSpell(); ok {
					if entity.WeaponSpellSource != sim.WeaponSpellItem || entity.WeaponSpell != spell || entity.WeaponSpellLevel != power {
						t.Fatalf("mission %d unit %d %s item/source mismatch: entity %+v item %+v", mission, u.UnitID, name, entity, worn[0])
					}
					if entity.SuppressCorpseLoot {
						castSuppressed++
					} else {
						castEligible++
					}
				}
			} else if entity.WeaponSpell != 0 || entity.WeaponSpellLevel != 0 {
				unitItemCasts++
				spell, power, cast := worn[0].CastSpell()
				if !cast || entity.WeaponSpellSource != sim.WeaponSpellItem || entity.WeaponSpell != spell || entity.WeaponSpellLevel != power {
					t.Fatalf("mission %d unit %d %s cast pair = (%d,%d) source %d, held item %+v",
						mission, u.UnitID, name, entity.WeaponSpell, entity.WeaponSpellLevel, entity.WeaponSpellSource, worn[0])
				}
			}
			if strings.Contains(name, "Dragon") {
				dragons++
				if entity.WeaponSpellSource != sim.WeaponSpellNone || entity.WeaponSpell != 0 || entity.WeaponSpellLevel != 0 {
					t.Fatalf("mission %d unit %d %s row %d strings %q invented spell state (%d,%d,%d)",
						mission, u.UnitID, name, resolution.Index, collection.EntryStrings(resolution.Index),
						entity.WeaponSpellSource, entity.WeaponSpell, entity.WeaponSpellLevel)
				}
				if worn[0].Code != uint16(flame.Code) || !worn[0].SourceEquipment.Definition.Present ||
					worn[0].SourceEquipment.Definition.Suitable != 0 {
					t.Fatalf("mission %d unit %d %s lost held innate Flame Thrower: %+v", mission, u.UnitID, name, worn[0])
				}
				if _, _, cast := worn[0].CastSpell(); cast {
					t.Fatalf("mission %d unit %d %s invented a cast suffix: %+v", mission, u.UnitID, name, worn[0])
				}
				dragonsToKill = append(dragonsToKill, entity.ID)
			}
			if mission == 100 && u.UnitID == 97 {
				brigand = name == "M_Brigand3" && entity.X == 14 && entity.Y == 107
			}
			if mission == 150 && u.UnitID == 13 {
				scrakan = name == "NPC_Scrakan" && entity.X == 14 && entity.Y == 103 && entity.SuppressCorpseLoot
				scrakanCopy := entity
				scrakanToKill = &scrakanCopy
				scrakanStaff = worn[0].Code
			}
		}

		for _, record := range loot.Records {
			elements += len(record.Elements)
			for _, element := range record.Elements {
				if element.TileMarkerIndex != 0 {
					links++
					linked[element.TileMarkerIndex] = true
				}
				want := releaseLootItem(t, element, m, f.Table)
				if !releaseWorldHoldsLoot(ms.World, m, record, want) {
					t.Fatalf("mission %d record owner %d cell (%d,%d) does not hold decoded item %+v",
						mission, record.Owner, record.CellX(), record.CellY(), want)
				}
				if mission == 40 && record.Ground() && record.CellX() == 96 && record.CellY() == 84 && element.ItemCode() == 0x0e1f {
					treasure = want.Price == 10000 && len(want.Effects) == 0 &&
						!containsString(itemInstanceInfoLines(want, f.Table), "#Value 10000")
				}
				if mission == 70 && record.Ground() && element.ItemCode() == 0x8134 {
					base := mapload.ItemInstanceFromCode(element.ItemCode(), f.Table)
					lines := itemInstanceInfoLines(want, f.Table, f.Words)
					shortBow = record.CellX() == 109 && record.CellY() == 110 && want.Price != base.Price &&
						reflect.DeepEqual(want.Effects, []sim.ItemEffect{{Kind: 12, Operand: 5}}) &&
						containsString(lines, fmt.Sprintf("#%s +5", f.Words.ItemStats[12])) &&
						!containsString(lines, fmt.Sprintf("#%s %d", f.Words.ItemStats[1], want.Price))
					if shortBow {
						shortBowValue = want.Price
					}
				}
				if mission == 70 && record.Ground() && element.ItemCode() == 0x4461 {
					base := mapload.ItemInstanceFromCode(element.ItemCode(), f.Table)
					lines := itemInstanceInfoLines(want, f.Table, f.Words)
					fireRing = record.CellX() == 128 && record.CellY() == 51 && want.Price != base.Price &&
						reflect.DeepEqual(want.Effects, []sim.ItemEffect{{Kind: 21, Operand: 5}}) &&
						containsString(lines, fmt.Sprintf("#%s +5", f.Words.ItemStats[21])) &&
						!containsString(lines, fmt.Sprintf("#%s %d", f.Words.ItemStats[1], want.Price))
					if fireRing {
						fireRingValue = want.Price
					}
				}
			}
		}
		flameBefore := releaseSackCodeCount(ms.World.Sacks(), uint16(flame.Code))
		for _, id := range dragonsToKill {
			if err := ms.World.HeadlessKill(id); err != nil {
				t.Fatal(err)
			}
			if worn, ok := ms.World.EquippedItems(id); !ok || worn[0].Code != uint16(flame.Code) {
				t.Fatalf("mission %d Dragon %d lost Flame Thrower on death: equipped=%v, ok=%v", mission, id, worn[0], ok)
			}
			sim.Step(ms.World, nil)
			remaining := releaseEntityByID(t, ms.World, id).Dwell
			for ; remaining > 0; remaining-- {
				sim.Step(ms.World, nil)
			}
		}
		if flameAfter := releaseSackCodeCount(ms.World.Sacks(), uint16(flame.Code)); flameAfter != flameBefore {
			t.Fatalf("mission %d Dragon deaths changed Flame Thrower sack count from %d to %d", mission, flameBefore, flameAfter)
		}
		if scrakanToKill != nil {
			before := releaseSackCodeCount(ms.World.Sacks(), scrakanStaff)
			headlessFell(t, ms.World, scrakanToKill.ID)
			sim.Step(ms.World, nil)
			if after := releaseSackCodeCount(ms.World.Sacks(), scrakanStaff); after != before {
				t.Fatalf("mission %d NPC_Scrakan death changed staff sack count from %d to %d", mission, before, after)
			}
		}
		linkedRecords += len(linked)
	}
	if records != 176 || elements != 177 || type9 != 199 || links != 74 {
		t.Fatalf("ALM population = %d records/%d elements/%d type9/%d links; want 176/177/199/74",
			records, elements, type9, links)
	}
	if castEligible != 49 || castSuppressed != 27 {
		t.Fatalf("Human cast-spell placements = %d eligible/%d suppressed; want 49/27", castEligible, castSuppressed)
	}
	if unitItemCasts != 3 {
		t.Fatalf("Unit held-item cast-spell placements = %d, want 3", unitItemCasts)
	}
	if releaseDragonDefinitions(f.Table.Units) != 4 || dragons != 21 {
		t.Fatalf("Dragon population = %d definitions/%d placements; want 4/21", releaseDragonDefinitions(f.Table.Units), dragons)
	}
	if !brigand || !scrakan || !treasure || !shortBow || !fireRing {
		t.Fatalf("named witnesses: M_Brigand3=%v NPC_Scrakan=%v Treasure=%v mission70-Short-Bow=%v mission70-Fire-Ring=%v",
			brigand, scrakan, treasure, shortBow, fireRing)
	}
	t.Logf("equipment cells=%d braced=%d effects=%d rejected=%d; maps=%d type8=%d/%d type9=%d linked=%d records (%d elements); Human staff placements=%d eligible + %d suppressed; Unit held-item cast placements=%d; Dragons=4 definitions/%d placements",
		cells, braced, effects, rejected, len(missions), records, elements, type9, linkedRecords, links, castEligible, castSuppressed, unitItemCasts, dragons)
	t.Logf("mission 70 linked values: Short Bow=%d (Attack +5), Fire Ring=%d (Fire protection +5)", shortBowValue, fireRingValue)
	t.Log("blind spots: computed image targets from EXP-0227 and post-pin item-description text are outside story 1040")
}

func releaseMissionPopulation(t *testing.T, f *FrontEnd) []int {
	t.Helper()
	var missions []int
	for _, entry := range f.Archives.Containers.Entries() {
		if !strings.HasPrefix(entry.Address, scenarioPrefix) || !strings.HasSuffix(entry.Address, ".alm") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(entry.Address, scenarioPrefix), ".alm")
		mission, err := strconv.Atoi(name)
		if err != nil {
			t.Fatalf("scenario archive entry %q does not name a numeric ALM: %v", entry.Address, err)
		}
		missions = append(missions, mission)
	}
	sort.Ints(missions)
	return missions
}

func releaseEquipmentCellPopulation(t *testing.T, table *mapload.Table) (cells, braced, effects, rejected int) {
	t.Helper()
	for _, collection := range []data.Collection{table.Humans, table.Units} {
		for i := 1; i < collection.Len(); i++ {
			for _, cell := range collection.EntryStrings(i) {
				cells++
				if strings.Contains(cell, "{") {
					braced++
				}
				_, parsed, n, err := mapload.ParseItemCell(cell, table)
				if err != nil {
					t.Fatalf("%s equipment cell %q: %v", collection.EntryName(i), cell, err)
				}
				effects += len(parsed)
				rejected += n
				if n != 0 {
					t.Logf("reported equipment token: %s %q rejected=%d", collection.EntryName(i), cell, n)
				}
			}
		}
	}
	return cells, braced, effects, rejected
}

func releaseLootItem(t *testing.T, element alm.LootElement, m *alm.Map, table *mapload.Table) sim.ItemInstance {
	t.Helper()
	item := mapload.ItemInstanceFromCode(element.ItemCode(), table)
	if element.TileMarkerIndex == 0 {
		return item
	}
	index := element.TileMarkerIndex - 1
	if index >= uint32(len(m.Enchantments)) {
		t.Fatalf("loot item %#04x link %d is outside %d type9 records", item.Code, element.TileMarkerIndex, len(m.Enchantments))
	}
	recipe := m.Enchantments[index]
	if recipe.X != 0 || recipe.Y != 0 {
		t.Fatalf("loot item %#04x link %d names ineligible type9 (%d,%d)", item.Code, element.TileMarkerIndex, recipe.X, recipe.Y)
	}
	addEffect := func(kind uint32, operand uint32) {
		if kind > 255 {
			t.Fatalf("type9 link %d kind %d exceeds u8", element.TileMarkerIndex, kind)
		}
		item.Effects = append(item.Effects, sim.ItemEffect{Kind: uint8(kind), Operand: operand})
	}
	if recipe.A != 0 {
		addEffect(uint32(recipe.A)+43, uint32(uint8(recipe.B))|uint32(uint8(recipe.C))<<8)
	}
	if uint16(recipe.SpellRaw) != 0 {
		kind := uint8(41)
		if item.Kind == 5 {
			kind = 42
		}
		item.Effects = append(item.Effects, sim.ItemEffect{Kind: kind, Operand: recipe.SpellRaw})
	}
	for _, effect := range recipe.Elements {
		kind := effect.Kind
		if kind == 41 {
			kind = 49
		}
		addEffect(uint32(kind), uint32(effect.Low)|uint32(effect.High)<<16)
	}
	item.Price = releaseStoredItemPrice(item, table)
	return item
}

func releaseStoredItemPrice(item sim.ItemInstance, table *mapload.Table) int32 {
	if item.Kind == 5 && len(item.Effects) == 1 {
		effect := item.Effects[0]
		if effect.Kind == 42 && effect.Mode == 0 && effect.Operand > 0 && effect.Operand < 32 {
			if table != nil && table.Spells != nil && int(effect.Operand) < table.Spells.Len() {
				params := table.Spells.EntryParams(int(effect.Operand))
				if len(params) > 21 {
					return params[21]
				}
			}
			return item.Price
		}
	}
	base := item.Price
	effects := item.Effects
	var points int32
	var cast, direct int64
	for _, effect := range effects {
		magnitude := int32(effect.Operand)
		if effect.Mode != 0 && effect.Mode != 8 {
			magnitude = int32(int16(effect.Operand))
		}
		if effect.Kind == 1 {
			direct += int64(magnitude)
			continue
		}
		if effect.Kind == 41 {
			if table == nil || table.Spells == nil {
				continue
			}
			spell := int(uint16(effect.Operand))
			if spell >= table.Spells.Len() {
				continue
			}
			params := table.Spells.EntryParams(spell)
			power := int32(int16(effect.Operand >> 16))
			if len(params) <= 20 || params[20] <= 0 || power < 0 {
				continue
			}
			rank := math.Log(float64(power)/30+1) / math.Log(1.2)
			cast += int64(10 * float64(params[20]) * math.Pow(2, rank))
			continue
		}
		if table == nil || table.Magic == nil || int(effect.Kind) >= table.Magic.Len() {
			continue
		}
		params := table.Magic.EntryParams(int(effect.Kind))
		if len(params) == 0 || params[0] <= 0 {
			continue
		}
		if effect.Kind >= 44 && effect.Kind <= 48 {
			magnitude = int32(uint8(effect.Operand)) + int32(uint8(effect.Operand>>8))
		}
		points += magnitude * params[0]
	}
	var ordinary int64
	if points > 0 {
		n := float64(points)
		ordinary = int64((math.Pow(1.5, n/70) + 1) * n * 50)
	}
	price := int64(base) + direct + ordinary + cast
	if item.Kind == 4 {
		// ITEM-VALUE-115 gives Scroll its own value branch: only kind-41
		// contributions, with the MagicItems base used on a zero sum.
		// Equipment still takes base + every supported effect above.
		price = cast
		if price == 0 {
			price = int64(base)
		}
	}
	if price >= 9_999_999 {
		return 9_999_999
	}
	if price < -1<<31 {
		return -1 << 31
	}
	return int32(price)
}

func TestScrollValueOracleKeepsSubtypeAndDefaultCastPower(t *testing.T) {
	table := &mapload.Table{Spells: enchantSpells{5: 80}, MagicItems: potionShopRows{{}, {"Scroll", ""}}}
	for _, tc := range []struct {
		power uint32
		want  int32
	}{{0, 800}, {6, 1600}} {
		power := tc.power
		item := sim.ItemInstance{Code: 0xe01, Kind: 4, Price: 37, Effects: []sim.ItemEffect{
			{Kind: 41, Operand: 5 | power<<16}, {Kind: 1, Operand: 17},
		}}
		want := tc.want
		if got := releaseStoredItemPrice(item, table); got != want {
			t.Fatalf("power%d independent oracle=%d want%d", power, got, want)
		}
		if got := mapload.RepriceItemInstance(item, table); got != want {
			t.Fatalf("power%d production value=%d want%d", power, got, want)
		}
		item.Kind = 1
		wantEquipment := 37 + 17 + tc.want
		if got := releaseStoredItemPrice(item, table); got != wantEquipment {
			t.Fatalf("equipment oracle lost additive valuation: %d want%d", got, wantEquipment)
		}
	}
}

func releaseWorldHoldsLoot(w *sim.World, m *alm.Map, record alm.LootRecord, want sim.ItemInstance) bool {
	if record.Ground() {
		for _, sack := range w.Sacks() {
			if sack.X != record.CellX() || sack.Y != record.CellY() {
				continue
			}
			if releaseItemsContain(sack.ItemInstances, want) {
				return true
			}
			for _, code := range sack.Items {
				if reflect.DeepEqual(sim.PlainItem(code), want) {
					return true
				}
			}
		}
		return false
	}
	for i, unit := range m.Units {
		if uint32(unit.UnitID) != record.Owner {
			continue
		}
		items, ok := w.CarriedItems(sim.EntityID(i))
		return ok && releaseItemsContain(items, want)
	}
	return false
}

func releaseItemsContain(items []sim.ItemInstance, want sim.ItemInstance) bool {
	for _, item := range items {
		if reflect.DeepEqual(item, want) {
			return true
		}
	}
	return false
}

func releaseDragonDefinitions(units data.Collection) int {
	n := 0
	for i := 1; i < units.Len(); i++ {
		if strings.Contains(units.EntryName(i), "Dragon") {
			n++
		}
	}
	return n
}

func containsString(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}

func releaseTerminalDeath(t *testing.T, w *sim.World, id sim.EntityID) {
	t.Helper()
	if err := w.HeadlessKill(id); err != nil {
		t.Fatal(err)
	}
	sim.Step(w, nil)
	remaining := releaseEntityByID(t, w, id).Dwell
	for ; remaining > 0; remaining-- {
		sim.Step(w, nil)
	}
}

// TestReleaseMBrigand3KeepsOneStaffThroughDeathPickupEquipSaveAndCast starts
// from the exact shipped actor and then drives every stateful boundary with the
// complete loaded instance. The final cast uses the production weapon-attack
// path and the installed spell table.
func TestReleaseMBrigand3KeepsOneStaffThroughDeathPickupEquipSaveAndCast(t *testing.T) {
	f := releaseFront(t)
	party := []mapload.PartyMember{{ID: "item-witness", PlayerCharacter: true, StartingHero: true,
		Mage: true, Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: data.NewCampaignHero(data.SkillBlade)}}
	ms, err := StartMission(f.Archives.Containers, 100, f.Table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatalf("StartMission(100): %v", err)
	}
	mage := entityByMapUnitID(t, ms.World, 97)
	if mage.X != 14 || mage.Y != 107 || mage.WeaponSpellSource != sim.WeaponSpellItem {
		t.Fatalf("M_Brigand3 u97 = %+v", mage)
	}
	worn, _ := ms.World.EquippedItems(mage.ID)
	staff := worn[0]
	spell, power, ok := staff.CastSpell()
	if !ok || spell != mage.WeaponSpell || power != mage.WeaponSpellLevel || power != 15 {
		t.Fatalf("M_Brigand3 staff/source = %+v / %+v", staff, mage)
	}

	releaseTerminalDeath(t, ms.World, mage.ID)
	if !releaseSackAtContains(ms.World.Sacks(), mage.X, mage.Y, staff) {
		t.Fatalf("M_Brigand3 death sack at (%d,%d) lost staff %+v", mage.X, mage.Y, staff)
	}
	taker := ms.Start.IDs[0]
	worldTakeAt(t, ms, taker, mage.X, mage.Y)
	stacks, _ := ms.World.CarriedStacks(taker)
	stackIndex := -1
	for i, stack := range stacks {
		if reflect.DeepEqual(stack.Instance(), staff) {
			stackIndex = i
			break
		}
	}
	if stackIndex < 0 {
		t.Fatalf("party pack after pickup = %+v, want staff %+v", stacks, staff)
	}
	sim.Step(ms.World, []sim.Command{{Kind: sim.KindEquip, Entity: taker, X: int32(stackIndex), Y: 1}})
	partyWorn, _ := ms.World.EquippedItems(taker)
	if !reflect.DeepEqual(partyWorn[0], staff) {
		t.Fatalf("party equipped staff = %+v, want %+v", partyWorn[0], staff)
	}

	form, _ := ms.World.MarshalBinary()
	var restored sim.World
	if err := restored.UnmarshalBinary(form); err != nil {
		t.Fatalf("reload item witness: %v", err)
	}
	restoredWorn, _ := restored.EquippedItems(taker)
	restoredEntity := releaseEntityByID(t, &restored, taker)
	if !reflect.DeepEqual(restoredWorn[0], staff) || restoredEntity.WeaponSpellSource != sim.WeaponSpellItem ||
		restoredEntity.WeaponSpell != spell || restoredEntity.WeaponSpellLevel != power {
		t.Fatalf("reloaded party source/item = %+v / %+v", restoredEntity, restoredWorn[0])
	}

	castWorld := releaseStaffCastWorld(t, restoredEntity, staff, mapload.SpellRules(f.Table))
	var event sim.CastEvent
	for tick := 0; tick < 512 && event.Spell == 0; tick++ {
		var commands []sim.Command
		if tick == 0 {
			commands = []sim.Command{{Kind: sim.KindAttack, Entity: 1, X: 2}}
		}
		for _, candidate := range sim.StepObserved(castWorld, commands) {
			if candidate.Weapon {
				event = candidate
				break
			}
		}
	}
	if !event.Weapon || event.Spell != spell || event.Caster != 1 || event.Target != 2 {
		t.Fatalf("reloaded equipped staff produced cast event %+v, want weapon spell %d from 1 to 2", event, spell)
	}
	t.Logf("100.alm u97 M_Brigand3 staff %#04x Lightning:%d survived death, pickup, equip and form reload, then emitted weapon cast %+v", staff.Code, power, event)
}

func releaseSackAtContains(sacks []sim.Sack, x, y int32, want sim.ItemInstance) bool {
	for _, sack := range sacks {
		if sack.X == x && sack.Y == y && releaseItemsContain(sack.ItemInstances, want) {
			return true
		}
	}
	return false
}

func releaseSackCodeCount(sacks []sim.Sack, code uint16) int {
	count := 0
	for _, sack := range sacks {
		for _, item := range sack.Items {
			if item == code {
				count++
			}
		}
	}
	return count
}

func releaseEntityByID(t *testing.T, w *sim.World, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, entity := range w.Entities() {
		if entity.ID == id {
			return entity
		}
	}
	t.Fatalf("world has no entity %d", id)
	return sim.Entity{}
}

func releaseStaffCastWorld(t *testing.T, loaded sim.Entity, staff sim.ItemInstance, spells []sim.SpellRule) *sim.World {
	t.Helper()
	caster := sim.Entity{ID: 1, X: 2, Y: 2, Owner: 1, HP: 100, MaxHP: 100,
		Mana: 100, MaxMana: 100, Mind: loaded.Mind, Reach: 1, ScanRange: 8,
		AttackCharge: 1, AttackRelax: 1, WeaponSpell: loaded.WeaponSpell,
		WeaponSpellLevel: loaded.WeaponSpellLevel, WeaponSpellSource: sim.WeaponSpellItem}
	target := sim.Entity{ID: 2, X: 3, Y: 2, Owner: 2, HP: 10000, MaxHP: 10000, DyingTime: 200}
	var relations sim.Relations
	relations.Set(1, 2, 1)
	stock := sim.Stock{ID: caster.ID}
	stock.EquippedItems[0] = staff
	w, err := sim.NewStockedSpelledWorld(1040, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{caster, target}, nil, relations, nil, []sim.Stock{stock}, spells)
	if err != nil {
		t.Fatalf("build staff cast world: %v", err)
	}
	return w
}

// TestReleaseGeneratedMagicStaffKeepsItsSpellAndExactPriceThroughTheWholePath
// starts at a forced-cast weapon triple from the installed shop table. The
// fixed draws make the generator's spell, power and price independently
// calculable, then the exact instance crosses every shop and simulation
// boundary a purchased staff reaches.
func TestReleaseGeneratedMagicStaffKeepsItsSpellAndExactPriceThroughTheWholePath(t *testing.T) {
	f := releaseFront(t)
	const ceiling int32 = 1_000_000

	var forced []data.ShopCandidate
	for _, candidate := range shopWeaponPool(f.Table, ceiling) {
		if candidate.ForcedCast {
			forced = append(forced, candidate)
		}
	}
	if len(forced) != 10 {
		t.Fatalf("forced-cast weapon triples = %d, want 10", len(forced))
	}

	const spellID uint16 = 13 // Lightning: fixed draw 1 from the mage spell set.
	spellParams := f.Table.Spells.EntryParams(int(spellID))
	if len(spellParams) <= 20 || spellParams[20] <= 0 {
		t.Fatalf("Lightning row %d has no cast-price scalar: %v", spellID, spellParams)
	}
	scalar := spellParams[20]
	var candidate data.ShopCandidate
	powerMax := 0
	for _, possible := range forced {
		name := f.Table.Weapons.EntryName(possible.Row)
		budget := releaseForcedCastBudget(possible, ceiling)
		maximum := releaseCastPowerMax(budget, scalar)
		if strings.Contains(name, "Staff") && maximum >= 15 {
			candidate, powerMax = possible, maximum
			break
		}
	}
	if candidate.Code == 0 {
		t.Fatalf("no installed forced-cast Staff triple can generate Lightning power 15: %+v", forced)
	}
	definition := f.Table.Weapons.EntryParams(candidate.Row)
	if len(definition) <= data.SutableForColumn || definition[data.SutableForColumn]&3 != 2 {
		t.Fatalf("generated Staff definition row %d is not mage-only: %v", candidate.Row, definition)
	}
	if candidate.Code != 0x812d || candidate.Price != 333 || powerMax != 19 ||
		releaseForcedCastBudget(candidate, ceiling) != 33_300 {
		t.Fatalf("installed Staff witness = code %#04x base %d budget %d powerMax %d, want 0x812d/333/33300/19",
			candidate.Code, candidate.Price, releaseForcedCastBudget(candidate, ceiling), powerMax)
	}

	draws := &fixedDraws{values: []int{1, 14}}
	generated, ok := shopEnchantedItem(candidate, ceiling, f.Table, draws)
	if !ok {
		t.Fatal("production shop generator refused the selected forced-cast Staff")
	}
	const power int32 = 15
	wantEffect := sim.ItemEffect{Kind: 41,
		Operand: uint32(spellID) | uint32(uint16(int16(power)))<<16}
	wantPrice := candidate.Price + releaseCastPrice(scalar, power)
	if castPrice := releaseCastPrice(scalar, power); castPrice != 23_357 || wantPrice != 23_690 {
		t.Fatalf("Lightning-15 price = cast %d final %d, want 23357/23690", castPrice, wantPrice)
	}
	if generated.Code != candidate.Code || generated.Kind != 2 || generated.Count != 1 ||
		generated.Price != wantPrice || !reflect.DeepEqual(generated.Effects, []sim.ItemEffect{wantEffect}) {
		t.Fatalf("generated Staff = %+v, want code %#04x kind 2 count 1 effect %+v price %d",
			generated, candidate.Code, wantEffect, wantPrice)
	}
	if !reflect.DeepEqual(draws.calls, []int{5, powerMax}) || len(draws.values) != 0 {
		t.Fatalf("forced Staff draw bounds = %v remaining=%v, want [5 %d] and no remaining draws",
			draws.calls, draws.values, powerMax)
	}
	spell, gotPower, cast := generated.Instance().CastSpell()
	if !cast || spell != spellID || gotPower != power || !generated.Instance().HasEnchantment() {
		t.Fatalf("generated Staff cast identity = (%d,%d,%v), enchanted=%v",
			spell, gotPower, cast, generated.Instance().HasEnchantment())
	}

	plain := shopItemFromInstance(mapload.ItemInstanceFromCode(uint16(candidate.Code), f.Table), 1)
	if plain.Price != candidate.Price || plain.Instance().HasEnchantment() {
		t.Fatalf("plain same-code control = %+v, want base price %d and no enchantment", plain, candidate.Price)
	}
	samePricePlain := plain.Clone()
	samePricePlain.Price = generated.Price
	if folded := shopStackShelf([]ShopItem{generated, samePricePlain}); len(folded) != 2 {
		t.Fatalf("enchanted and plain same-code controls folded together: %+v", folded)
	}

	shop := NewShop(ceiling)
	shop.tbl = f.Table
	shop.shelves[ShelfMagic] = []ShopItem{generated.Clone()}
	screen := f.bindTown(&townScreen{})
	shelfCell := screen.shopShelfCell(shop.Shelf(ShelfMagic)[0], wantPrice)
	valueLine := fmt.Sprintf("%s %d", f.Words.ItemStats[1], wantPrice)
	castSpell := f.Table.Spells.EntryName(int(spellID))
	if name := f.Words.ItemSpellNames[spellID]; name != "" {
		castSpell = name
	}
	castLine := fmt.Sprintf("%s %s", f.Words.ItemCasts, castSpell)
	spellRules := mapload.SpellRules(f.Table)
	spellBase, spellSpread, spellDamage := sim.WeaponSpellDamageFor(sim.Rules{}, sim.Entity{
		MaxMana: 1, WeaponSpell: spellID, WeaponSpellLevel: power,
	}, spellRules)
	if !spellDamage {
		t.Fatal("generated Staff's attached cast has no damage interval")
	}
	var castRule sim.SpellRule
	castRuleFound := false
	for _, rule := range spellRules {
		if rule.ID == spellID {
			castRule, castRuleFound = rule, true
			break
		}
	}
	if !castRuleFound {
		t.Fatalf("generated Staff's spell %d has no rule", spellID)
	}
	castDamageLine := fmt.Sprintf("%s %d-%d", f.Words.ItemDamage, spellBase, spellBase+spellSpread)
	castRangeLine := fmt.Sprintf("%s %d", f.Words.ItemRange, castRule.MaxRange)
	rawCastLine := fmt.Sprintf("#%s %s power %d", f.Words.ItemStats[41],
		f.Table.Spells.EntryName(int(spellID)), power)
	magicLine := roomCaptionRawLine(t, roomCaptionRaw(t, f, "main/text/main.txt"), 189)
	if shelfCell.Price != wantPrice || containsString(shelfCell.Info, valueLine) ||
		!containsString(shelfCell.Info, magicLine) || !containsString(shelfCell.Info, castLine) ||
		!containsString(shelfCell.Info, castDamageLine) || !containsString(shelfCell.Info, castRangeLine) ||
		containsString(shelfCell.Info, rawCastLine) {
		t.Fatalf("shelf cell = price %d info %v, want the grid price %d, no %q line and %q/%q/%q once",
			shelfCell.Price, shelfCell.Info, wantPrice, valueLine, castLine, castDamageLine, castRangeLine)
	}
	if !shop.TakeFromShelf(ShelfMagic, 0, 1) {
		t.Fatal("merchant table refused generated Staff")
	}
	table := shop.Table()
	if len(table) != 1 || table[0].Mine || !shopItemStateEqual(table[0].ShopItem, generated) ||
		shop.BuyTotal() != wantPrice {
		t.Fatalf("merchant table = %+v total %d, want exact generated Staff at %d", table, shop.BuyTotal(), wantPrice)
	}
	bought, spent, ok := shop.Buy(wantPrice)
	if !ok || spent != wantPrice || len(bought) != 1 || !shopItemStateEqual(bought[0], generated) {
		t.Fatalf("buy = items %+v spent %d ok=%v, want exact generated Staff at %d", bought, spent, ok, wantPrice)
	}

	caster := sim.Entity{ID: 1, X: 2, Y: 2, Owner: 1, HP: 100, MaxHP: 100,
		Mana: 100, MaxMana: 100, Mind: 100, Reach: 1, ScanRange: 8,
		AttackCharge: 1, AttackRelax: 1}
	target := sim.Entity{ID: 2, X: 3, Y: 2, Owner: 2, HP: 10000, MaxHP: 10000, DyingTime: 200}
	var relations sim.Relations
	relations.Set(1, 2, 1)
	w, err := sim.NewStockedSpelledWorld(1040, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{caster, target}, nil, relations, nil,
		[]sim.Stock{{ID: caster.ID, ItemInstances: []sim.ItemInstance{bought[0].Instance()}}}, mapload.SpellRules(f.Table))
	if err != nil {
		t.Fatalf("build generated Staff world: %v", err)
	}
	pack, _ := w.CarriedStacks(caster.ID)
	if len(pack) != 1 || pack[0].Count != 1 || !reflect.DeepEqual(pack[0].Instance(), generated.Instance()) {
		t.Fatalf("purchased Staff pack = %+v, want exact generated instance", pack)
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindEquip, Entity: caster.ID, X: 0, Y: 1}})
	worn, _ := w.EquippedItems(caster.ID)
	equipped := releaseEntityByID(t, w, caster.ID)
	if !reflect.DeepEqual(worn[0], generated.Instance()) || equipped.WeaponSpellSource != sim.WeaponSpellItem ||
		equipped.WeaponSpell != spellID || equipped.WeaponSpellLevel != power {
		t.Fatalf("equipped generated Staff/source = %+v / %+v", worn[0], equipped)
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal generated Staff world: %v", err)
	}
	if len(form) == 0 {
		t.Fatal("generated Staff form is empty")
	}
	var restored sim.World
	if err := restored.UnmarshalBinary(form); err != nil {
		t.Fatalf("unmarshal generated Staff world: %v", err)
	}
	restoredWorn, _ := restored.EquippedItems(caster.ID)
	restoredCaster := releaseEntityByID(t, &restored, caster.ID)
	if !reflect.DeepEqual(restoredWorn[0], generated.Instance()) ||
		restoredCaster.WeaponSpellSource != sim.WeaponSpellItem || restoredCaster.WeaponSpell != spellID ||
		restoredCaster.WeaponSpellLevel != power {
		t.Fatalf("restored generated Staff/source = %+v / %+v", restoredWorn[0], restoredCaster)
	}
	var event sim.CastEvent
	for tick := 0; tick < 512 && event.Spell == 0; tick++ {
		var commands []sim.Command
		if tick == 0 {
			commands = []sim.Command{{Kind: sim.KindAttack, Entity: caster.ID, X: int32(target.ID)}}
		}
		for _, possible := range sim.StepObserved(&restored, commands) {
			if possible.Weapon {
				event = possible
				break
			}
		}
	}
	if !event.Weapon || event.Spell != spellID || event.Caster != caster.ID || event.Target != target.ID {
		t.Fatalf("restored generated Staff cast event = %+v, want item weapon spell %d from %d to %d",
			event, spellID, caster.ID, target.ID)
	}

	if !shop.PutOnTable(bought[0]) {
		t.Fatal("player table refused purchased Staff")
	}
	playerPlace := shop.Table()[0]
	placeCell := screen.shopPlaceCell(playerPlace)
	wantSell := shopHalfUp(wantPrice)
	if wantSell != 11_845 {
		t.Fatalf("Staff sell price = %d, want 11845", wantSell)
	}
	if !playerPlace.Mine || shop.SellTotal() != wantSell || shop.SellPayout() != wantSell ||
		placeCell.Price != wantSell || containsString(placeCell.Info, valueLine) ||
		!containsString(placeCell.Info, castLine) {
		t.Fatalf("player table = %+v cell price/info=%d/%v totals=%d/%d, want sell %d, no %q line and stored price %d",
			playerPlace, placeCell.Price, placeCell.Info, shop.SellTotal(), shop.SellPayout(), wantSell, valueLine, wantPrice)
	}
	paid, ok := shop.Sell()
	if !ok || paid != wantSell || len(shop.Table()) != 0 {
		t.Fatalf("sell = paid %d ok=%v table %+v, want %d and an empty table", paid, ok, shop.Table(), wantSell)
	}
	for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
		items := shop.Shelf(shelf)
		if shelf == ShelfMagic {
			if len(items) != 1 || items[0].Count != 1 || !shopItemStateEqual(items[0], generated) {
				t.Fatalf("sold Magic shelf %+v, want one exact generated Staff", items)
			}
		} else if len(items) != 0 {
			t.Fatalf("sold %s shelf %+v, want empty", shelf, items)
		}
	}
	if !shop.TakeFromShelf(ShelfMagic, 0, 1) || shop.BuyTotal() != wantPrice {
		t.Fatal("sold generated Staff could not be repurchased at its exact price")
	}
	repurchased, repaid, ok := shop.Buy(wantPrice)
	if !ok || repaid != wantPrice || len(repurchased) != 1 || repurchased[0].Count != 1 || !shopItemStateEqual(repurchased[0], generated) {
		t.Fatalf("repurchase = items %+v paid %d ok=%v, want exact generated Staff at %d", repurchased, repaid, ok, wantPrice)
	}
	for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
		if len(shop.Shelf(shelf)) != 0 {
			t.Fatalf("repurchase left items on %s", shelf)
		}
	}

	t.Logf("%s %#04x: base=%d budget=%d Lightning power=%d/%d cast=%d final=%d sell=%d; exact instance survived shelf/table/buy/pack/equip/form/cast/sell",
		f.Table.Weapons.EntryName(candidate.Row), candidate.Code, candidate.Price,
		releaseForcedCastBudget(candidate, ceiling), power, powerMax, releaseCastPrice(scalar, power), wantPrice, wantSell)
}

func releaseForcedCastBudget(candidate data.ShopCandidate, ceiling int32) int64 {
	budget := int64(2)*int64(ceiling) - int64(candidate.Price)
	if capBudget := int64(candidate.Price) * 100; capBudget < budget {
		budget = capBudget
	}
	return budget
}

func releaseCastPowerMax(budget int64, scalar int32) int {
	ratio := float64(budget) / float64(10*int64(scalar))
	if ratio <= 1 {
		return 0
	}
	maximum := int32(30 * (math.Pow(1.2, math.Log2(ratio)) - 1))
	if maximum > 100 {
		maximum = 100
	}
	return int(maximum)
}

func releaseCastPrice(scalar, power int32) int32 {
	return int32(10 * float64(scalar) * math.Pow(2, math.Log(float64(power)/30+1)/math.Log(1.2)))
}

// TestReleaseGame0002PotionRetainsItsSavedEffect imports the exact Potion
// whose ROM1 stackability rule would otherwise erase its item state. The test
// runs independently against the EN and RU copies selected by the release gate.
func TestReleaseGame0002PotionRetainsItsSavedEffect(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: game0002.sav item witness needs a lawful install")
	}
	f := releaseFront(t)
	raw, err := os.ReadFile(filepath.Join(root, "game0002.sav"))
	if err != nil {
		t.Fatalf("read game0002.sav: %v", err)
	}
	sf, err := sav.Open(raw)
	if err != nil {
		t.Fatalf("open game0002.sav: %v", err)
	}
	characters, err := sf.Party()
	if err != nil {
		t.Fatalf("walk game0002.sav party: %v", err)
	}
	wantEffect := sim.ItemEffect{Kind: 8, Mode: 1, Operand: 0x03c00064}
	var imported []sim.ItemInstance
	witnessName := ""
	for _, character := range characters {
		for _, piece := range character.Items {
			if piece.Code != 0x0e06 || piece.Stack != 3 {
				continue
			}
			report := &RestoredParty{}
			_, carried := restoredItemLoadout(character, report)
			if report.UnsupportedItemEffects != 0 {
				t.Fatalf("Potion character reports %d unsupported effect states", report.UnsupportedItemEffects)
			}
			for _, item := range carried {
				if item.Code == piece.Code && item.Kind == 3 && len(item.Effects) == 1 && item.Effects[0] == wantEffect {
					imported = append(imported, item)
				}
			}
			if character.Class == "Human" && int(character.DefRow) < f.Table.Humans.Len() {
				witnessName = f.Table.Humans.EntryName(int(character.DefRow))
			}
		}
	}
	if len(imported) != 3 || !strings.Contains(witnessName, "Witch") {
		t.Fatalf("game0002 Potion witness = %d imported items on %q, want three on the Witch row", len(imported), witnessName)
	}
	w, err := sim.NewStockedWorld(1040, sim.Bounds{Width: 4, Height: 4}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{{ID: 1, X: 1, Y: 1}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 1, ItemInstances: imported}})
	if err != nil {
		t.Fatalf("stack imported Potion: %v", err)
	}
	stacks, _ := w.CarriedStacks(1)
	if len(stacks) != 1 || stacks[0].Count != 3 || !reflect.DeepEqual(stacks[0].Instance(), imported[0]) {
		t.Fatalf("imported Potion stack = %+v, want one complete three-count instance", stacks)
	}
	t.Logf("game0002.sav Witch: Potion %#04x count=%d kind=%d effect=%+v price=%d; unsupported states=0",
		stacks[0].Code, stacks[0].Count, stacks[0].Kind, stacks[0].Effects[0], stacks[0].Price)
}
