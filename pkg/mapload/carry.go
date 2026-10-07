package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// Carry is what a party member brings OUT of one mission and INTO the next:
// the state he EARNED while a world was running, as opposed to the state he
// was minted from.
//
// IT EXISTS BECAUSE PartyMember CANNOT SAY ANY OF IT. Every other field of a
// member is an INPUT to the mint — four statistics, a profile, a body name, a
// weapon — and each is a fact about who he is rather than about what has
// happened to him. Experience, a pack and a worn set are the opposite: they
// are what the SIMULATION wrote, they exist only after a world has run, and
// before this type there was nowhere in a member to put them. So a second
// mission opened on a freshly minted party and everything the first one earned
// was thrown away.
//
// IT IS A LOADER INPUT LIKE Body AND Profile ARE, and it is carried by
// POINTER for the one reason a pointer is ever right here: the zero value has
// to mean "this member earned nothing yet", and a member who really did end a
// mission holding nothing and wearing nothing is a DIFFERENT state from one
// who was never in a mission at all. The first arrives bare; the second
// arrives wearing the weapon his class row gave him. A struct value could not
// tell those two apart, and the mint would have re-armed a hero who had
// deliberately dropped his sword.
//
// NIL IS EVERY PARTY THIS TREE BUILT BEFORE THE CONTINUITY HOTFIX, and a nil
// carry changes nothing at all: the mint takes exactly the arm it always took.
//
// Party saves persist this value through PartyMember. The code arrays remain
// validated compatibility projections; ItemInstances and EquippedItems carry
// canonical item values. CarryRoster's data-only world boundary drops registry
// handles, not item values. CarryRosterIDs retains them inside a running world.
type Carry struct {
	// SkillXP is the six per-slot experience integers the entity ENDED the
	// previous mission with, exactly (0125's own field).
	//
	// IT IS THE EXACT INTEGER AND NOT THE LEVEL IT IMPLIES. A level and its
	// experience are two spellings of one value joined by S(n) and S's
	// inverse (data.Derived.SkillXP), and the inverse FLOORS — so a member
	// carried forward through his level alone would silently lose every
	// point of progress he had made toward the next one, at every mission
	// boundary, for the whole campaign. CarryParty writes the LEVELS too, so
	// the member's derived combat numbers rise with what he earned; this
	// field is what keeps the residue.
	SkillXP [data.SkillSlots]int32

	// Items is his container as the previous mission left it: what he picked
	// up off the ground and off the bodies, in the order the simulation holds
	// it.
	Items         []uint16
	ItemInstances []sim.ItemInstance
	OrderedStacks []sim.ItemStack
	LiveLoad      *sim.ActorLoadSnapshot

	// Equipped is the twelve equipment slots as the previous mission left
	// them — sim.Stock.Equipped's own shape and its own meaning, slot 1 at
	// index 0 and the zero code an empty slot.
	//
	// A WHOLLY ZERO ARRAY IS A MEMBER WEARING NOTHING and is honoured as
	// such, which is the whole reason this type is reached by pointer: see
	// the type's own doc.
	Equipped      [sim.EquipSlots]uint16
	EquippedItems [sim.EquipSlots]sim.ItemInstance
}

// CarryParty is the party that finished a mission, expressed as the party that
// STARTS the next one: the same members, each holding what his entity ended
// with.
//
// party and ids are POSITIONAL AND PARALLEL, which is exactly the pair
// Mission.Party and Start.IDs already are — member i was minted as entity
// ids[i]. Nothing is re-derived from the world's shape, because "the last
// len(party) entities" is a rule StartMission owns and a second copy of it
// here would be kept true only by nobody changing the append.
//
// A MEMBER WHOSE ENTITY DID NOT SURVIVE IS CARRIED FORWARD UNCHANGED — the
// member as he ENTERED that mission, not as he left it. A world that no longer
// holds the id and one that holds it dead answer the same way, on
// settleNotices' own test. Nothing is invented for him and nothing is taken
// away: he keeps what he came in with. That arm is not reachable for the party
// this tree builds today, because it holds one member and his death is the
// mission's own loss condition, so it is written to a rule rather than left to
// whatever a nil entity happened to do.
//
// A nil world, or a party longer than the id list, carries the members it
// cannot speak for forward unchanged for the same reason.
//
// IT RETURNS A FRESH SLICE and never writes through its argument: the mission
// that has just been won still holds the party it was started with, and a
// reader asking what a member began that mission as must still get that
// answer.
//
// A Saved DOES NOT CROSS THIS BOUNDARY. A Saved describes exactly the
// mission the save was taken in: a cell on THAT map and the pools the
// character held THERE. PartyMember is copied by value here and Saved is a
// pointer field, so before this it travelled into the next mission verbatim
// — and StartMission places a member carrying one at Saved.Cell with
// Saved's pools. A hero who resumed mission 10 from a save therefore arrived
// in mission 20 at his mission-10 cell, which on that map is inside the
// rocks, carrying his mission-10 health. It is cleared for EVERY member,
// including one whose entity did not survive: the whole party is moving to a
// map the save says nothing about.
//
// WHAT SURVIVES INSTEAD IS Carry, which is the right shape for the boundary: it
// is what the member EARNED rather than where he stood. The next mission places
// him by its own map's drop cell and mints his pools from the fold, exactly as
// it does for a party that came from no save at all.
func CarryParty(party []PartyMember, w *sim.World, ids []sim.EntityID) []PartyMember {
	out := OwnParty(party)
	for i := range out {
		out[i].Saved = nil
		// World item/effect/award producers do not yet maintain the original
		// city's full modifier history. Never carry a stale basis back to town.
		out[i].OriginalHuman = nil
	}
	if w == nil {
		return out
	}
	live := make(map[sim.EntityID]sim.Entity, len(ids))
	for _, e := range w.EntityView() {
		live[e.ID] = e
	}
	for i := range out {
		if i >= len(ids) {
			break
		}
		e, ok := live[ids[i]]
		if !ok || !e.Alive() {
			continue
		}
		c := Carry{SkillXP: e.SkillXP, Equipped: worn(w, ids[i]), EquippedItems: wornItems(w, ids[i])}
		c.LiveLoad = e.CurrentActorLoad()
		if c.LiveLoad != nil {
			c.OrderedStacks, _ = w.CarriedStacks(ids[i])
		}
		if items, ok := w.Carried(ids[i]); ok {
			c.Items = items
		}
		if items, ok := w.CarriedItems(ids[i]); ok {
			c.ItemInstances = items
		}
		// Carry trained inputs; bounded effective levels can hide an award.
		out[i].Hero.Skill = e.TrainedSkills(EquippedSkillBonus(c.EquippedItems, out[i].Profile.Fighter))
		out[i].Hero = PotionHero(out[i].Hero, e.PotionStats)
		out[i].PotionEffect = nil
		for _, effect := range w.ActiveEffects() {
			if effect.Target == e.ID && effect.Spell == 0 {
				out[i].PotionEffect = &effect
				break
			}
		}
		out[i].KnownSpells = e.KnownSpells
		out[i].Book = e.Book
		if e.Book.State != sim.BookLegacy {
			out[i].SpellbookRestored = true
			out[i].SpellbookPresent = e.Book.HasInstances()
		}
		out[i].Carry = &c
		setCarryHoldings(&out[i])
		if c.LiveLoad != nil && c.LiveLoad.Inventory.Source.Class == 2 {
			out[i].Hero = SourceHumanState(c.LiveLoad.Inventory.Source, c.LiveLoad.Inventory.Accumulator).Hero()
		}
	}
	return out
}

// worn is id's equipment set, and the empty set where the world does not hold
// id — Equipped's own second result, dropped here because CarryParty has
// already established that it does.
func worn(w *sim.World, id sim.EntityID) [sim.EquipSlots]uint16 {
	eq, _ := w.Equipped(id)
	return eq
}

func wornItems(w *sim.World, id sim.EntityID) [sim.EquipSlots]sim.ItemInstance {
	eq, _ := w.EquippedItems(id)
	return eq
}

// CarryRoster is CarryParty plus the actors that JOINED the party while the
// mission ran: the party that finished a mission, expressed as the party
// that starts the next one, including members who were nobody's party member
// when it began.
//
// THE THREE TESTS ARE THE SIMULATION'S, NOT THIS FUNCTION'S. w.BoundarySurvivors
// answers which of the human participant's actors cross — alive, owned by him,
// and carrying a person's server type id — over every actor the world holds
// rather than over the ids the mission was started with. An actor a script
// handed over is judged by exactly the same three tests as one that walked in,
// which is the whole reason the question is asked of the world.
//
// A SURVIVOR THAT WAS ALREADY A MEMBER IS NOT APPENDED AGAIN: ids is the party's
// own entity list and every id in it is skipped here, because CarryParty above
// has already carried that member with his own experience, pack and worn set.
//
// roster IS WHAT THE MAP SUPPLIED AT LOAD (Start.Roster): the four statistics,
// the profile, the spellbook and the weapon of the row each person placement
// resolved to. A survivor with no entry there is SKIPPED rather than appended
// bare (0159 D-12) — minting him on the next map would fold his combat numbers
// from zeroes. A creature never reaches this arm at all, being out of band.
//
// A JOINER ARRIVES CARRYING WHAT HIS ENTITY HELD, through the same Carry
// CarryParty writes for a member who walked in: his six experience integers,
// his six skill levels, his container and his twelve worn slots. He is placed on
// the next map by that map's own drop cell, mints his pools from the fold, and
// therefore arrives at full health and full mana — which is what the original's
// own boundary does to a kept actor by resetting each pool from its maximum.
//
// THE APPEND IS IN ASCENDING ENTITY-ID ORDER, after every member that walked
// in, so which party this returns does not depend on the order the world
// stores entities in. Identity is the template's own — the runtime id the
// joiner carried in the mission he joined in — so the same companion is
// the same roster entry in every later mission, and OwnParty's uniqueness
// pass finds nothing to disambiguate.
//
// A nil world, a nil roster or an empty survivor set makes this exactly
// CarryParty, which is what every caller that has no roster to hand gets.
func CarryRoster(party []PartyMember, w *sim.World, ids []sim.EntityID,
	roster map[sim.EntityID]PartyMember) []PartyMember {
	out, _ := CarryRosterIDs(party, w, ids, roster)
	// A new mission or the town has no copy of the completed World's object
	// registry. Its local handles cannot accompany this value-only projection.
	// Full cross-world Token/child identity ownership remains DIV804 debt.
	for i := range out {
		c := out[i].Carry
		if c == nil {
			continue
		}
		for j := range c.ItemInstances {
			c.ItemInstances[j].ObjectID = 0
		}
		for j := range c.OrderedStacks {
			c.OrderedStacks[j].ObjectID = 0
		}
		for j := range c.EquippedItems {
			c.EquippedItems[j].ObjectID = 0
		}
		setCarryHoldings(&out[i])
	}
	return out
}

// rosterJoiners lists, in ascending id order, the living self-owned roster
// actors that ids does not yet hold: the members CarryRosterIDs appends.
func rosterJoiners(w *sim.World, ids []sim.EntityID, roster map[sim.EntityID]PartyMember) []sim.EntityID {
	if w == nil || len(roster) == 0 {
		return nil
	}
	was := make(map[sim.EntityID]bool, len(ids))
	for _, id := range ids {
		was[id] = true
	}
	var out []sim.EntityID
	for _, id := range w.BoundarySurvivors(sim.SelfSlot) {
		if _, ok := roster[id]; ok && !was[id] {
			out = append(out, id)
		}
	}
	return out
}

// RosterJoinPending reports whether CarryRosterIDs would append a member,
// without cloning the party.
func RosterJoinPending(w *sim.World, ids []sim.EntityID, roster map[sim.EntityID]PartyMember) bool {
	return len(rosterJoiners(w, ids, roster)) != 0
}

// CarryRosterIDs is CarryRoster with the parallel runtime ids of the returned
// members. A running mission uses the pair immediately when a script hands an
// actor over; the ordinary boundary needs only the party and calls CarryRoster.
func CarryRosterIDs(party []PartyMember, w *sim.World, ids []sim.EntityID,
	roster map[sim.EntityID]PartyMember) ([]PartyMember, []sim.EntityID) {

	out := CarryParty(party, w, ids)
	n := len(out)
	if len(ids) < n {
		n = len(ids)
	}
	outIDs := append([]sim.EntityID(nil), ids[:n]...)
	if w == nil || len(roster) == 0 {
		return out, outIDs
	}
	for _, id := range rosterJoiners(w, ids, roster) {
		p := roster[id]
		c := Carry{Equipped: worn(w, id), EquippedItems: wornItems(w, id)}
		if items, ok := w.Carried(id); ok {
			c.Items = items
		}
		if items, ok := w.CarriedItems(id); ok {
			c.ItemInstances = items
		}
		for _, e := range w.EntityView() {
			if e.ID != id {
				continue
			}
			// HIS EXPERIENCE AND HIS LEVELS ARE THE ENTITY'S OWN, neither
			// recomputed from the other — CarryParty's own rule for a member
			// who walked in, applied to one who did not.
			c.SkillXP = e.SkillXP
			c.LiveLoad = e.CurrentActorLoad()
			if c.LiveLoad != nil {
				c.OrderedStacks, _ = w.CarriedStacks(id)
			}
			p.Hero = PotionHero(p.Hero, e.PotionStats)
			p.PotionEffect = nil
			for _, effect := range w.ActiveEffects() {
				if effect.Target == id && effect.Spell == 0 {
					p.PotionEffect = &effect
					break
				}
			}
			p.Hero.Skill = e.TrainedSkills(EquippedSkillBonus(c.EquippedItems, p.Profile.Fighter))
			p.KnownSpells = e.KnownSpells
			p.Book = e.Book
			break
		}
		p.Carry = &c
		setCarryHoldings(&p)
		if c.LiveLoad != nil && c.LiveLoad.Inventory.Source.Class == 2 {
			p.Hero = SourceHumanState(c.LiveLoad.Inventory.Source, c.LiveLoad.Inventory.Accumulator).Hero()
		}
		out = append(out, p)
		outIDs = append(outIDs, id)
	}
	return OwnParty(out), outIDs
}

// Keep compatibility views on the same current inventory as Carry. They own
// detached containers; clearing local handles at a boundary updates both views.
func setCarryHoldings(p *PartyMember) {
	c := p.Carry
	if c == nil {
		return
	}
	p.Worn, p.WornItems = c.Equipped, c.EquippedItems
	for i := range p.WornItems {
		p.WornItems[i] = p.WornItems[i].Clone()
	}
	p.Carried, p.CarriedItems = nil, nil
	if c.Items != nil {
		p.Carried = append(make([]uint16, 0, len(c.Items)), c.Items...)
	}
	if c.ItemInstances != nil {
		p.CarriedItems = make([]sim.ItemInstance, len(c.ItemInstances))
	}
	for i := range p.CarriedItems {
		p.CarriedItems[i] = c.ItemInstances[i].Clone()
	}
}
