package terrain_test

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// A metadata mask bit admits its four slots and nothing else; a clear bit
// admits none (TERR-SLOTADMIT-166).
func TestAdmitFollowsTheMaskFourSlotsPerBit(t *testing.T) {
	full := terrain.LoadTileset(shippedSource())
	for bit := 0; bit < 32; bit++ {
		got := full.Admit(1 << bit)
		for i := 0; i < terrain.SlotCount; i++ {
			want := full.Slot(i) != nil && i>>2 == bit
			if (got.Slot(i) != nil) != want {
				t.Fatalf("mask bit %d: slot %d held=%v, want %v", bit, i, got.Slot(i) != nil, want)
			}
		}
	}
	if n := full.Admit(0).Loaded; n != 0 {
		t.Fatalf("mask 0 admits %d slots, want none", n)
	}
	if got := full.Admit(terrain.AllGroups); got.Loaded != full.Loaded {
		t.Fatalf("all-one mask admits %d slots, want the loaded %d", got.Loaded, full.Loaded)
	}
	if d := full.Admit(0).Dirt; d == nil || d != full.Dirt {
		t.Fatal("the dirt overlay is not governed by the mask")
	}
}

// A tileset bound again under a mask that excludes a slot it held clears the
// slot, and the set it came from keeps it.
func TestAdmitClearsASlotAReusedSetHeld(t *testing.T) {
	full := terrain.LoadTileset(shippedSource())
	first := full.Admit(1<<0 | 1<<1)
	if first.Slot(0) == nil || first.Slot(4) == nil {
		t.Fatal("first mask does not admit its slots")
	}
	second := first.Admit(1 << 1)
	for i := 0; i < 4; i++ {
		if second.Slot(i) != nil {
			t.Fatalf("slot %d survived a mask that excludes it", i)
		}
	}
	if second.Slot(4) == nil || second.Loaded != 4 {
		t.Fatalf("second mask holds %d slots, want group 1's four", second.Loaded)
	}
	if full.Slot(0) == nil || first.Slot(0) == nil {
		t.Fatal("binding a mask changed the set it came from")
	}
}
