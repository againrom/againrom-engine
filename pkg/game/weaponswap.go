package game

import (
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// heldPair is what a unit holds in its two hands: equipment slot 1, the
// weapon, and slot 2, the shield. A bow is a slot-1 weapon that leaves slot 2
// empty.
type heldPair struct {
	weapon, shield sim.ItemInstance
}

// weaponSwapState is the weapon swap's memory and its unfinished swaps. It is
// App state: it lives on the running mapWorld, is not hashed and is not saved,
// so a LOAD starts every unit with no previous weapon (DIV-2948).
type weaponSwapState struct {
	// seen is the pair each party member held at the last observation;
	// previous is the pair it held before its weapon last changed.
	seen, previous map[sim.EntityID]heldPair
	plans          []weaponSwapPlan
	// forced makes rearm and refreshAppearance recompute although the
	// subject's trackers were seeded from the state already changed.
	forced bool
}

// weaponSwapPlan is one unit's swap in progress. Each step is one command
// whose container index is read from the state the previous command left, so
// a step waits until the tick after the one it was queued for.
type weaponSwapPlan struct {
	id     sim.EntityID
	want   heldPair
	stage  int
	issued uint64
}

const (
	swapStageWeapon = iota
	swapStageShield
	swapStageDone
)

// swapMember is the party member a swap may act on: the population whose
// inventory the player can open (switchInventorySubject's own gate).
func (mw *mapWorld) swapMember(id sim.EntityID) (mapload.PartyMember, bool) {
	if mw == nil || mw.mission == nil || mw.world == nil {
		return mapload.PartyMember{}, false
	}
	for i, mid := range mw.mission.ids {
		if mid == id && i < len(mw.mission.party) {
			m := mw.mission.party[i]
			return m, !m.Hired() && data.ComposesFigure(m.Class)
		}
	}
	return mapload.PartyMember{}, false
}

// observeWeapons records, for every party member the swap may act on, the pair
// it held before its weapon changed. An empty weapon is never remembered as a
// previous weapon. It reads the world and writes nothing to it.
func (mw *mapWorld) observeWeapons() {
	if mw == nil || mw.mission == nil || mw.world == nil {
		return
	}
	s := &mw.weaponSwap
	for i, id := range mw.mission.ids {
		if i >= len(mw.mission.party) {
			break
		}
		if _, ok := mw.swapMember(id); !ok {
			continue
		}
		worn, ok := mw.world.EquippedItems(id)
		if !ok {
			continue
		}
		cur := heldPair{weapon: worn[0], shield: worn[1]}
		if s.seen == nil {
			s.seen, s.previous = make(map[sim.EntityID]heldPair), make(map[sim.EntityID]heldPair)
		}
		if old, had := s.seen[id]; had && !old.weapon.Empty() && !sim.WornItemEqual(old.weapon, cur.weapon) {
			s.previous[id] = old
		}
		s.seen[id] = cur
	}
}

// swapWeapons is the ui.MapWeaponSwap the loader hands the viewer: each named
// unit with a previous weapon still in its own pack starts a swap. A unit whose
// earlier swap is still waiting on its tick is left to finish it.
func (mw *mapWorld) swapWeapons(ids []uint32) {
	mw.observeWeapons()
	s := &mw.weaponSwap
	for _, raw := range ids {
		id := sim.EntityID(raw)
		if _, ok := mw.swapMember(id); !ok {
			continue
		}
		if slices.ContainsFunc(s.plans, func(p weaponSwapPlan) bool { return p.id == id }) {
			continue
		}
		want, ok := s.previous[id]
		if !ok || want.weapon.Empty() || mw.packIndex(id, want.weapon) < 0 {
			continue
		}
		s.plans = append(s.plans, weaponSwapPlan{id: id, want: want, stage: swapStageWeapon, issued: ^uint64(0)})
	}
	mw.advanceWeaponSwaps()
}

// advanceWeaponSwaps queues the next step of every swap whose last command
// has been applied and whose unit has no earlier inventory order still
// pending, and drops each swap that is finished. Waiting for those orders
// means a step's container index is read from the pack its command meets: an
// earlier queued drop cannot shift another item under it, and a remembered
// item that order removed ends the swap with no command.
func (mw *mapWorld) advanceWeaponSwaps() {
	s := &mw.weaponSwap
	if len(s.plans) == 0 || mw.world == nil {
		return
	}
	tick := mw.world.Tick()
	kept := s.plans[:0]
	for _, p := range s.plans {
		if p.issued != ^uint64(0) && tick <= p.issued || mw.inventoryOrderPending(p.id) {
			kept = append(kept, p)
			continue
		}
		cmd, ok := mw.nextSwapCommand(&p)
		if !ok {
			continue
		}
		mw.pending = append(mw.pending, cmd)
		p.issued = tick
		kept = append(kept, p)
	}
	s.plans = kept
}

// nextSwapCommand is the plan's next order, read from the unit's current
// equipment and pack, or false when the swap is finished or can go no further.
//
// The weapon step wears the previous weapon and sends the current shield to
// the pack unless it is the previous shield and the weapon leaves a hand free.
// The shield step wears the previous shield, or empties slot 2 when the
// previous pair had none, so the hands end holding the previous pair.
func (mw *mapWorld) nextSwapCommand(p *weaponSwapPlan) (sim.Command, bool) {
	var none sim.Command
	member, ok := mw.swapMember(p.id)
	if !ok {
		return none, false
	}
	worn, ok := mw.world.EquippedItems(p.id)
	if !ok {
		return none, false
	}
	t := mw.invParty.table
	w, shield := p.want.weapon, p.want.shield
	freeHand := mapload.WeaponAllowsShield(data.ItemCode(w.Code), t)
	if p.stage == swapStageWeapon {
		p.stage = swapStageShield
		if !sim.WornItemEqual(worn[0], w) {
			idx := mw.packIndex(p.id, w)
			if idx < 0 || !mw.swapWears(member, w, 1) {
				return none, false
			}
			var displace sim.EquipSlot
			if !worn[1].Empty() && (shield.Empty() || !sim.WornItemEqual(worn[1], shield) || !freeHand) {
				displace = 2
			}
			return sim.EquipDisplacing(p.id, sim.ItemSlot(idx), 1, displace), true
		}
	}
	if p.stage != swapStageShield || !sim.WornItemEqual(worn[0], w) {
		return none, false
	}
	p.stage = swapStageDone
	if !shield.Empty() && sim.WornItemEqual(worn[1], shield) {
		return none, false
	}
	if !shield.Empty() && freeHand && mw.swapWears(member, shield, 2) {
		if idx := mw.packIndex(p.id, shield); idx >= 0 {
			return sim.Equip(p.id, sim.ItemSlot(idx), 2), true
		}
	}
	if !worn[1].Empty() {
		return sim.Unequip(p.id, 2), true
	}
	return none, false
}

// swapWears reports whether item goes to equipment slot slot and member's
// class may wear it: enqueueEquip's own two questions, asked of any member.
func (mw *mapWorld) swapWears(member mapload.PartyMember, item sim.ItemInstance, slot int) bool {
	code := data.ItemCode(item.Code)
	t := mw.invParty.table
	if got, ok := EquipTarget(code, t); !ok || got != slot {
		return false
	}
	if t == nil {
		return true
	}
	allowed, _ := data.AllowsItem(code, member.Mage, t.Weapons, t.Shields, t.Armors)
	return allowed
}

// packIndex is the first element of id's pack holding item, or -1.
func (mw *mapWorld) packIndex(id sim.EntityID, item sim.ItemInstance) int {
	stacks, ok := mw.world.CarriedStacks(id)
	if !ok {
		return -1
	}
	for i, st := range stacks {
		if st.Count > 0 && sim.WornItemEqual(st.Instance(), item) {
			return i
		}
	}
	return -1
}

// inventoryOrderPending reports whether a pending order of id's can change
// its pack or its worn items.
func (mw *mapWorld) inventoryOrderPending(id sim.EntityID) bool {
	return slices.ContainsFunc(mw.pending, func(c sim.Command) bool {
		return c.Entity == id && changesInventory(c.Kind)
	})
}

func changesInventory(k uint8) bool {
	switch k {
	case sim.KindEquip, sim.KindUnequip, sim.KindDropCarried, sim.KindDropWorn, sim.KindPickUp,
		sim.KindReadBook, sim.KindUsePotion, sim.KindUseScroll, sim.KindUseScrollAt, sim.KindRearrangeCarried:
		return true
	}
	return false
}

// wornBefore is a party member's worn items before the step that applies its
// pending equip order.
type wornBefore struct {
	id   sim.EntityID
	worn [sim.EquipSlots]sim.ItemInstance
}

// queuedEquipActors records, before the step, the worn items of every party
// member a pending order equips or unequips. It reads only the World and the
// pending orders, which a SAV holds, so a LOAD between queueing and applying
// re-derives the same units.
func (mw *mapWorld) queuedEquipActors() []wornBefore {
	var out []wornBefore
	for i, c := range mw.pending {
		if c.Kind != sim.KindEquip && c.Kind != sim.KindUnequip && c.Kind != sim.KindDropWorn ||
			i < len(mw.pendingIgnored) && mw.pendingIgnored[i] ||
			slices.ContainsFunc(out, func(b wornBefore) bool { return b.id == c.Entity }) {
			continue
		}
		if _, ok := mw.swapMember(c.Entity); !ok {
			continue
		}
		if worn, ok := mw.world.EquippedItems(c.Entity); ok {
			out = append(out, wornBefore{id: c.Entity, worn: worn})
		}
	}
	return out
}

// rearmEquipped re-derives, after the step, each recorded unit whose worn
// items changed and that is not the inventory subject. rearm and
// refreshAppearance follow only the subject, so each such unit becomes the
// subject for one forced pass of both, and the subject the player had is
// restored. The pass is the same derivation an equip from that unit's own
// inventory runs.
func (mw *mapWorld) rearmEquipped(before []wornBefore) {
	if len(before) == 0 || !mw.invSubjectSet {
		return
	}
	s := &mw.weaponSwap
	subject := mw.invSubject.ID
	for _, b := range before {
		worn, ok := mw.world.EquippedItems(b.id)
		if uint32(b.id) == subject || !ok || itemEquipmentStateEqual(worn, b.worn) {
			continue
		}
		mw.switchInventorySubject(uint32(b.id))
		if !mw.invSubjectSet || mw.invSubject.ID != uint32(b.id) {
			continue
		}
		s.forced = true
		mw.rearm()
		mw.refreshAppearance()
		s.forced = false
	}
	if mw.invSubject.ID != subject {
		mw.switchInventorySubject(subject)
	}
}
