package game

import (
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

var bankVisits = visitRule{
	rooms: map[sim.VisitClass]string{sim.VisitShop: "shop", sim.VisitInn: "inn"},
	stock: func(t *townScreen) *Shop {
		c := secondOf(t.sess)
		if c == nil {
			return nil
		}
		return cloneShopMutation(&c.shop)
	},
	seat: func(n *CampaignSession, stock *Shop, town int, pack []sim.ItemInstance) (*[]sim.ItemInstance, func()) {
		orig := n.Town.second
		c := *orig
		c.available = slices.Clone(orig.available)
		c.party = []mapload.PartyMember{secondHero()}
		c.party[0].CarriedItems = pack
		c.current, c.room = partyPlace{kind: placeTown, id: town}, secondTownSquare
		c.innBits, c.talking, c.mapOpen = nil, nil, nil
		c.shop = *stock
		n.Town.second = &c
		return c.heroBag(), func() { orig.bank, orig.available = c.bank, c.available }
	},
	view: func(table *mapload.Table, u sim.ItemInstance) (sim.ItemInstance, bool) {
		u.ObjectID = 0
		return secondPartyItems.keptForm(u, table)
	},
	world: func(table *mapload.Table, u sim.ItemInstance) sim.ItemInstance {
		return secondPartyItems.missionForm(u, table)
	},
	table: func(s *Shop, p ShopPlace) []sim.ItemInstance {
		units, _ := s.tables.Units(secondItem(p.ShopItem))
		return units
	},
}
