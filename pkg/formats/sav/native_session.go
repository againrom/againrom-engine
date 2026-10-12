package sav

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// NativeSession is the session leaf's payload: the session seed, the mode it
// ran in, the shared stream's state and the session's reseed count. Version 1
// holds five dwords and no count; version 2 adds the count as a sixth and is
// written only when the count is not zero. The meaning of each value belongs
// to the game package; the descriptor frames and bounds it.
type NativeSession struct {
	Seed    uint64
	Mode    uint32
	Shared  uint32
	Reseeds uint32
}

// nativeSessionLen is each version's length in bytes, by version.
var nativeSessionLen = map[uint32]int{1: 20, 2: 24}

const nativeSessionRefusal = "sav: AgainromSeed requires version 1 with five dwords or version 2 with six"

func validateNativeSession(b []byte) error {
	if len(b) < 4 || nativeSessionLen[binary.LittleEndian.Uint32(b)] != len(b) {
		return errors.New(nativeSessionRefusal)
	}
	if mode := binary.LittleEndian.Uint32(b[12:]); mode > 1 {
		return fmt.Errorf("sav: AgainromSeed names random mode %d, which is not defined", mode)
	}
	return nil
}

// ReadNativeSession reads the optional session leaf.
func ReadNativeSession(state DocumentStateData) (NativeSession, bool, error) {
	b, found, err := nativeSessionLeaf.lookup(state)
	if err != nil || !found {
		return NativeSession{}, false, err
	}
	out := NativeSession{Seed: binary.LittleEndian.Uint64(b[4:]), Mode: binary.LittleEndian.Uint32(b[12:]),
		Shared: binary.LittleEndian.Uint32(b[16:])}
	if len(b) > 20 {
		out.Reseeds = binary.LittleEndian.Uint32(b[20:])
	}
	return out, true, nil
}

// SetNativeSession writes the session leaf, replacing any earlier one.
func SetNativeSession(state *DocumentStateData, value NativeSession) error {
	if state == nil || value.Mode > 1 {
		return fmt.Errorf("sav: the random session requires an application registry and a defined mode")
	}
	version := uint32(1)
	if value.Reseeds != 0 {
		version = 2
	}
	b := make([]byte, nativeSessionLen[version])
	binary.LittleEndian.PutUint32(b, version)
	binary.LittleEndian.PutUint64(b[4:], value.Seed)
	binary.LittleEndian.PutUint32(b[12:], value.Mode)
	binary.LittleEndian.PutUint32(b[16:], value.Shared)
	if version == 2 {
		binary.LittleEndian.PutUint32(b[20:], value.Reseeds)
	}
	return nativeSessionLeaf.store(state, b)
}
