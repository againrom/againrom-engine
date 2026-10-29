package game

import (
	"cmp"
	"fmt"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/town"
)

// TavernOffer is one whole-squad cell on the town screen.
type TavernOffer struct {
	Type       int
	Count      int
	Capacity   int
	Price      int
	Hired      bool
	Affordable bool
}

func (t *townScreen) tavernMercenaries() []TavernOffer {
	if t == nil || t.sess == nil || t.sess.Town == nil {
		return nil
	}
	listed := make(map[int]bool)
	for _, typ := range t.sess.Town.ChapterData().Mercenaries {
		listed[typ] = true
	}
	var out []TavernOffer
	for _, typ := range tavernStockWalk(t.sess.Town.tavernStockIDs) {
		if !listed[typ] || !t.sess.Town.MercenaryEnabled(typ) || t.sess.Town.mercenaryHiredElsewhere(typ) {
			continue
		}
		count := t.sess.Town.MercenaryPool(typ)
		hired := t.sess.Town.MercenaryHired(typ)
		if count <= 0 {
			continue
		}
		capacity := t.sess.Town.MercenaryCapacity(typ)
		if capacity < count {
			capacity = count
		}
		price, ok := t.mercenaryPrice(typ, count)
		if !ok {
			continue
		}
		out = append(out, TavernOffer{Type: typ, Count: count, Capacity: capacity, Price: price,
			Hired: hired, Affordable: hired || price <= t.sess.Town.Gold()})
	}
	return out
}

// tavernRule is the mercenary rule of the shipped catalog that states one.
// Its stock-id-phase is the stock unit id of type 1 when no LOAD gave the
// allocator's state: the original builds one stock unit per type 1..15 in
// type order at tavern entry and lists them in the walk of its actor map
// (TAVERN-ORDER-015). The phase is fitted to one original screenshot, where
// types 1..10 share one sixteen-id block and 11..15 the next (DIV-1410).
// Its stock-types is the number of stock units command 0x37 builds.
var tavernRule *town.MercenaryRule

// tavernStockWalk is the mercenary types 1..15 in the order the actor-map walk
// visits their stock units, given the ids of types 1..15: bucket
// `(id >> 4) % size` from 0, each bucket's chain from its head, and the client
// inserts at the head, so units of one sixteen-id block come out newest first.
// Without ids it takes the fitted phase. It assumes the blocks land in
// ascending buckets, which holds unless an id passes the table's last bucket.
func tavernStockWalk(ids []uint32) []int {
	types := tavernRule.StockTypes
	if len(ids) != types {
		ids = make([]uint32, types)
		for i := range ids {
			ids[i] = uint32(tavernRule.StockIDPhase + i)
		}
	}
	out := make([]int, 0, types)
	for typ := 1; typ <= types; typ++ {
		out = append(out, typ)
	}
	slices.SortStableFunc(out, func(a, b int) int {
		if c := cmp.Compare(ids[a-1]&0xffff>>4, ids[b-1]&0xffff>>4); c != 0 {
			return c
		}
		return cmp.Compare(b, a)
	})
	return out
}

// loadedTavernStockIDs is the ids command 0x37 gives the stock units of types
// 1..15 at the first tavern entry after a LOAD of saved. The bitmap is the one
// LOAD leaves (SAV-1372); the command first removes each party actor whose
// type byte is non-zero and frees its id (SAV-1136), and only a loaded Human
// mercenary of an NPC row keeps that byte (SAV-1137); the units then take the
// lowest clear bits in type order (SAV-926, SAV-927, MOVE-ID-016). Nil when
// the document does not decode.
func loadedTavernStockIDs(saved []byte, chars []sav.Character, table *mapload.Table) []uint32 {
	doc, origins := decodeSavedDocument(saved)
	if doc.Document == nil {
		return nil
	}
	var released []uint32
	for _, c := range chars {
		if _, ok := originalMercenaryType(c, table); ok && uint8(c.DisplayBacking) != 0 {
			released = append(released, c.RuntimeID)
		}
	}
	ids := sim.RuntimeIDsAfterLoad(runtimeIDLoads(doc.Document, origins), released, tavernRule.StockTypes)
	if len(ids) != tavernRule.StockTypes {
		return nil
	}
	return ids
}

func (t *townScreen) mercenaryPrice(typ, count int) (int, bool) {
	if t == nil || t.sess == nil || t.sess.Town == nil {
		return 0, false
	}
	return mercenarySquadPrice(t.in.Table, t.sess.Town.Chapter(), typ, count)
}

// mercenarySquadPrice is the one whole-squad price used by card presentation,
// affordability and the hire/return action. Keeping the mission argument here
// also lets the installed campaign census exercise every shipped row without
// mutating live town progress.
func mercenarySquadPrice(table *mapload.Table, mission, typ, count int) (int, bool) {
	if table == nil || table.NPC == nil || count <= 0 {
		return 0, false
	}
	terms, ok := table.NPC.Mercenary(int32(typ))
	if !ok {
		return 0, false
	}
	factor := mercenaryUnitPrice(mission)
	if factor <= 0 {
		return 0, false
	}
	return (int(terms.PriceA) + count*int(terms.PriceB)) * factor, true
}

func (t *townScreen) mercenaryPartyCount(typ int) int {
	n := 0
	for _, member := range t.sess.Carried {
		if int(member.MercenaryType) == typ {
			n++
		}
	}
	return n
}

// tavernSlotHireRefused is the tavern's sound slot offset requested when a
// hire is refused for a price above the money left (TAVERN-LINES-022). The
// slot's file is not identified by any claim.
const tavernSlotHireRefused = 0xa0

// requestTavernSlotSound records one request of a numbered tavern sound slot.
func (t *townScreen) requestTavernSlotSound(slot int) {
	if t.tavernSlotRequests == nil {
		t.tavernSlotRequests = make(map[int]int)
	}
	t.tavernSlotRequests[slot]++
}

// pressMercenary hires or returns a squad the way the tavern's upper button
// does. A hire refused for its price requests the refusal sound slot and posts
// no line; no other refusal posts a line either (TAVERN-LINES-022).
func (t *townScreen) pressMercenary(typ int) {
	if t == nil || t.sess == nil || t.sess.Town == nil {
		return
	}
	hired := t.sess.Town.MercenaryHired(typ)
	if _, ok := t.toggleMercenary(typ); ok {
		if hired {
			t.roomSound("tavern", "dismiss")
		} else {
			t.roomSound("tavern", "hire")
		}
		return
	}
	if !hired {
		if price, priced := t.mercenaryPrice(typ, t.sess.Town.MercenaryPool(typ)); priced && price > t.sess.Town.Gold() {
			t.requestTavernSlotSound(tavernSlotHireRefused)
			t.roomSound("tavern", "hire-short")
		}
	}
}

func (t *townScreen) toggleMercenary(typ int) (string, bool) {
	if t == nil || t.sess == nil || t.sess.Town == nil {
		return "tavern unavailable", false
	}
	if t.sess.Town.MercenaryHired(typ) {
		count := t.mercenaryPartyCount(typ)
		price, ok := t.mercenaryPrice(typ, t.sess.Town.MercenaryPool(typ))
		if !ok || !t.sess.Town.mercenaryReturn(typ, count, price) {
			return "the squad cannot be returned", false
		}
		var returned []string
		kept := t.sess.Carried[:0]
		for _, member := range t.sess.Carried {
			if int(member.MercenaryType) != typ {
				kept = append(kept, member)
			} else {
				returned = append(returned, member.ID)
			}
		}
		t.sess.Carried = kept
		t.sess.Town.refundCityGroups(t.sess.Carried, returned)
		t.clampTownMember()
		t.composeShopFaces()
		return "", true
	}

	count := t.sess.Town.MercenaryPool(typ)
	price, ok := t.mercenaryPrice(typ, count)
	if !ok {
		return "mercenary data unavailable", false
	}
	members, ok := t.buildMercenarySquad(typ, count)
	if !ok {
		return "mercenary template unavailable", false
	}
	if _, ok := t.sess.Town.mercenaryHire(typ, price); !ok {
		return "the squad cannot be hired", false
	}
	t.sess.Carried = mapload.OwnParty(append(t.sess.Carried, members...))
	t.sess.Town.settleCityGroups(t.sess.Carried)
	t.composeShopFaces()
	return "", true
}

// mercenaryLevel is the squad level at mission, by the catalog's mercenary
// rule.
func mercenaryLevel(mission int) int { return tavernRule.Level(mission) }

func (t *townScreen) buildMercenarySquad(typ, count int) ([]mapload.PartyMember, bool) {
	if t == nil || t.sess == nil {
		return nil, false
	}
	return buildMercenarySquad(t.in.Table, t.sess.Town, typ, count)
}

// buildMercenarySquad builds count members of tavern type typ from the
// install's table, at the chapter the town is in.
func buildMercenarySquad(table *mapload.Table, town *Town, typ, count int) ([]mapload.PartyMember, bool) {
	if table == nil || count <= 0 {
		return nil, false
	}
	if typ <= 2 {
		return buildSiegeSquad(table, typ, count)
	}
	name := fmt.Sprintf("NPC%02d_%d", typ, mercenaryLevel(town.Chapter()))
	i := data.FindHumanByName(table.Humans, name)
	if i == data.NotFound {
		return nil, false
	}
	def, err := data.NewHumanDef(name, table.Humans.EntryParams(i))
	if err != nil {
		return nil, false
	}
	dir, face := data.FigureFor(def.TypeID, def.Face, def.Gender)
	wornItems, carriedItems, weapon, err := mapload.HumanRowEquipment(table.Humans.EntryStrings(i), table)
	if err != nil {
		return nil, false
	}
	base := mapload.PartyMember{
		Name: name, Temporary: true, MercenaryType: uint8(typ),
		DefinitionRow:      uint8(i),
		SuppressCorpseLoot: mapload.SuppressesCorpseLoot(name),
		Mage:               dir.Mage(), Profile: def.Profile(),
		FigureDir: string(dir), FigureFace: face, Hero: def.Hero(), KnownSpells: def.KnownSpells,
		Weapon: weapon, WornItems: wornItems, CarriedItems: carriedItems,
		HiredRotationSpeed: def.RotationSpeed,
		Class:              def.TypeID,
	}
	joinedWornSpells(base.WornItems[:], table)
	for slot, item := range wornItems {
		base.Worn[slot] = item.Code
	}
	base.Carried = make([]uint16, len(carriedItems))
	for i, item := range carriedItems {
		base.Carried[i] = item.Code
	}
	out := make([]mapload.PartyMember, count)
	for i := range out {
		out[i] = base
	}
	return out, true
}

func buildSiegeSquad(table *mapload.Table, typ, count int) ([]mapload.PartyMember, bool) {
	base, ok := siegeHireMember(table.Units, typ)
	if !ok {
		return nil, false
	}
	out := make([]mapload.PartyMember, count)
	for i := range out {
		out[i] = base
	}
	return out, true
}

// siegeHireName is the Units row name of a hired siege engine (tavern type 1
// is the Catapult, type 2 the Ballista).
func siegeHireName(typ int) string {
	if typ == 2 {
		return "Ballista"
	}
	return "Catapult"
}

// siegeHireMember builds the party member of one hired siege engine from its
// Units row. A tavern hire and a restored saved Unit share it, so a loaded hire
// equals a fresh one.
func siegeHireMember(c data.Collection, typ int) (mapload.PartyMember, bool) {
	name := siegeHireName(typ)
	idx := data.NotFound
	if c != nil {
		for i := 1; i < c.Len(); i++ {
			if c.EntryName(i) == name {
				idx = i
				break
			}
		}
	}
	if idx == data.NotFound {
		return mapload.PartyMember{}, false
	}
	def, err := data.NewUnitDef(name, c.EntryParams(idx))
	if err != nil {
		return mapload.PartyMember{}, false
	}
	return mapload.PartyMember{
		Name: name, Temporary: true, MercenaryType: uint8(typ), Class: def.TypeID,
		FigureFace:         int(def.Face),
		Hero:               data.Hero{Body: def.Body, Reaction: def.Reaction, Mind: def.Mind, Spirit: def.Spirit},
		Profile:            data.Profile{HealthColumn: true, ManaColumn: def.ManaMax > 0},
		HiredRotationSpeed: def.RotationSpeed,
	}, true
}
