package sav

import "bytes"

// MaxNativeMods bounds the payload of the mod mark.
const MaxNativeMods = 1 << 20

// NativeMods reads the optional mod mark payload.
func NativeMods(state DocumentStateData) ([]byte, bool, error) {
	value, found, err := nativeModsLeaf.lookupPayload(state)
	return bytes.Clone(value), found, err
}

// SetNativeMods writes the mod mark payload, replacing any earlier one.
func SetNativeMods(state *DocumentStateData, value []byte) error {
	return nativeModsLeaf.storePayload(state, value)
}

// ClearNativeMods removes the mod mark, so a document the current mod set did
// not touch carries none.
func ClearNativeMods(state *DocumentStateData) {
	nativeModsLeaf.remove(state)
}
