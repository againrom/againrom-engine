package sav

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"strconv"
)

const (
	maxWorldStateBytes       = 64 << 20
	maxWorldStateProjectiles = 1 << 16
	maxWorldStateRecords     = 29 + 17*maxWorldStateProjectiles
)

// SAV-TAILEXT-062 and SAV-FOG-061 add Fog to the city state. SAV-PROJSTORE-428
// adds Projectiles and one Prj<decimal u16 id> section per referenced projectile.
// Nine directories and 28 records describe an empty manager, not every world.
var worldProjectileLeaves = []stateLeaf{
	{"x", 2}, {"y", 2}, {"z", 2}, {"picture", 2}, {"dir", 2}, {"phase", 2},
	{"lastaction", 2}, {"action", 2}, {"actiondir", 2}, {"actiontarget", 2},
	{"actionx", 2}, {"actiony", 2}, {"actionz", 2}, {"actionphase", 2},
	{"actionsegments", 2}, {"actionspell", 2},
}

func parseWorldState(raw []byte) (*cityState, error) {
	if len(raw) > maxWorldStateBytes {
		return nil, fmt.Errorf("sav: world state store has %d bytes, exceeds %d", len(raw), maxWorldStateBytes)
	}
	state, err := parseStateStore(raw, -1, -1, validateWorldState)
	if err != nil {
		return nil, err
	}
	return narrowWorldProjectiles(state)
}

// narrowProjectileLeaf is the width a record holds a Prj leaf in: action and
// actiondir are bytes and actiontarget a word. SAVE writes the bytes
// sign-extended and the word zero-extended, LOAD keeps the low byte or word
// (SAV-1322); the other thirteen leaves are dwords (SAV-1202).
func narrowProjectileLeaf(name string, v int32) int32 {
	switch name {
	case "action", "actiondir":
		return int32(int8(v))
	case "actiontarget":
		return int32(uint16(v))
	}
	return v
}

// narrowWorldProjectiles gives a world state store the values the original's
// LOAD keeps and its SAVE writes back: the low word of FreeIndex (SAV-1191),
// each IDs element once as a zero-extended word, because a repeated ID
// replaces the earlier record in the one record map SAVE walks (SAV-1322,
// SAV-PROJLOAD-429), and the record widths of narrowProjectileLeaf. A state
// that already holds those values is returned unchanged; otherwise a copy is.
func narrowWorldProjectiles(state *cityState) (*cityState, error) {
	if _, present := state.directoryKinds["/Projectiles"]; !present {
		return state, nil
	}
	lookup := state.leafIndex()
	changed := map[string]cityStateValue{}
	if r, ok, err := lookup.read("/Projectiles/FreeIndex"); err != nil {
		return nil, err
	} else if ok && r.Value.Kind == 2 && r.Value.Int32 != int32(uint16(r.Value.Int32)) {
		changed[r.Path] = cityStateValue{kind: 2, int32: int32(uint16(r.Value.Int32))}
	}
	var ids []uint16
	if r, ok, err := lookup.read("/Projectiles/IDs"); err != nil {
		return nil, err
	} else if ok && r.Value.Kind == 2 {
		ids = []uint16{uint16(r.Value.Int32)}
	} else if ok && r.Value.Kind == 6 && len(r.Value.Bytes)%4 == 0 {
		seen := map[uint16]bool{}
		var kept []byte
		for at := 0; at < len(r.Value.Bytes); at += 4 {
			id := uint16(binary.LittleEndian.Uint32(r.Value.Bytes[at:]))
			if seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
			kept = binary.LittleEndian.AppendUint32(kept, uint32(id))
		}
		if !bytes.Equal(kept, r.Value.Bytes) {
			changed[r.Path] = cityStateValue{kind: 6, int32: r.Value.Int32, bytes: kept}
		}
	}
	for _, id := range ids {
		for _, name := range []string{"action", "actiondir", "actiontarget"} {
			r, ok, err := lookup.read("/Prj" + strconv.FormatUint(uint64(id), 10) + "/" + name)
			if err != nil {
				return nil, err
			}
			if ok && r.Value.Kind == 2 && r.Value.Int32 != narrowProjectileLeaf(name, r.Value.Int32) {
				changed[r.Path] = cityStateValue{kind: 2, int32: narrowProjectileLeaf(name, r.Value.Int32)}
			}
		}
	}
	if len(changed) == 0 {
		return state, nil
	}
	out := &cityState{rootKind: state.rootKind, directoryKinds: state.directoryKinds, values: make(map[string]cityStateValue, len(state.values))}
	for path, v := range state.values {
		out.values[path] = v
	}
	for path, v := range changed {
		out.values[path] = v
	}
	return out, nil
}

// worldStateSerializedSize bounds the record table and all retained leaf bytes
// without constructing a pool or output buffer. The envelope may impose a
// smaller combined-file budget. The result is exact for a valid typed shape;
// shape validation remains the serializer's responsibility.
func worldStateSerializedSize(state *cityState) (uint64, error) {
	if state == nil {
		return 0, fmt.Errorf("sav: world state is nil")
	}
	records := uint64(len(state.directoryKinds)) + uint64(len(state.values))
	if records > maxWorldStateRecords {
		return 0, fmt.Errorf("sav: world state has %d records, exceeds %d", records, maxWorldStateRecords)
	}
	total := uint64(28) + 32*records
	for path, value := range state.values {
		if value.kind == 2 && len(value.bytes) != 0 {
			return 0, fmt.Errorf("sav: world state integer %s carries inactive bytes", path)
		}
		if uint64(len(value.bytes)) > maxWorldStateBytes || total > maxWorldStateBytes-uint64(len(value.bytes)) {
			return 0, fmt.Errorf("sav: world state exceeds %d serialized bytes", maxWorldStateBytes)
		}
		total += uint64(len(value.bytes))
	}
	if total > maxWorldStateBytes {
		return 0, fmt.Errorf("sav: world state exceeds %d serialized bytes", maxWorldStateBytes)
	}
	return total, nil
}

func serializeWorldState(state *cityState) ([]byte, error) {
	state, err := narrowWorldProjectiles(state)
	if err != nil {
		return nil, err
	}
	shape, err := worldStateShape(state)
	if err != nil {
		return nil, err
	}
	if err := validateStateShape(state, shape); err != nil {
		return nil, err
	}
	return serializeStateStore(state, shape)
}

func validateWorldState(state *cityState) error {
	shape, err := worldStateShape(state)
	if err != nil {
		return err
	}
	return validateStateShape(state, shape)
}

func worldStateShape(state *cityState) ([]stateDirectory, error) {
	if _, err := worldStateSerializedSize(state); err != nil {
		return nil, err
	}
	shape, err := placeSupplementLeaves(state, append([]stateDirectory(nil), cityStateShape...), worldForm)
	if err != nil {
		return nil, err
	}
	shape = append(shape, stateDirectory{"Fog", []stateLeaf{{"FirstState", 2}, {"Data", 6}}})
	for path, value := range state.values {
		if value.kind == 6 && len(value.bytes)%4 != 0 {
			return nil, fmt.Errorf("sav: world state array %s has %d unaligned bytes", path, len(value.bytes))
		}
	}
	// SAV-PROJLOAD-429 also establishes missing leaves/the entire section and
	// a kind-2 singleton IDs compatibility arm. Preserve those typed forms;
	// newly constructed canonical state supplies both leaves with IDs kind 6.
	// Missing per-projectile fields are different: their constructor defaults
	// are not supplied by this codec, so each referenced section must be whole.
	lookup := state.leafIndex()
	if _, present := state.directoryKinds["/Projectiles"]; present {
		projectiles := stateDirectory{directory: "Projectiles"}
		if _, ok, err := lookup.read("/Projectiles/FreeIndex"); err != nil {
			return nil, err
		} else if ok {
			projectiles.children = append(projectiles.children, stateLeaf{"FreeIndex", 2})
		}
		var ids []uint16
		if r, ok, err := lookup.read("/Projectiles/IDs"); err != nil {
			return nil, err
		} else if ok {
			value := r.Value
			switch value.Kind {
			case 2:
				ids = []uint16{uint16(value.Int32)}
			case 6:
				if len(value.Bytes)/4 > maxWorldStateProjectiles {
					return nil, fmt.Errorf("sav: world state has more than 65536 projectile IDs")
				}
				ids = make([]uint16, len(value.Bytes)/4)
				for i := range ids {
					ids[i] = uint16(binary.LittleEndian.Uint32(value.Bytes[4*i:]))
				}
			default:
				return nil, fmt.Errorf("sav: world state Projectiles/IDs has unsupported kind %d", value.Kind)
			}
			projectiles.children = append(projectiles.children, stateLeaf{"IDs", value.Kind})
		}
		shape = append(shape, projectiles)
		seen := make(map[uint16]bool, len(ids))
		for _, id := range ids {
			if seen[id] {
				continue // A repeated ID names the same section; narrowWorldProjectiles drops it.
			}
			seen[id] = true
			shape = append(shape, stateDirectory{"Prj" + strconv.FormatUint(uint64(id), 10), worldProjectileLeaves})
		}
	}
	// Stable name order is transport, not projectile traversal order. The IDs
	// array keeps its order; only narrowWorldProjectiles drops a repeat.
	sort.Slice(shape, func(i, j int) bool { return shape[i].directory < shape[j].directory })
	return shape, nil
}
