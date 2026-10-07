package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// declareItemWeights resolves what every item code the world can name weighs
// and declares the answers into it.
//
// IT IS THE SEAM BETWEEN THE TABLES AND THE SIMULATION. pkg/sim holds no
// definition table and may not open one — the weight arithmetic is
// floating-point and pkg/data sits below the determinism wall — so the world
// carries a code-to-weight table instead, and this function is where that
// table is filled.
//
// THE POPULATION IS EVERY PRODUCER OF A CODE INTO A WORLD, and it is complete
// as follows. A code reaches a world's containers, its equipment slots or its
// ground through exactly four doors:
//
//   - AN ENTITY'S EQUIPMENT SLOTS, twelve per entity, written at construction
//     from a placement's worn cells or a party member's own doll.
//   - AN ENTITY'S CONTAINER, written at construction from the map's own stock
//     records and the class rows' loadouts.
//   - A SACK ON THE GROUND, written at construction from the map's type-8
//     loot section, and later by any drop — a drop moves a code out of a
//     container this pass already reached.
//   - THE COMPILED MISSION SCRIPT, whose add-item and take-item instants name
//     a code as a literal. This is the one door whose codes need not appear
//     anywhere else in the world at construction, which is why the script is
//     walked rather than only the world.
//
// Every runtime act that moves an item — a pick-up, a drop, a transfer, an
// equip, a corpse's loot, a give-all pour — moves a code that was already
// inside one of those four, so no runtime path can introduce a code this pass
// did not see. The one act that CAN is a caller replacing an actor's whole
// holdings from outside the mission (sim.ReplaceStock, used by the campaign's
// own between-mission carry), and that caller declares its own codes;
// sim.DeclareItemWeights is additive precisely so it can.
//
// WHAT THIS INSTRUMENT CANNOT SEE: a code whose class field names no equipment
// slot at all — a quest document, a carried token — resolves to no item and is
// declared no weight, so it weighs nothing. That is data.ItemCodeWeight's own
// answer rather than this function's, and it is the same answer the simulation
// gives a code nobody declared.
//
// A TABLE THAT CANNOT RESOLVE A CODE COSTS ONLY THAT CODE. The whole point of
// this pass is that a world still loads over a partial install: a code the
// tables refuse is skipped and the world carries no weight for it, exactly as
// a world built before this story carried none for anything. A refusal here
// would make an unreadable Armors collection fail a mission that never equips
// armour.
func declareItemWeights(w *sim.World, t *Table) {
	if w == nil || t == nil {
		return
	}
	codes := make(map[uint16]struct{})
	add := func(c uint16) {
		if c != 0 {
			codes[c] = struct{}{}
		}
	}
	for _, e := range w.Entities() {
		if worn, ok := w.Equipped(e.ID); ok {
			for _, c := range worn {
				add(c)
			}
		}
		if held, ok := w.CarriedStacks(e.ID); ok {
			for _, st := range held {
				add(st.Code)
			}
		}
	}
	for _, s := range w.Sacks() {
		for _, c := range s.Items {
			add(c)
		}
	}
	if s := w.Script(); s != nil {
		for _, in := range s.Instants() {
			add(in.Item)
		}
	}
	ordered := make([]uint16, 0, len(codes))
	for c := range codes {
		ordered = append(ordered, c)
	}
	DeclareCodeWeights(w, t, ordered)
}

// DeclareCodeWeights resolves an explicit list of item codes against t and
// declares the answers into w.
//
// IT EXISTS FOR THE ONE PRODUCER declareItemWeights above cannot see: a caller
// that replaces an actor's whole holdings from OUTSIDE the mission, through
// sim.ReplaceStock, with codes that were in no container, no slot, no sack and
// no script instant when the world was built. pkg/game has two — the original
// save's own restored loadout, and the party's fallback starting weapon — and
// each hands its own codes here rather than this package guessing at them.
//
// It is additive and idempotent, sim.DeclareItemWeights' own standing, so a
// caller may declare codes the world already carries.
func DeclareCodeWeights(w *sim.World, t *Table, codes []uint16) {
	if w == nil || t == nil || len(codes) == 0 {
		return
	}
	// The codes are sorted and de-duplicated before they are resolved, so the
	// list handed over is in one order whatever order the caller found them
	// in. sim's own normalisation sorts too, so this is not what makes the
	// world deterministic — it is what makes a resolution FAILURE
	// deterministic, since a partial table refuses the same codes in the same
	// order every time and the world that results is the same world.
	ordered := append([]uint16(nil), codes...)
	sortCodes(ordered)

	out := make([]sim.ItemWeight, 0, len(ordered))
	var last uint16
	for _, c := range ordered {
		if c == 0 || c == last {
			continue
		}
		last = c
		weight, ok, err := data.ItemCodeWeight(data.ItemCode(c), t.Shapes, t.Materials, t.Armors, t.Shields, t.Weapons)
		if err != nil || !ok {
			continue
		}
		out = append(out, sim.ItemWeight{Code: c, Weight: weight})
	}
	// The error is dropped for the reason a refusal is: the only error
	// DeclareItemWeights returns is two entries naming one code at two
	// weights, and this list is built from a set of codes each resolved once.
	_ = w.DeclareItemWeights(out)
	DeclareSourceConstructors(w, t)
}

// sortCodes is an ascending insertion sort over a code list. It is written out
// rather than reached through sort.Slice because the lists it runs on are the
// distinct codes of one mission — the largest shipped map reaches a few dozen
// — and an insertion sort states the order it produces without a comparator
// closure to read.
func sortCodes(c []uint16) {
	for i := 1; i < len(c); i++ {
		v := c[i]
		j := i - 1
		for j >= 0 && c[j] > v {
			c[j+1] = c[j]
			j--
		}
		c[j+1] = v
	}
}

// PartyLoad is one party member's carried load, for a sheet drawn OUTSIDE a
// mission.
//
// A member between missions has a worn set and a carried list and no entity
// anywhere, so there is no Entity.Load to read. This resolves the two sums off
// his own two arrays and hands them to sim.CarriedLoad, which is the same
// single statement of the law a live actor's load goes through. No arithmetic
// of the law is restated here.
//
// An explicit signed instance weight wins, including zero. Only a native or
// legacy instance without that field uses the table, then zero if unresolved.
//
// THE CARRIED LIST IS A FLAT CODE LIST and each occurrence is one unit, which
// is what sim's own container fold makes of it: a count of n is n elements
// naming one code, and summing the list is summing weight x count over the
// folded stacks.
func PartyLoad(p PartyMember, t *Table) int32 {
	if p.Carry != nil && p.Carry.LiveLoad != nil {
		return p.Carry.LiveLoad.Load
	}
	weight := func(item sim.ItemInstance) int32 {
		if item.WeightPresent {
			return int32(item.Weight)
		}
		code := item.Code
		if code == 0 || t == nil {
			return 0
		}
		w, ok, err := data.ItemCodeWeight(data.ItemCode(code), t.Shapes, t.Materials, t.Armors, t.Shields, t.Weapons)
		if err != nil || !ok {
			return 0
		}
		return w
	}
	var worn, container int32
	for _, item := range MemberItemEquipment(p, t) {
		worn += weight(item)
	}
	for _, item := range MemberCarriedItems(p, t) {
		container += weight(item)
	}
	return sim.CarriedLoad(worn, container)
}
