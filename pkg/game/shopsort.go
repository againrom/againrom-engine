package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/shopstock"
	"againrom/pkg/sim"
)

// The first game's SHOP-154 profile uses the row's suitability and AttackType
// for flag and weapon group. Equal items keep their order (DIV-4164).

// shopItemOrder holds a code's first three keys.
type shopItemOrder struct {
	class int
	kind  int
	group int
}

// shopSortShelf leaves a shelf unchanged without its definition table.
func shopSortShelf(items []ShopItem, t *mapload.Table) {
	if t == nil {
		return
	}
	shopstock.ShelfOrder[ShopItem]{
		Flag:  func(it ShopItem) bool { return shopOrderOf(it.Code, t).class == shopFlagFighter },
		Kind:  func(it ShopItem) int { return it.Code.B() },
		Group: func(it ShopItem) int { return shopWeaponGroup(it.Code, t) },
		Price: func(it ShopItem) int64 { return int64(it.Price) },
	}.Sort(items)
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
