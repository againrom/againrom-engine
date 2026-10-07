package sav

import (
	"fmt"
	"slices"
)

const restFine = 0x80

// Actor is a Unit-derived record reached through a Player's actor list.
// Raw bytes elsewhere in the document never introduce actors.
type Actor struct {
	Off              int
	Class            string
	Cell             uint16
	FineX, FineY     uint8
	Field6           uint16
	State, RuntimeID uint32
	MapUnitID        uint16
	// OwnerSlot resolves the Token owner key to an already-read Player.
	// Null, absent and not-yet-bound keys resolve to zero (SAV-PTRMAP-035).
	OwnerSlot uint16
	Stage     uint8
	HP        int16
	// Facing is the current mover byte, not Unit's face/portrait selector.
	// SAV-UNITPROG-156 restores the raw mover; MOVE-TURN-031 names byte zero.
	Facing uint8
}

// Dead also excludes dying owner-graph records from a living position join.
func (a Actor) Dead() bool { return a.RuntimeID == 0 || a.Stage != 0 || a.HP <= 0 }

// Dying names the first dying stage while the actor still belongs to a
// Player list. This is distinct from the top-level dead manager. The stage
// and health remain independent stored values (SAV-DEADLOAD-126/128).
func (a Actor) Dying() bool  { return a.RuntimeID != 0 && a.Stage == 1 && a.HP <= 0 }
func (a Actor) Row() int     { return int(a.Cell >> 8) }
func (a Actor) Col() int     { return int(a.Cell & 0xff) }
func (a Actor) AtRest() bool { return a.FineX == restFine && a.FineY == restFine }

// actorRecords traverses graph roots once. First-stage dying records detached
// from a native Group retain the complete Unit/Human graph in the dead list;
// later corpses keep the separately bounded dead-manager import.
func actorRecords(players []*Record, dead ...[]*Record) []*Record {
	return currentActorRecords(players, nil, dead...)
}

func currentActorRecords(players []*Record, current []uint16, dead ...[]*Record) []*Record {
	var out []*Record
	seen := map[*Record]bool{}
	add := func(r *Record) {
		if r != nil && !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	for _, p := range players {
		if p != nil {
			for _, r := range p.Refs["Actors"] {
				add(r)
			}
		}
	}
	for _, roots := range dead {
		for _, r := range roots {
			if r != nil && (slices.Contains(current, r.Index) || r.value("Stage") == 1 && int16(r.value("Health")) <= 0 && r.value("RuntimeID") != 0) {
				add(r)
			}
		}
	}
	return out
}

func ownerActors(players []*Record, dead ...[]*Record) ([]Actor, error) {
	return currentOwnerActors(players, nil, dead...)
}

func currentOwnerActors(players []*Record, current []uint16, dead ...[]*Record) ([]Actor, error) {
	var out []Actor
	owners := make(map[uint32]*Record)
	for _, player := range players {
		key := player.value("This")
		if key == 0 {
			continue
		}
		if prior := owners[key]; prior != nil && prior != player {
			return nil, fmt.Errorf("sav: duplicate Player identity %#x", key)
		}
		owners[key] = player
	}
	seen := make(map[*Record]bool)
	for _, r := range currentActorRecords(players, current, dead...) {
		if r == nil || seen[r] {
			continue
		}
		seen[r] = true
		if !groundClass(r.Class, "Unit") {
			continue
		}
		p := r.Raw["Block12"]
		if len(p) != 12 {
			return nil, fmt.Errorf("sav: incomplete actor Position at %d", r.Off)
		}
		mover := r.Raw["U154"]
		if len(mover) != 180 {
			return nil, fmt.Errorf("sav: incomplete actor mover at %d", r.Off)
		}
		var owner uint16
		// The reference, not its enclosing list or the ALM placement,
		// names the owner (SAV-OWNER-048). Token load resolves immediately;
		// a later Player cannot retroactively bind a missing key.
		if player := owners[r.value("Reference")]; player != nil && player.Off < r.Off {
			owner = uint16(player.value("Slot"))
		}
		out = append(out, Actor{Off: r.Off, Class: r.Class, Cell: u16(p, 0),
			FineX: p[4], FineY: p[5], Field6: u16(p, 6), State: u32(p, 8),
			RuntimeID: r.value("RuntimeID"), MapUnitID: uint16(r.value("T08")),
			OwnerSlot: owner, Stage: uint8(r.value("Stage")), HP: int16(r.value("Health")), Facing: mover[0]})
	}
	return out, nil
}

// SetActorPosition writes both coordinate representations and the fine bytes
// of the saved raw Position object (SAV-TOKENPOS-074).
func (f *File) SetActorPosition(i int, row, col int, fineX, fineY uint8) error {
	if i < 0 || i >= len(f.Actors) {
		return fmt.Errorf("sav: actor %d of %d", i, len(f.Actors))
	}
	if row < 0 || row > 0xff || col < 0 || col > 0xff {
		return fmt.Errorf("sav: cell (%d,%d) outside the 256x256 plane", col, row)
	}
	cell := uint16(row)<<8 | uint16(col)
	a := &f.Actors[i]
	put16(f.Body, a.Off, cell)
	put16(f.Body, a.Off+2, cell)
	f.Body[a.Off+4] = fineX
	f.Body[a.Off+5] = fineY
	a.Cell, a.FineX, a.FineY = cell, fineX, fineY
	return nil
}

// ActorByMapUnitID returns only an unambiguous living owner-graph match.
// Consumers that must distinguish ambiguity from absence validate the full join.
func (f *File) ActorByMapUnitID(id uint16) (int, bool) {
	found := -1
	for i, a := range f.Actors {
		if a.MapUnitID == id && !a.Dead() {
			if found >= 0 {
				return 0, false
			}
			found = i
		}
	}
	return found, found >= 0
}
