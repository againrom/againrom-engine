package sav

import (
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
)

// NativeRandomPath is Againrom-authored state in the existing application
// registry. REG-099..102 permit this bounded name/type shape; SAV-914..916 do
// not establish preservation by the original application's fresh SAVE.
const NativeRandomPath = "/CurrentState/AgainromRng"

func validateNativeRandom(kind uint32, value []byte) error {
	if kind != 6 || len(value) != 12 || binary.LittleEndian.Uint32(value) != 1 {
		return fmt.Errorf("sav: AgainromRng requires version 1 and exactly three dwords")
	}
	return nil
}

// NativeRandomState reads the optional version, low word, high word tuple.
// Every uint64, including zero, is a valid SplitMix64 state. Absence means
// the historical native bootstrap policy, never an inferred original RNG law.
func NativeRandomState(state DocumentStateData) (uint64, bool, error) {
	var result uint64
	found := false
	for _, r := range state.ValueRecords {
		if !strings.EqualFold(r.Path, NativeRandomPath) {
			continue
		}
		if found || r.Path != NativeRandomPath || r.Value.Int32 != 0 {
			return 0, false, fmt.Errorf("sav: ambiguous or invalid AgainromRng leaf")
		}
		if err := validateNativeRandom(r.Value.Kind, r.Value.Bytes); err != nil {
			return 0, false, err
		}
		found, result = true, binary.LittleEndian.Uint64(r.Value.Bytes[4:])
	}
	return result, found, nil
}

// SetNativeRandomState replaces historical metadata from the current World.
// It never reuses an old leaf as current simulation authority.
func SetNativeRandomState(state *DocumentStateData, value uint64) error {
	if state == nil {
		return fmt.Errorf("sav: native random state requires an application registry")
	}
	if _, _, err := NativeRandomState(*state); err != nil {
		return err
	}
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data, 1)
	binary.LittleEndian.PutUint64(data[4:], value)
	r := CityStateRecordData{Path: NativeRandomPath, Value: CityStateValueData{Kind: 6, Bytes: data}}
	records := slices.Clone(state.ValueRecords)
	replaced := false
	for i := range records {
		if records[i].Path == NativeRandomPath {
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
