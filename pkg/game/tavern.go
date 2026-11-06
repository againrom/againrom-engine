package game

import (
	"cmp"
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
	var out []TavernOffer
	for _, o := range innOffers(nil, innState{hire: &tavernHires, home: t.sess.Town, table: t.in.Table}) {
		out = append(out, TavernOffer{Type: o.npc, Count: o.count, Capacity: o.capacity, Price: o.price,
			Hired: o.high, Affordable: o.affordable})
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
			t.raiseRoomEvent(roomTavern, roomEventDismiss)
		} else {
			t.raiseRoomEvent(roomTavern, roomEventHire)
		}
		return
	}
	if !hired {
		if price, priced := t.mercenaryPrice(typ, t.sess.Town.MercenaryPool(typ)); priced && price > t.sess.Town.Gold() {
			t.requestTavernSlotSound(tavernSlotHireRefused)
			t.raiseRoomEvent(roomTavern, roomEventHireShort)
		}
	}
}

func (t *townScreen) toggleMercenary(typ int) (string, bool) {
	if t == nil || t.sess == nil || t.sess.Town == nil {
		return "tavern unavailable", false
	}
	tw := t.sess.Town
	if tw.MercenaryHired(typ) {
		price, ok := t.mercenaryPrice(typ, tw.MercenaryPool(typ))
		if !ok {
			return "the squad cannot be returned", false
		}
		d := tw.squadDeal(typ, price, true)
		d.party = &t.sess.Carried
		_, ok, returned := d.run()
		if !ok {
			return "the squad cannot be returned", false
		}
		tw.refundCityGroups(t.sess.Carried, returned)
		t.clampTownMember()
		t.composeShopFaces()
		return "", true
	}

	count := tw.MercenaryPool(typ)
	price, ok := t.mercenaryPrice(typ, count)
	if !ok {
		return "mercenary data unavailable", false
	}
	members, ok := t.tavernSquad(typ, count)
	if !ok {
		return "mercenary template unavailable", false
	}
	d := tw.squadDeal(typ, price, false)
	d.party, d.members = &t.sess.Carried, members
	if _, ok, _ := d.run(); !ok {
		return "the squad cannot be hired", false
	}
	tw.settleCityGroups(t.sess.Carried)
	t.composeShopFaces()
	return "", true
}

// squadDeal is the tavern's deal for type typ's whole squad at price: a gold
// cost from the town's purse and the type's hired flag, refused while
// another town holds the type or the pool is empty. fire dismisses it.
func (t *Town) squadDeal(typ, price int, fire bool) hireDeal {
	row, ok := tavernHires.row(typ)
	d := hireDeal{fire: fire, cost: tavernHires.cost, price: price, row: row}
	if t == nil || !ok || typ >= len(t.mercPool) {
		return d
	}
	d.purse, d.held = &t.gold, &t.mercHired[typ]
	if t.mercenaryHiredElsewhere(typ) || t.mercPool[typ] <= 0 {
		d.refusal = "the squad cannot be hired"
	}
	return d
}

// mercenaryLevel is the squad level at mission, by the catalog's mercenary
// rule.
func mercenaryLevel(mission int) int { return tavernRule.Level(mission) }

// tavernSquad is count members of tavern type typ through the hire
// producer, at the chapter the town is in.
func (t *townScreen) tavernSquad(typ, count int) ([]mapload.PartyMember, bool) {
	if t == nil || t.sess == nil {
		return nil, false
	}
	return hireMembers(&tavernHires, hireState{town: t.sess.Town, table: t.in.Table}, typ, count)
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
	var member mapload.PartyMember
	members, ok := hireMembers(&tavernHires, hireState{table: &mapload.Table{Units: c}}, typ, 1)
	if ok {
		member = members[0]
	}
	return member, ok
}
