package sim

import (
	"encoding/binary"
	"fmt"
)

// Version 70 adds fixed 173-byte records followed by a four-byte span, just
// before the relation. The backward length keeps every old variable-section
// cursor intact and gives the script decoder its exact former endpoint.
// The original 74-byte prefix stays fixed; +74 is held-Weapon presence and
// +75..172 is its bounded 98-byte provenance. This form was not published at
// the earlier 74-byte checkpoint.
const originalDeadRecordLen = 173

func (w *World) originalDeadSectionLen() int { return len(w.originalDead)*originalDeadRecordLen + 4 }

func putDeadState(b []byte, s DeadActorState) {
	binary.LittleEndian.PutUint32(b, s.RuntimeID)
	binary.LittleEndian.PutUint16(b[4:], s.Cell)
	b[6], b[7], b[8] = s.FineX, s.FineY, s.Stage
	binary.LittleEndian.PutUint16(b[9:], uint16(s.HP))
	b[11] = uint8(s.Timer)
}

func readDeadState(b []byte) DeadActorState {
	return DeadActorState{RuntimeID: binary.LittleEndian.Uint32(b), Cell: binary.LittleEndian.Uint16(b[4:]),
		FineX: b[6], FineY: b[7], Stage: b[8], HP: int16(binary.LittleEndian.Uint16(b[9:])), Timer: int8(b[11])}
}

func (w *World) encodeOriginalDead(b []byte, off int) int {
	start := off
	for _, r := range w.OriginalDeadActors() {
		s := r.Source
		binary.LittleEndian.PutUint32(b[off:], uint32(r.ID))
		binary.LittleEndian.PutUint32(b[off+4:], s.Identity)
		binary.LittleEndian.PutUint32(b[off+8:], s.TerrainKey)
		binary.LittleEndian.PutUint32(b[off+12:], s.OwnerKey)
		for i, v := range s.References {
			binary.LittleEndian.PutUint32(b[off+16+i*4:], v)
		}
		binary.LittleEndian.PutUint16(b[off+36:], s.ArchiveIndex)
		binary.LittleEndian.PutUint16(b[off+38:], s.MapUnitID)
		b[off+40] = s.Class
		if s.ContainerPresent {
			b[off+41] = 1
		}
		for i, v := range s.ContainerTail {
			binary.LittleEndian.PutUint32(b[off+42+i*4:], v)
		}
		putDeadState(b[off+50:], s.State)
		putDeadState(b[off+62:], r.Current)
		if s.HeldWeapon.Present {
			b[off+74] = 1
			putDeadWeapon(b[off+75:], s.HeldWeapon)
		}
		off += originalDeadRecordLen
	}
	binary.LittleEndian.PutUint32(b[off:], uint32(off-start))
	return off + 4
}

func decodeOriginalDead(data []byte, b Bounds, ents []Entity, carried [][]ItemStack, equipment [][EquipSlots]ItemInstance) ([]originalDeadRecord, error) {
	if len(data)%originalDeadRecordLen != 0 {
		return nil, fmt.Errorf("sim: invalid original dead record span")
	}
	out := make([]originalDeadRecord, 0, len(data)/originalDeadRecordLen)
	ids, keys, maps, indices := map[EntityID]bool{}, map[uint32]bool{}, map[uint16]bool{}, map[uint16]bool{}
	for off := 0; off < len(data); off += originalDeadRecordLen {
		s := OriginalDeadSource{Identity: binary.LittleEndian.Uint32(data[off+4:]), TerrainKey: binary.LittleEndian.Uint32(data[off+8:]), OwnerKey: binary.LittleEndian.Uint32(data[off+12:]),
			ArchiveIndex: binary.LittleEndian.Uint16(data[off+36:]), MapUnitID: binary.LittleEndian.Uint16(data[off+38:]), Class: data[off+40], ContainerPresent: data[off+41] == 1,
			State: readDeadState(data[off+50:])}
		for i := range s.References {
			s.References[i] = binary.LittleEndian.Uint32(data[off+16+i*4:])
		}
		for i := range s.ContainerTail {
			s.ContainerTail[i] = binary.LittleEndian.Uint32(data[off+42+i*4:])
		}
		r := originalDeadRecord{OriginalDeadActor: OriginalDeadActor{ID: EntityID(binary.LittleEndian.Uint32(data[off:])), Source: s}}
		current := readDeadState(data[off+62:])
		s.HeldWeapon = readDeadWeapon(data[off+75:])
		s.HeldWeapon.Present = data[off+74] == 1
		r.Source = s
		if data[off+74] > 1 {
			return nil, fmt.Errorf("sim: invalid original dead weapon presence")
		}
		if data[off+41] > 1 {
			return nil, fmt.Errorf("sim: invalid dead container presence")
		}
		if err := deadSourceFault(s, b); err != nil {
			return nil, err
		}
		if err := deadStateFaultWithPosition(current, b, s.MapUnitID != 0); err != nil {
			return nil, err
		}
		if ids[r.ID] || keys[s.Identity] || s.ArchiveIndex != 0 && indices[s.ArchiveIndex] || s.MapUnitID != 0 && maps[s.MapUnitID] {
			return nil, fmt.Errorf("sim: duplicate original dead binding")
		}
		ids[r.ID], keys[s.Identity], indices[s.ArchiveIndex] = true, true, true
		if s.MapUnitID != 0 {
			maps[s.MapUnitID] = true
		}
		if weapon := s.HeldWeapon; weapon.Present {
			if keys[weapon.Identity] || indices[weapon.ArchiveIndex] {
				return nil, fmt.Errorf("sim: duplicate original dead held-Weapon identity")
			}
			keys[weapon.Identity], indices[weapon.ArchiveIndex] = true, true
		}
		ei := indexOfEntity(ents, r.ID)
		switch {
		case s.MapUnitID == 0:
			if ei >= 0 {
				return nil, fmt.Errorf("sim: virtual dead actor %d is still an entity", r.ID)
			}
			if !validHeldDeadContinuation(s.State, current) {
				return nil, fmt.Errorf("sim: virtual dead actor %d has invalid continuation", r.ID)
			}
			r.terminal = current
		case current.Stage == 5:
			if ei >= 0 {
				return nil, fmt.Errorf("sim: terminal dead actor %d is still an entity", r.ID)
			}
			if s.State.Stage == 5 && current != s.State {
				return nil, fmt.Errorf("sim: imported terminal tuple changed")
			}
			r.terminal = current
		default:
			if ei < 0 {
				return nil, fmt.Errorf("sim: late corpse %d is absent", r.ID)
			}
			e := ents[ei]
			if e.OffMap || e.MapUnitID != s.MapUnitID || int16(e.HP) != current.HP || uint8(e.Decay) != current.Stage || e.Dwell != 0 ||
				e.X != int32(current.Cell&255) || e.Y != int32(current.Cell>>8) || !e.SuppressCorpseLoot || len(carried[ei]) != 0 ||
				current.Stage < s.State.Stage || current.RuntimeID != s.State.RuntimeID {
				return nil, fmt.Errorf("sim: late corpse %d disagrees with current dead record", r.ID)
			}
			for _, item := range equipment[ei] {
				if item.Code != 0 || len(item.Effects) != 0 {
					return nil, fmt.Errorf("sim: late corpse %d retains equipment", r.ID)
				}
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func validHeldDeadContinuation(source, current DeadActorState) bool {
	if source.Stage == 5 || current == source {
		return current == source
	}
	if current.Cell != source.Cell || current.FineX != source.FineX || current.FineY != source.FineY ||
		current.Timer != source.Timer || current.HP > source.HP || current.Stage < source.Stage {
		return false
	}
	if current.Stage == 5 {
		return current.HP == -10001 && current.RuntimeID == 0
	}
	return current.RuntimeID == source.RuntimeID && current.Stage == max(source.Stage, uint8(decayStageFor(int32(current.HP))))
}
