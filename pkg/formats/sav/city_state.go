package sav

import (
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
)

type cityStateValue struct {
	kind  uint32
	int32 int32
	bytes []byte
}

type cityState struct {
	rootKind       uint32
	directoryKinds map[string]uint32
	values         map[string]cityStateValue
}

type stateDirectory struct {
	directory string
	children  []stateLeaf
}

type stateLeaf struct {
	name string
	kind uint32
}

var cityStateShape = []stateDirectory{
	{"Character", []stateLeaf{{"Name", 0}}},
	{"CurrentState", []stateLeaf{{"InBattle", 2}}},
	{"GameOptions", []stateLeaf{{"FlyingHP", 2}, {"Formation", 2}, {"ShowHP", 2}, {"ShowTimeFlow", 2}, {"Speed", 2}, {"Wimpy", 2}}},
	{"Inventory", []stateLeaf{{"IsOpen", 2}}},
	{"Objects", []stateLeaf{{"Selection", 6}}},
	{"SpellBook", []stateLeaf{{"IsOpen", 2}, {"Pressed", 2}, {"Shortcuts", 6}}},
	{"View", []stateLeaf{{"X", 2}, {"Y", 2}}},
}

type cityStateRecord struct {
	value uint32
	size  uint32
	kind  uint32
	name  string
}

func parseCityState(raw []byte) (*cityState, error) {
	return parseStateStore(raw, len(cityStateShape), -1, validateCityState)
}

// parseStateStore detaches typed values from YA1 transport. A negative expected
// count selects the bounded variable world shape; city counts remain exact.
func parseStateStore(raw []byte, expectedRoots, expectedRecords int, validate func(*cityState) error) (*cityState, error) {
	if len(raw) < 28 || string(raw[:4]) != "&YA1" {
		return nil, fmt.Errorf("sav: state store has no &YA1 header")
	}
	rootStart := binary.LittleEndian.Uint32(raw[4:8])
	rootCount := binary.LittleEndian.Uint32(raw[8:12])
	rootKind := binary.LittleEndian.Uint32(raw[12:16])
	n := binary.LittleEndian.Uint32(raw[16:20])
	if rootKind != 17 || rootStart != 0 || rootCount > n || (expectedRoots >= 0 && rootCount != uint32(expectedRoots)) {
		return nil, fmt.Errorf("sav: state store root is start=%d count=%d kind=%d", rootStart, rootCount, rootKind)
	}
	if (expectedRecords >= 0 && n != uint32(expectedRecords)) || n > maxWorldStateRecords {
		return nil, fmt.Errorf("sav: state store has unsupported record count %d", n)
	}
	recordEnd64 := uint64(24) + uint64(n)*32
	if recordEnd64+4 > uint64(len(raw)) {
		return nil, fmt.Errorf("sav: city state record table overruns %d bytes", len(raw))
	}
	if recordEnd64+4 > maxWorldStateBytes {
		return nil, fmt.Errorf("sav: state store record table exceeds %d detached bytes", maxWorldStateBytes)
	}
	recordEnd := int(recordEnd64)
	// Pool spans may alias. Bound the sum of every declared leaf copy, not
	// just the input pool, before allocating records, maps or detached values.
	// Unreachable/ill-typed records are still rejected by the ordinary walk;
	// none may cause an earlier allocation amplification.
	declaredBytes := recordEnd64 + 4
	for i := uint32(0); i < n; i++ {
		off := 24 + 32*int(i)
		kind := binary.LittleEndian.Uint32(raw[off+12 : off+16])
		if kind != 0 && kind != 6 {
			continue
		}
		size := uint64(binary.LittleEndian.Uint32(raw[off+8 : off+12]))
		if size > maxWorldStateBytes-declaredBytes {
			return nil, fmt.Errorf("sav: state store declared detached state bytes exceed %d at record %d", maxWorldStateBytes, i)
		}
		declaredBytes += size
	}
	records := make([]cityStateRecord, int(n))
	for i := range records {
		off := 24 + 32*i
		name := string(raw[off+16 : off+32])
		if at := strings.IndexByte(name, 0); at >= 0 {
			name = name[:at]
		}
		records[i] = cityStateRecord{
			value: binary.LittleEndian.Uint32(raw[off+4 : off+8]),
			size:  binary.LittleEndian.Uint32(raw[off+8 : off+12]),
			kind:  binary.LittleEndian.Uint32(raw[off+12 : off+16]),
			name:  name,
		}
	}
	poolLen := binary.LittleEndian.Uint32(raw[recordEnd : recordEnd+4])
	end := uint64(recordEnd) + 4 + uint64(poolLen)
	if end != uint64(len(raw)) {
		return nil, fmt.Errorf("sav: city state extent is %d bytes, Store has %d", end, len(raw))
	}
	pool := raw[recordEnd+4:]
	state := &cityState{rootKind: rootKind, directoryKinds: make(map[string]uint32, int(rootCount)), values: make(map[string]cityStateValue)}
	used := make([]bool, len(records))
	detachedBytes := recordEnd64 + 4
	var walk func(uint32, uint32, string) error
	walk = func(start, count uint32, prefix string) error {
		if uint64(start)+uint64(count) > uint64(len(records)) {
			return fmt.Errorf("sav: city state child block %s [%d,%d) exceeds %d", prefix, start, start+count, len(records))
		}
		for i := start; i < start+count; i++ {
			if used[i] {
				return fmt.Errorf("sav: city state record %d (%s) is reached twice", i, records[i].name)
			}
			used[i] = true
			r := records[i]
			path := prefix + "/" + r.name
			if _, exists := state.directoryKinds[path]; exists {
				return fmt.Errorf("sav: state store repeats path %s", path)
			}
			if _, exists := state.values[path]; exists {
				return fmt.Errorf("sav: state store repeats path %s", path)
			}
			if r.kind&1 != 0 {
				if prefix != "" {
					return fmt.Errorf("sav: state store has nested directory %s", path)
				}
				if r.kind&0x0e != 0 {
					return fmt.Errorf("sav: city state directory %s kind %#x claims a value type", path, r.kind)
				}
				state.directoryKinds[path] = r.kind
				if err := walk(r.value, r.size, path); err != nil {
					return err
				}
				continue
			}
			v := cityStateValue{kind: r.kind, int32: int32(r.value)}
			switch r.kind {
			case 0, 6:
				if uint64(r.value)+uint64(r.size) > uint64(len(pool)) {
					return fmt.Errorf("sav: city state value %s [%d,%d) exceeds pool %d", path, r.value, r.value+r.size, len(pool))
				}
				if r.kind == 6 && r.size%4 != 0 {
					return fmt.Errorf("sav: city state int array %s has %d bytes", path, r.size)
				}
				if uint64(r.size) > maxWorldStateBytes-detachedBytes {
					return fmt.Errorf("sav: state store detached state bytes exceed %d before copying %s", maxWorldStateBytes, path)
				}
				detachedBytes += uint64(r.size)
				v.bytes = append([]byte(nil), pool[r.value:r.value+r.size]...)
			case 2:
			default:
				return fmt.Errorf("sav: city state value %s has unsupported kind %d", path, r.kind)
			}
			state.values[path] = v
		}
		return nil
	}
	if err := walk(rootStart, rootCount, ""); err != nil {
		return nil, err
	}
	for i, ok := range used {
		if !ok {
			return nil, fmt.Errorf("sav: city state record %d (%s) is unreachable", i, records[i].name)
		}
	}
	if int(rootCount) != len(state.directoryKinds) {
		return nil, fmt.Errorf("sav: state store has nested or non-directory roots")
	}
	if err := validate(state); err != nil {
		return nil, err
	}
	return state, nil
}

func validateCityState(state *cityState) error {
	shape, err := currentCityStateShape(state)
	if err != nil {
		return err
	}
	return validateStateShape(state, shape)
}

func currentCityStateShape(state *cityState) ([]stateDirectory, error) {
	shape, err := withSessionLeaf(state, append([]stateDirectory(nil), cityStateShape...))
	if err != nil {
		return nil, err
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
	if state != nil {
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
	}
	return shape, nil
}

func validateStateShape(state *cityState, shape []stateDirectory) error {
	if state == nil {
		return fmt.Errorf("sav: city state is nil")
	}
	if state.rootKind != 17 {
		return fmt.Errorf("sav: city state root kind is %d, want measured value 17", state.rootKind)
	}
	if len(state.directoryKinds) != len(shape) {
		return fmt.Errorf("sav: state store has %d directories, want %d", len(state.directoryKinds), len(shape))
	}
	leaves := 0
	for _, directory := range shape {
		leaves += len(directory.children)
	}
	if len(state.values) != leaves {
		return fmt.Errorf("sav: state store has %d leaves, want %d", len(state.values), leaves)
	}
	for _, directory := range shape {
		directoryPath := "/" + directory.directory
		kind, ok := state.directoryKinds[directoryPath]
		if !ok {
			return fmt.Errorf("sav: city state lacks directory %s", directoryPath)
		}
		if kind&1 == 0 || kind&0x0e != 0 {
			return fmt.Errorf("sav: city state directory %s has non-directory kind %#x", directoryPath, kind)
		}
		for _, child := range directory.children {
			path := "/" + directory.directory + "/" + child.name
			value, ok := state.values[path]
			if !ok {
				return fmt.Errorf("sav: city state lacks %s", path)
			}
			if value.kind != child.kind {
				return fmt.Errorf("sav: city state %s kind is %d, want %d", path, value.kind, child.kind)
			}
		}
	}
	name := state.values["/Character/Name"].bytes
	if len(name) == 0 || name[len(name)-1] != 0 || strings.IndexByte(string(name[:len(name)-1]), 0) >= 0 {
		return fmt.Errorf("sav: city state Character/Name is not one NUL-terminated string")
	}
	if shortcuts := state.values["/SpellBook/Shortcuts"].bytes; len(shortcuts) != 16 {
		return fmt.Errorf("sav: city state SpellBook/Shortcuts has %d bytes, want 16", len(shortcuts))
	}
	return nil
}

func cloneCityState(source *cityState) *cityState {
	out := &cityState{rootKind: source.rootKind, directoryKinds: make(map[string]uint32, len(source.directoryKinds)), values: make(map[string]cityStateValue, len(source.values))}
	for path, kind := range source.directoryKinds {
		out.directoryKinds[path] = kind
	}
	for path, value := range source.values {
		value.bytes = append([]byte(nil), value.bytes...)
		out.values[path] = value
	}
	return out
}

func serializeCityState(state *cityState) ([]byte, error) {
	if err := validateCityState(state); err != nil {
		return nil, err
	}
	shape, err := currentCityStateShape(state)
	if err != nil {
		return nil, err
	}
	return serializeStateStore(state, shape)
}

// serializeStateStore writes names, indices, record tables and the pool anew.
// The caller validates its city or world shape before entering this helper.
func serializeStateStore(state *cityState, shape []stateDirectory) ([]byte, error) {
	type record struct {
		name        string
		value, size uint32
		kind        uint32
	}
	records := make([]record, 0, len(shape)+len(state.values))
	childAt := len(shape)
	for _, directory := range shape {
		records = append(records, record{name: directory.directory, value: uint32(childAt), size: uint32(len(directory.children)), kind: state.directoryKinds["/"+directory.directory]})
		childAt += len(directory.children)
	}
	var pool []byte
	for _, directory := range shape {
		for _, child := range directory.children {
			path := "/" + directory.directory + "/" + child.name
			value := state.values[path]
			r := record{name: child.name, kind: value.kind, value: uint32(value.int32)}
			if value.kind == 0 || value.kind == 6 {
				r.value = uint32(len(pool))
				r.size = uint32(len(value.bytes))
				pool = append(pool, value.bytes...)
			}
			records = append(records, r)
		}
	}
	if len(records) != len(shape)+len(state.values) {
		return nil, fmt.Errorf("sav: state store writer built an inconsistent record count")
	}
	var out []byte
	out = append(out, "&YA1"...)
	out = cityAppendU32(out, 0)
	out = cityAppendU32(out, uint32(len(shape)))
	out = cityAppendU32(out, state.rootKind)
	out = cityAppendU32(out, uint32(len(records)))
	out = cityAppendU32(out, 0)
	for _, r := range records {
		if r.name == "" || len(r.name) > 15 || strings.IndexByte(r.name, 0) >= 0 {
			return nil, fmt.Errorf("sav: city state node name %q does not fit its field", r.name)
		}
		out = cityAppendU32(out, 0)
		out = cityAppendU32(out, r.value)
		out = cityAppendU32(out, r.size)
		out = cityAppendU32(out, r.kind)
		name := make([]byte, 16)
		copy(name, r.name)
		out = append(out, name...)
	}
	out = cityAppendU32(out, uint32(len(pool)))
	out = append(out, pool...)
	return out, nil
}
