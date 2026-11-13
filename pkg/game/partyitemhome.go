package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
)

// partyItemHome is the one home of what a campaign party carries between
// missions. Each member's pack lives in the session's party; the campaign's
// bag, where the game's profile has one, lives in the second campaign's
// record and saves unchanged there. Every unit conversion and pack
// replacement of a crossing happens in this file (R2-ENGINE-402,
// R2-ENGINE-404, DIV-4191).
type partyItemHome struct {
	sess    *CampaignSession
	table   *mapload.Table
	profile partyItemProfile
}

// partyItemProfile is what the games differ in about those items, read by
// every operation. bagUnits bounds the campaign's shared bag, its unit
// counting the hero's bag and his table (DIV-3043); zero is a game with no
// bag, whose members' packs are the whole home and where nothing crosses.
type partyItemProfile struct {
	bagUnits  int
	shopGraph bool
	// departs is a party that leaves its town for a map of places before a
	// mission, the bag joining the hero's pack: the departure keeps the
	// state at the mission's entry for a return without completion. False
	// is a game whose missions open from a town that stands, moving nothing.
	departs bool
	// kept is a pack unit in its bag form, false when the bag would lose it.
	kept func(sim.ItemInstance, func(uint16) sim.ItemInstance) (sim.ItemInstance, bool)
	// mission is a bag unit in the form a mission holds it.
	mission func(sim.ItemInstance, func(uint16) sim.ItemInstance) sim.ItemInstance
	// shopUnit is an item in the form the shop gives its bag.
	shopUnit func(sim.ItemInstance) sim.ItemInstance
	// bagForm reports a loaded file's piece the bag keeps.
	bagForm func(sim.ItemInstance) bool
}

var (
	// firstPartyItems is the first game: a pack per member and no bag.
	firstPartyItems = partyItemProfile{shopGraph: true}
	// secondPartyItems is the second game's shared bag, the units' forms
	// that of its shop and mission tables, bounded by the save's bound.
	secondPartyItems = partyItemProfile{bagUnits: maxSecondBagUnits, departs: true, kept: secondshop.KeptUnit,
		mission: secondshop.MissionUnit, shopUnit: secondshop.Unit, bagForm: secondBagUnitForm}
)

// partyItems is the session's home: the second game's row where the session
// holds the second campaign's record, the only store of a bag, else the row
// of a game with none. It is the one place a row is picked from a session.
func (s *CampaignSession) partyItems(table *mapload.Table) partyItemHome {
	if secondOf(s) != nil {
		return secondPartyItems.over(s, table)
	}
	return firstPartyItems.over(s, table)
}

// over is profile p's home over session s.
func (p partyItemProfile) over(s *CampaignSession, table *mapload.Table) partyItemHome {
	return partyItemHome{sess: s, table: table, profile: p}
}

// departure is the session's record of the party's departure from its town,
// which keeps the state at a mission's entry; nil where the row's game does
// not depart.
func (h partyItemHome) departure() *secondCampaign {
	if !h.profile.departs {
		return nil
	}
	return secondOf(h.sess)
}

// departedTown reports whether the departure record holds the town the
// running mission was entered from.
func (h partyItemHome) departedTown() bool {
	c := h.departure()
	return c != nil && c.entryTown.kind == placeTown
}

// keepEntry keeps the party and the bag as they stand at a departure from
// town, before leaveLocation joins the bag to the hero's pack.
func (h partyItemHome) keepEntry() {
	if c := h.departure(); c != nil {
		c.entryParty, c.entryBag = mapload.CloneParty(h.sess.Carried), mapload.CloneParty(c.party)
	}
}

// dropEntry forgets the kept entry, as when the session adopts another party.
func (h partyItemHome) dropEntry() {
	if c := h.departure(); c != nil {
		c.entryParty, c.entryBag = nil, nil
	}
}

// returnEntry brings the party and the bag home as they stood at the
// mission's entry: the copies the departure kept, member by member in place
// so a companion the departure appended stays; without copies, as for a
// mission loaded from a SAV or a game that does not depart, the arrival walk.
func (h partyItemHome) returnEntry() error {
	c := h.departure()
	if c == nil || c.entryParty == nil {
		return h.arriveTown()
	}
	copy(h.sess.Carried, mapload.CloneParty(c.entryParty))
	c.party = mapload.CloneParty(c.entryBag)
	return nil
}

// secondMissionItem is the engine's one item decoder over table, the base
// object a mission builds for an item ID.
func secondMissionItem(table *mapload.Table) func(uint16) sim.ItemInstance {
	return func(code uint16) sim.ItemInstance { return mapload.ItemInstanceFromCode(code, table) }
}

// keptForm is a pack unit in its bag form, false when the bag would lose it.
func (p partyItemProfile) keptForm(it sim.ItemInstance, table *mapload.Table) (sim.ItemInstance, bool) {
	return p.kept(it, secondMissionItem(table))
}

// missionForm is a bag unit in the form a mission holds it.
func (p partyItemProfile) missionForm(u sim.ItemInstance, table *mapload.Table) sim.ItemInstance {
	return p.mission(u, secondMissionItem(table))
}

// leaveLocation follows a departure from a location: the bag's units join
// the end of the campaign hero's pack, each in its mission form, so a bought
// potion can be drunk there; the bag stays when the hero's carried load
// refuses them, and a game with no bag crosses nothing.
func (h partyItemHome) leaveLocation() {
	c := secondOf(h.sess)
	if h.profile.bagUnits == 0 || c == nil {
		return
	}
	s := h.sess
	i := secondHeroIndex(s.Carried)
	if i < 0 || len(c.party) == 0 || len(c.party[0].CarriedItems) == 0 {
		return
	}
	pack := mapload.MemberCarriedItems(s.Carried[i], h.table)
	for _, u := range c.party[0].CarriedItems {
		pack = append(pack, h.profile.missionForm(u, h.table))
	}
	if hero, ok := memberWithPack(s.Carried[i], pack, h.table, true); ok {
		s.Carried[i] = hero
		c.party[0].CarriedItems = nil
	}
}

// arriveTown follows the party's arrival at a town: each hero's pack unit
// the bag takes home without a loss becomes the bag again, in pack order,
// while the bag holds fewer units than its bound; the others, such as a unit
// at a negative price, stay in his pack (DIV-4191). Only the hero's pack
// comes home; a companion keeps his. When the town's item graph cannot follow
// the pack, nothing moves and the error says why.
func (h partyItemHome) arriveTown() error {
	c := secondOf(h.sess)
	if h.profile.bagUnits == 0 || c == nil {
		return nil
	}
	s := h.sess
	i := secondHeroIndex(s.Carried)
	if i < 0 {
		return nil
	}
	var bag, kept []sim.ItemInstance
	if len(c.party) != 0 {
		bag = slices.Clone(c.party[0].CarriedItems)
	}
	held := len(bag)
	for _, it := range mapload.MemberCarriedItems(s.Carried[i], h.table) {
		if unit, ok := h.profile.keptForm(it, h.table); ok && len(bag) < h.profile.bagUnits {
			bag = append(bag, unit)
		} else {
			kept = append(kept, it)
		}
	}
	if len(bag) == held {
		return nil
	}
	hero, ok := memberWithPack(s.Carried[i], kept, h.table, true)
	if !ok {
		return nil
	}
	if err := h.commitPack(i, hero); err != nil {
		return fmt.Errorf("the bag cannot come home: %w", err)
	}
	*c.heroBag() = bag
	return nil
}

// takeLoadedPack empties the hero's pack of a loaded file's party: a town
// LOAD reads the second game's hero bag into the campaign's bag (DIV-4153).
// A hero's unit with no bag form stays in his pack, as at a town's arrival
// (DIV-4191). A companion keeps his own pack, the file's Human Inventory, as
// he keeps it at a town's arrival.
func (p partyItemProfile) takeLoadedPack(party []mapload.PartyMember, table *mapload.Table) error {
	hero := secondHeroIndex(party)
	if p.bagUnits == 0 || hero < 0 {
		return nil
	}
	m := &party[hero]
	var kept []sim.ItemInstance
	for _, it := range mapload.MemberCarriedItems(*m, table) {
		if !p.bagForm(it) {
			kept = append(kept, it)
		}
	}
	if len(kept) != 0 {
		next, ok := memberWithPack(*m, kept, table, true)
		if !ok {
			return fmt.Errorf("second-game save: the hero's pack cannot keep the %d units the bag has no form for", len(kept))
		}
		*m = next
		return nil
	}
	m.Carried, m.CarriedItems = nil, nil
	if m.Carry != nil {
		m.Carry.Items, m.Carry.ItemInstances, m.Carry.OrderedStacks = nil, nil, nil
	}
	return nil
}

// shopView is the units of member i's own pack the shop shows, each in its
// bag form: the units the bag keeps without a loss, in pack order.
func (h partyItemHome) shopView(i int) []sim.ItemInstance {
	if i < 0 || i >= len(h.sess.Carried) {
		return nil
	}
	var out []sim.ItemInstance
	for _, it := range mapload.MemberCarriedItems(h.sess.Carried[i], h.table) {
		if unit, ok := h.profile.keptForm(it, h.table); ok {
			out = append(out, unit)
		}
	}
	return out
}

// writeShopView makes items the units member i's pack shows: a pack unit
// still among them stays as it is, a new one enters in its mission form, and
// the units the shop does not show stay in his pack. The town's item graph
// follows the pack. It is false when the member's carried load refuses the
// pack.
func (h partyItemHome) writeShopView(i int, items []sim.ItemInstance, refresh bool) bool {
	s := h.sess
	if i < 0 || i >= len(s.Carried) {
		return false
	}
	var out, shown, held []sim.ItemInstance
	for _, it := range mapload.MemberCarriedItems(s.Carried[i], h.table) {
		if unit, ok := h.profile.keptForm(it, h.table); ok {
			shown, held = append(shown, unit), append(held, it)
		} else {
			out = append(out, it)
		}
	}
	for _, it := range items {
		at := slices.IndexFunc(shown, func(u sim.ItemInstance) bool { return sim.SameUnit(u, it, sim.UnitForm) })
		if at < 0 {
			out = append(out, h.profile.missionForm(h.profile.shopUnit(it), h.table))
			continue
		}
		out = append(out, held[at])
		shown, held = slices.Delete(shown, at, at+1), slices.Delete(held, at, at+1)
	}
	member, ok := memberWithPack(s.Carried[i], out, h.table, refresh)
	if !ok {
		return false
	}
	return h.commitPack(i, member) == nil
}

// writePack makes items, as they are, member i's pack in party, the write of
// a shop whose goods the pack holds in their own form. It is false when the
// member's carried load refuses the pack.
func (h partyItemHome) writePack(party []mapload.PartyMember, i int, items []sim.ItemInstance, refresh bool) bool {
	next, ok := memberWithPack(party[i], items, h.table, refresh)
	if !ok {
		return false
	}
	party[i] = next
	return true
}

func (h partyItemHome) writeShopStacks(index int, stacks []sim.ItemStack, refresh bool) error {
	before := mapload.CloneParty([]mapload.PartyMember{h.sess.Carried[index]})[0]
	next := mapload.CloneParty([]mapload.PartyMember{before})[0]
	next = mapload.MaterializePartyCarry(next, h.table)
	next.RetireOriginalHuman()
	if err := mapload.UpdatePartyLoadOrdered(before, &next, h.table, refresh, false, stacks); err != nil {
		return err
	}
	return h.commitPack(index, next)
}

// commitPack makes next member i, the town's item graph following his pack,
// the one point where a crossing's pack replaces the party's. When the graph
// cannot follow, nothing changes.
func (h partyItemHome) commitPack(i int, next mapload.PartyMember) error {
	s := h.sess
	var graph *cityObjectTopology
	if !h.profile.shopGraph {
		party := slices.Clone(s.Carried)
		party[i] = next
		var err error
		graph, err = s.followHeroPack(s.Carried[i], party, i, h.table)
		if err != nil {
			return err
		}
	}
	s.Carried[i] = next
	if graph != nil {
		s.Town.cityObjects = graph
	}
	if h.profile.shopGraph && s.originalCity != nil {
		s.originalCity.trade = nil
	}
	return nil
}

// followHeroPack builds the town's item graph for party, where member i's
// pack lost units outside the shop (before is the member as he was): a stack
// that stayed keeps its pack edge, a stack that left takes its edge with it,
// and an Item no holder reaches retires with its record, as a shop sale
// retires it. Positional edges would write a kept unit under a departed
// unit's record in the town SAV. A town without a graph returns none.
func (s *CampaignSession) followHeroPack(before mapload.PartyMember, party []mapload.PartyMember, i int, table *mapload.Table) (*cityObjectTopology, error) {
	if s.Town == nil || s.Town.cityObjects == nil {
		return nil, nil
	}
	g := s.Town.cityObjects.Clone()
	for k := range g.Roots {
		root := &g.Roots[k]
		if string(root.PartyID) != before.ID {
			continue
		}
		old, at := cityMemberStacks(before, table), 0
		var pack []sim.SavedObjectID
		for _, stack := range cityMemberStacks(party[i], table) {
			for at < len(old) && !sim.SameUnit(old[at], stack, sim.UnitStack) {
				at++
			}
			id := sim.SavedObjectID(0)
			if at < len(old) && at < len(root.Pack) {
				id = root.Pack[at]
			}
			pack = append(pack, id)
			at++
		}
		root.Pack = pack
	}
	p, err := newCityObjectProjection(g, party, table)
	if err != nil {
		return nil, err
	}
	p.retireUnreached()
	if err := p.graph.Validate(); err != nil {
		return nil, err
	}
	return p.graph, nil
}

// memberWithPack is member holding items as his pack: in the carried form
// when a mission wrote one, whose load follows, else in the town form. It is
// false when the carried load refuses the pack.
func memberWithPack(member mapload.PartyMember, items []sim.ItemInstance, table *mapload.Table, refresh bool) (mapload.PartyMember, bool) {
	before := mapload.CloneParty([]mapload.PartyMember{member})[0]
	next := mapload.CloneParty([]mapload.PartyMember{before})[0]
	next.RetireOriginalHuman()
	cloned := make([]sim.ItemInstance, len(items))
	codes := make([]uint16, len(items))
	for k := range items {
		cloned[k] = items[k].Clone()
		codes[k] = items[k].Code
	}
	if next.Carry != nil {
		next.Carry.ItemInstances = cloned
		next.Carry.Items = codes
		if err := mapload.UpdatePartyLoad(before, &next, table, refresh, false); err != nil {
			return mapload.PartyMember{}, false
		}
		// The town views hold the carried form's pack, as the carry
		// writes them, so a SAVE and its LOAD read the same member.
		mapload.SetCarriedFromCarry(&next)
	} else {
		next.CarriedItems = cloned
		next.Carried = codes
	}
	// A worn layer is a unit the pack backs: a pack that no longer holds it
	// takes it off, so buying the item back does not wear it again.
	next.Layers = mapload.MemberLayers(next, table)
	return next, true
}

// savedBagView is the campaign's bag as a town SAVE reads it from the saved
// party: one member holding every unit, nil when none is held.
func savedBagView(party []currentPartyMember) mapload.PartyMember {
	bag := mapload.PartyMember{ID: "bag"}
	for _, m := range party {
		if m.Member != nil {
			bag.CarriedItems = append(bag.CarriedItems, m.Member.CarriedItems...)
		}
	}
	return bag
}

// secondHeroBag is the hero's bag of a file's characters, as the second
// campaign's bag units: one unit per stacked piece, each the bag's item
// form.
func secondHeroBag(chars []sav.Character) ([]sim.ItemInstance, error) {
	var out []sim.ItemInstance
	for _, c := range chars {
		if !c.Hero {
			continue
		}
		for _, p := range c.Items {
			unit := sim.ItemInstance{Code: p.Code, Kind: p.Kind, Price: p.Price}
			for _, e := range p.Effects {
				unit.Effects = append(unit.Effects, sim.ItemEffect{Kind: e.Kind, Mode: e.Mode, Operand: e.Operand})
			}
			// A piece with no bag form stays in the hero's pack
			// (takeLoadedPack, DIV-4191).
			if !secondBagUnitForm(unit) {
				continue
			}
			n := max(int(p.Stack), 1)
			if len(out)+n > maxSecondBagUnits {
				return nil, fmt.Errorf("second-game save: bag holds more than %d units", maxSecondBagUnits)
			}
			for range n {
				out = append(out, unit.Clone())
			}
		}
	}
	return out, nil
}

// secondBagUnitForm reports a file's hero piece the bag keeps: its ID,
// type, price and effects are a bag unit's. A piece at a negative price,
// such as magic item 0x0e4d at the original's 0xffffffff, has none.
func secondBagUnitForm(in sim.ItemInstance) bool {
	_, ok := secondshop.FromInstance(sim.ItemInstance{Code: in.Code, Kind: in.Kind, Price: in.Price, Effects: in.Effects})
	return ok
}
