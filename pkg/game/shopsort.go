package game

import (
	"sort"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/shopstock"
	"againrom/pkg/sim"
)

// The order a shop shelf is PRESENTED in (owner, `DIV-319`).
//
// IT IS AN OWNER DIRECTIVE, NOT A DECODED FACT. SHOP-SCREEN-036 reads the one
// routine that paints every cell of all three grids walking its list in index
// order, and no claim names how that list is built from the shelf the
// generator filled, so the original's order is Unknown.
//
// The order is three keys, outermost first:
//
//  1. CLASS -- fighter items, then mage items, then items neither bit
//     separates. The bits are the item's own sutableFor column (Suitability),
//     so only the POSITION of each group is authored.
//  2. KIND -- the fixed list below. The owner named six armour kinds and five
//     weapon kinds; rings and amulets are shipped kinds he did not name,
//     placed after his six. Shields lead (his later ruling).
//  3. PRICE, ascending: inside one class and kind the first cell holds the
//     cheapest element. Price never moves an element out of its group.
//
// Elements equal on all three keep the order they had: draw order, which
// keeps a shelf reproducible from its seed, then a returned element.
//
// THE KINDS ARE READ OFF THE ITEM CODE AND ITS ROW, NOT OFF A NAME. An
// armour's field B is its Armors row's own Slot column (ITEM-ARMSLOT-031), and
// the six the owner named are slots 6, 7, 8, 9, 10 and 12 on both preserved
// roots -- helms, mails, cuirasses, bracers, gauntlets and boots. A weapon's
// kind is its row's AttackType cell, and the five he named are 1 through 5,
// which are also SkillBlade through SkillShoot: a melee weapon ASSIGNS its
// kind into the wielder's active-skill field, so the two are one number.

// The class ranks. Fighter and Mage are the item's two sutableFor bits read
// apart; an item setting both -- a ring, an amulet, bare hands, the plasma
// sword -- and an item setting neither are the same rank, because neither is
// the fighter's or the mage's more than the other's.
const (
	shopClassFighter = iota
	shopClassMage
	shopClassNeither
)

// The kind ranks, in the order the owner gave them. Shields come first (the
// armour shelf of the original opens on them, owner ruling on `DIV-319`); the
// next six are his armour list; then the two shipped kinds he did not name;
// the five after those are his weapon list.
const (
	shopKindShield = iota
	shopKindHelm
	shopKindMail
	shopKindCuirass
	shopKindBracer
	shopKindGlove
	shopKindBoot
	shopKindRing
	shopKindAmulet
	shopKindSword
	shopKindAxe
	shopKindMace
	shopKindSpear
	shopKindRanged
	shopKindOtherWeapon
	shopKindOther
)

// The armour slots the six named kinds occupy -- an armour code's field B,
// which is its row's Slot column (ITEM-ARMSLOT-031). Slots 3 and 11 ship no
// content on either root and reach shopKindOther, as does a field B of 0,
// which is what ArmorShopClass composes for a row whose Slot names no
// equipment place.
const (
	shopSlotRing    = 4
	shopSlotAmulet  = 5
	shopSlotHelm    = 6
	shopSlotMail    = 7
	shopSlotCuirass = 8
	shopSlotBracer  = 9
	shopSlotGlove   = 10
	shopSlotBoot    = 12
)

// shopItemOrder is one element's place in that order, computed once per element
// rather than once per comparison.
type shopItemOrder struct {
	class int
	kind  int
}

// shopSortShelf puts items in the owner's order, in place: by class, then kind,
// and inside one class and kind cheapest first.
//
// A NIL TABLE SORTS NOTHING and leaves the shelf exactly as it was. The class
// and the weapon kind are both cells of a definition row, so a shop with no
// table to read has no groups to run price inside, and generation order is
// what it has.
func shopSortShelf(items []ShopItem, t *mapload.Table) {
	if t == nil || len(items) < 2 {
		return
	}
	type entry struct {
		item  ShopItem
		order shopItemOrder
	}
	sorted := make([]entry, len(items))
	for i, item := range items {
		sorted[i] = entry{item: item, order: shopOrderOf(item.Code, t)}
	}
	sort.SliceStable(sorted, func(a, b int) bool {
		x, y := sorted[a], sorted[b]
		if x.order.class != y.order.class {
			return x.order.class < y.order.class
		}
		if x.order.kind != y.order.kind {
			return x.order.kind < y.order.kind
		}
		return x.item.Price < y.item.Price
	})
	for i := range sorted {
		items[i] = sorted[i].item
	}
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

// shopOrderOf is one item code's class and kind.
func shopOrderOf(c data.ItemCode, t *mapload.Table) shopItemOrder {
	suit, known := data.SuitabilityFromCode(c, t.Weapons, t.Shields, t.Armors)
	return shopItemOrder{class: shopClassRank(suit, known), kind: shopKindOf(c, t)}
}

// shopClassRank is which of the three class groups an item belongs to.
//
// AN UNKNOWN SUITABILITY IS shopClassNeither AND NOT A REFUSAL. ok is false for
// a code this build cannot reach a row for -- the carried class among them --
// and such an item is neither the fighter's nor the mage's as far as this order
// can tell.
func shopClassRank(s data.Suitability, known bool) int {
	if !known {
		return shopClassNeither
	}
	switch {
	case s.Fighter && !s.Mage:
		return shopClassFighter
	case s.Mage && !s.Fighter:
		return shopClassMage
	}
	return shopClassNeither
}

// shopKindOf is which kind an item code names.
func shopKindOf(c data.ItemCode, t *mapload.Table) int {
	switch c.B() {
	case shopWeaponClass:
		return shopWeaponKind(c, t)
	case shopShieldClass:
		return shopKindShield
	case shopSlotRing:
		return shopKindRing
	case shopSlotAmulet:
		return shopKindAmulet
	case shopSlotHelm:
		return shopKindHelm
	case shopSlotMail:
		return shopKindMail
	case shopSlotCuirass:
		return shopKindCuirass
	case shopSlotBracer:
		return shopKindBracer
	case shopSlotGlove:
		return shopKindGlove
	case shopSlotBoot:
		return shopKindBoot
	}
	return shopKindOther
}

// shopWeaponKind is which of the owner's five weapon kinds a weapon code names,
// off its row's AttackType cell.
//
// AN ATTACK TYPE OUTSIDE 1..5 IS shopKindOtherWeapon AND IS NOT FORCED INTO ONE
// OF THE FIVE. The shipped Weapons collection carries one such row on both
// roots -- the flame thrower, at 11 -- and a build that made it a bow would be
// stating something about it that no column says.
func shopWeaponKind(c data.ItemCode, t *mapload.Table) int {
	kind, ok := data.AttackTypeFromCode(c, t.Weapons)
	if !ok {
		return shopKindOtherWeapon
	}
	switch kind {
	case data.SkillBlade:
		return shopKindSword
	case data.SkillAxe:
		return shopKindAxe
	case data.SkillBludgen:
		return shopKindMace
	case data.SkillPike:
		return shopKindSpear
	case data.SkillShoot:
		return shopKindRanged
	}
	return shopKindOtherWeapon
}
