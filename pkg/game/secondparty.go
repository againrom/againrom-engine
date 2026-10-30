package game

import (
	"fmt"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
)

// The second game's shop party is one member, the hero; his bag is the
// member's pack of the engine's item instances (R2-ENGINE-402,
// R2-ENGINE-404). The shop deals with that pack in a town. Outside a town
// the bag's units are the campaign hero's pack, the one the party carry
// takes into a mission and brings back (DIV-4191).

// leftLocation follows a departure from a location: the bag joins the
// hero's pack, and a nonzero slot 773 asks the addition producer for that
// identity (R2-ENGINE-106, R2-ENGINE-379). Every ordinary departure zeroes
// the slot before its case (R2-SESSION-047), and the producer reuses a
// carried identity.
func (s *CampaignSession) leftLocation(in townInstall) {
	c := secondOf(s)
	if c == nil {
		return
	}
	s.packBag(c, in.table)
	if c.bank[773] != 0 {
		s.addPartyIdentity(in, int(c.bank[773]))
	}
}

// enteredTown follows the party's arrival at a town: each hero's pack unit
// secondshop.KeptUnit takes home without a loss becomes the bag again, in
// pack order, while the bag holds fewer than maxSecondBagUnits; the others,
// such as a unit at a negative price, stay in his pack (DIV-4191). Only the
// hero's pack comes home; a companion keeps his. When the town's item graph
// cannot follow the pack, nothing moves and the error says why.
func (s *CampaignSession) enteredTown(in townInstall) error {
	c := secondOf(s)
	i := secondHeroIndex(s.Carried)
	if c == nil || i < 0 {
		return nil
	}
	var bag, kept []sim.ItemInstance
	if len(c.party) != 0 {
		bag = slices.Clone(c.party[0].CarriedItems)
	}
	held := len(bag)
	for _, it := range mapload.MemberCarriedItems(s.Carried[i], in.table) {
		if unit, ok := secondshop.KeptUnit(it, secondMissionItem(in.table)); ok && len(bag) < maxSecondBagUnits {
			bag = append(bag, unit)
		} else {
			kept = append(kept, it)
		}
	}
	if len(bag) == held {
		return nil
	}
	hero, ok := memberWithPack(s.Carried[i], kept, in.table, true)
	if !ok {
		return nil
	}
	party := slices.Clone(s.Carried)
	party[i] = hero
	graph, err := s.followHeroPack(s.Carried[i], party, i, in.table)
	if err != nil {
		return fmt.Errorf("the bag cannot come home: %w", err)
	}
	s.Carried[i] = hero
	*c.heroBag() = bag
	if graph != nil {
		s.Town.cityObjects = graph
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
			for at < len(old) && !sim.StackStateEqual(old[at], stack) {
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

// packBag moves the bag's units to the end of the campaign hero's pack, each
// in its mission form (secondshop.MissionUnit), so a bought potion can be
// drunk there; the bag stays when the hero's carried load refuses them.
func (s *CampaignSession) packBag(c *secondCampaign, table *mapload.Table) {
	i := secondHeroIndex(s.Carried)
	if i < 0 || len(c.party) == 0 || len(c.party[0].CarriedItems) == 0 {
		return
	}
	pack := mapload.MemberCarriedItems(s.Carried[i], table)
	for _, u := range c.party[0].CarriedItems {
		pack = append(pack, secondshop.MissionUnit(u, secondMissionItem(table)))
	}
	if hero, ok := memberWithPack(s.Carried[i], pack, table, true); ok {
		s.Carried[i] = hero
		c.party[0].CarriedItems = nil
	}
}

// secondMissionItem is the engine's one item decoder over table, the base
// object a mission builds for an item ID.
func secondMissionItem(table *mapload.Table) func(uint16) sim.ItemInstance {
	return func(code uint16) sim.ItemInstance { return mapload.ItemInstanceFromCode(code, table) }
}

// secondHeroIndex is the campaign party's starting hero, else its first
// member; -1 for an empty party.
func secondHeroIndex(party []mapload.PartyMember) int {
	for i, m := range party {
		if m.StartingHero {
			return i
		}
	}
	if len(party) == 0 {
		return -1
	}
	return 0
}

// secondHero is a new campaign's hero: the player's character with an
// empty bag.
func secondHero() mapload.PartyMember {
	return mapload.PartyMember{ID: "hero", PlayerCharacter: true, StartingHero: true}
}

// leaveBagToCampaign empties each member's pack: a town LOAD reads the
// second game's hero bag into the campaign's bag (DIV-4153). A hero's unit
// with no bag form (secondBagUnitForm) stays in his pack, as at a town's
// arrival (DIV-4191).
func leaveBagToCampaign(party []mapload.PartyMember, table *mapload.Table) error {
	hero := secondHeroIndex(party)
	for i := range party {
		m := &party[i]
		var kept []sim.ItemInstance
		if i == hero {
			for _, it := range mapload.MemberCarriedItems(*m, table) {
				if !secondBagUnitForm(it) {
					kept = append(kept, it)
				}
			}
		}
		if len(kept) != 0 {
			next, ok := memberWithPack(*m, kept, table, true)
			if !ok {
				return fmt.Errorf("second-game save: the hero's pack cannot keep the %d units the bag has no form for", len(kept))
			}
			*m = next
			continue
		}
		m.Carried, m.CarriedItems = nil, nil
		if m.Carry != nil {
			m.Carry.Items, m.Carry.ItemInstances, m.Carry.OrderedStacks = nil, nil, nil
		}
	}
	return nil
}

// heroBag is the hero's pack, made when the campaign has no party yet.
func (c *secondCampaign) heroBag() *[]sim.ItemInstance {
	if len(c.party) == 0 {
		c.party = []mapload.PartyMember{secondHero()}
	}
	return &c.party[0].CarriedItems
}

// adoptLegacyHero moves the hero list of a document written before the bag
// into the bag, once the install's tables can code its items. An item with
// no bag form is dropped.
func (c *secondCampaign) adoptLegacyHero(tables secondshop.Tables) {
	if len(c.legacyHero) == 0 {
		return
	}
	bag := c.heroBag()
	for _, it := range c.legacyHero {
		if units, ok := tables.Units(it.item()); ok {
			*bag = append(*bag, units...)
		}
	}
	c.legacyHero = nil
}

// captureSecondParty writes the hero through the party member producer,
// with the deal settled by the one open-deal policy on a copy of it: the
// offered stacks back in his bag in the order they were offered; nil while
// the bag is empty.
func captureSecondParty(c *secondCampaign) []currentPartyMember {
	if len(c.party) == 0 {
		return nil
	}
	hero := mapload.CloneParty(c.party[:1])[0]
	// The bag's units are the record; the shop's pack writer keeps their
	// codes beside them, which the bag does not save.
	hero.Carried = nil
	deal := c.shop.shopDeal
	deal.table = slices.Clone(deal.table)
	for i := range deal.shelves {
		deal.shelves[i] = slices.Clone(deal.shelves[i])
	}
	var back []sim.ItemInstance
	_ = deal.settle(c.shop.rules(), func(_ int, it ShopItem) error {
		units, _ := c.shop.tables.Units(secondItem(it))
		back = append(units, back...)
		return nil
	})
	hero.CarriedItems = append(hero.CarriedItems, back...)
	if len(hero.CarriedItems) == 0 {
		return nil
	}
	return []currentPartyMember{captureCurrentParty(0, hero)}
}

// maxSecondBagUnits bounds the hero's units the SAVE producer writes: his
// bag with the units he offers on the deal's table. It is the engine's
// choice (DIV-3043): twice the older hero list's entry bound, so an older
// stack past that bound still loads, while a full bag's SAV stays near 4 MB
// at about 500 bytes a unit. The shop refuses a purchase past it.
const maxSecondBagUnits = 8192

// secondBagUnits is the units the party's packs hold.
func secondBagUnits(party []currentPartyMember) int {
	n := 0
	for _, p := range party {
		if p.Member != nil {
			n += len(p.Member.CarriedItems)
		}
	}
	return n
}

// validateSecondParty bounds the saved party to one member whose bag holds
// at most maxSecondBagUnits units, each one the bag's item form.
func validateSecondParty(party []currentPartyMember) error {
	if len(party) > 1 {
		return fmt.Errorf("current second campaign party holds more than the hero")
	}
	for _, p := range party {
		if p.Member == nil || p.Member.Carry != nil || len(p.Member.Carried) != 0 {
			return fmt.Errorf("current second campaign hero has no bag")
		}
		if len(p.Member.CarriedItems) > maxSecondBagUnits {
			return fmt.Errorf("current second campaign bag holds too many units")
		}
		for _, in := range p.Member.CarriedItems {
			if _, ok := secondshop.FromInstance(in); !ok {
				return fmt.Errorf("current second campaign bag holds an invalid item")
			}
		}
	}
	return nil
}

func restoreSecondParty(party []currentPartyMember) []mapload.PartyMember {
	var out []mapload.PartyMember
	for _, p := range party {
		out = append(out, p.restore())
	}
	return out
}
