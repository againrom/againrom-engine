package sav

import (
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
)

// NativeSessionPath carries the random session of a save: the session seed,
// the mode it ran in and the shared stream's state, in an ordinary
// application-registry dword array beside AgainromRng. The meaning of each
// value belongs to the game package; this package frames and bounds it.
const NativeSessionPath = "/CurrentState/AgainromSeed"

// NativeSession is the leaf's payload.
type NativeSession struct {
	Seed   uint64
	Mode   uint32
	Shared uint32
}

const nativeSessionLen = 20

func validateNativeSession(kind uint32, b []byte) error {
	if kind != 6 || len(b) != nativeSessionLen || binary.LittleEndian.Uint32(b) != 1 {
		return fmt.Errorf("sav: AgainromSeed requires version 1 and exactly five dwords")
	}
	if mode := binary.LittleEndian.Uint32(b[12:]); mode > 1 {
		return fmt.Errorf("sav: AgainromSeed names random mode %d, which is not defined", mode)
	}
	return nil
}

// ReadNativeSession reads the optional session leaf.
func ReadNativeSession(state DocumentStateData) (NativeSession, bool, error) {
	var out NativeSession
	found := false
	for _, r := range state.ValueRecords {
		if !strings.EqualFold(r.Path, NativeSessionPath) {
			continue
		}
		if found || r.Path != NativeSessionPath || r.Value.Int32 != 0 {
			return NativeSession{}, false, fmt.Errorf("sav: ambiguous or invalid AgainromSeed leaf")
		}
		if err := validateNativeSession(r.Value.Kind, r.Value.Bytes); err != nil {
			return NativeSession{}, false, err
		}
		b := r.Value.Bytes
		out = NativeSession{Seed: binary.LittleEndian.Uint64(b[4:]), Mode: binary.LittleEndian.Uint32(b[12:]),
			Shared: binary.LittleEndian.Uint32(b[16:])}
		found = true
	}
	return out, found, nil
}

// SetNativeSession writes the session leaf, replacing any earlier one.
func SetNativeSession(state *DocumentStateData, value NativeSession) error {
	if state == nil || value.Mode > 1 {
		return fmt.Errorf("sav: the random session requires an application registry and a defined mode")
	}
	if _, _, err := ReadNativeSession(*state); err != nil {
		return err
	}
	b := make([]byte, nativeSessionLen)
	binary.LittleEndian.PutUint32(b, 1)
	binary.LittleEndian.PutUint64(b[4:], value.Seed)
	binary.LittleEndian.PutUint32(b[12:], value.Mode)
	binary.LittleEndian.PutUint32(b[16:], value.Shared)
	r := CityStateRecordData{Path: NativeSessionPath, Value: CityStateValueData{Kind: 6, Bytes: b}}
	records := slices.Clone(state.ValueRecords)
	replaced := false
	for i := range records {
		if records[i].Path == NativeSessionPath {
			records[i], replaced = r, true
		}
	}
	if !replaced {
		records = append(records, r)
	}
	slices.SortFunc(records, func(a, b CityStateRecordData) int { return strings.Compare(a.Path, b.Path) })
	state.ValueRecords = records
	return nil
}

// withSessionLeaf inserts the session leaf's shape entry before InBattle, the
// place its name sorts to among the CurrentState children.
func withSessionLeaf(state *cityState, shape []stateDirectory) ([]stateDirectory, error) {
	if state == nil {
		return shape, nil
	}
	value, ok := state.values[NativeSessionPath]
	if !ok {
		return shape, nil
	}
	if err := validateNativeSession(value.kind, value.bytes); err != nil {
		return nil, err
	}
	for i := range shape {
		if shape[i].directory != "CurrentState" {
			continue
		}
		children := slices.Clone(shape[i].children)
		at := slices.IndexFunc(children, func(l stateLeaf) bool { return l.name == "InBattle" })
		if at < 0 {
			at = len(children)
		}
		shape[i].children = slices.Insert(children, at, stateLeaf{"AgainromSeed", 6})
	}
	return shape, nil
}
