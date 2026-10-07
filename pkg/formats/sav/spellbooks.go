package sav

import (
	"encoding/binary"
	"fmt"
)

// ActorSpellbook is the narrow Unit-owned book projection, independent of
// Character and first-Player party policy. HP retains its signed wire meaning.
type ActorSpellbook struct {
	Off          int
	RuntimeID    uint32
	MapUnitID    uint16
	Cell         uint16
	Stage        uint8
	HP           int16
	HasSpellbook bool
	SpellCount   int
	Spells       []SavedSpell
	// CreatureSpells are the three class spell slot pairs the order block
	// carries raw at +0x78..+0x8c (SAV-1066). Absent when the record has no
	// complete order block.
	CreatureSpells    [CreatureSpellSlots]SavedCreatureSpell
	HasCreatureSpells bool
}

// CreatureSpellSlots is the number of class spell slots in an order block.
const CreatureSpellSlots = 3

// Order block offsets of the three spell ids and the three scaled
// probabilities (SAV-1066).
const (
	CreatureSpellIDOffset        = 0x78
	CreatureSpellThresholdOffset = 0x84
)

// SavedCreatureSpell is one slot pair read raw from the order block.
type SavedCreatureSpell struct{ ID, Threshold uint32 }

func (a ActorSpellbook) KnownSpells() uint32 { return savedSpellMembership(a.Spells) }

// ActorSpellbooks follows every Player under one shared archive state
// (SAV-DOC-053, SAV-ROSTER-024). Repeated actor references name one actor;
// shared Spell references produce detached per-actor values (DIV-691).
// A malformed late Player cannot publish a partial book population.
func (f *File) ActorSpellbooks(current ...uint16) ([]ActorSpellbook, error) {
	doc, _, err := f.exactDocument()
	if err != nil {
		return nil, err
	}
	seen := make(map[*Record]bool)
	var out []ActorSpellbook
	for _, r := range currentActorRecords(doc.players, current, doc.dead) {
		if seen[r] {
			continue
		}
		seen[r] = true
		if !groundClass(r.Class, "Unit") {
			return nil, fmt.Errorf("sav: Player actor at %d is %s, not a Unit", r.Off, r.Class)
		}
		position := r.Raw["Block12"]
		if len(position) != 12 {
			return nil, fmt.Errorf("sav: actor at %d has no complete Position", r.Off)
		}
		spells, err := recordSpells(r)
		if err != nil {
			return nil, err
		}
		book := ActorSpellbook{Off: r.Off, RuntimeID: r.value("RuntimeID"),
			MapUnitID: uint16(r.value("T08")), Cell: u16(position, 0),
			Stage: uint8(r.value("Stage")), HP: int16(r.value("Health")),
			HasSpellbook: r.value("HasSpellbook") == 1, SpellCount: r.Counts["Spells"], Spells: spells}
		if order := r.Raw["U158"]; len(order) == 148 {
			book.HasCreatureSpells = true
			for i := range book.CreatureSpells {
				book.CreatureSpells[i] = SavedCreatureSpell{
					ID:        binary.LittleEndian.Uint32(order[CreatureSpellIDOffset+4*i:]),
					Threshold: binary.LittleEndian.Uint32(order[CreatureSpellThresholdOffset+4*i:])}
			}
		}
		out = append(out, book)
	}
	return out, nil
}

// The walker bounds presence, count and reference widths. This projection
// validates semantic membership before any caller can subscript a sim book.
func recordSpells(r *Record) ([]SavedSpell, error) {
	var spells []SavedSpell
	for i, ref := range r.SpellSlots {
		if ref == nil {
			continue
		}
		slot := i + 1
		if ref.Class != "Spell" {
			return nil, fmt.Errorf("%w: actor at %d slot %d refers to %s", ErrSpellbook, r.Off, slot, ref.Class)
		}
		id := ref.value("S08")
		if id == 0 || id > 28 || int(id) != slot {
			return nil, fmt.Errorf("%w: actor at %d slot %d has spell ID %d (supported IDs 1..28 must match their slot)", ErrSpellbook, r.Off, slot, id)
		}
		spells = append(spells, SavedSpell{Slot: slot, ID: uint8(id),
			Range: uint8(ref.value("S09")), Defensive: uint8(ref.value("S0A")),
			ManaCost: uint16(ref.value("S0C")), ArchiveIndex: ref.Index, Key: ref.value("This")})
	}
	return spells, nil
}

func savedSpellMembership(spells []SavedSpell) uint32 {
	var mask uint32
	for _, spell := range spells {
		if spell.ID > 0 && spell.ID <= 28 {
			mask |= uint32(1) << spell.ID
		}
	}
	return mask
}
