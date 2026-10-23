package game

import (
	"fmt"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
)

// The second game's party is one member, the hero; his bag is the member's
// pack of the engine's item instances (R2-ENGINE-402, R2-ENGINE-404). The
// shop deals with that pack. The bag does not reach a mission (DIV-3043).

// secondHero is a new campaign's hero: the player's character with an
// empty bag.
func secondHero() mapload.PartyMember {
	return mapload.PartyMember{ID: "hero", PlayerCharacter: true, StartingHero: true}
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
