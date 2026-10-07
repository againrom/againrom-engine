package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// A current book owns S08/S09/S0A/S0C (MAGIC-SPELL-001). A shared Spell
// object can be reused only when every incoming edge asks for that same value.
// In particular an independently changed book must not rewrite a weapon Spell.
func projectCurrentBookGraph(b *generatedDocumentBuilder, state *SnapshotSAVDocument, world *sim.World, ordinaryRange bool) error {
	b.doc.Objects = slices.Clone(b.doc.Objects)
	entities := map[sim.EntityID]sim.Entity{}
	for _, e := range world.Entities() {
		entities[e.ID] = e
	}
	incoming := savedDocumentIncoming(&b.doc)
	type request struct {
		actor int
		slot  int
		old   uint16
		value sim.SourceItemSpell
	}
	var requests []request
	uses := map[uint16][]sim.SourceItemSpell{}
	roots := map[sim.EntityID][28]sim.SavedObjectID{}
	indices := map[sim.SavedObjectID]uint16{}
	if registry := world.SavedObjects(); registry != nil {
		for _, root := range registry.BookRoots {
			roots[root.Entity] = root.Slots
		}
		if state.Objects != nil {
			indices = savedObjectIndices(state.Objects)
		}
	}
	for _, a := range state.Actors {
		if a.Retired {
			continue
		}
		e, ok := entities[a.EntityID]
		if !ok {
			return fmt.Errorf("current book actor %d is absent", a.EntityID)
		}
		if e.Book.State == sim.BookLegacy {
			e.Book.State = sim.BookPresent
			e.KnownSpells &= 0x1ffffffe
			for _, rule := range world.Spells() {
				if rule.ID >= 1 && rule.ID <= 28 && e.KnownSpells&(1<<rule.ID) != 0 {
					e.Book.Slots[rule.ID-1].ManaCost = uint16(rule.ManaCost)
					if rule.Defensive {
						e.Book.Slots[rule.ID-1].Defensive = 1
					}
				}
			}
			if !ordinaryRange || world.Rules().HasSpellFormulas() {
				sim.RefreshBook(world.Rules(), &e, world.Spells())
			} else {
				sim.RefreshOriginalBook(&e, world.Spells())
			}
		}
		if err := e.Book.Validate(e.KnownSpells); err != nil {
			return err
		}
		r := &b.doc.Objects[a.ObjectIndex-1]
		var old []uint16
		for _, refs := range r.RefSlots {
			if refs.Name == "Spells" {
				old = refs.Objects
			}
		}
		for _, ref := range old {
			if ref != 0 {
				incoming[ref]--
			}
		}
		savedObjectSetValue(r, "HasSpellbook", 0)
		r.RefSlots = slices.DeleteFunc(slices.Clone(r.RefSlots), func(v sav.DocumentRefsData) bool { return v.Name == "Spells" })
		r.Counts = slices.DeleteFunc(slices.Clone(r.Counts), func(v sav.DocumentCountData) bool { return v.Name == "Spells" })
		if !e.Book.HasInstances() {
			r.Values = slices.DeleteFunc(slices.Clone(r.Values), func(v sav.DocumentValueData) bool { return v.Name == "SpellsHeader" })
			continue
		}
		savedObjectSetValue(r, "HasSpellbook", 1)
		if _, err := savedStructureValue(r, "SpellsHeader"); err != nil {
			// No native field owns this opaque constructor word. Use the same
			// initial record as the existing fresh actor/book producer.
			initial := mustNewRecord(r.Class, "HasSpellbook")
			header, err := savedStructureValue(&initial, "SpellsHeader")
			if err != nil {
				return err
			}
			savedObjectSetValue(r, "SpellsHeader", header)
		}
		var slots []uint16
		for slot, value := range e.Book.Slots {
			if e.KnownSpells&(1<<uint(slot+1)) == 0 {
				continue
			}
			for len(slots) <= slot {
				slots = append(slots, 0)
			}
			previous := uint16(0)
			if slot < len(old) {
				previous = old[slot]
			}
			if id := roots[a.EntityID][slot]; id != 0 {
				previous = indices[id]
				if previous == 0 {
					return fmt.Errorf("current book root has no ordinary Spell binding")
				}
			}
			v := sim.SourceItemSpell{Present: true, ID: uint8(slot + 1), Range: value.Range, Defensive: value.Defensive, ManaCost: value.ManaCost}
			if previous != 0 {
				if int(previous) > len(b.doc.Objects) || b.doc.Objects[previous-1].Class != "Spell" {
					return fmt.Errorf("book slot has invalid Spell binding")
				}
				uses[previous] = append(uses[previous], v)
			}
			requests = append(requests, request{int(a.ObjectIndex) - 1, slot, previous, v})
		}
		savedObjectSetRefs(r, "Spells", slots, true)
		mustSetCount(r, "Spells", uint32(len(slots)+1))
	}
	for _, q := range requests {
		index := q.old
		unchanged := false
		if index != 0 {
			previous, err := savedSpellRecord(&b.doc.Objects[index-1])
			if err != nil {
				return err
			}
			unchanged = previous.Value == q.value
		}
		exclusive := index != 0 && incoming[index] == 0
		for _, other := range uses[index] {
			exclusive = exclusive && other == q.value
		}
		if unchanged {
			// Other owners already reference these exact current operands.
		} else if exclusive {
			this, err := savedStructureValue(&b.doc.Objects[index-1], "This")
			if err != nil {
				return err
			}
			b.doc.Objects[index-1] = savedCurrentSpellRecord(sim.SavedSpellObject{Value: q.value, This: this})
		} else {
			var err error
			index, err = b.spell(q.value)
			if err != nil {
				return err
			}
		}
		r := &b.doc.Objects[q.actor]
		for i := range r.RefSlots {
			if r.RefSlots[i].Name == "Spells" {
				r.RefSlots[i].Objects[q.slot] = index
			}
		}
	}
	return nil
}
