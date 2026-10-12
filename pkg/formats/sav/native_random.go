package sav

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const nativeRandomRefusal = "sav: AgainromRng requires version 1 and exactly three dwords"

func validateNativeRandom(value []byte) error {
	if len(value) != 12 || binary.LittleEndian.Uint32(value) != 1 {
		return errors.New(nativeRandomRefusal)
	}
	return nil
}

// NativeRandomState reads the optional version, low word, high word tuple.
// Every uint64, including zero, is a valid SplitMix64 state. Absence means
// the historical native bootstrap policy, never an inferred original RNG law.
func NativeRandomState(state DocumentStateData) (uint64, bool, error) {
	b, found, err := nativeRandomLeaf.lookup(state)
	if err != nil || !found {
		return 0, false, err
	}
	return binary.LittleEndian.Uint64(b[4:]), true, nil
}

// SetNativeRandomState replaces historical metadata from the current World.
// It never reuses an old leaf as current simulation authority.
func SetNativeRandomState(state *DocumentStateData, value uint64) error {
	if state == nil {
		return fmt.Errorf("sav: native random state requires an application registry")
	}
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data, 1)
	binary.LittleEndian.PutUint64(data[4:], value)
	return nativeRandomLeaf.store(state, data)
}
