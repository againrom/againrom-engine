package sav

import (
	"encoding/binary"
	"fmt"
	"slices"
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
	// CreatureSpells are the three class spell slot pairs the record's
	// profile places in its order block (creatureSpellWindow). Absent when
	// the record has no complete block.
	CreatureSpells    [CreatureSpellSlots]SavedCreatureSpell
	HasCreatureSpells bool
	// Corpse marks a dead-list record of decay stage 2..4 outside the current
	// actor population; Identity joins it to the World's dead record.
	Corpse   bool
	Identity uint32
}

// CreatureSpellSlots is the number of class spell slots in an order block.
const CreatureSpellSlots = 3

// Order block offsets of the three spell ids and the three scaled
// probabilities in the first game's records and every engine document
// (SAV-1066).
const (
	CreatureSpellIDOffset        = 0x78
	CreatureSpellThresholdOffset = 0x84
)

// SavedCreatureSpell is one slot pair read raw from the order block.
type SavedCreatureSpell struct{ ID, Threshold uint32 }

// creatureSpellWindow is where a profile's Unit record holds the three slot
// pairs: the raw member, its width and the offsets of the ids and the
// thresholds.
type creatureSpellWindow struct {
	block           string
	width           int
	ids, thresholds int
}

// firstCreatureSpells is the first game's order block U158, the block every
// engine document writes; secondCreatureSpells is the second game's command
// block, the ids at +0x7c and the thresholds at +0x88 (R2-ENGINE-652,
// R2-ENGINE-832).
var (
	firstCreatureSpells  = creatureSpellWindow{block: "U158", width: 148, ids: CreatureSpellIDOffset, thresholds: CreatureSpellThresholdOffset}
	secondCreatureSpells = creatureSpellWindow{block: "URaw184", width: 184, ids: 0x7c, thresholds: 0x88}
)

// slots reads the window's pairs from b; false when b is not the block's
// width.
func (w creatureSpellWindow) slots(b []byte) ([CreatureSpellSlots]SavedCreatureSpell, bool) {
	var out [CreatureSpellSlots]SavedCreatureSpell
	if len(b) != w.width {
		return out, false
	}
	for i := range out {
		out[i] = SavedCreatureSpell{ID: binary.LittleEndian.Uint32(b[w.ids+4*i:]),
			Threshold: binary.LittleEndian.Uint32(b[w.thresholds+4*i:])}
	}
	return out, true
}

// put writes slots into b, a block of the window's width.
func (w creatureSpellWindow) put(b []byte, slots [CreatureSpellSlots]SavedCreatureSpell) {
	for i, s := range slots {
		binary.LittleEndian.PutUint32(b[w.ids+4*i:], s.ID)
		binary.LittleEndian.PutUint32(b[w.thresholds+4*i:], s.Threshold)
	}
}

// CreatureSpellBlock is the raw member and width of the block holding p's
// creature spell slots.
func (p *Profile) CreatureSpellBlock() (string, int) {
	return p.creatureSpells.block, p.creatureSpells.width
}

// PutCreatureSpells writes slots into block, p's CreatureSpellBlock, at
// p's offsets.
func (p *Profile) PutCreatureSpells(block []byte, slots [CreatureSpellSlots]SavedCreatureSpell) error {
	if len(block) != p.creatureSpells.width {
		return fmt.Errorf("sav: %s block of %d bytes, want %d", p.creatureSpells.block, len(block), p.creatureSpells.width)
	}
	p.creatureSpells.put(block, slots)
	return nil
}

func (a ActorSpellbook) KnownSpells() uint32 { return savedSpellMembership(a.Spells) }

// ActorSpellbooks follows every Player under one shared archive state
// (SAV-DOC-053, SAV-ROSTER-024). Repeated actor references name one actor;
// shared Spell references produce detached per-actor values (DIV-691).
// A malformed late Player cannot publish a partial book population. The dead
// list's corpse records follow, marked Corpse, for the corpse book step.
func (f *File) ActorSpellbooks(current ...uint16) ([]ActorSpellbook, error) {
	doc, _, err := f.exactDocument()
	if err != nil {
		return nil, err
	}
	seen := make(map[*Record]bool)
	records := currentActorRecords(doc.players, current, doc.dead)
	population := len(records)
	for _, r := range doc.dead {
		if r != nil && !slices.Contains(records, r) && r.value("Stage") >= 2 && r.value("Stage") <= 4 {
			records = append(records, r)
		}
	}
	var out []ActorSpellbook
	for i, r := range records {
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
			HasSpellbook: r.value("HasSpellbook") == 1, SpellCount: r.Counts["Spells"], Spells: spells,
			Corpse: i >= population, Identity: r.value("Identity")}
		window := r.profile().creatureSpells
		book.CreatureSpells, book.HasCreatureSpells = window.slots(r.Raw[window.block])
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
		if bound := r.profile().maxSpellID; id == 0 || id > bound || int(id) != slot {
			return nil, fmt.Errorf("%w: actor at %d slot %d has spell ID %d (supported IDs 1..%d must match their slot)", ErrSpellbook, r.Off, slot, id, bound)
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
		if spell.ID > 0 && uint32(spell.ID) <= SecondGame.maxSpellID {
			mask |= uint32(1) << spell.ID
		}
	}
	return mask
}
