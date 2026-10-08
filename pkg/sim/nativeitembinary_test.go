package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestNativeItemRecordColdHashCloneAndAtomicDecode(t *testing.T) {
	w := cbWorld(t, 17, cbEnt(0, 0, 0))
	old, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	history := NativeItemRecord{Token: SavedObjectToken{Position: [12]byte{0: 0xa5, 11: 0x5a}, Identity: 17, RuntimeID: 0xf1234567, Reference: 33, T0C: 9, T0E: 0x4321, T08: 0xfedcba98, T18: 0xa721}, F45: 21, F46: 31, F47: 41, F48: 0xabcd}
	item := PlainItem(0x201)
	item.NativeRecord = &history
	w.carried[0] = []ItemStack{StackItem(item, 3)}
	raw, err := w.MarshalBinary()
	if err != nil || binary.Size(history)+4 != nativeItemRecordLen {
		t.Fatal("bounded native Item layout", err)
	}
	if err := CheckSaveForm(raw); err != nil {
		t.Fatal("current Item form refused", err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() || !StackStateEqual(w.carried[0][0], cold.carried[0][0]) {
		t.Fatal("cold current token/service history lost", err)
	}
	clone := cold.carried[0][0].Clone()
	clone.NativeRecord.Token.T18++
	if clone.NativeRecord.Token.T18 == cold.carried[0][0].NativeRecord.Token.T18 {
		t.Fatal("clone aliases current token history")
	}
	before := cold.Hash()
	cold.carried[0][0].NativeRecord.Token.T18++
	if cold.Hash() == before || CanMergeItemValues(clone.Instance(), item) {
		t.Fatal("token history excluded from hash or discarded by a native merge")
	}
	start := len(raw) - 9 - int(binary.LittleEndian.Uint32(raw[len(raw)-9:]))
	for _, at := range []int{start, start + 4, len(raw) - 9} {
		bad := bytes.Clone(raw)
		bad[at] = 255
		if CheckSaveForm(bad) == nil {
			t.Fatal("malformed Item form admitted")
		}
		prior := cold.Hash()
		if cold.UnmarshalBinary(bad) == nil || cold.Hash() != prior {
			t.Fatal("malformed Item ordinal/span accepted or mutated state")
		}
	}
	if err := cold.UnmarshalBinary(old); err != nil || len(cold.carried[0]) != 0 {
		t.Fatal("old byte form invented current token history", err)
	}
}
