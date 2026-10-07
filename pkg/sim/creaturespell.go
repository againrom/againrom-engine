package sim

import (
	"bytes"
	"encoding/binary"
)

// CreatureSpellSlots is the number of class spell slots a creature carries.
const CreatureSpellSlots = 3

// CreatureSpellScale multiplies a class Probability column into the stored
// threshold the per-slot draw compares against (UNIT-SPELL-007).
const CreatureSpellScale = 0x147

// creatureDrawMax is the top of the draw: the CRT generator's 0..0x7fff
// (AI-341).
const creatureDrawMax = 0x7fff

// CreatureSpell is one class spell slot: the spell id and the Probability
// column scaled by CreatureSpellScale. A zero ID is an empty slot.
type CreatureSpell struct {
	ID, Threshold uint32
}

// CreatureSpellFor builds a slot from a class's Spell and Probability cells.
// An empty (non-positive) spell cell leaves the slot empty. The threshold
// keeps the signed product, so a probability below one never draws a hit.
func CreatureSpellFor(spell, probability int32) CreatureSpell {
	if spell <= 0 {
		return CreatureSpell{}
	}
	return CreatureSpell{ID: uint32(spell), Threshold: uint32(probability * CreatureSpellScale)}
}

func (c CreatureSpell) present() bool { return c.ID != 0 || c.Threshold != 0 }

// hasCreatureSpells reports whether any slot of the class spellbook is set.
func (e Entity) hasCreatureSpells() bool {
	for _, s := range e.CreatureSpells {
		if s.present() {
			return true
		}
	}
	return false
}

// bookCaster reports whether e may cast from its book at all: a mage, or a
// creature whose class spellbook holds slots. Only a mage pays mana
// (MAGIC-CAST-003): a creature casts for nothing and is not gated by a pool.
func bookCaster(e Entity) bool {
	return isMage(e) || e.Book.HasInstances() && e.hasCreatureSpells()
}

// creatureSpellPick is the per-slot draw (AI-341): every non-empty slot draws
// once, in slot order, and a draw below its threshold selects the slot's
// spell. A later selection replaces an earlier one. Zero means none selected.
func (w *World) creatureSpellPick(e Entity) uint32 {
	var pick uint32
	for _, s := range e.CreatureSpells {
		if s.ID == 0 {
			continue
		}
		if w.rng.uniform(creatureDrawMax) < int32(s.Threshold) {
			pick = s.ID
		}
	}
	return pick
}

// The order block carries the three class spell slots in one window: the ids
// at 0x78 and the scaled probabilities at 0x84. The World-held order keeps the
// loaded bytes, and Entity.CreatureSpells is the slots' only in-memory truth.
// The byte form writes the window as zero while it equals the entity's slots,
// so a loaded world and its own reloaded save hash alike.
const (
	orderSlotIDStart        = 0x78
	orderSlotThresholdStart = 0x84
	orderSlotWindowStart    = 0x78
	orderSlotWindowEnd      = 0x90
)

// slotWindow is the order bytes 0x78..0x8f the entity's slots project to.
func (e Entity) slotWindow() (win [orderSlotWindowEnd - orderSlotWindowStart]byte) {
	for i, s := range e.CreatureSpells {
		binary.LittleEndian.PutUint32(win[orderSlotIDStart-orderSlotWindowStart+4*i:], s.ID)
		binary.LittleEndian.PutUint32(win[orderSlotThresholdStart-orderSlotWindowStart+4*i:], s.Threshold)
	}
	return win
}

// maskedOrderRaw is the order bytes as the byte form writes them: a window
// equal to the holder's own slots is written as zero.
func (w *World) maskedOrderRaw(o SavedActorOrder) [144]byte {
	raw := o.Raw
	i := indexOfEntity(w.entities, o.Entity)
	if i < 0 || !w.entities[i].hasCreatureSpells() {
		return raw
	}
	if win := w.entities[i].slotWindow(); bytes.Equal(raw[orderSlotWindowStart:orderSlotWindowEnd], win[:]) {
		clear(raw[orderSlotWindowStart:orderSlotWindowEnd])
	}
	return raw
}

// fillOrderSlotWindows restores the window of each held order whose holder has
// slots and whose window was written as zero.
func (w *World) fillOrderSlotWindows() {
	if w.savedGroups == nil {
		return
	}
	for k := range w.savedGroups.Orders {
		o := &w.savedGroups.Orders[k]
		i := indexOfEntity(w.entities, o.Entity)
		if i < 0 || !w.entities[i].hasCreatureSpells() {
			continue
		}
		win := o.Raw[orderSlotWindowStart:orderSlotWindowEnd]
		if !bytes.Equal(win, make([]byte, len(win))) {
			continue
		}
		slots := w.entities[i].slotWindow()
		copy(win, slots[:])
	}
}
