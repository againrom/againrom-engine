package sav

import (
	"encoding/binary"
	"fmt"
	"slices"
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
	return parseStateStore(raw, -1, -1, validateWorldState)
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
	shape := append([]stateDirectory(nil), cityStateShape...)
	if value, ok := state.values[NativeRandomPath]; ok {
		if err := validateNativeRandom(value.kind, value.bytes); err != nil {
			return nil, err
		}
		// REG-100: sorted lookup compares case-sensitive bytes. Both names
		// fit the original 15-byte insertion limit, with an actual NUL tail.
		for i := range shape {
			if shape[i].directory == "CurrentState" {
				shape[i].children = []stateLeaf{{"AgainromRng", 6}, {"InBattle", 2}}
			}
		}
	}
	mods, err := modsLeaf(state)
	if err != nil {
		return nil, err
	}
	if mods != nil {
		for i := range shape {
			if shape[i].directory == "CurrentState" {
				shape[i].children = append(slices.Clone(mods), shape[i].children...)
			}
		}
	}
	if value, ok := state.values[NativeActionsPath]; ok {
		if _, _, err := NativeActions(DocumentStateData{ValueRecords: []CityStateRecordData{{Path: NativeActionsPath, Value: CityStateValueData{Kind: value.kind, Bytes: value.bytes}}}}); err != nil {
			return nil, err
		}
		for i := range shape {
			if shape[i].directory == "CurrentState" {
				shape[i].children = append([]stateLeaf{{"AgainromActions", 6}}, shape[i].children...)
			}
		}
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
	if _, present := state.directoryKinds["/Projectiles"]; present {
		projectiles := stateDirectory{directory: "Projectiles"}
		if _, ok := state.values["/Projectiles/FreeIndex"]; ok {
			projectiles.children = append(projectiles.children, stateLeaf{"FreeIndex", 2})
		}
		var ids []uint16
		if value, ok := state.values["/Projectiles/IDs"]; ok {
			switch value.kind {
			case 2:
				ids = []uint16{uint16(value.int32)}
			case 6:
				if len(value.bytes)/4 > maxWorldStateProjectiles {
					return nil, fmt.Errorf("sav: world state has more than 65536 projectile IDs")
				}
				ids = make([]uint16, len(value.bytes)/4)
				for i := range ids {
					ids[i] = uint16(binary.LittleEndian.Uint32(value.bytes[4*i:]))
				}
			default:
				return nil, fmt.Errorf("sav: world state Projectiles/IDs has unsupported kind %d", value.kind)
			}
			projectiles.children = append(projectiles.children, stateLeaf{"IDs", value.kind})
		}
		shape = append(shape, projectiles)
		seen := make(map[uint16]bool, len(ids))
		for _, id := range ids {
			if seen[id] {
				continue // Repeated IDs remain in the value; YA1 has one named section.
			}
			seen[id] = true
			shape = append(shape, stateDirectory{"Prj" + strconv.FormatUint(uint64(id), 10), worldProjectileLeaves})
		}
	}
	// Stable name order is transport, not projectile traversal order. The IDs
	// array is never sorted, deduplicated, narrowed or reconstructed on output.
	sort.Slice(shape, func(i, j int) bool { return shape[i].directory < shape[j].directory })
	return shape, nil
}
