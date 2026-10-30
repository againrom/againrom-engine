package game

import (
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// missionShopState is a mission's Shop: the party member whose Shop structure
// use waits for the page, and the stock the mission's visits trade with. The
// stock is a copy of the town's current stock taken at the first visit, or the
// stock the town would draw when the session holds none (DIV-4218).
type missionShopState struct {
	customer sim.EntityID
	waiting  bool
	stock    *Shop
}

// observeShopUses raises the Shop page for a Shop structure use by a member
// of the party the mission opened with.
func (mw *mapWorld) observeShopUses(uses []sim.StructureUse) {
	if mw == nil || mw.mission == nil || mw.view == nil {
		return
	}
	for _, u := range uses {
		if mw.managedMember(u.Entity) >= 0 {
			mw.shop.customer, mw.shop.waiting = u.Entity, true
			mw.view.RaiseShop()
		}
	}
}

// missionShopVisit is the Shop page shown over a mission: the town screen as
// it stood before, the mission's world, and the customer's pack as the page
// trades with it.
type missionShopVisit struct {
	saved townScreen
	live  *mapWorld
	hero  sim.EntityID
	owner uint32
	bag   []sim.ItemInstance
}

// OpenMissionShop shows the town's Shop room for the waiting customer. The
// room runs over a session of its own: the town's state with the customer's
// purse as its gold and the customer as its party, and the mission's stock as
// its deal. The town is not open in it, so no offer speaks and no save is
// admitted while the page shows.
func (t *townScreen) OpenMissionShop() bool {
	if t == nil || t.visit != nil || t.sess == nil || t.sess.Town == nil || t.hostRules().Shop != "chapter" {
		return false
	}
	if d := t.townDescription(); d == nil || d.Room("shop") == nil {
		return false
	}
	mw := t.sess.live
	if mw == nil || mw.world == nil || mw.mission == nil || !mw.shop.waiting {
		return false
	}
	mw.shop.waiting = false
	hero := mw.shop.customer
	k := mw.managedMember(hero)
	e, alive := mw.world.Entity(hero)
	items, held := mw.world.CarriedItems(hero)
	if k < 0 || !alive || !e.Alive() || !held {
		return false
	}
	if mw.shop.stock == nil {
		if t.sess.Shop != nil {
			mw.shop.stock = cloneShopMutation(t.sess.Shop)
		} else {
			mw.shop.stock = t.sess.Town.newShop(t.switches.townInstall(t.in))
		}
		mw.shop.stock.inputEpoch = new(byte)
	}
	owner := uint32(e.Owner)
	v := &missionShopVisit{saved: *t, live: mw, hero: hero, owner: owner, bag: items}
	n := *t.sess
	town := *t.sess.Town
	town.cityObjects, town.open, town.gold = nil, false, int(mw.world.Purse(owner))
	n.Town = &town
	n.Carried = mapload.CarryParty(mw.mission.party[k:k+1], mw.world, []sim.EntityID{hero})
	n.originalCity = nil
	n.Shop = mw.shop.stock
	t.visit, t.sess = v, &n
	t.bag = func(*CampaignSession) *[]sim.ItemInstance { return &v.bag }
	t.settle = t.settleMissionDeal
	t.wornFixed = true
	t.shopMember = 0
	t.squareView().EnterRoom("shop")
	if t.room != roomShop {
		t.dropMissionShop()
		return false
	}
	return true
}

// MissionShopOpen reports whether the Shop page shows over a mission.
func (t *townScreen) MissionShopOpen() bool { return t != nil && t.visit != nil }

// closeMissionShop leaves the Shop room: its exit steps return the table,
// what the visit traded is committed, and the town screen stands as it stood
// before. The mission's pace restarts from the frame it resumes on.
func (t *townScreen) closeMissionShop() bool {
	v := t.visit
	t.squareView().ExitRoom("shop")
	t.commitMissionDeal(v)
	t.dropMissionShop()
	v.live.last = time.Time{}
	return true
}

// dropMissionShop restores the town screen the visit stood over.
func (t *townScreen) dropMissionShop() {
	v := t.visit
	if v == nil {
		return
	}
	v.live.shop.stock = t.sess.Shop
	t.destroyRoomAudio()
	*t = v.saved
	t.roomAudio, t.audioRoom = nil, ""
}

// settleMissionDeal runs one Shop gesture and commits what it changed to the
// mission's world. A deal the world refuses leaves the page as it stood.
func (t *townScreen) settleMissionDeal(act func() ui.TownAction) ui.TownAction {
	v := t.visit
	shop, bag, gold := cloneShopMutation(t.sess.Shop), cloneItemInstances(v.bag), t.sess.Town.gold
	a := act()
	if t.visit != v {
		return a
	}
	if !t.commitMissionDeal(v) {
		t.sess.Shop, v.bag, t.sess.Town.gold = shop, bag, gold
		v.live.shop.stock = shop
		return ui.TownAction{}
	}
	return a
}

// commitMissionDeal makes the customer's pack and purse in the world what the
// page holds for him: his pack and his places on the table. It is one
// TradeCarried call, or none when nothing changed.
func (t *townScreen) commitMissionDeal(v *missionShopVisit) bool {
	mw := v.live
	mw.shop.stock = t.sess.Shop
	held := cloneItemInstances(v.bag)
	for _, p := range t.sess.Shop.Table() {
		for n := int32(0); p.Mine && n < p.ShopItem.Count; n++ {
			held = append(held, p.ShopItem.Instance())
		}
	}
	stacks, ok := mw.world.CarriedStacks(v.hero)
	if !ok {
		return false
	}
	sold, bought := shopDealDiff(stacks, held)
	purse := uint32(max(t.sess.Town.gold, 0))
	if len(sold) == 0 && len(bought) == 0 && purse == mw.world.Purse(v.owner) {
		return true
	}
	if !mw.world.TradeCarried(v.hero, sold, bought, purse) {
		return false
	}
	mw.refreshPack()
	mw.push()
	return true
}

// shopDealDiff matches the world's pack, unit by unit, against the units the
// page holds. An unmatched world unit is sold, named by its stack's index; an
// unmatched held unit is bought. A unit matches by every field the pack
// merges on but its saved object and its price, which the page's table does
// not keep.
func shopDealDiff(stacks []sim.ItemStack, held []sim.ItemInstance) (sold []int, bought []sim.ItemStack) {
	used := make([]bool, len(held))
	for e, s := range stacks {
		unit := s.Instance()
		for range s.Count {
			j := 0
			for j < len(held) && (used[j] || !sameShopUnit(unit, held[j])) {
				j++
			}
			if j == len(held) {
				sold = append(sold, e)
				continue
			}
			used[j] = true
		}
	}
	for j, h := range held {
		if !used[j] {
			h.ObjectID = 0
			bought = append(bought, sim.StackItem(h, 1))
		}
	}
	return sold, bought
}

func sameShopUnit(a, b sim.ItemInstance) bool {
	a.ObjectID, a.Price, b.ObjectID, b.Price = 0, 0, 0, 0
	return a.Kind == b.Kind && sim.ItemEqual(a, b)
}

func cloneItemInstances(items []sim.ItemInstance) []sim.ItemInstance {
	out := make([]sim.ItemInstance, len(items))
	for i, item := range items {
		out[i] = item.Clone()
	}
	return out
}
