package sav

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
)

// NativeModsPath carries the mod mark of a save written under an active mod
// set, in an ordinary application-registry dword array beside AgainromRng and
// AgainromActions. The payload's meaning belongs to the game package; this
// package frames and bounds it.
const NativeModsPath = "/CurrentState/AgainromMods"

// MaxNativeMods bounds the payload of the mod mark.
const MaxNativeMods = 1 << 20

func validateNativeMods(kind uint32, b []byte) error {
	if kind != 6 || len(b) < 8 || len(b) > MaxNativeMods+12 || len(b)%4 != 0 || binary.LittleEndian.Uint32(b) != 1 {
		return fmt.Errorf("sav: invalid mod mark framing")
	}
	n := uint64(binary.LittleEndian.Uint32(b[4:]))
	if n == 0 || n > MaxNativeMods || n > uint64(len(b)-8) || uint64(len(b)) != (n+11)&^3 {
		return fmt.Errorf("sav: invalid mod mark length")
	}
	for _, pad := range b[8+n:] {
		if pad != 0 {
			return fmt.Errorf("sav: nonzero mod mark padding")
		}
	}
	return nil
}

// NativeMods reads the optional mod mark payload.
func NativeMods(state DocumentStateData) ([]byte, bool, error) {
	var value []byte
	found := false
	for _, r := range state.ValueRecords {
		if !strings.EqualFold(r.Path, NativeModsPath) {
			continue
		}
		if found || r.Path != NativeModsPath || r.Value.Int32 != 0 {
			return nil, false, fmt.Errorf("sav: ambiguous mod mark leaf")
		}
		if err := validateNativeMods(r.Value.Kind, r.Value.Bytes); err != nil {
			return nil, false, err
		}
		n := binary.LittleEndian.Uint32(r.Value.Bytes[4:])
		value, found = bytes.Clone(r.Value.Bytes[8:8+n]), true
	}
	return value, found, nil
}

// SetNativeMods writes the mod mark payload, replacing any earlier one.
func SetNativeMods(state *DocumentStateData, value []byte) error {
	if state == nil || len(value) == 0 || len(value) > MaxNativeMods {
		return fmt.Errorf("sav: the mod mark requires a bounded application leaf")
	}
	if _, _, err := NativeMods(*state); err != nil {
		return err
	}
	b := make([]byte, (len(value)+11)&^3)
	binary.LittleEndian.PutUint32(b, 1)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(value)))
	copy(b[8:], value)
	r := CityStateRecordData{Path: NativeModsPath, Value: CityStateValueData{Kind: 6, Bytes: b}}
	records := slices.Clone(state.ValueRecords)
	index := -1
	for i := range records {
		if records[i].Path == r.Path {
			index = i
		}
	}
	if index < 0 {
		records = append(records, r)
	} else {
		records[index] = r
	}
	slices.SortFunc(records, func(a, b CityStateRecordData) int { return strings.Compare(a.Path, b.Path) })
	state.ValueRecords = records
	return nil
}

// ClearNativeMods removes the mod mark, so a document the current mod set did
// not touch carries none.
func ClearNativeMods(state *DocumentStateData) {
	if state == nil {
		return
	}
	state.ValueRecords = slices.DeleteFunc(slices.Clone(state.ValueRecords), func(r CityStateRecordData) bool { return r.Path == NativeModsPath })
}

// modsLeaf is the shape entry of the mod mark; it sorts between
// AgainromActions and AgainromRng.
func modsLeaf(state *cityState) ([]stateLeaf, error) {
	if state == nil {
		return nil, nil
	}
	value, ok := state.values[NativeModsPath]
	if !ok {
		return nil, nil
	}
	if err := validateNativeMods(value.kind, value.bytes); err != nil {
		return nil, err
	}
	return []stateLeaf{{"AgainromMods", 6}}, nil
}
