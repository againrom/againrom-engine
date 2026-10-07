package sav

import (
	"encoding/binary"
	"fmt"
	"slices"
)

// ActorReference is a file-local identity lookup at a particular LOAD boundary.
// A missing key stays unresolved; the enclosing Player is not a fallback.
type ActorReference struct {
	Key          uint32
	ArchiveIndex uint16
	Class        string
	PlayerSlot   uint16
	PlayerF58    uint32
	Resolved     bool
}

// ActorGroup preserves the named Group fields separately from actor ownership.
// AI and the two lists are separate restored values, never pointer identities.
// Ordinal is this graph's one-based inline Group index, never an archive index.
type ActorGroup struct {
	Ordinal          uint32
	Off              int
	Selector         uint32
	Reference, Owner ActorReference
	AI               [80]byte
	Words, AIWords   []uint16
	Members          []uint16
}

// ActorOffMapFlag is U4C bit 3: opcode 16 sets it when it unlinks an actor
// from the on-map list, opcode 17 clears it, and LOAD skips a flagged actor
// when it rebuilds that list (TRIG-OFFMAP-041, MOVE-TICK-017).
const ActorOffMapFlag = 0x08

// ActorRecord is one exact source object. Actor.OwnerSlot is the effective
// owner after Player suffixes; TokenOwnerSlot is the earlier Token lookup.
// Group is a one-based ActorGroup ordinal. Zero is no final membership.
type ActorRecord struct {
	Actor
	// CurrentEntity is explicit adapter metadata: this archive node is still
	// owned by the current Entity population even at a late lifecycle stage.
	CurrentEntity  bool
	CurrentOffMap  bool // explicit current presence policy, absent from ordinary Position
	TerminalActor  bool // a current terminal DeadActors root: never an Entity, whatever its Stage
	ArchiveIndex   uint16
	Identity       uint32
	Group          uint32
	TokenOwnerSlot uint16
	Owner          ActorReference
	TypeID         uint16
	DefRow         uint8
	TokenSize      uint8
	Domain         uint8
	Face           uint8
	ClassSelector  uint8
	DisplayBacking uint32
	Character      Character
	Order          [148]byte
	Patrol         []uint16
	ActorState     uint32
	HasOrder       bool
	DyingTimer     int8 // Unit +0x6c, serialized independently (SAV-DEADLOAD-126).
	// Byte locators only; consumers can read independent layouts at these
	// substructure starts without using decoded field values.
	ControlOff, StateOff int

	// Mover is the raw 180-byte block Unit's `*(+0x154)` pointer names
	// (SAV-UNITPROG-156). Two byte fields inside it are named: Mover[1] is
	// DesiredFacing, Mover[5] is the terrain passability mask (MOVE-TURN-031,
	// TERR-PASS-051); the rest, including the turn-progress and turn-cost
	// bytes MOVE-TURN-031 also names, stay raw pending a byte-width citation
	// (DIV-950). StaticRoute/DynamicRoute are the two embedded u16 lists at
	// `+0x15c`/`+0x178`, each element the packed cell `(y<<8)|x`
	// MOVE-ROUTE-004 defines and SAV-630 confirms by both the save and the
	// load arm; an empty list is the corpus mode (85.0%/94.9%, SAV-630).
	HasMover     bool
	Mover        [180]byte
	StaticRoute  []uint16
	DynamicRoute []uint16
}

// DesiredFacing is Mover[1], the direction the unit is turning toward, in
// 32-unit steps over 256 (MOVE-TURN-031). It differs from Actor.Facing
// (Mover[0]), the unit's current facing.
func (a ActorRecord) DesiredFacing() uint8 { return a.Mover[1] }

// PassabilityMask is Mover[5], the byte `TERR-PASS-051` reads as
// `block[cell] & mask` to test the mover's own footprint. 0x41 (ground, the
// constructor default), 0x44 and 0x82 (TERR-MOVE-057) are read across the
// whole preserved corpus (gameversions/saves); this is what the corpus
// happens to hold, not a closed set TERR-PASS-051 itself names, so a fourth
// value is not refused here on the strength of this list alone.
func (a ActorRecord) PassabilityMask() uint8 { return a.Mover[5] }

// RotationSpeed is Mover[0xa], the Data.bin-sourced turn rate MOVE-TURN-031
// names (Units slot 9, Humans slot 7); shipped values are 8..23 over a
// mover-constructor default of 0x10.
func (a ActorRecord) RotationSpeed() uint8 { return a.Mover[0xa] }

// TurnInProgress reads the full active DWORD at Mover+0xa0 (MOVE-TURN-044).
func (a ActorRecord) TurnInProgress() bool { return binary.LittleEndian.Uint32(a.Mover[0xa0:]) != 0 }

// RouteCell decodes one StaticRoute/DynamicRoute element into its map column
// and row (MOVE-ROUTE-004's packed cell `(y<<8)|x`, confirmed by SAV-630).
func RouteCell(packed uint16) (x, y int) { return int(packed & 0xff), int(packed >> 8) }

// SetActorMoverRoute patches one archived actor's mover block and both saved
// route lists in place, keyed by the archive index ActorRecord.ArchiveIndex
// names. staticRoute/dynamicRoute must match the file's own list lengths — a
// list's element count is written once, by Unit's own store arm
// (SAV-UNITPROG-156), and this setter never rewrites it; a length mismatch
// is refused rather than silently truncated or padded. It does not itself
// judge whether a route cell is in bounds; that is a live-state admission
// question, not a file-patching one.
func (f *File) SetActorMoverRoute(archiveIndex uint16, mover [180]byte, staticRoute, dynamicRoute []uint16) error {
	doc, _, err := f.exactDocument()
	if err != nil {
		return err
	}
	record, ok := doc.objects[archiveIndex]
	if !ok || !groundClass(record.Class, "Unit") {
		return fmt.Errorf("sav: actor %d has no Unit projection to patch", archiveIndex)
	}
	if len(record.Raw["U154"]) != len(mover) {
		return fmt.Errorf("sav: actor %d has no 180-byte mover to patch", archiveIndex)
	}
	routes := []struct {
		name   string
		values []uint16
	}{{"U15C", staticRoute}, {"U178", dynamicRoute}}
	// Every input is checked before the first byte is written (F-5): a
	// refused call must leave the file exactly as it found it, not the mover
	// block patched with one or both route lists still refused, which a
	// caller retrying after fixing only the reported list would otherwise
	// compound into a mover/route pair the source archive never had.
	offsets := make([]int, len(routes))
	for i, route := range routes {
		if record.Counts[route.name] != len(route.values) {
			return fmt.Errorf("sav: actor %d %s has %d elements, not %d", archiveIndex, route.name, record.Counts[route.name], len(route.values))
		}
		if len(route.values) == 0 {
			continue
		}
		off, known := record.ListOff[route.name]
		if !known {
			return fmt.Errorf("sav: actor %d %s has no known file offset", archiveIndex, route.name)
		}
		offsets[i] = off
	}
	copy(record.Raw["U154"], mover[:])
	for i, route := range routes {
		for j, v := range route.values {
			put16(f.Body, offsets[i]+2*j, v)
		}
	}
	return nil
}

// DefinitionRow is the local class-specific LOAD binding, not an ALM lookup.
// Exact Humanoid's null definition is an unsupported constructor boundary.
func (a ActorRecord) DefinitionRow() (uint8, error) {
	switch a.Class {
	case "Unit":
		return a.DefRow, nil
	case "Human":
		if a.TypeID >= 33 {
			return 5, nil
		}
		return a.DefRow, nil
	default:
		return 0, fmt.Errorf("sav: actor %d class %s has no supported definition binding", a.ArchiveIndex, a.Class)
	}
}

type SavedActorGraph struct {
	Actors []ActorRecord
	Groups []ActorGroup
	// CurrentPopulation marks an explicit adapter-owned population, including
	// an empty one. It supplies membership, never ordinary actor field values.
	CurrentPopulation bool
}

// ActorGraph follows the exact shared archive. It does not scan raw bytes for
// actors or infer identity from MapUnitID (SAV-ACTORBIND-544, SAV-GRPLOAD-560).
func (f *File) ActorGraph() (SavedActorGraph, error) {
	return f.CurrentActorGraph(nil)
}

func (f *File) CurrentActorGraph(current []uint16) (SavedActorGraph, error) {
	doc, _, err := f.exactDocument()
	if err != nil {
		return SavedActorGraph{}, err
	}
	return currentActorGraph(doc.players, current, doc.dead)
}

func savedActorGraph(players []*Record, dead ...[]*Record) (SavedActorGraph, error) {
	return currentActorGraph(players, nil, dead...)
}

func currentActorGraph(players []*Record, current []uint16, dead ...[]*Record) (SavedActorGraph, error) {
	out := SavedActorGraph{CurrentPopulation: current != nil}
	actors, err := currentOwnerActors(players, current, dead...)
	if err != nil {
		return out, err
	}
	byOff := make(map[int]Actor, len(actors))
	for _, actor := range actors {
		byOff[actor.Off] = actor
	}
	all, seen := map[uint32][]*Record{}, map[*Record]bool{}
	var collect func(*Record)
	collect = func(r *Record) {
		if r == nil || seen[r] {
			return
		}
		seen[r] = true
		key := r.value("Identity")
		if key == 0 {
			key = r.value("This")
		}
		if key != 0 {
			all[key] = append(all[key], r)
		}
		for _, refs := range r.Refs {
			for _, ref := range refs {
				collect(ref)
			}
		}
	}
	for _, player := range players {
		collect(player)
	}
	for _, roots := range dead {
		for _, r := range roots {
			collect(r)
		}
	}
	resolve := func(key uint32, end int) (ActorReference, error) {
		ref := ActorReference{Key: key}
		for _, target := range all[key] {
			if target.Off >= end {
				continue
			}
			if ref.Resolved {
				return ActorReference{}, fmt.Errorf("sav: ambiguous Group reference %#x at %d", key, end)
			}
			ref.Resolved, ref.ArchiveIndex, ref.Class = true, target.Index, target.Class
			if target.Class == "Player" {
				ref.PlayerSlot = uint16(target.value("Slot"))
				ref.PlayerF58 = target.value("F58")
			}
		}
		return ref, nil
	}
	byIndex, byKey := map[uint16]int{}, map[uint32]uint16{}
	for _, record := range currentActorRecords(players, current, dead...) {
		if record == nil {
			continue
		}
		if record.Index == 0 {
			return SavedActorGraph{}, fmt.Errorf("sav: actor at %d has no archive index", record.Off)
		}
		if prior, known := byIndex[record.Index]; known {
			if out.Actors[prior].Off != record.Off || out.Actors[prior].Identity != record.value("Identity") {
				return SavedActorGraph{}, fmt.Errorf("sav: distinct actors repeat archive index %d", record.Index)
			}
			continue
		}
		key := record.value("Identity")
		if prior, known := byKey[key]; key != 0 && known {
			return SavedActorGraph{}, fmt.Errorf("sav: actors %d and %d repeat identity %#x", prior, record.Index, key)
		}
		byKey[key] = record.Index
		actor, ok := byOff[record.Off]
		if !ok {
			return SavedActorGraph{}, fmt.Errorf("sav: actor %d has no Unit projection", record.Index)
		}
		entry := ActorRecord{Actor: actor, ArchiveIndex: record.Index, Identity: key,
			CurrentEntity: slices.Contains(current, record.Index),
			ControlOff:    record.UnitControlOff, StateOff: record.UnitStateOff,
			TokenOwnerSlot: actor.OwnerSlot, TypeID: uint16(record.value("T0E")), DefRow: uint8(record.value("T0C")),
			TokenSize: uint8(record.value("U49")), Domain: uint8(record.value("U4A")),
			Face: uint8(record.value("U4B")), ClassSelector: uint8(record.value("U4C")), DisplayBacking: record.value("U148"), DyingTimer: int8(record.value("U6C"))}
		entry.HasOrder = len(record.Raw["U158"]) != 0
		if entry.HasOrder && (len(record.Raw["U158"]) != len(entry.Order) || len(record.Raw["U50"]) != 4 || len(record.Raw["U158_90"]) != 2*record.Counts["U158_90"]) {
			return SavedActorGraph{}, fmt.Errorf("sav: actor %d has incomplete order operands: raw %d state %d ring %d/%d", record.Index, len(record.Raw["U158"]), len(record.Raw["U50"]), len(record.Raw["U158_90"]), record.Counts["U158_90"])
		}
		copy(entry.Order[:], record.Raw["U158"])
		if len(record.Raw["U50"]) == 4 {
			entry.ActorState = u32(record.Raw["U50"], 0)
		}
		for i := 0; i < len(record.Raw["U158_90"]); i += 2 {
			entry.Patrol = append(entry.Patrol, u16(record.Raw["U158_90"], i))
		}
		entry.HasMover = len(record.Raw["U154"]) != 0
		if entry.HasMover && len(record.Raw["U154"]) != len(entry.Mover) {
			return SavedActorGraph{}, fmt.Errorf("sav: actor %d has incomplete mover: raw %d", record.Index, len(record.Raw["U154"]))
		}
		copy(entry.Mover[:], record.Raw["U154"])
		for _, route := range []struct {
			name   string
			target *[]uint16
		}{{"U15C", &entry.StaticRoute}, {"U178", &entry.DynamicRoute}} {
			raw := record.Raw[route.name]
			if len(raw) != 2*record.Counts[route.name] {
				return SavedActorGraph{}, fmt.Errorf("sav: actor %d has incomplete %s route list: raw %d count %d", record.Index, route.name, len(raw), record.Counts[route.name])
			}
			for i := 0; i < len(raw); i += 2 {
				*route.target = append(*route.target, u16(raw, i))
			}
		}
		entry.Owner, err = resolve(record.value("Reference"), record.Off+37)
		if err != nil {
			return SavedActorGraph{}, err
		}
		if !actor.Dead() || actor.Dying() || entry.CurrentEntity {
			entry.Character, err = actorCharacter(record)
			if err != nil {
				return SavedActorGraph{}, err
			}
			entry.Character.Basis, err = actorBasisFields(record)
			if err != nil {
				return SavedActorGraph{}, err
			}
		}
		byIndex[record.Index] = len(out.Actors)
		out.Actors = append(out.Actors, entry)
	}
	seenPlayer := map[*Record]bool{}
	for _, player := range players {
		if player == nil || seenPlayer[player] {
			continue
		}
		seenPlayer[player] = true
		first := len(out.Groups)
		for _, record := range player.Groups {
			group := ActorGroup{Ordinal: uint32(len(out.Groups) + 1), Off: record.Off, Selector: record.value("G1C")}
			if len(record.Raw["G3C"]) != len(group.AI) {
				return SavedActorGraph{}, fmt.Errorf("sav: Group at %d has incomplete AI block", record.Off)
			}
			copy(group.AI[:], record.Raw["G3C"])
			for _, field := range []struct {
				name   string
				target *[]uint16
			}{{"G20", &group.Words}, {"G4C", &group.AIWords}} {
				raw := record.Raw[field.name]
				if len(raw) != 2*record.Counts[field.name] {
					return SavedActorGraph{}, fmt.Errorf("sav: Group at %d has incomplete %s list", record.Off, field.name)
				}
				for i := 0; i < len(raw); i += 2 {
					*field.target = append(*field.target, u16(raw, i))
				}
			}
			group.Reference, err = resolve(record.value("G40"), record.End)
			if err != nil {
				return SavedActorGraph{}, err
			}
			group.Owner, err = resolve(record.value("G44"), record.End)
			if err != nil {
				return SavedActorGraph{}, err
			}
			out.Groups = append(out.Groups, group)
			members, hasSlots := record.RefSlots["Actors"]
			if !hasSlots {
				members = record.Refs["Actors"]
			}
			for _, member := range members {
				if member == nil {
					out.Groups[group.Ordinal-1].Members = append(out.Groups[group.Ordinal-1].Members, 0)
					continue
				}
				index, ok := byIndex[member.Index]
				if !ok {
					return SavedActorGraph{}, fmt.Errorf("sav: Group member %d has no actor", member.Index)
				}
				actor := &out.Actors[index]
				if actor.Group != 0 {
					prior := &out.Groups[actor.Group-1]
					prior.Members = slices.DeleteFunc(prior.Members, func(id uint16) bool { return id == member.Index })
				}
				actor.Group = group.Ordinal
				current := &out.Groups[group.Ordinal-1]
				current.Members = append(current.Members, member.Index)
			}
		}
		// The Player suffix overwrites actors still in these Groups, after each
		// Group's independent owner reference was restored (SAV-GRPOWNER-561).
		for _, group := range out.Groups[first:] {
			for _, member := range group.Members {
				if member == 0 {
					continue
				}
				actor := &out.Actors[byIndex[member]]
				actor.OwnerSlot = uint16(player.value("Slot"))
				actor.Owner = ActorReference{Key: player.value("This"), ArchiveIndex: player.Index,
					Class: "Player", PlayerSlot: actor.OwnerSlot, PlayerF58: player.value("F58"), Resolved: true}
			}
		}
	}
	slices.SortFunc(out.Actors, func(a, b ActorRecord) int { return int(a.ArchiveIndex) - int(b.ArchiveIndex) })
	for i := range out.Actors {
		a := &out.Actors[i]
		if a.Character.Basis != nil {
			a.Character.Basis.Human.HasOwner = a.Owner.Resolved && a.Owner.Class == "Player"
			if a.Character.Basis.Human.HasOwner {
				a.Character.Basis.Human.ManaReservePercent = a.Owner.PlayerF58
			}
		}
	}
	return out, nil
}
