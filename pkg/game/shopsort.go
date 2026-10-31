package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/shopstock"
	"againrom/pkg/sim"
)

// The first game's shelf order is SHOP-154's four keys, outermost first: the
// fighter flag set first, the code's kind nibble ascending, for weapons a row
// group ascending, then price ascending. The flag is the row's sutableFor
// bit 0x1 (ITEM-SUIT-035); the weapon row group stands in as the row's
// AttackType rank; equal elements keep their order (DIV-4164).
//
// THE KIND IS THE CODE'S FIELD B, NOT A NAME: weapons 1, shields 2, then the
// armour slots (ITEM-ARMSLOT-031), so rings 4 and amulets 5 stand before
// helms 6 and boots 12.

// shopItemOrder is one element's place in that order, computed once per element
// rather than once per comparison.
type shopItemOrder struct {
	class int
	kind  int
	group int
}

// shopSortShelf puts items in that order, in place, keeping the order of
// elements equal on all four keys: draw order, then a returned element.
//
// A NIL TABLE SORTS NOTHING and leaves the shelf exactly as it was: the flag
// and the weapon group are cells of a definition row.
func shopSortShelf(items []ShopItem, t *mapload.Table) {
	if t == nil {
		return
	}
	shopstock.SortBy(items, func(item ShopItem) []int64 {
		o := shopOrderOf(item.Code, t)
		return []int64{int64(o.class), int64(o.kind), int64(o.group), int64(item.Price)}
	})
}

// shopStackShelf folds every repeated item into one element carrying the
// count, and returns the shortened shelf (owner, `DIV-322`).
//
// IDENTITY IS CODE AND UNIT PRICE, WHICH IS shopFindMinePlace's OWN RULE
// (shoproom.go) restated one container over. Both fields must agree for the
// same reason they must agree on the table: every sum in this package reads
// Count x the element's own Price, once, so two lots at different prices
// folded into one element would charge the wrong price for one of them.
//
// THE ORIGINAL SHOWS A QUANTITY TOO, AND THIS FOLDS IT ONE CONTAINER EARLIER.
// SHOP-DUP-028: the original's shelf container holds one element per drawn
// unit and its screen merges display rows (Medium), so the player-visible
// result agrees at that confidence and the mechanism differs (DIV-322).
// Complete-equal instances fold, effect list included.
//
// The first occurrence keeps its position. It needs no table.
func shopStackShelf(items []ShopItem) []ShopItem {
	if len(items) < 2 {
		return items
	}
	return shopstock.Insert(make([]ShopItem, 0, len(items)), shopShelfStack(nil), items...)
}

// shopStackKey holds the fields shopItemEqual requires to agree, so only
// elements sharing a key are compared and a long shelf stacks in linear time.
type shopStackKey struct {
	cityID, objectID sim.SavedObjectID
	code             data.ItemCode
	price            int32
}

// The two flag ranks: the fighter flag set, then everything else.
const (
	shopFlagFighter = iota
	shopFlagNone
)

// The weapon row-group ranks, by the row's AttackType: blade, axe, bludgeon,
// pike, shoot, then any other attack type or a code with no row.
const (
	shopGroupSword = iota
	shopGroupAxe
	shopGroupMace
	shopGroupSpear
	shopGroupRanged
	shopGroupOtherWeapon
)

// shopOrderOf is one item code's flag, kind and weapon group.
func shopOrderOf(c data.ItemCode, t *mapload.Table) shopItemOrder {
	suit, known := data.SuitabilityFromCode(c, t.Weapons, t.Shields, t.Armors)
	o := shopItemOrder{class: shopFlagRank(suit, known), kind: c.B()}
	if o.kind == shopWeaponClass {
		o.group = shopWeaponGroup(c, t)
	}
	return o
}

// shopFlagRank is whether the row sets the fighter bit. An item for both
// classes sets it; a code this build reaches no row for does not.
func shopFlagRank(s data.Suitability, known bool) int {
	if known && s.Fighter {
		return shopFlagFighter
	}
	return shopFlagNone
}

// shopWeaponGroup is a weapon code's row group, off its row's AttackType cell.
// An attack type outside 1..5 -- the shipped flame thrower, at 11 -- is
// shopGroupOtherWeapon and is not forced into one of the five.
func shopWeaponGroup(c data.ItemCode, t *mapload.Table) int {
	kind, ok := data.AttackTypeFromCode(c, t.Weapons)
	if !ok {
		return shopGroupOtherWeapon
	}
	switch kind {
	case data.SkillBlade:
		return shopGroupSword
	case data.SkillAxe:
		return shopGroupAxe
	case data.SkillBludgen:
		return shopGroupMace
	case data.SkillPike:
		return shopGroupSpear
	case data.SkillShoot:
		return shopGroupRanged
	}
	return shopGroupOtherWeapon
}
